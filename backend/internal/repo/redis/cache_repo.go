package redis

import "github.com/redis/go-redis/v9"

type CacheRepo struct {
	client *redis.Client
}

func NewCacheRepo(client *redis.Client) *CacheRepo {
	return &CacheRepo{client: client}
}
