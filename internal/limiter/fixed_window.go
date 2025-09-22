package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
)

const fixedWindowScript = `
local key = KEYS[1]
local config_hash = ARGV[1]
local limit = tonumber(ARGV[2])
local window_size = tonumber(ARGV[3])
local now = tonumber(ARGV[4])

local window_start = math.floor(now / window_size) * window_size

-- Check if config has changed or if the window has rolled over
local bucket = redis.call('HMGET', key, 'count', 'window_start', 'config_hash')
local stored_count = tonumber(bucket[1]) or 0
local stored_window_start = tonumber(bucket[2]) or 0
local stored_config = bucket[3]

if stored_config and stored_config ~= config_hash then
    -- Config changed, reset the bucket
    redis.call('HMSET', key, 'count', 0, 'window_start', window_start, 'config_hash', config_hash)
    stored_count = 0
    stored_window_start = window_start
end

-- If the window has rolled over, reset the count for the new window.
if stored_window_start < window_start then
    redis.call('HMSET', key, 'count', 0, 'window_start', window_start, 'config_hash', config_hash)
    stored_count = 0
end

-- Get the correct count after potential resets
local current_count = tonumber(redis.call('HGET', key, 'count')) or 0

if current_count < limit then
    local new_count = redis.call('HINCRBY', key, 'count', 1)
    
    -- Ensure the window_start is set if it's the first request
    if stored_window_start == 0 then
        redis.call('HSET', key, 'window_start', window_start)
    end
    
    redis.call('EXPIRE', key, math.ceil(window_size / 1000) + 60)
    local remaining = limit - new_count
    return {1, remaining, 0}
else
    local window_end = stored_window_start + window_size
    local retry_after = window_end - now
    return {0, 0, retry_after}
end
`

func (rl *RateLimiter) FixedWindowLimiter(ctx context.Context, key string, algoConfig config.AlgorithmConfig, configHash string) (*LimitResult, error) {
	if algoConfig.Limit == nil || algoConfig.WindowSize == nil {
		return nil, fmt.Errorf("fixed window requires limit and window_size")
	}

	now := time.Now().UnixMilli()

	result := rl.redisClient.Eval(ctx, fixedWindowScript, []string{key},
		configHash, *algoConfig.Limit, *algoConfig.WindowSize, now)

	if result.Err() != nil {
		return nil, result.Err()
	}

	values, ok := result.Val().([]interface{})
	if !ok || len(values) != 3 {
		return nil, fmt.Errorf("unexpected response format")
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
