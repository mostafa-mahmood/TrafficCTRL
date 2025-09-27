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

const improvedReputationScript = `
local reputation_key = KEYS[1]
local is_violation = tonumber(ARGV[1])
local now = tonumber(ARGV[2])

-- Get current reputation data
local rep_data = redis.call('HMGET', reputation_key, 'score', 'violation_count', 'good_requests', 'last_activity')
local current_score = tonumber(rep_data[1]) or 1.0
local violation_count = tonumber(rep_data[2]) or 0
local good_requests = tonumber(rep_data[3]) or 0
local last_activity = tonumber(rep_data[4]) or now

-- Time-based reputation decay for legitimate users caught in bot traffic
-- Only apply if no violations in last 10 minutes (600000ms) and score < 1.0
local time_since_last = now - last_activity
if violation_count == 0 and current_score < 1.0 and time_since_last > 600000 then
    -- Slow natural recovery for users with no violations
    local time_recovery = math.min(0.05, (time_since_last / 3600000) * 0.1) -- Max 0.05 per hour
    current_score = math.min(1.0, current_score + time_recovery)
end

if is_violation == 1 then
    -- Anti-bot violation handling
    violation_count = violation_count + 1
    
    -- Progressive punishment - gets worse with each violation
    local base_impact = math.max(0.05, math.min(0.15, 1.0 / (good_requests + 1)))
    
    -- Escalating punishment for repeat offenders (bot-like behavior)
    local escalation_factor = 1.0
    if violation_count >= 10 then
        escalation_factor = 2.0  -- Double punishment for persistent bots
    elseif violation_count >= 5 then
        escalation_factor = 1.5  -- 50% more punishment for suspicious behavior
    end
    
    local violation_impact = base_impact * escalation_factor
    current_score = math.max(0.0, current_score - violation_impact)
    
    -- Immediate severe punishment for rapid-fire violations (bot detection)
    -- If multiple violations within 1 second, assume bot behavior
    local last_violation = tonumber(redis.call('HGET', reputation_key, 'last_violation') or 0)
    if last_violation > 0 and (now - last_violation) < 1000 then
        current_score = math.max(0.0, current_score - 0.2) -- Extra 20% penalty
    end
    
    redis.call('HMSET', reputation_key, 
        'score', current_score,
        'violation_count', violation_count,
        'last_violation', now,
        'good_requests', good_requests,
        'last_activity', now)
else
    -- Handle good request
    good_requests = good_requests + 1
    
    -- Recovery system - slower for users with violations (anti-bot)
    if violation_count > 0 then
        -- Very slow recovery for violators to prevent bot adaptation
        local recovery_rate = 0.005  -- Base recovery rate (0.5%)
        
        -- Reduce recovery rate based on violation count (punish bots more)
        local violation_penalty = math.min(0.8, violation_count * 0.1)
        recovery_rate = recovery_rate * (1.0 - violation_penalty)
        
        -- Apply recovery
        local improvement = math.min(0.02, recovery_rate / math.sqrt(violation_count))
        current_score = math.min(1.0, current_score + improvement)
    else
        -- Fast recovery for clean users (likely legitimate users caught in traffic)
        if current_score < 1.0 then
            current_score = math.min(1.0, current_score + 0.02)
        end
    end
    
    redis.call('HMSET', reputation_key,
        'score', current_score,
        'violation_count', violation_count,
        'good_requests', good_requests,
        'last_activity', now)
end

-- Anti-bot TTL strategy
local ttl
if current_score < 0.1 then
    ttl = 14400   -- 4h for confirmed bots (very long monitoring)
elseif current_score < 0.3 then
    ttl = 7200    -- 2h for suspicious actors
elseif current_score < 0.7 then
    ttl = 3600    -- 1h for questionable actors
else
    ttl = 1800    -- 30min for good actors
end

-- Extend TTL for repeat offenders (bot-like patterns)
if violation_count >= 10 then
    ttl = ttl * 2  -- Double monitoring time for persistent violators
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

	result := rl.redisClient.Eval(ctx, improvedReputationScript,
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

func (rl *RateLimiter) GetReputationThreshold() float64 {
	return 0.3 // Block requests from users with reputation below 30%
}
