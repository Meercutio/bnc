package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/yourname/bulls-cows/internal/realtime"
)

// Redis keys
const (
	queueKey   = "mm:queue"
	inqueueKey = "mm:inqueue"
)

type Service struct {
	rdb      *redis.Client
	pub      *realtime.Publisher
	matchTTL time.Duration
}

func New(rdb *redis.Client, pub *realtime.Publisher) *Service {
	return &Service{
		rdb:      rdb,
		pub:      pub,
		matchTTL: 1 * time.Hour,
	}
}

// Lua: если пользователь уже в очереди -> "already"
// иначе пытаемся LPOP соперника из queue
// если никого нет -> RPUSH user, SADD inqueue, "queued"
// если соперник найден -> SREM inqueue opponent, "matched" + opponent
var startSearchLua = redis.NewScript(`
local user = ARGV[1]

if redis.call("SISMEMBER", KEYS[2], user) == 1 then
  return {"already", ""}
end

local opp = redis.call("LPOP", KEYS[1])
if not opp then
  redis.call("RPUSH", KEYS[1], user)
  redis.call("SADD", KEYS[2], user)
  return {"queued", ""}
end

redis.call("SREM", KEYS[2], opp)
return {"matched", opp}
`)

type StartResult struct {
	Status     string
	MatchID    string
	OpponentID string
}

// StartSearch : либо ставим в очередь, либо матчим сразу
func (s *Service) StartSearch(ctx context.Context, userID string) (StartResult, error) {
	// если уже есть матч - сразу ответим matched
	if mid, _ := s.rdb.Get(ctx, "mm:match:"+userID).Result(); mid != "" {
		opp, _ := s.rdb.Get(ctx, "mm:opp:"+mid+":"+userID).Result()
		return StartResult{Status: "matched", MatchID: mid, OpponentID: opp}, nil
	}

	res, err := startSearchLua.Run(ctx, s.rdb, []string{queueKey, inqueueKey}, userID).Result()
	if err != nil {
		return StartResult{}, err
	}

	arr := res.([]any)
	status := arr[0].(string)
	opp := arr[1].(string)

	switch status {
	case "queued":
		return StartResult{Status: "queued"}, nil
	case "already":
		return StartResult{Status: "already"}, nil
	case "matched":
		matchID := uuid.NewString()

		pipe := s.rdb.Pipeline()
		pipe.Set(ctx, "mm:match:"+userID, matchID, s.matchTTL)
		pipe.Set(ctx, "mm:match:"+opp, matchID, s.matchTTL)
		pipe.Set(ctx, "mm:opp:"+matchID+":"+userID, opp, s.matchTTL)
		pipe.Set(ctx, "mm:opp:"+matchID+":"+opp, userID, s.matchTTL)
		_, err := pipe.Exec(ctx)
		if err != nil {
			return StartResult{}, err
		}

		// publish events for both users
		ev1, _ := json.Marshal(map[string]any{
			"type": "match_found",
			"payload": map[string]any{
				"match_id":    matchID,
				"opponent_id": opp,
			},
		})
		ev2, _ := json.Marshal(map[string]any{
			"type": "match_found",
			"payload": map[string]any{
				"match_id":    matchID,
				"opponent_id": userID,
			},
		})

		_ = s.pub.PublishUser(ctx, userID, ev1)
		_ = s.pub.PublishUser(ctx, opp, ev2)

		// (опционально) матч-канал
		created, _ := json.Marshal(map[string]any{
			"type": "match_created",
			"payload": map[string]any{
				"match_id": matchID,
				"players":  []string{userID, opp},
			},
		})
		_ = s.pub.PublishMatch(ctx, matchID, created)

		return StartResult{Status: "matched", MatchID: matchID, OpponentID: opp}, nil
	default:
		return StartResult{Status: "idle"}, nil
	}
}

type CancelResult struct {
	Status  string
	MatchID string
}

func (s *Service) CancelSearch(ctx context.Context, userID string) (CancelResult, error) {
	// если уже матч — нельзя “отменить”
	if mid, _ := s.rdb.Get(ctx, "mm:match:"+userID).Result(); mid != "" {
		return CancelResult{Status: "matched", MatchID: mid}, nil
	}

	pipe := s.rdb.Pipeline()
	pipe.SRem(ctx, inqueueKey, userID)
	pipe.LRem(ctx, queueKey, 0, userID)
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return CancelResult{}, err
	}

	// грубо: если хотя бы одна операция что-то удалила — считаем canceled
	removed := false
	for _, c := range cmds {
		if ic, ok := c.(*redis.IntCmd); ok && ic.Val() > 0 {
			removed = true
		}
	}

	if removed {
		return CancelResult{Status: "canceled"}, nil
	}
	return CancelResult{Status: "not_in_queue"}, nil
}

type StatusResult struct {
	Status     string
	MatchID    string
	OpponentID string
}

func (s *Service) Status(ctx context.Context, userID string) (StatusResult, error) {
	if mid, _ := s.rdb.Get(ctx, "mm:match:"+userID).Result(); mid != "" {
		opp, _ := s.rdb.Get(ctx, "mm:opp:"+mid+":"+userID).Result()
		return StatusResult{Status: "matched", MatchID: mid, OpponentID: opp}, nil
	}

	inQ, err := s.rdb.SIsMember(ctx, inqueueKey, userID).Result()
	if err != nil {
		return StatusResult{}, err
	}
	if inQ {
		return StatusResult{Status: "searching"}, nil
	}
	return StatusResult{Status: "idle"}, nil
}
