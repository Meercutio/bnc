package realtime

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Registry struct {
	rdb *redis.Client
}

func NewRegistry(rdb *redis.Client) *Registry {
	return &Registry{rdb: rdb}
}

func (r *Registry) Add(ctx context.Context, userID, connID string) error {
	key := "ws:user:" + userID
	pipe := r.rdb.Pipeline()
	pipe.SAdd(ctx, key, connID)
	pipe.Expire(ctx, key, 24*time.Hour) // чтобы ключи не висели вечно
	_, err := pipe.Exec(ctx)
	return err
}

func (r *Registry) Remove(ctx context.Context, userID, connID string) error {
	key := "ws:user:" + userID
	return r.rdb.SRem(ctx, key, connID).Err()
}
