package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type RateRepo struct {
	client *goredis.Client
}

func NewRateRepo(client *goredis.Client) *RateRepo {
	return &RateRepo{client: client}
}

func (r *RateRepo) IncrementWindow(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	if r.client == nil {
		return 0, 0, fmt.Errorf("redis client is nil")
	}
	if key == "" || window <= 0 {
		return 0, 0, fmt.Errorf("invalid rate window payload")
	}

	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("increment rate key: %w", err)
	}
	if count == 1 {
		if err := r.client.Expire(ctx, key, window).Err(); err != nil {
			return 0, 0, fmt.Errorf("set rate key ttl: %w", err)
		}
	}

	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("read rate key ttl: %w", err)
	}
	if ttl < 0 {
		ttl = 0
	}

	return count, ttl, nil
}

func (r *RateRepo) WindowState(ctx context.Context, key string) (int64, time.Duration, error) {
	if r.client == nil {
		return 0, 0, fmt.Errorf("redis client is nil")
	}
	if key == "" {
		return 0, 0, fmt.Errorf("rate key is required")
	}

	count, err := r.client.Get(ctx, key).Int64()
	if err != nil && err != goredis.Nil {
		return 0, 0, fmt.Errorf("get rate key state: %w", err)
	}
	if err == goredis.Nil {
		return 0, 0, nil
	}

	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("read rate key ttl: %w", err)
	}
	if ttl < 0 {
		ttl = 0
	}

	return count, ttl, nil
}
