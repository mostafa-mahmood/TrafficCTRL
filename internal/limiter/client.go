package limiter

import (
	"context"
	"crypto/tls"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient(config *config.RedisConfig) *redis.Client {
	opts := &redis.Options{
		Addr:     config.Address,
		Password: config.Password,
		DB:       config.DB,
		PoolSize: config.PoolSize,
	}

	if config.UseTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: config.TLSSkipVerify,
		}
	}

	return redis.NewClient(opts)
}

func (r *RateLimiter) Ping(ctx context.Context) error {
	_, err := r.redisClient.Ping(ctx).Result()
	return err
}
