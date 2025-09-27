package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/metrics"
)

const leakyBucketScript = `
local key = KEYS[1]
local config_hash = ARGV[1]
local capacity = tonumber(ARGV[2])
local leak_rate = tonumber(ARGV[3])
local leak_period = tonumber(ARGV[4])
local now = tonumber(ARGV[5])

-- Check if config has changed
local stored_config = redis.call('HGET', key, 'config_hash')
if stored_config and stored_config ~= config_hash then
    -- Config changed, reset the bucket to empty and add current request
    redis.call('HMSET', key, 'level', 1, 'last_leak', now, 'config_hash', config_hash)
    redis.call('EXPIRE', key, math.ceil((capacity / leak_rate) * (leak_period / 1000)) + 60)
    return {1, capacity - 1, 0}
end

-- Get current bucket state
local bucket = redis.call('HMGET', key, 'level', 'last_leak')
local current_level = tonumber(bucket[1])
local last_leak = tonumber(bucket[2])

-- Initialize if this is the first request
if current_level == nil or last_leak == nil then
    current_level = 0
    last_leak = now
    redis.call('HMSET', key, 'level', current_level, 'last_leak', last_leak, 'config_hash', config_hash)
    redis.call('EXPIRE', key, math.ceil((capacity / leak_rate) * (leak_period / 1000)) + 60)
end

-- Calculate leaking based on elapsed time
local time_elapsed = now - last_leak
if time_elapsed > 0 then
    local periods_elapsed = math.floor(time_elapsed / leak_period)
    if periods_elapsed > 0 then
        local amount_to_leak = periods_elapsed * leak_rate
        current_level = math.max(0, current_level - amount_to_leak)
        last_leak = last_leak + (periods_elapsed * leak_period)
    end
end

-- Check if we can add the request (bucket has space)
if current_level < capacity then
    -- Add request to bucket
    current_level = current_level + 1
    
    -- Update state
    redis.call('HMSET', key, 'level', current_level, 'last_leak', last_leak, 'config_hash', config_hash)
    redis.call('EXPIRE', key, math.ceil((capacity / leak_rate) * (leak_period / 1000)) + 60)
    
    -- Return remaining capacity
    return {1, capacity - current_level, 0}
else
    -- Bucket is full, request rejected
    redis.call('HMSET', key, 'level', current_level, 'last_leak', last_leak, 'config_hash', config_hash)
    redis.call('EXPIRE', key, math.ceil((capacity / leak_rate) * (leak_period / 1000)) + 60)
    
    -- Calculate when space will be available
    -- We need to wait for at least one item to leak out
    local next_leak = last_leak + leak_period
    local retry_after = math.max(0, next_leak - now)
    
    return {0, 0, retry_after}
end
`

func (rl *RateLimiter) LeakyBucketLimiter(ctx context.Context, key string,
	algoConfig config.AlgorithmConfig, configHash string) (*LimitResult, error) {
	now := time.Now().UnixMilli()

	result := rl.redisClient.Eval(ctx, leakyBucketScript, []string{key},
		configHash, *algoConfig.Capacity, *algoConfig.LeakRate, algoConfig.LeakPeriod.Milliseconds(), now)

	if result.Err() != nil {
		//==========================Metrics=======================
		metrics.RedisErrors.Inc()
		//========================================================
		return &LimitResult{Allowed: true}, result.Err()
	}

	values, ok := result.Val().([]interface{})
	if !ok || len(values) != 3 {
		//==========================Metrics=======================
		metrics.RedisErrors.Inc()
		//========================================================
		return &LimitResult{Allowed: true}, fmt.Errorf("unexpected response format from Redis script")
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
