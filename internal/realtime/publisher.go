package realtime

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Publisher struct {
	rdb *redis.Client
}

func NewPublisher(rdb *redis.Client) *Publisher {
	return &Publisher{rdb: rdb}
}

func (p *Publisher) PublishUser(ctx context.Context, userID string, payload []byte) error {
	ch := "ws:user:" + userID
	return p.rdb.Publish(ctx, ch, payload).Err()
}

func (p *Publisher) PublishMatch(ctx context.Context, matchID string, payload []byte) error {
	ch := "ws:match:" + matchID
	return p.rdb.Publish(ctx, ch, payload).Err()
}
