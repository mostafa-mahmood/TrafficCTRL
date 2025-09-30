package e2e

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/stretchr/testify/assert"
)

func (s *E2ETestSuite) TestConcurrentRateLimiting() {
	t := s.T()

	var wg sync.WaitGroup
	successCount := 0
	rateLimitCount := 0
	var mu sync.Mutex

	start := time.Now()

	for userID := 0; userID < 50; userID++ {
		wg.Add(1)
		go func(user int) {
			defer wg.Done()

			tenantKey := fmt.Sprintf("user-%d", user)

			for req := 0; req < 10; req++ {
				result, err := s.rateLimiter.CheckTenantLimit(
					context.Background(),
					tenantKey,
					&s.proxyConfig.Limiter.PerTenant,
				)
				assert.NoError(t, err)

				mu.Lock()
				if result.Allowed {
					successCount++
				} else {
					rateLimitCount++
				}
				mu.Unlock()

				time.Sleep(10 * time.Millisecond)
			}
		}(userID)
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("Concurrent test: %d success, %d rate limited in %v\n",
		successCount, rateLimitCount, duration)

	assert.Equal(t, 500, successCount+rateLimitCount, "Should process all requests")
	assert.True(t, duration < 10*time.Second, "Should complete in reasonable time")
}

func (s *E2ETestSuite) TestRedisConnectionPool() {
	t := s.T()
	var wg sync.WaitGroup
	errors := 0
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_, err := s.rateLimiter.CheckGlobalLimit(ctx, &s.proxyConfig.Limiter.Global)
			if err != nil {
				mu.Lock()
				errors++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 0, errors, "Should handle concurrent Redis operations without errors")
}
