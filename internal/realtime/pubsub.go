package realtime

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
)

type PubSub struct {
	rdb *redis.Client
	hub *Hub
}

func NewPubSub(rdb *redis.Client, hub *Hub) *PubSub {
	return &PubSub{rdb: rdb, hub: hub}
}

func (p *PubSub) Run(ctx context.Context) error {
	// Паттерн-подписка: слушаем все user и match каналы
	ps := p.rdb.PSubscribe(ctx, "ws:user:*", "ws:match:*")
	ch := ps.Channel()

	for {
		select {
		case <-ctx.Done():
			_ = ps.Close()
			return nil
		case msg, ok := <-ch:
			if !ok {
				return nil
			}

			// channel format:
			// ws:user:<userId>
			// ws:match:<matchId>
			if strings.HasPrefix(msg.Channel, "ws:user:") {
				userID := strings.TrimPrefix(msg.Channel, "ws:user:")
				p.hub.SendToUser(userID, []byte(msg.Payload))
				continue
			}
			if strings.HasPrefix(msg.Channel, "ws:match:") {
				matchID := strings.TrimPrefix(msg.Channel, "ws:match:")
				p.hub.SendToMatch(matchID, []byte(msg.Payload))
				continue
			}
		}
	}
}
