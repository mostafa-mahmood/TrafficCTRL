package limiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/metrics"
)

type Reputation struct {
	Score          float64
	ViolationCount int64
	GoodRequests   int64
	TTL            int64
}

const updateReputationScript = `
local reputation_key = KEYS[1]
local is_violation = tonumber(ARGV[1])
local now = tonumber(ARGV[2])

-- Get current reputation data
local rep_data = redis.call('HMGET', reputation_key, 'score', 'violation_count', 'good_requests')
local current_score = tonumber(rep_data[1]) or 1.0
local violation_count = tonumber(rep_data[2]) or 0
local good_requests = tonumber(rep_data[3]) or 0

if is_violation == 1 then
    -- Handle violation
    violation_count = violation_count + 1
    
    local violation_impact = math.min(0.1, 1.0 / (good_requests + 1))
    current_score = math.max(0.0, current_score - violation_impact)
    
    redis.call('HMSET', reputation_key, 
        'score', current_score,
        'violation_count', violation_count,
        'last_violation', now,
        'good_requests', good_requests)
else
    -- Handle good request
    good_requests = good_requests + 1
    
    if violation_count > 0 then
        local improvement = math.min(0.01, 0.5 / violation_count)
        current_score = math.min(1.0, current_score + improvement)
    end
    
    redis.call('HMSET', reputation_key,
        'score', current_score,
        'violation_count', violation_count,
        'good_requests', good_requests)
end

-- Decide TTL dynamically based on score
local ttl
if current_score < 0.3 then
    ttl = 7200   -- 2h for bad actors
elseif current_score < 0.7 then
    ttl = 3600   -- 1h for mid actors
else
    ttl = 1800   -- 30min for good actors
end

redis.call('EXPIRE', reputation_key, ttl)

return {
    math.floor(current_score * 1000) / 1000,
    violation_count,
    good_requests,
    ttl
}
`

func (rl *RateLimiter) UpdateReputation(ctx context.Context, tenantKey string, isViolation bool) (*Reputation, error) {
	reputationKey := fmt.Sprintf("ctrl:reputation:%s", tenantKey)

	now := time.Now().UnixMilli()
	violationFlag := 0
	if isViolation {
		violationFlag = 1
	}

	result := rl.redisClient.Eval(ctx, updateReputationScript,
		[]string{reputationKey},
		violationFlag, now)

	if result.Err() != nil {
		//==========================Metrics=======================
		metrics.RedisErrors.Inc()
		//========================================================
		return nil, result.Err()
	}

	values, ok := result.Val().([]interface{})
	if !ok || len(values) < 4 {
		//==========================Metrics=======================
		metrics.RedisErrors.Inc()
		//========================================================
		return nil, fmt.Errorf("unexpected result from lua script: %v", result.Val())
	}

	score, _ := strconv.ParseFloat(fmt.Sprint(values[0]), 64)
	violationCount, _ := strconv.ParseInt(fmt.Sprint(values[1]), 10, 64)
	goodRequests, _ := strconv.ParseInt(fmt.Sprint(values[2]), 10, 64)
	ttl, _ := strconv.ParseInt(fmt.Sprint(values[3]), 10, 64)

	return &Reputation{
		Score:          score,
		ViolationCount: violationCount,
		GoodRequests:   goodRequests,
		TTL:            ttl,
	}, nil
}

func (rl *RateLimiter) GetTenantReputation(ctx context.Context, tenantKey string) (*Reputation, error) {
	reputationKey := fmt.Sprintf("ctrl:reputation:%s", tenantKey)

	result := rl.redisClient.HMGet(ctx, reputationKey, "score", "violation_count", "good_requests")

	if result.Err() != nil {
		return &Reputation{Score: 1.0, TTL: 0}, nil
	}

	values := result.Val()

	score := 1.0
	if values[0] != nil {
		if s, err := strconv.ParseFloat(values[0].(string), 64); err == nil {
			score = s
		}
	}

	violationCount := int64(0)
	if values[1] != nil {
		if v, err := strconv.ParseInt(values[1].(string), 10, 64); err == nil {
			violationCount = v
		}
	}

	goodRequests := int64(0)
	if values[2] != nil {
		if g, err := strconv.ParseInt(values[2].(string), 10, 64); err == nil {
			goodRequests = g
		}
	}

	ttlCmd := rl.redisClient.TTL(ctx, reputationKey)
	ttlSeconds := int64(0)
	if err := ttlCmd.Err(); err == nil {
		ttlSeconds = int64(ttlCmd.Val().Seconds())
	}

	return &Reputation{
		Score:          score,
		ViolationCount: violationCount,
		GoodRequests:   goodRequests,
		TTL:            ttlSeconds,
	}, nil
}
