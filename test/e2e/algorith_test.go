package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/stretchr/testify/assert"
)

func (s *E2ETestSuite) TestAlgorithmBehaviorComparison() {
	t := s.T()

	tenantKey := "algorithm-test"

	results := make(map[string]int)

	t.Run("TokenBucket", func(t *testing.T) {
		allowed := 0
		algoConfig := config.AlgorithmConfig{
			Algorithm:    "token_bucket",
			Capacity:     intPtr(10),
			RefillRate:   intPtr(5),
			RefillPeriod: &config.Duration{Duration: time.Minute},
		}

		globalConfig := &config.Global{
			Enabled:         true,
			AlgorithmConfig: algoConfig,
		}

		for i := 0; i < 15; i++ {
			result, err := s.rateLimiter.CheckGlobalLimit(context.Background(), globalConfig)
			assert.NoError(t, err)

			if result.Allowed {
				allowed++
			}
		}
		results["token_bucket"] = allowed
	})

	t.Run("FixedWindow", func(t *testing.T) {
		allowed := 0
		algoConfig := config.AlgorithmConfig{
			Algorithm:  "fixed_window",
			WindowSize: &config.Duration{Duration: time.Minute},
			Limit:      intPtr(10),
		}

		globalConfig := &config.Global{
			Enabled:         true,
			AlgorithmConfig: algoConfig,
		}

		for i := 0; i < 15; i++ {
			result, err := s.rateLimiter.CheckGlobalLimit(context.Background(), globalConfig)
			assert.NoError(t, err)

			if result.Allowed {
				allowed++
			}
		}
		results["fixed_window"] = allowed
	})

	t.Run("SlidingWindow", func(t *testing.T) {
		allowed := 0
		algoConfig := config.AlgorithmConfig{
			Algorithm:  "sliding_window",
			WindowSize: &config.Duration{Duration: time.Minute},
			Limit:      intPtr(10),
		}

		globalConfig := &config.Global{
			Enabled:         true,
			AlgorithmConfig: algoConfig,
		}

		for i := 0; i < 15; i++ {
			result, err := s.rateLimiter.CheckGlobalLimit(context.Background(), globalConfig)
			assert.NoError(t, err)

			if result.Allowed {
				allowed++
			}
		}
		results["sliding_window"] = allowed
	})

	t.Run("LeakyBucket", func(t *testing.T) {
		allowed := 0
		algoConfig := config.AlgorithmConfig{
			Algorithm:  "leaky_bucket",
			Capacity:   intPtr(10),
			LeakRate:   intPtr(5),
			LeakPeriod: &config.Duration{Duration: time.Minute},
		}

		tenantConfig := &config.PerTenant{
			Enabled:         true,
			AlgorithmConfig: algoConfig,
		}

		for i := 0; i < 15; i++ {
			result, err := s.rateLimiter.CheckTenantLimit(context.Background(), tenantKey, tenantConfig)
			assert.NoError(t, err)

			if result.Allowed {
				allowed++
			}
		}
		results["leaky_bucket"] = allowed
	})

	fmt.Printf("Algorithm comparison - allowed requests in burst:\n")
	for algo, count := range results {
		fmt.Printf("  %s: %d\n", algo, count)
	}

	assert.Equal(t, 10, results["token_bucket"], "Token bucket should allow exactly capacity in immediate burst")
	assert.True(t, results["fixed_window"] <= 10, "Fixed window should allow up to limit")
	assert.True(t, results["sliding_window"] <= 10, "Sliding window should allow up to limit")
	assert.True(t, results["leaky_bucket"] <= 10, "Leaky bucket should allow up to capacity")
}
