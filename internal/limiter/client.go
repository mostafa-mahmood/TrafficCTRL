package limiter

import (
	"context"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient(config config.RedisConfig) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Password: config.Password,
		DB:       config.DB,
		PoolSize: config.PoolSize,
	})
	return rdb
}

func (r *RateLimiter) Ping(ctx context.Context) error {
	_, err := r.redisClient.Ping(ctx).Result()
	return err
}
