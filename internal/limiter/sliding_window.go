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

-- Use 1-second buckets for granularity
local bucket_size = 1000
local current_bucket = math.floor(now / bucket_size) * bucket_size
local window_start = now - window_size

-- Check if config has changed
local stored_config = redis.call('HGET', key, 'config_hash')
if stored_config and stored_config ~= config_hash then
    -- Config changed, clear all data
    redis.call('DEL', key)
    redis.call('HSET', key, 'config_hash', config_hash)
end

-- Ensure config hash is set
if not stored_config then
    redis.call('HSET', key, 'config_hash', config_hash)
end

-- Count requests in sliding window and clean up old buckets
local total_requests = 0
local oldest_request_time = now
local buckets_to_delete = {}

-- Get all fields (buckets)
local all_data = redis.call('HGETALL', key)
for i = 1, #all_data, 2 do
    local field = all_data[i]
    local value = tonumber(all_data[i + 1])
    
    -- Skip config_hash field
    if field ~= 'config_hash' then
        local bucket_time = tonumber(field)
        
        if bucket_time and bucket_time >= window_start then
            -- Bucket is within sliding window
            total_requests = total_requests + value
            if bucket_time < oldest_request_time then
                oldest_request_time = bucket_time
            end
        elseif bucket_time then
            -- Bucket is outside window, mark for deletion
            table.insert(buckets_to_delete, field)
        end
    end
end

-- Clean up old buckets
if #buckets_to_delete > 0 then
    redis.call('HDEL', key, unpack(buckets_to_delete))
end

-- Check if request can be allowed
if total_requests < limit then
    -- Increment current bucket
    redis.call('HINCRBY', key, tostring(current_bucket), 1)
    
    -- Set appropriate TTL
    redis.call('EXPIRE', key, math.ceil(window_size / 1000) + 60)
    
    -- Calculate remaining (note: this is approximate since we just added one)
    local remaining = limit - total_requests - 1
    return {1, math.max(0, remaining), 0}
else
    -- Request denied - calculate when oldest request will expire
    local retry_after = (oldest_request_time + window_size) - now
    return {0, 0, math.max(0, retry_after)}
end
`

func (rl *RateLimiter) SlidingWindowLimiter(ctx context.Context, key string, algoConfig config.AlgorithmConfig, configHash string) (*LimitResult, error) {
	if algoConfig.Limit == nil || algoConfig.WindowSize == nil {
		return &LimitResult{Allowed: true}, fmt.Errorf("sliding window requires limit and window_size")
	}

	if *algoConfig.Limit <= 0 || *algoConfig.WindowSize <= 0 {
		return &LimitResult{Allowed: true}, fmt.Errorf("sliding window parameters must be positive")
	}

	now := time.Now().UnixMilli()

	result := rl.redisClient.Eval(ctx, slidingWindowScript, []string{key},
		configHash, *algoConfig.Limit, *algoConfig.WindowSize, now)

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
