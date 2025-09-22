package limiter

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	redisClient *redis.Client
}

type LimitResult struct {
	Allowed    bool
	Remaining  int64
	RetryAfter time.Duration
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redisClient: redisClient,
	}
}

func (rl *RateLimiter) CheckGlobalLimit(ctx context.Context,
	globalConfig config.Global) (*LimitResult, error) {

	if !globalConfig.Enabled {
		return &LimitResult{Allowed: true}, nil
	}

	redisKey := constructRedisKey(config.GlobalLevel, "", []string{}, "")
	algoConfig := globalConfig.AlgorithmConfig
	configHash, err := generateConfigHash(algoConfig)
	if err != nil {
		return &LimitResult{Allowed: true}, fmt.Errorf("error generating config hash")
	}

	return rl.checkLimit(ctx, redisKey, algoConfig, configHash)
}

func (rl *RateLimiter) CheckTenantLimit(ctx context.Context, tenantKey string,
	tenantConfig config.PerTenant) (*LimitResult, error) {

	if !tenantConfig.Enabled {
		return &LimitResult{Allowed: true}, nil
	}

	redisKey := constructRedisKey(config.PerTenantLevel, "", []string{}, tenantKey)
	algoConfig := tenantConfig.AlgorithmConfig
	configHash, err := generateConfigHash(algoConfig)
	if err != nil {
		return &LimitResult{Allowed: true}, fmt.Errorf("error generating config hash")
	}

	return rl.checkLimit(ctx, redisKey, algoConfig, configHash)
}

func (rl *RateLimiter) CheckEndpointLimit(ctx context.Context, tenantKey string,
	endpointConfig config.EndpointRules) (*LimitResult, error) {

	methods := endpointConfig.Methods
	path := endpointConfig.Path
	redisKey := constructRedisKey(config.PerEndpointLevel, path, methods, tenantKey)
	algoConfig := endpointConfig.AlgorithmConfig
	configHash, err := generateConfigHash(algoConfig)
	if err != nil {
		return &LimitResult{Allowed: true}, fmt.Errorf("error generating config hash")
	}

	return rl.checkLimit(ctx, redisKey, algoConfig, configHash)
}

func (rl *RateLimiter) checkLimit(ctx context.Context, redisKey string,
	algoConfig config.AlgorithmConfig, configHash string) (*LimitResult, error) {
	switch algoConfig.Algorithm {
	case string(config.TokenBucket):
		return rl.TokenBucketLimiter(ctx, redisKey, algoConfig, configHash)
	case string(config.LeakyBucket):
		return rl.LeakyBucketLimiter(ctx, redisKey, algoConfig, configHash)
	case string(config.FixedWindow):
		return rl.FixedWindowLimiter(ctx, redisKey, algoConfig, configHash)
	case string(config.SlidingWindow):
		return rl.SlidingWindowLimiter(ctx, redisKey, algoConfig, configHash)
	default:
		return nil, fmt.Errorf("unknown rate limiting algorithm")
	}
}

func constructRedisKey(LevelType config.LimitLevelType, endpointPath string, endpointMethod []string,
	tenantKey string) string {
	prefix := "ctrl:limiter:"
	methodsString := strings.Join(endpointMethod, "_")

	switch LevelType {
	case config.GlobalLevel:
		//ctrl:limiter:global
		return fmt.Sprintf("%sglobal", prefix)
	case config.PerTenantLevel:
		//ctrl:limiter:pertenant:user123
		return fmt.Sprintf("%spertenant:%s", prefix, tenantKey)
	case config.PerEndpointLevel:
		//ctrl:limiter:perendpoint:GET_POST|/api/v2|user123
		return fmt.Sprintf("%sperendpoint:%s:%s:%s", prefix, methodsString, endpointPath, tenantKey)
	default:
		return ""
	}
}

func generateConfigHash(algoConfig config.AlgorithmConfig) (string, error) {
	data, err := json.Marshal(algoConfig)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}
