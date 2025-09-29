package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRateLimiterleaky(t *testing.T) (*RateLimiter, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	rl := &RateLimiter{
		redisClient: rdb,
	}

	return rl, mr
}

func TestLeakyBucketLimiter_BasicFunctionality(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:leaky:basic"
	configHash := "leaky_config_v1"

	capacity := 5
	leakRate := 1
	leakPeriod := &config.Duration{Duration: 1000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Capacity:   &capacity,
		LeakRate:   &leakRate,
		LeakPeriod: leakPeriod,
	}

	// First request should be allowed (bucket starts empty)
	result, err := rl.LeakyBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(4), result.Remaining) // 5 - 1 = 4 remaining capacity
}

func TestLeakyBucketLimiter_BucketOverflow(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:leaky:overflow"
	configHash := "leaky_config_v1"

	capacity := 3
	leakRate := 1
	leakPeriod := &config.Duration{Duration: 1000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Capacity:   &capacity,
		LeakRate:   &leakRate,
		LeakPeriod: leakPeriod,
	}

	// Fill the bucket completely
	for i := 0; i < 3; i++ {
		result, err := rl.LeakyBucketLimiter(ctx, key, algoConfig, configHash)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Next request should be denied (bucket full)
	result, err := rl.LeakyBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)
	assert.Greater(t, result.RetryAfter, time.Duration(0))
}

func TestLeakyBucketLimiter_LeakingBehavior(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:leaky:leak"
	configHash := "leaky_config_v1"

	capacity := 3
	leakRate := 2
	leakPeriod := &config.Duration{Duration: 100 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Capacity:   &capacity,
		LeakRate:   &leakRate,
		LeakPeriod: leakPeriod,
	}

	// Fill the bucket
	for i := 0; i < 3; i++ {
		result, err := rl.LeakyBucketLimiter(ctx, key, algoConfig, configHash)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Bucket should be full
	result, err := rl.LeakyBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Wait for leak period
	time.Sleep(leakPeriod.Duration + 10*time.Millisecond)

	// Should have space now (2 items leaked out)
	result, err = rl.LeakyBucketLimiter(ctx, key, algoConfig, configHash)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(1), result.Remaining) // 3 - 2 leaked - 1 new = 1 space left
}

func TestLeakyBucketLimiter_ConfigChange(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	key := "test:leaky:config"

	// Initial config
	capacity1 := 2
	leakRate1 := 1
	leakPeriod1 := &config.Duration{Duration: 1000 * time.Millisecond}
	configHash1 := "leaky_config_v1"

	algoConfig1 := config.AlgorithmConfig{
		Capacity:   &capacity1,
		LeakRate:   &leakRate1,
		LeakPeriod: leakPeriod1,
	}

	// Fill bucket
	for i := 0; i < 2; i++ {
		result, err := rl.LeakyBucketLimiter(ctx, key, algoConfig1, configHash1)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Should be full
	result, err := rl.LeakyBucketLimiter(ctx, key, algoConfig1, configHash1)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Change config
	capacity2 := 4
	leakRate2 := 2
	leakPeriod2 := &config.Duration{Duration: 500 * time.Millisecond}
	configHash2 := "leaky_config_v2"

	algoConfig2 := config.AlgorithmConfig{
		Capacity:   &capacity2,
		LeakRate:   &leakRate2,
		LeakPeriod: leakPeriod2,
	}

	// Should reset bucket and allow request
	result, err = rl.LeakyBucketLimiter(ctx, key, algoConfig2, configHash2)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(3), result.Remaining) // New capacity 4 - 1 = 3
}

func TestLeakyBucketLimiter_RedisError(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	mr.Close() // Close Redis to simulate error

	ctx := context.Background()
	key := "test:leaky:error"
	configHash := "config_v1"

	capacity := 5
	leakRate := 1
	leakPeriod := &config.Duration{Duration: 1000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Capacity:   &capacity,
		LeakRate:   &leakRate,
		LeakPeriod: leakPeriod,
	}

	// Should return error and NIL result on system failure
	result, err := rl.LeakyBucketLimiter(ctx, key, algoConfig, configHash)
	assert.Error(t, err)
	// FIX: Assert that the result pointer is NIL, as the Fail-Open logic is outside the Limiter function
	assert.Nil(t, result)
}

func BenchmarkLeakyBucketLimiter(b *testing.B) {
	rl, mr := setupTestRateLimiterleaky(nil)
	defer mr.Close()

	ctx := context.Background()
	key := "bench:leaky"
	configHash := "config_v1"

	capacity := 1000000
	leakRate := 1000
	leakPeriod := &config.Duration{Duration: 1000 * time.Millisecond}

	algoConfig := config.AlgorithmConfig{
		Capacity:   &capacity,
		LeakRate:   &leakRate,
		LeakPeriod: leakPeriod,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := rl.LeakyBucketLimiter(ctx, key, algoConfig, configHash)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
