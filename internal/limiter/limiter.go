package limiter

import (
	"context"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
)

type RateLimiter struct{}

type LimitResult struct {
	Allowed    bool
	Remaining  int64
	RetryAfter time.Duration
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{}
}

func (rl *RateLimiter) CheckGlobalLimit(ctx context.Context,
	globalConfig config.Global) (*LimitResult, error) {

	if !globalConfig.Enabled {
		return &LimitResult{Allowed: true}, nil
	}

	return &LimitResult{
		Allowed:    true,
		Remaining:  10,
		RetryAfter: time.Duration(time.Hour),
	}, nil
}

func (rl *RateLimiter) CheckTenantLimit(ctx context.Context, tenantKey string,
	tenantConfig config.PerTenant) (*LimitResult, error) {

	if !tenantConfig.Enabled {
		return &LimitResult{Allowed: true}, nil
	}

	return &LimitResult{
		Allowed:    true,
		Remaining:  10,
		RetryAfter: time.Duration(time.Hour),
	}, nil
}

func (rl *RateLimiter) CheckEndpointLimit(ctx context.Context, tenantKey string,
	endpointConfig config.EndpointRules) (*LimitResult, error) {

	return &LimitResult{
		Allowed:    true,
		Remaining:  10,
		RetryAfter: time.Duration(time.Hour),
	}, nil
}
