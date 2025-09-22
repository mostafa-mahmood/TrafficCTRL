package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
)

const slidingWindowScript = `
local key = KEYS[1]
local config_hash = ARGV[1]
local limit = tonumber(ARGV[2])
local window_size = tonumber(ARGV[3])
local now = tonumber(ARGV[4])

-- Use a small, fixed-size bucket for counting requests
local bucket_size = 1000 -- 1-second buckets
local current_bucket_start = math.floor(now / bucket_size) * bucket_size
local window_start = now - window_size

-- Check if config has changed
local stored_config = redis.call('HGET', key, 'config_hash')
if stored_config and stored_config ~= config_hash then
    -- Config changed, wipe all buckets and reset state
    redis.call('DEL', key)
    redis.call('HMSET', key, 'config_hash', config_hash)
end

-- Always ensure the current config hash is stored
redis.call('HSET', key, 'config_hash', config_hash)

local current_count = 0
local oldest_timestamp = now

-- Get all buckets within the hash map (excluding config_hash)
local buckets = redis.call('HGETALL', key)

if buckets then
    local fields_to_delete = {}
    for i = 1, #buckets, 2 do
        local field = buckets[i]
        if field ~= 'config_hash' then
            local bucket_time = tonumber(field)
            local bucket_count = tonumber(buckets[i+1])
            
            if bucket_time < window_start then
                -- Outside the window, mark for deletion
                table.insert(fields_to_delete, field)
            else
                -- Inside the window
                current_count = current_count + bucket_count
                if bucket_time < oldest_timestamp then
                    oldest_timestamp = bucket_time
                end
            end
        end
    end
    -- Clean up old buckets
    if #fields_to_delete > 0 then
        redis.call('HDEL', key, unpack(fields_to_delete))
    end
end

if current_count < limit then
    -- Allow: increment this bucket
    local new_count = redis.call('HINCRBY', key, tostring(current_bucket_start), 1)
    -- Update TTL dynamically
    redis.call('EXPIRE', key, math.ceil(window_size / 1000) + 60)

    local remaining = limit - new_count
    return {1, remaining, 0}
else
    -- Deny: calculate retry_after
    local retry_after = (oldest_timestamp + window_size) - now
    return {0, 0, retry_after}
end
`

func (rl *RateLimiter) SlidingWindowLimiter(ctx context.Context, key string, algoConfig config.AlgorithmConfig, configHash string) (*LimitResult, error) {
	if algoConfig.Limit == nil || algoConfig.WindowSize == nil {
		return &LimitResult{Allowed: true}, fmt.Errorf("sliding window requires limit and window_size")
	}

	now := time.Now().UnixMilli()

	result := rl.redisClient.Eval(ctx, slidingWindowScript, []string{key},
		configHash, *algoConfig.Limit, *algoConfig.WindowSize, now)

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
