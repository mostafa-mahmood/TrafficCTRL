package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/redis/go-redis/v9"
)

type RedisTokenBucketStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisTokenBucketStore(client *redis.Client) *RedisTokenBucketStore {
	return &RedisTokenBucketStore{
		client: client,
		ttl:    time.Hour,
	}
}

func (r *RedisTokenBucketStore) GetState(ctx context.Context, key string) (*limiter.TokenBucketState, error) {
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var state limiter.TokenBucketState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func (r *RedisTokenBucketStore) UpdateState(ctx context.Context, key string, state *limiter.TokenBucketState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, r.ttl).Err()
}
