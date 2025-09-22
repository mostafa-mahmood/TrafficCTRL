package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
)

const leakyBucketScript = `
local key = KEYS[1]
local config_hash = ARGV[1]
local capacity = tonumber(ARGV[2])
local leak_rate = tonumber(ARGV[3])
local leak_period = tonumber(ARGV[4])
local now = tonumber(ARGV[5])

-- Check if config has changed. If so, reset the bucket.
local stored_config = redis.call('HGET', key, 'config_hash')
if stored_config and stored_config ~= config_hash then
    -- Config changed, reset the bucket level to 1 (for the current request).
    redis.call('HMSET', key, 'level', 1, 'last_leak', now, 'config_hash', config_hash)
    -- Set the expiration dynamically based on the time it takes to empty.
    redis.call('EXPIRE', key, math.ceil((capacity / leak_rate) * leak_period / 1000) + 60)
    return {1, capacity - 1, 0}
end

-- Get the current state of the bucket. If the key doesn't exist, start with level 0.
local bucket = redis.call('HMGET', key, 'level', 'last_leak')
local level = tonumber(bucket[1]) or 0
local last_leak = tonumber(bucket[2]) or now

-- Calculate how much should leak based on time elapsed
local time_elapsed = now - last_leak
local periods_elapsed = math.floor(time_elapsed / leak_period)
local amount_to_leak = periods_elapsed * leak_rate

-- Update level after leaking, ensuring it doesn't go below 0.
level = math.max(0, level - amount_to_leak)
local new_last_leak = last_leak + (periods_elapsed * leak_period)

-- Check if request can be allowed (bucket has space)
if level < capacity then
    level = level + 1
    redis.call('HMSET', key, 'level', level, 'last_leak', new_last_leak, 'config_hash', config_hash)
    redis.call('EXPIRE', key, math.ceil((capacity / leak_rate) * leak_period / 1000) + 60)
    return {1, capacity - level, 0}
else
    redis.call('HMSET', key, 'level', level, 'last_leak', new_last_leak, 'config_hash', config_hash)
    redis.call('EXPIRE', key, math.ceil((capacity / leak_rate) * leak_period / 1000) + 60)
    local retry_after = (new_last_leak + leak_period) - now
    return {0, 0, retry_after}
end
`

func (rl *RateLimiter) LeakyBucketLimiter(ctx context.Context, key string, algoConfig config.AlgorithmConfig, configHash string) (*LimitResult, error) {
	if algoConfig.Capacity == nil || algoConfig.LeakRate == nil || algoConfig.LeakPeriod == nil {
		return &LimitResult{Allowed: true}, fmt.Errorf("leaky bucket requires capacity, leak_rate, and leak_period")
	}

	now := time.Now().UnixMilli()

	result := rl.redisClient.Eval(ctx, leakyBucketScript, []string{key},
		configHash, *algoConfig.Capacity, *algoConfig.LeakRate, *algoConfig.LeakPeriod, now)

	if result.Err() != nil {
		return &LimitResult{Allowed: true}, result.Err()
	}

	values, ok := result.Val().([]interface{})
	if !ok || len(values) != 3 {
		return &LimitResult{Allowed: true}, fmt.Errorf("unexpected response format")
	}

	allowed := values[0].(int64) == 1
	remaining := values[1].(int64)
	retryAfterMs := values[2].(int64)

	return &LimitResult{
		Allowed:    allowed,
		Remaining:  remaining,
		RetryAfter: time.Duration(retryAfterMs) * time.Millisecond,
	}, nil
}
