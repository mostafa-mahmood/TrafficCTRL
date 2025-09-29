package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenBucketLimiter_BasicFunctionality(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:user:123"
	configHash := "config_hash_v1"

	capacity := 10
	refillRate := 2
	refillPeriod := &config.Duration{Duration: 1000 * time.Millisecond} // 1 second

	algoConfig := config.AlgorithmConfig{
		Capacity:     &capacity,
		RefillRate:   &refillRate,
		RefillPeriod: refillPeriod,
	}

	// First request should be allowed (bucket starts full)
	result, err := rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(9), result.Remaining) // 10 - 1 = 9
	assert.Equal(t, time.Duration(0), result.RetryAfter)
}

func TestTokenBucketLimiter_BucketExhaustion(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:user:exhaustion"
	configHash := "config_hash_v1"

	capacity := 3
	refillRate := 1
	refillPeriod := &config.Duration{Duration: 1000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Capacity:     &capacity,
		RefillRate:   &refillRate,
		RefillPeriod: refillPeriod,
	}

	// Consume all tokens
	for i := int64(3); i > 0; i-- {
		result, err := rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, i-1, result.Remaining)
	}

	// Next request should be denied
	result, err := rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)
	assert.Greater(t, result.RetryAfter, time.Duration(0))
	assert.LessOrEqual(t, result.RetryAfter, refillPeriod.Duration)
}

func TestTokenBucketLimiter_TokenRefill(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:user:refill"
	configHash := "config_hash_v1"

	capacity := 5
	refillRate := 2
	refillPeriod := &config.Duration{Duration: 100 * time.Millisecond} // 100ms for faster testing

	algoConfig := config.AlgorithmConfig{
		Capacity:     &capacity,
		RefillRate:   &refillRate,
		RefillPeriod: refillPeriod,
	}

	// Consume all tokens
	for i := 0; i < 5; i++ {
		result, err := rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Verify bucket is empty
	result, err := rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)

	// Wait for one refill period
	time.Sleep(refillPeriod.Duration + 10*time.Millisecond) // Add small buffer for timing

	// Should have 2 tokens now (refill rate)
	result, err = rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(1), result.Remaining) // 2 refilled - 1 consumed = 1

	// Consume the other token
	result, err = rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)

	// Should be denied again
	result, err = rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
}

func TestTokenBucketLimiter_ConfigChange(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:user:config_change"

	// Initial config
	capacity1 := 2
	refillRate1 := 1
	refillPeriod1 := &config.Duration{Duration: 1000 * time.Millisecond}
	configHash1 := "config_hash_v1"

	algoConfig1 := config.AlgorithmConfig{
		Capacity:     &capacity1,
		RefillRate:   &refillRate1,
		RefillPeriod: refillPeriod1,
	}

	// Consume both tokens
	result, err := rl.TokenBucketLimiter(ctx, key, algoConfig1, configHash1)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(1), result.Remaining)

	result, err = rl.TokenBucketLimiter(ctx, key, algoConfig1, configHash1)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)

	// Verify bucket is empty
	result, err = rl.TokenBucketLimiter(ctx, key, algoConfig1, configHash1)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Change config (increase capacity)
	capacity2 := 5
	refillRate2 := 2
	refillPeriod2 := &config.Duration{Duration: 1000 * time.Millisecond}
	configHash2 := "config_hash_v2"

	algoConfig2 := config.AlgorithmConfig{
		Capacity:     &capacity2,
		RefillRate:   &refillRate2,
		RefillPeriod: refillPeriod2,
	}

	// Should reset to new capacity and allow request
	result, err = rl.TokenBucketLimiter(ctx, key, algoConfig2, configHash2)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(4), result.Remaining) // New capacity 5 - 1 consumed = 4
}

func TestTokenBucketLimiter_MultipleRefillPeriods(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:user:multiple_refill"
	configHash := "config_hash_v1"

	capacity := 10
	refillRate := 2
	refillPeriod := &config.Duration{Duration: 50 * time.Millisecond} // Very short for testing

	algoConfig := config.AlgorithmConfig{
		Capacity:     &capacity,
		RefillRate:   &refillRate,
		RefillPeriod: refillPeriod,
	}

	// Consume all tokens
	for i := 0; i < 10; i++ {
		result, err := rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Wait for 3 refill periods (should add 6 tokens total)
	time.Sleep(3*refillPeriod.Duration + 10*time.Millisecond)

	// Should have 6 tokens available
	result, err := rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(5), result.Remaining) // 6 - 1 = 5
}

func TestTokenBucketLimiter_CapacityLimit(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:user:capacity_limit"
	configHash := "config_hash_v1"

	capacity := int(3)
	refillRate := int(10)                                             // High refill rate
	refillPeriod := &config.Duration{Duration: 10 * time.Millisecond} // Very short period

	algoConfig := config.AlgorithmConfig{
		Capacity:     &capacity,
		RefillRate:   &refillRate,
		RefillPeriod: refillPeriod,
	}

	// Initial state - should have full capacity
	result, err := rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(2), result.Remaining)

	// Wait for multiple refill periods - tokens should not exceed capacity
	time.Sleep(100 * time.Millisecond) // Much longer than refill period

	// Should still be limited by capacity
	result, err = rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(2), result.Remaining) // Should be at capacity (3) - 1 = 2
}

func TestTokenBucketLimiter_ZeroCapacity(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:user:zero_capacity"
	configHash := "config_hash_v1"

	capacity := 0
	refillRate := 1
	refillPeriod := &config.Duration{Duration: 1000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Capacity:     &capacity,
		RefillRate:   &refillRate,
		RefillPeriod: refillPeriod,
	}

	// Should be denied immediately
	result, err := rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)
	assert.Greater(t, result.RetryAfter, time.Duration(0))
}

func TestTokenBucketLimiter_DifferentKeys(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	configHash := "config_hash_v1"

	capacity := 2
	refillRate := 1
	refillPeriod := &config.Duration{Duration: 1000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Capacity:     &capacity,
		RefillRate:   &refillRate,
		RefillPeriod: refillPeriod,
	}

	// Different keys should have independent buckets
	key1 := "test:user:123"
	key2 := "test:user:456"

	// Exhaust key1 bucket
	for i := 0; i < 2; i++ {
		result, err := rl.TokenBucketLimiter(ctx, key1, algoConfig, configHash)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Key1 should be exhausted
	result, err := rl.TokenBucketLimiter(ctx, key1, algoConfig, configHash)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Key2 should still have full bucket
	result, err = rl.TokenBucketLimiter(ctx, key2, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(1), result.Remaining)
}

func TestTokenBucketLimiter_RedisError(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	mr.Close() // Close Redis to simulate error

	ctx := context.Background()
	key := "test:user:redis_error"
	configHash := "config_hash_v1"

	capacity := 10
	refillRate := 2
	refillPeriod := &config.Duration{Duration: 1000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Capacity:     &capacity,
		RefillRate:   &refillRate,
		RefillPeriod: refillPeriod,
	}

	// Should return error and NIL result on system failure
	result, err := rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
	assert.Error(t, err)
	// FIX: Assert that the result pointer is NIL, as the Fail-Open logic is outside the Limiter function
	assert.Nil(t, result)
}

func TestTokenBucketLimiter_ConcurrentAccess(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:user:concurrent"
	configHash := "config_hash_v1"

	capacity := 100
	refillRate := 10
	refillPeriod := &config.Duration{Duration: 100 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Capacity:     &capacity,
		RefillRate:   &refillRate,
		RefillPeriod: refillPeriod,
	}

	// Run concurrent requests
	numGoroutines := 50
	results := make(chan *LimitResult, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			result, err := rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}()
	}

	// Collect results
	allowedCount := 0
	deniedCount := 0

	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			if result.Allowed {
				allowedCount++
			} else {
				deniedCount++
			}
		case err := <-errors:
			t.Fatalf("Unexpected error: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	// Should not exceed capacity
	assert.LessOrEqual(t, allowedCount, int(capacity))
	assert.Equal(t, numGoroutines, allowedCount+deniedCount)
}

// Benchmark tests
func BenchmarkTokenBucketLimiter(b *testing.B) {
	rl, mr := setupTestRateLimiterleaky(nil)
	defer mr.Close()

	ctx := context.Background()
	key := "bench:user:123"
	configHash := "config_hash_v1"

	capacity := 1000000 // Large capacity for benchmarking
	refillRate := 1000
	refillPeriod := &config.Duration{Duration: 1000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Capacity:     &capacity,
		RefillRate:   &refillRate,
		RefillPeriod: refillPeriod,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := rl.TokenBucketLimiter(ctx, key, algoConfig, configHash)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
