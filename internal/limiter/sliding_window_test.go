package limiter

import (
	"testing"
	"time"
)

func TestSlidingWindow_BasicFunctionality(t *testing.T) {
	sw := NewSlidingWindow(3, 10*time.Second)
	tenantKey := "test-tenant"

	// Test 1: Fill up the limit
	for i := 0; i < 3; i++ {
		allowed, remaining, retryAfter := sw.Allow(tenantKey)
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
		expectedRemaining := 3 - (i + 1)
		if remaining != expectedRemaining {
			t.Errorf("Request %d: expected remaining %d, got %d", i+1, expectedRemaining, remaining)
		}
		if retryAfter != 0 {
			t.Errorf("Request %d: expected retryAfter 0, got %v", i+1, retryAfter)
		}
	}

	// Test 2: Next request should be rejected
	allowed, remaining, retryAfter := sw.Allow(tenantKey)
	if allowed {
		t.Error("Request should be rejected when limit exceeded")
	}
	if remaining != 0 {
		t.Errorf("Expected remaining 0 when rejected, got %d", remaining)
	}
	if retryAfter <= 0 {
		t.Errorf("Expected positive retryAfter when rejected, got %v", retryAfter)
	}
}

func TestSlidingWindow_WindowSliding(t *testing.T) {
	sw := NewSlidingWindow(2, 5*time.Second)
	tenantKey := "test-tenant"

	// Fill the limit
	allowed1, remaining1, _ := sw.Allow(tenantKey)
	allowed2, remaining2, _ := sw.Allow(tenantKey)

	if !allowed1 || !allowed2 {
		t.Fatal("Initial requests should be allowed")
	}
	if remaining1 != 1 || remaining2 != 0 {
		t.Fatalf("Expected remaining [1, 0], got [%d, %d]", remaining1, remaining2)
	}

	// Should be rejected now
	allowed3, remaining3, _ := sw.Allow(tenantKey)
	if allowed3 || remaining3 != 0 {
		t.Error("Third request should be rejected")
	}

	// Wait for window to slide (wait longer than window size)
	time.Sleep(6 * time.Second)

	// Now should be allowed with full capacity
	allowed4, remaining4, retryAfter4 := sw.Allow(tenantKey)
	if !allowed4 {
		t.Error("Request should be allowed after window slides")
	}
	if remaining4 != 1 {
		t.Errorf("Expected remaining 1 after window slides, got %d", remaining4)
	}
	if retryAfter4 != 0 {
		t.Errorf("Expected retryAfter 0 after window slides, got %v", retryAfter4)
	}

	t.Logf("Window sliding test completed successfully")
}

func TestSlidingWindow_PartialWindowSliding(t *testing.T) {
	sw := NewSlidingWindow(3, 6*time.Second)
	tenantKey := "test-tenant"

	// Make 3 requests with 2-second gaps
	times := []time.Time{}
	for i := 0; i < 3; i++ {
		allowed, _, _ := sw.Allow(tenantKey)
		times = append(times, time.Now())
		if !allowed {
			t.Fatalf("Request %d should be allowed", i+1)
		}
		if i < 2 {
			time.Sleep(2 * time.Second)
		}
	}

	// Should be rejected now
	allowed, remaining, _ := sw.Allow(tenantKey)
	if allowed || remaining != 0 {
		t.Error("Fourth request should be rejected")
	}

	// Wait for first request to expire (total ~7 seconds from start)
	time.Sleep(3 * time.Second)

	// Now should have 1 slot available (first request expired)
	allowed, remaining, retryAfter := sw.Allow(tenantKey)
	if !allowed {
		t.Error("Request should be allowed after first timestamp expires")
	}
	if remaining != 0 {
		t.Errorf("Expected remaining 0 (using the available slot), got %d", remaining)
	}
	if retryAfter != 0 {
		t.Errorf("Expected retryAfter 0, got %v", retryAfter)
	}

	t.Log("Partial window sliding test completed successfully")
}

func TestSlidingWindow_MultipleTenantsIsolation(t *testing.T) {
	sw := NewSlidingWindow(2, 5*time.Second)

	// Fill limit for tenant1
	allowed1, _, _ := sw.Allow("tenant1")
	allowed2, _, _ := sw.Allow("tenant1")
	allowed3, _, _ := sw.Allow("tenant1") // Should be rejected

	if !allowed1 || !allowed2 || allowed3 {
		t.Error("Tenant1 rate limiting not working correctly")
	}

	// tenant2 should have fresh limit
	allowed4, remaining4, retryAfter4 := sw.Allow("tenant2")
	if !allowed4 || remaining4 != 1 || retryAfter4 != 0 {
		t.Error("Tenant2 should have independent rate limit")
	}
}

func TestSlidingWindow_EdgeCases(t *testing.T) {
	// Test with limit 1
	sw := NewSlidingWindow(1, 2*time.Second)
	tenantKey := "test"

	allowed1, remaining1, _ := sw.Allow(tenantKey)
	allowed2, remaining2, _ := sw.Allow(tenantKey)

	if !allowed1 || remaining1 != 0 {
		t.Error("First request with limit=1 should be allowed with remaining=0")
	}
	if allowed2 || remaining2 != 0 {
		t.Error("Second request with limit=1 should be rejected")
	}

	// Wait and try again
	time.Sleep(3 * time.Second)
	allowed3, remaining3, _ := sw.Allow(tenantKey)
	if !allowed3 || remaining3 != 0 {
		t.Error("Request should be allowed after window expires")
	}
}

// Manual test for visual inspection (similar to your original)
func TestSlidingWindow_Visual(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping visual test in short mode")
	}

	t.Log("Visual test - watch the remaining counter behavior:")
	sw := NewSlidingWindow(3, 8*time.Second)
	tenantKey := "visual-test"

	for i := 0; i < 8; i++ {
		allowed, remaining, retryAfter := sw.Allow(tenantKey)
		t.Logf("Request %d: allowed=%v, remaining=%d, retryAfter=%v",
			i+1, allowed, remaining, retryAfter)
		time.Sleep(2 * time.Second)
	}
}
