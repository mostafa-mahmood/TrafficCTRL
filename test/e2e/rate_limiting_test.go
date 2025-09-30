package e2e

import (
	"context"
	"fmt"
	"sync"

	"github.com/stretchr/testify/assert"
)

func (s *E2ETestSuite) TestGlobalRateLimiting() {
	t := s.T()

	var wg sync.WaitGroup
	successCount := 0
	rateLimitCount := 0
	var mu sync.Mutex

	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			result, err := s.rateLimiter.CheckGlobalLimit(context.Background(), &s.proxyConfig.Limiter.Global)
			assert.NoError(t, err)

			mu.Lock()
			if result.Allowed {
				successCount++
			} else {
				rateLimitCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	assert.True(t, rateLimitCount > 0, "Should have rate limited some requests")
	assert.True(t, successCount <= 100, "Should not exceed global capacity")

	fmt.Printf("Global limiting: %d allowed, %d rate limited\n", successCount, rateLimitCount)
}
func (s *E2ETestSuite) TestPerTenantRateLimiting() {
	t := s.T()

	goodTenant := "tenant-good"
	badTenant := "tenant-bad"

	for i := 0; i < 10; i++ {
		result, err := s.rateLimiter.CheckTenantLimit(
			context.Background(),
			goodTenant,
			&s.proxyConfig.Limiter.PerTenant,
		)
		assert.NoError(t, err)
		assert.True(t, result.Allowed, "Good tenant should be allowed")
	}

	blockedAt := 0
	for i := 0; i < 40; i++ {
		result, err := s.rateLimiter.CheckTenantLimit(
			context.Background(),
			badTenant,
			&s.proxyConfig.Limiter.PerTenant,
		)
		assert.NoError(t, err)

		if !result.Allowed {
			blockedAt = i
			break
		}
	}

	assert.True(t, blockedAt > 0, "Bad tenant should eventually be blocked")

	result, err := s.rateLimiter.CheckTenantLimit(
		context.Background(),
		goodTenant,
		&s.proxyConfig.Limiter.PerTenant,
	)
	assert.NoError(t, err)
	assert.True(t, result.Allowed, "Good tenant should still work after bad tenant blocked")
}

func (s *E2ETestSuite) TestEndpointSpecificRateLimiting() {
	t := s.T()

	tenantKey := "test-user-123"

	loginBlocks := 0
	for i := 0; i < 10; i++ {
		result, err := s.rateLimiter.CheckEndpointLimit(
			context.Background(),
			tenantKey,
			&s.proxyConfig.Limiter.PerEndpoint.Rules[0],
		)
		assert.NoError(t, err)

		if !result.Allowed {
			loginBlocks++
		}
	}

	assert.True(t, loginBlocks > 0, "Should block excessive login attempts")

	result, err := s.rateLimiter.CheckEndpointLimit(
		context.Background(),
		tenantKey,
		&s.proxyConfig.Limiter.PerEndpoint.Rules[1],
	)
	assert.NoError(t, err)
	assert.True(t, result.Allowed, "Should allow requests to different endpoint")
}

func (s *E2ETestSuite) TestReputationSystem() {
	t := s.T()

	tenantKey := "reputation-test-user"

	for i := 0; i < 5; i++ {
		rep, err := s.rateLimiter.UpdateReputation(context.Background(), tenantKey, false)
		assert.NoError(t, err)
		assert.Greater(t, rep.Score, 0.5, "Score should be good with no violations")
	}

	for i := 0; i < 10; i++ {
		rep, err := s.rateLimiter.UpdateReputation(context.Background(), tenantKey, true)
		assert.NoError(t, err)

		if i > 5 {
			assert.Less(t, rep.Score, 0.5, "Score should degrade with violations")
		}
	}

	rep, err := s.rateLimiter.GetTenantReputation(context.Background(), tenantKey)
	assert.NoError(t, err)

	assert.Less(t, rep.Score, s.rateLimiter.GetReputationThreshold(),
		"Should be below threshold after many violations")
	assert.Greater(t, rep.ViolationCount, int64(0), "Should track violation count")
}
