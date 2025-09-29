package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedWindowLimiter_BasicFunctionality(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:fixed:basic"
	configHash := "fixed_config_v1"

	limit := 5
	windowSize := &config.Duration{Duration: 1000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Limit:      &limit,
		WindowSize: windowSize,
	}

	// First request should be allowed
	result, err := rl.FixedWindowLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(4), result.Remaining) // 5 - 1 = 4
}

func TestFixedWindowLimiter_WindowExhaustion(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:fixed:exhaust"
	configHash := "fixed_config_v1"

	limit := 3
	windowSize := &config.Duration{Duration: 1000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Limit:      &limit,
		WindowSize: windowSize,
	}

	// Consume all requests in window
	for i := 0; i < 3; i++ {
		result, err := rl.FixedWindowLimiter(ctx, key, algoConfig, configHash)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, int64(3-i-1), result.Remaining)
	}

	// Next request should be denied
	result, err := rl.FixedWindowLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)
	assert.Greater(t, result.RetryAfter, time.Duration(0))
}

func TestFixedWindowLimiter_WindowReset(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:fixed:reset"
	configHash := "fixed_config_v1"

	limit := 2
	windowSize := &config.Duration{Duration: 200 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Limit:      &limit,
		WindowSize: windowSize,
	}

	// Use up the window
	for i := 0; i < 2; i++ {
		result, err := rl.FixedWindowLimiter(ctx, key, algoConfig, configHash)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Should be exhausted
	result, err := rl.FixedWindowLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Wait for window to reset
	time.Sleep(windowSize.Duration + 50*time.Millisecond)

	// Should be allowed again (new window)
	result, err = rl.FixedWindowLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(1), result.Remaining)
}

func TestFixedWindowLimiter_ConfigChange(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:fixed:config"

	// Initial config
	limit1 := 2
	windowSize1 := &config.Duration{Duration: 1000 * time.Millisecond}
	configHash1 := "fixed_config_v1"

	algoConfig1 := config.AlgorithmConfig{
		Limit:      &limit1,
		WindowSize: windowSize1,
	}

	// Use up window
	for i := 0; i < 2; i++ {
		result, err := rl.FixedWindowLimiter(ctx, key, algoConfig1, configHash1)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Should be exhausted
	result, err := rl.FixedWindowLimiter(ctx, key, algoConfig1, configHash1)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Change config
	limit2 := 5
	windowSize2 := &config.Duration{Duration: 2000 * time.Millisecond}
	configHash2 := "fixed_config_v2"

	algoConfig2 := config.AlgorithmConfig{
		Limit:      &limit2,
		WindowSize: windowSize2,
	}

	// Should reset and allow request
	result, err = rl.FixedWindowLimiter(ctx, key, algoConfig2, configHash2)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(4), result.Remaining) // 5 - 1 = 4
}

func TestFixedWindowLimiter_RedisError(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	mr.Close() // Close Redis to simulate error

	ctx := context.Background()
	key := "test:fixed:error"
	configHash := "config_v1"

	limit := 5
	windowSize := &config.Duration{Duration: 1000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Limit:      &limit,
		WindowSize: windowSize,
	}

	// Should return error and NIL result on system failure
	result, err := rl.FixedWindowLimiter(ctx, key, algoConfig, configHash)
	assert.Error(t, err)
	// FIX: Assert that the result pointer is NIL, as the Fail-Open logic is outside the Limiter function
	assert.Nil(t, result)
}

func BenchmarkFixedWindowLimiter(b *testing.B) {
	rl, mr := setupTestRateLimiter(nil)
	defer mr.Close()

	ctx := context.Background()
	key := "bench:fixed"
	configHash := "config_v1"

	limit := 1000000
	windowSize := &config.Duration{Duration: 60000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Limit:      &limit,
		WindowSize: windowSize,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := rl.FixedWindowLimiter(ctx, key, algoConfig, configHash)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
