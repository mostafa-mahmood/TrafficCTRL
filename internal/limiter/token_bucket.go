package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
)

const tokenBucketScript = `
local key = KEYS[1]
local config_hash = ARGV[1]
local capacity = tonumber(ARGV[2])
local refill_rate = tonumber(ARGV[3])
local refill_period = tonumber(ARGV[4])
local now = tonumber(ARGV[5])

-- Check if config has changed by comparing the hash
local stored_config = redis.call('HGET', key, 'config_hash')

if stored_config and stored_config ~= config_hash then
    -- Config changed, reset the bucket
    redis.call('HMSET', key, 'tokens', capacity, 'last_refill', now, 'config_hash', config_hash)
    redis.call('EXPIRE', key, math.ceil((capacity / refill_rate) * (refill_period / 1000)) + 60)
    
    -- Now consume a token for this request
    if capacity >= 1 then
        redis.call('HSET', key, 'tokens', capacity - 1)
        return {1, capacity - 1, 0}
    else
        return {0, 0, refill_period}
    end
end

-- Get current bucket state
local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
local current_tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

-- Initialize if this is the first request
if current_tokens == nil or last_refill == nil then
    current_tokens = capacity
    last_refill = now
    redis.call('HMSET', key, 'tokens', current_tokens, 'last_refill', last_refill, 'config_hash', config_hash)
    redis.call('EXPIRE', key, math.ceil((capacity / refill_rate) * (refill_period / 1000)) + 60)
end

-- Calculate tokens to add based on elapsed time
local time_elapsed = now - last_refill
if time_elapsed > 0 then
    local periods_elapsed = math.floor(time_elapsed / refill_period)
    if periods_elapsed > 0 then
        local tokens_to_add = periods_elapsed * refill_rate
        current_tokens = math.min(capacity, current_tokens + tokens_to_add)
        last_refill = last_refill + (periods_elapsed * refill_period)
    end
end

-- Try to consume a token
if current_tokens >= 1 then
    -- Consume token
    current_tokens = current_tokens - 1
    
    -- Update state
    redis.call('HMSET', key, 'tokens', current_tokens, 'last_refill', last_refill, 'config_hash', config_hash)
    redis.call('EXPIRE', key, math.ceil((capacity / refill_rate) * (refill_period / 1000)) + 60)
    
    return {1, current_tokens, 0}
else
    -- No tokens available
    redis.call('HMSET', key, 'tokens', current_tokens, 'last_refill', last_refill, 'config_hash', config_hash)
    redis.call('EXPIRE', key, math.ceil((capacity / refill_rate) * (refill_period / 1000)) + 60)
    
    -- Calculate retry after time
    local next_refill = last_refill + refill_period
    local retry_after = math.max(0, next_refill - now)
    
    return {0, current_tokens, retry_after}
end
`

func (rl *RateLimiter) TokenBucketLimiter(ctx context.Context, key string, algoConfig config.AlgorithmConfig,
	configHash string) (*LimitResult, error) {
	if algoConfig.Capacity == nil || algoConfig.RefillRate == nil || algoConfig.RefillPeriod == nil {
		return &LimitResult{Allowed: true}, fmt.Errorf("token bucket requires capacity, refill_rate, and refill_period")
	}

	if *algoConfig.Capacity <= 0 || *algoConfig.RefillRate <= 0 || algoConfig.RefillPeriod.Duration <= 0 {
		return &LimitResult{Allowed: true}, fmt.Errorf("token bucket parameters must be positive")
	}

	now := time.Now().UnixMilli()

	result := rl.redisClient.Eval(ctx, tokenBucketScript, []string{key},
		configHash, *algoConfig.Capacity, *algoConfig.RefillRate, algoConfig.RefillPeriod.Milliseconds(), now)

	if result.Err() != nil {
		return &LimitResult{Allowed: true}, result.Err()
	}

	values, ok := result.Val().([]interface{})
	if !ok || len(values) != 3 {
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
