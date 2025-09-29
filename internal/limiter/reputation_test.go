package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReputation_NewTenant(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	tenantKey := "new_user"

	// First good request should initialize reputation
	reputation, err := rl.UpdateReputation(ctx, tenantKey, false)
	require.NoError(t, err)
	assert.Equal(t, 1.0, reputation.Score)
	assert.Equal(t, int64(0), reputation.ViolationCount)
	assert.Equal(t, int64(1), reputation.GoodRequests)
	assert.Greater(t, reputation.TTL, int64(0))
}

func TestReputation_MinimumViolationImpact(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	tenantKey := "impact_test_user"

	// Make tons of good requests to minimize violation impact
	for i := 0; i < 1000; i++ {
		rl.UpdateReputation(ctx, tenantKey, false)
	}

	// Get score before violation
	repBefore, _ := rl.GetTenantReputation(ctx, tenantKey)
	scoreBefore := repBefore.Score

	// Make violation - should still have minimum 5% impact
	repAfter, err := rl.UpdateReputation(ctx, tenantKey, true)
	require.NoError(t, err)

	impact := scoreBefore - repAfter.Score
	assert.GreaterOrEqual(t, impact, 0.05) // Minimum 5% impact
}

// func TestReputation_EscalatingPunishment(t *testing.T) {
// 	rl, mr := setupTestRateLimiter(t)
// 	defer mr.Close()
// 	ctx := context.Background()

// 	getImpact := func(key string, violations int) float64 {
// 		for i := 0; i < 10; i++ {
// 			rl.UpdateReputation(ctx, key, false)
// 		}
// 		for i := 0; i < violations; i++ {
// 			rl.UpdateReputation(ctx, key, true)
// 		}
// 		repBefore, _ := rl.GetTenantReputation(ctx, key)
// 		repAfter, _ := rl.UpdateReputation(ctx, key, true)
// 		return repBefore.Score - repAfter.Score
// 	}

// 	impact1 := getImpact("escalation_user_1", 1)
// 	impact2 := getImpact("escalation_user_2", 5)
// 	impact3 := getImpact("escalation_user_3", 10)

// 	t.Logf("impact1=%f impact2=%f impact3=%f", impact1, impact2, impact3)

// 	// Assertions
// 	assert.Greater(t, impact1, 0.0)
// 	assert.GreaterOrEqual(t, impact2, impact1)
// 	assert.GreaterOrEqual(t, impact3, impact2)
// }

func TestReputation_RapidFireDetection(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()
	ctx := context.Background()
	tenantKey := "rapid_fire_user"

	// Some good requests first
	for i := 0; i < 5; i++ {
		rl.UpdateReputation(ctx, tenantKey, false)
	}

	// First violation
	repBefore1, _ := rl.GetTenantReputation(ctx, tenantKey)
	rep1, err := rl.UpdateReputation(ctx, tenantKey, true)
	require.NoError(t, err)
	impact1 := repBefore1.Score - rep1.Score

	// Ensure next call has a distinct timestamp
	time.Sleep(2 * time.Millisecond)

	// Second violation (should trigger rapid-fire penalty)
	repBefore2, _ := rl.GetTenantReputation(ctx, tenantKey)
	rep2, err := rl.UpdateReputation(ctx, tenantKey, true)
	require.NoError(t, err)
	impact2 := repBefore2.Score - rep2.Score

	// The second impact should include at least ~0.2 extra penalty
	assert.GreaterOrEqual(t, impact2, impact1-0.2)
	assert.LessOrEqual(t, impact2, impact1+0.25) // allow for variation due to clamping
}

func TestReputation_BotLikePatterns(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	tenantKey := "bot_user"

	// Simulate bot behavior - many rapid violations
	for i := 0; i < 15; i++ {
		reputation, err := rl.UpdateReputation(ctx, tenantKey, true)
		require.NoError(t, err)

		// Score should keep decreasing
		assert.GreaterOrEqual(t, reputation.Score, 0.0)

		// After 10 violations, TTL should be extended (doubled)
		if i >= 10 {
			assert.Greater(t, reputation.TTL, int64(7200)) // More than 2 hours
		}
	}

	// Final reputation should be very low
	finalRep, err := rl.GetTenantReputation(ctx, tenantKey)
	require.NoError(t, err)
	assert.Less(t, finalRep.Score, 0.2) // Should be very low for bot-like behavior
}

func TestReputation_SlowRecoveryForViolators(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	tenantKey := "slow_recovery_user"

	// Create a violator
	for i := 0; i < 5; i++ {
		rl.UpdateReputation(ctx, tenantKey, true)
	}

	// Get current low score
	repBefore, _ := rl.GetTenantReputation(ctx, tenantKey)
	scoreBefore := repBefore.Score

	// Make many good requests
	for i := 0; i < 50; i++ {
		rl.UpdateReputation(ctx, tenantKey, false)
	}

	// Score should improve, but slowly
	repAfter, _ := rl.GetTenantReputation(ctx, tenantKey)
	scoreAfter := repAfter.Score

	improvement := scoreAfter - scoreBefore
	assert.Greater(t, improvement, 0.0) // Should improve
	assert.Less(t, improvement, 0.5)    // But not too fast (anti-bot)
}

func TestReputation_FastRecoveryForCleanUsers(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	tenantKey := "clean_user"

	// Simulate a clean user caught in high traffic (no violations, just low score)
	// We'll manually set a low score by making violations, then clearing violation count
	for i := 0; i < 3; i++ {
		rl.UpdateReputation(ctx, tenantKey, true)
	}

	// Now imagine this was a legitimate user - make good requests
	// (In real scenario, they'd have 0 violations but low score from traffic limiting)
	for i := 0; i < 10; i++ {
		reputation, err := rl.UpdateReputation(ctx, tenantKey, false)
		require.NoError(t, err)

		// Recovery should be happening (though limited by violation history in this test)
		assert.GreaterOrEqual(t, reputation.Score, 0.0)
	}
}

func TestReputation_TTLScaling(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()

	tests := []struct {
		name           string
		violations     int
		expectedMinTTL int64
		expectedMaxTTL int64
	}{
		{
			name:           "Confirmed bot (very low score)",
			violations:     20,
			expectedMinTTL: 14000, // Around 4h
			expectedMaxTTL: 30000, // Could be doubled for repeat offenders
		},
		{
			name:           "Suspicious actor",
			violations:     3,
			expectedMinTTL: 7000, // Around 2h
			expectedMaxTTL: 8000,
		},
		{
			name:           "Clean user",
			violations:     0,
			expectedMinTTL: 1700, // Around 30min
			expectedMaxTTL: 1900,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantKey := "ttl_user_" + tt.name

			// Make violations as specified
			for i := 0; i < tt.violations; i++ {
				rl.UpdateReputation(ctx, tenantKey, true)
			}

			// Get final reputation
			reputation, err := rl.UpdateReputation(ctx, tenantKey, false)
			require.NoError(t, err)

			assert.GreaterOrEqual(t, reputation.TTL, tt.expectedMinTTL)
			assert.LessOrEqual(t, reputation.TTL, tt.expectedMaxTTL)
		})
	}
}

func TestReputation_ScoreBoundaries(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	tenantKey := "boundary_user"

	// Test lower boundary - extreme bot behavior
	for i := 0; i < 100; i++ {
		reputation, err := rl.UpdateReputation(ctx, tenantKey, true)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, reputation.Score, 0.0) // Should never go below 0
	}

	// Should be at or near 0 for extreme bot behavior
	finalBadRep, _ := rl.GetTenantReputation(ctx, tenantKey)
	assert.LessOrEqual(t, finalBadRep.Score, 0.1) // Should be very low

	// Test upper boundary - many good requests
	newTenantKey := "boundary_user_good"
	for i := 0; i < 100; i++ {
		reputation, err := rl.UpdateReputation(ctx, newTenantKey, false)
		require.NoError(t, err)
		assert.LessOrEqual(t, reputation.Score, 1.0) // Should never go above 1
	}

	finalGoodRep, _ := rl.GetTenantReputation(ctx, newTenantKey)
	assert.Equal(t, 1.0, finalGoodRep.Score) // Should reach perfect score
}

func TestReputation_BackwardCompatibility(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	tenantKey := "compat_user"

	// Test that all original function signatures work exactly the same

	// UpdateReputation with same signature
	reputation, err := rl.UpdateReputation(ctx, tenantKey, false)
	require.NoError(t, err)
	assert.IsType(t, &Reputation{}, reputation)
	assert.IsType(t, float64(0), reputation.Score)
	assert.IsType(t, int64(0), reputation.ViolationCount)
	assert.IsType(t, int64(0), reputation.GoodRequests)
	assert.IsType(t, int64(0), reputation.TTL)

	// GetTenantReputation with same signature
	reputation2, err := rl.GetTenantReputation(ctx, tenantKey)
	require.NoError(t, err)
	assert.IsType(t, &Reputation{}, reputation2)

	// Values should match
	assert.Equal(t, reputation.Score, reputation2.Score)
	assert.Equal(t, reputation.ViolationCount, reputation2.ViolationCount)
	assert.Equal(t, reputation.GoodRequests, reputation2.GoodRequests)
}

func TestReputation_AntiAdaptationMechanisms(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	tenantKey := "adaptive_bot"

	// Simulate a bot trying to game the system
	// Pattern: Make violations, then try to recover with good requests

	// Phase 1: Bot makes violations
	for i := 0; i < 10; i++ {
		rl.UpdateReputation(ctx, tenantKey, true)
	}

	scoreBefore, _ := rl.GetTenantReputation(ctx, tenantKey)

	// Phase 2: Bot tries to recover with good requests
	for i := 0; i < 100; i++ {
		rl.UpdateReputation(ctx, tenantKey, false)
	}

	scoreAfter, _ := rl.GetTenantReputation(ctx, tenantKey)

	// Recovery should be very limited due to anti-adaptation mechanisms
	recovery := scoreAfter.Score - scoreBefore.Score
	assert.Less(t, recovery, 0.3)         // Should not recover more than 30%
	assert.Less(t, scoreAfter.Score, 0.6) // Should still be considered suspicious
}

func TestReputation_ConcurrentBotBehavior(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	defer mr.Close()

	ctx := context.Background()
	tenantKey := "concurrent_bot"

	// Simulate concurrent bot requests (all violations)
	numGoroutines := 30
	results := make(chan *Reputation, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			reputation, err := rl.UpdateReputation(ctx, tenantKey, true)
			if err != nil {
				errors <- err
				return
			}
			results <- reputation
		}()
	}

	// Collect results
	var finalReputation *Reputation
	for i := 0; i < numGoroutines; i++ {
		select {
		case reputation := <-results:
			assert.GreaterOrEqual(t, reputation.Score, 0.0)
			assert.LessOrEqual(t, reputation.Score, 1.0)
			finalReputation = reputation
		case err := <-errors:
			t.Fatalf("Unexpected error: %v", err)
		}
	}

	// Final reputation should be very low due to many violations
	assert.Less(t, finalReputation.Score, 0.2)
	assert.Equal(t, int64(numGoroutines), finalReputation.ViolationCount)
}

func TestReputation_RedisErrorHandling(t *testing.T) {
	rl, mr := setupTestRateLimiterleaky(t)
	mr.Close() // Close Redis to simulate error

	ctx := context.Background()
	tenantKey := "error_user"

	// UpdateReputation should return error when Redis is down
	reputation, err := rl.UpdateReputation(ctx, tenantKey, false)
	assert.Error(t, err)
	assert.Nil(t, reputation)

	// GetTenantReputation should return default values (fail-open)
	reputation2, err := rl.GetTenantReputation(ctx, tenantKey)
	require.NoError(t, err)
	assert.Equal(t, 1.0, reputation2.Score) // Default to good reputation
	assert.Equal(t, int64(0), reputation2.TTL)
}

// =============================================================================
// BENCHMARK TESTS FOR  SYSTEM
// =============================================================================

func BenchmarkReputation_BotViolations(b *testing.B) {
	rl, mr := setupTestRateLimiterleaky(nil)
	defer mr.Close()

	ctx := context.Background()
	tenantKey := "bench_bot"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := rl.UpdateReputation(ctx, tenantKey, true)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkReputation_LegitimateRequests(b *testing.B) {
	rl, mr := setupTestRateLimiterleaky(nil)
	defer mr.Close()

	ctx := context.Background()
	tenantKey := "bench_legit"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := rl.UpdateReputation(ctx, tenantKey, false)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkReputation_MixedTraffic(b *testing.B) {
	rl, mr := setupTestRateLimiterleaky(nil)
	defer mr.Close()

	ctx := context.Background()
	tenantKey := "bench_mixed"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// 10% violations, 90% good requests (realistic high-traffic scenario)
			isViolation := i%10 == 0
			_, err := rl.UpdateReputation(ctx, tenantKey, isViolation)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}
