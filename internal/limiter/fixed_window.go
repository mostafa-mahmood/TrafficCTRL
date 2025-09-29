package limiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/metrics"
)

const fixedWindowScript = `
local key = KEYS[1]
local config_hash = ARGV[1]
local limit = tonumber(ARGV[2])
local window_size = tonumber(ARGV[3])
local now = tonumber(ARGV[4])

-- Calculate current window start
local window_start = math.floor(now / window_size) * window_size

-- Get current state
local bucket = redis.call('HMGET', key, 'count', 'window_start', 'config_hash')
local current_count = tonumber(bucket[1]) or 0
local stored_window_start = tonumber(bucket[2]) or 0
local stored_config = bucket[3]

-- Check if config changed
if stored_config and stored_config ~= config_hash then
    -- Reset everything for new config
    current_count = 0
    stored_window_start = window_start
    redis.call('HMSET', key, 'count', 0, 'window_start', window_start, 'config_hash', config_hash)
end

-- Check if window has rolled over (new window)
if stored_window_start < window_start then
    current_count = 0
    stored_window_start = window_start
    redis.call('HMSET', key, 'count', 0, 'window_start', window_start, 'config_hash', config_hash)
end

-- Initialize if this is the first request
if stored_window_start == 0 then
    stored_window_start = window_start
    redis.call('HSET', key, 'window_start', window_start)
    if not stored_config then
        redis.call('HSET', key, 'config_hash', config_hash)
    end
end

-- Check if request can be allowed
if current_count < limit then
    -- Increment count and allow request
    local new_count = redis.call('HINCRBY', key, 'count', 1)
    
    -- Set expiration to window end + buffer
    local window_end = stored_window_start + window_size
    local ttl_seconds = math.ceil((window_end - now) / 1000) + 60
    redis.call('EXPIRE', key, ttl_seconds)
    
    return {1, limit - new_count, 0}
else
    -- Request denied - calculate retry after
    local window_end = stored_window_start + window_size
    local retry_after = window_end - now
    
    return {0, 0, math.max(0, retry_after)}
end
`

func (rl *RateLimiter) FixedWindowLimiter(ctx context.Context, key string,
	algoConfig config.AlgorithmConfig, configHash string) (*LimitResult, error) {
	now := time.Now().UnixMilli()

	result := rl.redisClient.Eval(ctx, fixedWindowScript, []string{key},
		configHash, *algoConfig.Limit, algoConfig.WindowSize.Milliseconds(), now)

	if result.Err() != nil {
		//==========================Metrics=======================
		metrics.RedisErrors.Inc()
		//========================================================
		return nil, result.Err()
	}

	values, ok := result.Val().([]interface{})
	if !ok || len(values) != 3 {
		//==========================Metrics=======================
		metrics.RedisErrors.Inc()
		//========================================================
		return nil, fmt.Errorf("unexpected response format from Redis script")
	}

	allowedInt, _ := strconv.ParseInt(fmt.Sprint(values[0]), 10, 64)
	remaining, _ := strconv.ParseInt(fmt.Sprint(values[1]), 10, 64)
	retryAfterMs, _ := strconv.ParseInt(fmt.Sprint(values[2]), 10, 64)

	return &LimitResult{
		Allowed:    allowedInt == 1,
		Remaining:  remaining,
		RetryAfter: time.Duration(retryAfterMs) * time.Millisecond,
	}, nil
}
