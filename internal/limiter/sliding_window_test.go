package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlidingWindowLimiter_BasicFunctionality(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:sliding:basic"
	configHash := "sliding_config_v1"

	limit := 5
	windowSize := &config.Duration{Duration: 5000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Limit:      &limit,
		WindowSize: windowSize,
	}

	// First request should be allowed
	result, err := rl.SlidingWindowLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(4), result.Remaining) // 5 - 1 = 4
}

func TestSlidingWindowLimiter_WindowExhaustion(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:sliding:exhaust"
	configHash := "sliding_config_v1"

	limit := 3
	windowSize := &config.Duration{Duration: 5000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Limit:      &limit,
		WindowSize: windowSize,
	}

	// Consume all requests
	for i := 0; i < 3; i++ {
		result, err := rl.SlidingWindowLimiter(ctx, key, algoConfig, configHash)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Next request should be denied
	result, err := rl.SlidingWindowLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)
	assert.Greater(t, result.RetryAfter, time.Duration(0))
}

func TestSlidingWindowLimiter_WindowSliding(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:sliding:slide"
	configHash := "sliding_config_v1"

	limit := 2
	windowSize := &config.Duration{Duration: 2000 * time.Millisecond} // 2 seconds

	algoConfig := config.AlgorithmConfig{
		Limit:      &limit,
		WindowSize: windowSize,
	}

	// Make first request
	result, err := rl.SlidingWindowLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Wait a bit, then make second request
	time.Sleep(500 * time.Millisecond)
	result, err = rl.SlidingWindowLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Should be at limit
	result, err = rl.SlidingWindowLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Wait for first request to slide out of window
	time.Sleep(1600 * time.Millisecond) // Total 2.1 seconds since first request

	// Should be allowed again
	result, err = rl.SlidingWindowLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestSlidingWindowLimiter_ConfigChange(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:sliding:config"

	// Initial config
	limit1 := 2
	windowSize1 := &config.Duration{Duration: 5000 * time.Millisecond}
	configHash1 := "sliding_config_v1"

	algoConfig1 := config.AlgorithmConfig{
		Limit:      &limit1,
		WindowSize: windowSize1,
	}

	// Use up limit
	for i := 0; i < 2; i++ {
		result, err := rl.SlidingWindowLimiter(ctx, key, algoConfig1, configHash1)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Should be exhausted
	result, err := rl.SlidingWindowLimiter(ctx, key, algoConfig1, configHash1)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Change config
	limit2 := 5
	windowSize2 := &config.Duration{Duration: 10000 * time.Millisecond}
	configHash2 := "sliding_config_v2"

	algoConfig2 := config.AlgorithmConfig{
		Limit:      &limit2,
		WindowSize: windowSize2,
	}

	// Should reset and allow request
	result, err = rl.SlidingWindowLimiter(ctx, key, algoConfig2, configHash2)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(4), result.Remaining) // 5 - 1 = 4
}

func TestSlidingWindowLimiter_RedisError(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	mr.Close() // Close Redis to simulate error

	ctx := context.Background()
	key := "test:sliding:error"
	configHash := "config_v1"

	limit := 5
	windowSize := &config.Duration{Duration: 1000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Limit:      &limit,
		WindowSize: windowSize,
	}

	// Should return error and NIL result on system failure
	result, err := rl.SlidingWindowLimiter(ctx, key, algoConfig, configHash)
	assert.Error(t, err)
	// FIX: Assert that the result pointer is NIL, as the Fail-Open logic is outside the Limiter function
	assert.Nil(t, result)
}

func BenchmarkSlidingWindowLimiter(b *testing.B) {
	rl, mr := setupTestRateLimiter(nil)
	defer mr.Close()

	ctx := context.Background()
	key := "bench:sliding"
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
			_, err := rl.SlidingWindowLimiter(ctx, key, algoConfig, configHash)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
