package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
)

type TokenBucketStore interface {
	GetState(ctx context.Context, key string) (*TokenBucketState, error)
	UpdateState(ctx context.Context, key string, state *TokenBucketState) error
}

type TokenBucketState struct {
	Tokens     int64     `json:"tokens"`
	LastRefill time.Time `json:"last_refill"`
	ConfigHash string    `json:"config_hash"`
}

func (rl *RateLimiter) TokenBucketLimiter(ctx context.Context, key string, algConfig config.AlgorithmConfig, configHash string) (*LimitResult, error) {
	if algConfig.Capacity == nil || algConfig.RefillRate == nil || algConfig.RefillPeriod == nil {
		return nil, fmt.Errorf("token bucket requires capacity, refill_rate, and refill_period")
	}

	capacity := int64(*algConfig.Capacity)
	refillRate := int64(*algConfig.RefillRate)
	refillPeriod := time.Duration(*algConfig.RefillPeriod) * time.Second

	state, err := rl.TokenBucketStore.GetState(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get token bucket state: %w", err)
	}

	if state == nil || state.ConfigHash != configHash {
		state = &TokenBucketState{
			Tokens:     capacity,
			LastRefill: time.Now(),
			ConfigHash: configHash,
		}
	}

	now := time.Now()
	timePassed := now.Sub(state.LastRefill)

	refillPeriodsPassed := timePassed / refillPeriod
	tokensToAdd := int64(refillPeriodsPassed) * refillRate

	if tokensToAdd > 0 {
		state.Tokens = min(capacity, state.Tokens+tokensToAdd)
		state.LastRefill = state.LastRefill.Add(refillPeriodsPassed * refillPeriod)
	}

	if state.Tokens >= 1 {
		state.Tokens--

		if err := rl.TokenBucketStore.UpdateState(ctx, key, state); err != nil {
			return nil, fmt.Errorf("failed to update token bucket state: %w", err)
		}

		return &LimitResult{
			Allowed:    true,
			Remaining:  state.Tokens,
			RetryAfter: 0,
		}, nil
	}

	timeSinceLastRefill := now.Sub(state.LastRefill)
	timeUntilNextRefill := refillPeriod - (timeSinceLastRefill % refillPeriod)

	if err := rl.TokenBucketStore.UpdateState(ctx, key, state); err != nil {
		return nil, fmt.Errorf("failed to update token bucket state: %w", err)
	}

	return &LimitResult{
		Allowed:    false,
		Remaining:  0,
		RetryAfter: timeUntilNextRefill,
	}, nil
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
