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



-- Check if config has changed by comparing the hash. If so, reset the bucket.

local stored_config = redis.call('HGET', key, 'config_hash')

if stored_config and stored_config ~= config_hash then

-- Config changed, reset the bucket to capacity - 1 and consume one token.

redis.call('HMSET', key, 'tokens', capacity - 1, 'last_refill', now, 'config_hash', config_hash)

-- Set the expiration dynamically based on the refill time plus a buffer.

redis.call('EXPIRE', key, math.ceil((capacity / refill_rate) * refill_period / 1000) + 60)

return {1, capacity - 1, 0}

end



-- Get the current state of the bucket. If the key doesn't exist, initialize with a full bucket.

local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')

local tokens = tonumber(bucket[1]) or capacity

local last_refill = tonumber(bucket[2]) or now



-- Calculate the number of tokens to add based on elapsed time

local time_elapsed = now - last_refill

local periods_elapsed = math.floor(time_elapsed / refill_period)

local tokens_to_add = periods_elapsed * refill_rate



-- Update tokens, but do not exceed capacity.

tokens = math.min(capacity, tokens + tokens_to_add)



-- Calculate the timestamp of the last actual refill.

local new_last_refill = last_refill + (periods_elapsed * refill_period)



-- Check if a token can be consumed

if tokens >= 1 then

-- Request allowed: consume one token and update the bucket state.

tokens = tokens - 1

redis.call('HMSET', key, 'tokens', tokens, 'last_refill', new_last_refill, 'config_hash', config_hash)

-- Set the expiration dynamically based on the refill time plus a buffer.

redis.call('EXPIRE', key, math.ceil((capacity / refill_rate) * refill_period / 1000) + 60)

return {1, tokens, 0}

else

-- Request denied: update the bucket state (in case a refill occurred)

redis.call('HMSET', key, 'tokens', tokens, 'last_refill', new_last_refill, 'config_hash', config_hash)

-- Set the expiration dynamically based on the refill time plus a buffer.

redis.call('EXPIRE', key, math.ceil((capacity / refill_rate) * refill_period / 1000) + 60)

-- Calculate the time until the next token is available.

local retry_after = (new_last_refill + refill_period) - now

return {0, tokens, retry_after}

end

`

func (rl *RateLimiter) TokenBucketLimiter(ctx context.Context, key string, algoConfig config.AlgorithmConfig,
	configHash string) (*LimitResult, error) {

	if algoConfig.Capacity == nil || algoConfig.RefillRate == nil || algoConfig.RefillPeriod == nil {
		return &LimitResult{Allowed: true}, fmt.Errorf("token bucket requires capacity, refill_rate, and refill_period")
	}

	now := time.Now().UnixMilli()

	result := rl.redisClient.Eval(ctx, tokenBucketScript, []string{key},
		configHash, *algoConfig.Capacity, *algoConfig.RefillRate, *algoConfig.RefillPeriod, now)

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
