package limiter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
)

type RateLimiter struct {
	TokenBucketStore TokenBucketStore
}

type LimitResult struct {
	Allowed    bool
	Remaining  int64
	RetryAfter time.Duration
}

func NewRateLimiter(tokenBucketStore TokenBucketStore) *RateLimiter {
	return &RateLimiter{
		TokenBucketStore: tokenBucketStore,
	}
}

func (rl *RateLimiter) CheckGlobalLimit(ctx context.Context, globalConfig config.Global) (*LimitResult, error) {
	if !globalConfig.Enabled {
		return &LimitResult{Allowed: true}, nil
	}

	key := "ctrl:limiter:global"
	configHash := rl.generateConfigHash(globalConfig.AlgorithmConfig)

	return rl.checkLimit(ctx, key, globalConfig.AlgorithmConfig, configHash)
}

func (rl *RateLimiter) checkLimit(ctx context.Context, key string, algConfig config.AlgorithmConfig,
	configHash string) (*LimitResult, error) {
	algorithm := algConfig.Algorithm

	switch algorithm {
	case string(config.TokenBucket):
		return rl.TokenBucketLimiter(ctx, key, algConfig, configHash)
	default:
		return nil, fmt.Errorf("unknown algorithm: %s", algConfig.Algorithm)
	}
}

func (rl *RateLimiter) generateConfigHash(algConfig config.AlgorithmConfig) string {
	algorithm := config.AlgorithmType(algConfig.Algorithm)

	var configString string
	switch algorithm {
	case config.TokenBucket:
		configString = fmt.Sprintf("tb|%d|%d|%d",
			safeIntValue(algConfig.Capacity),
			safeIntValue(algConfig.RefillRate),
			safeIntValue(algConfig.RefillPeriod))
	case config.LeakyBucket:
		configString = fmt.Sprintf("lb|%d|%d|%d",
			safeIntValue(algConfig.Capacity),
			safeIntValue(algConfig.LeakRate),
			safeIntValue(algConfig.LeakPeriod))
	case config.FixedWindow:
		configString = fmt.Sprintf("fw|%d|%d",
			safeIntValue(algConfig.WindowSize),
			safeIntValue(algConfig.Limit))
	case config.SlidingWindow:
		configString = fmt.Sprintf("sw|%d|%d",
			safeIntValue(algConfig.WindowSize),
			safeIntValue(algConfig.Limit))
	default:
		configString = fmt.Sprintf("unknown|%s", algConfig.Algorithm)
	}

	hash := sha256.Sum256([]byte(configString))
	return hex.EncodeToString(hash[:8])
}

func safeIntValue(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}
