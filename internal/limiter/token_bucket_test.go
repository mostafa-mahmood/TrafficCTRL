package limiter

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
)

// Simple in-memory mock store - no import cycles
type mockStore struct {
	mu        sync.RWMutex
	states    map[string]*TokenBucketState
	getErr    error
	updateErr error
}

func newMockStore() *mockStore {
	return &mockStore{
		states: make(map[string]*TokenBucketState),
	}
}

func (m *mockStore) GetState(ctx context.Context, key string) (*TokenBucketState, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, exists := m.states[key]
	if !exists {
		return nil, nil
	}
	stateCopy := *state
	return &stateCopy, nil
}

func (m *mockStore) UpdateState(ctx context.Context, key string, state *TokenBucketState) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	stateCopy := *state
	m.states[key] = &stateCopy
	return nil
}

func (m *mockStore) setState(key string, state *TokenBucketState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[key] = state
}

func (m *mockStore) getStoredState(key string) *TokenBucketState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.states[key]
}

func TestTokenBucketLimiter_NewBucket(t *testing.T) {
	mockStore := newMockStore()
	limiter := NewRateLimiter(mockStore)

	algConfig := config.AlgorithmConfig{
		Algorithm:    "token_bucket",
		Capacity:     intPtr(10),
		RefillRate:   intPtr(5),
		RefillPeriod: intPtr(1),
	}

	result, err := limiter.TokenBucketLimiter(context.Background(), "test-key", algConfig, "hash123")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.Allowed {
		t.Error("Expected first request to be allowed")
	}

	if result.Remaining != 9 { // 10 capacity - 1 consumed
		t.Errorf("Expected 9 remaining tokens, got %d", result.Remaining)
	}

	if result.RetryAfter != 0 {
		t.Errorf("Expected no retry delay, got %v", result.RetryAfter)
	}
}

func TestTokenBucketLimiter_ExhaustTokens(t *testing.T) {
	mockStore := newMockStore()
	limiter := NewRateLimiter(mockStore)

	algConfig := config.AlgorithmConfig{
		Algorithm:    "token_bucket",
		Capacity:     intPtr(2),
		RefillRate:   intPtr(1),
		RefillPeriod: intPtr(10), // Long refill period
	}

	// First request should succeed
	result1, err := limiter.TokenBucketLimiter(context.Background(), "test-key", algConfig, "hash123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !result1.Allowed {
		t.Error("Expected first request to be allowed")
	}
	if result1.Remaining != 1 {
		t.Errorf("Expected 1 remaining token, got %d", result1.Remaining)
	}

	// Second request should succeed
	result2, err := limiter.TokenBucketLimiter(context.Background(), "test-key", algConfig, "hash123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !result2.Allowed {
		t.Error("Expected second request to be allowed")
	}
	if result2.Remaining != 0 {
		t.Errorf("Expected 0 remaining tokens, got %d", result2.Remaining)
	}

	// Third request should fail
	result3, err := limiter.TokenBucketLimiter(context.Background(), "test-key", algConfig, "hash123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result3.Allowed {
		t.Error("Expected third request to be denied")
	}
	if result3.Remaining != 0 {
		t.Errorf("Expected 0 remaining tokens, got %d", result3.Remaining)
	}
	if result3.RetryAfter == 0 {
		t.Error("Expected retry delay to be set")
	}
}

func TestTokenBucketLimiter_Refill(t *testing.T) {
	mockStore := newMockStore()
	limiter := NewRateLimiter(mockStore)

	algConfig := config.AlgorithmConfig{
		Algorithm:    "token_bucket",
		Capacity:     intPtr(10),
		RefillRate:   intPtr(5),
		RefillPeriod: intPtr(1), // 1 second
	}

	// Consume all tokens
	for i := 0; i < 10; i++ {
		limiter.TokenBucketLimiter(context.Background(), "test-key", algConfig, "hash123")
	}

	// Next request should fail
	result, err := limiter.TokenBucketLimiter(context.Background(), "test-key", algConfig, "hash123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result.Allowed {
		t.Error("Expected request to be denied when bucket is empty")
	}

	// Simulate time passing by manually updating the stored state
	storedState := mockStore.getStoredState("test-key")
	storedState.LastRefill = storedState.LastRefill.Add(-2 * time.Second) // 2 seconds ago
	mockStore.setState("test-key", storedState)

	// Now request should succeed because 2 refill periods passed (2 * 5 = 10 tokens added)
	result, err = limiter.TokenBucketLimiter(context.Background(), "test-key", algConfig, "hash123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !result.Allowed {
		t.Error("Expected request to be allowed after refill")
	}
	if result.Remaining != 9 { // 10 tokens added, 1 consumed
		t.Errorf("Expected 9 remaining tokens after refill, got %d", result.Remaining)
	}
}

func TestTokenBucketLimiter_ConfigChange(t *testing.T) {
	mockStore := newMockStore()
	limiter := NewRateLimiter(mockStore)

	// Set initial state with old config
	oldState := &TokenBucketState{
		Tokens:     5,
		LastRefill: time.Now(),
		ConfigHash: "old-hash",
	}
	mockStore.setState("test-key", oldState)

	// New config with different hash
	algConfig := config.AlgorithmConfig{
		Algorithm:    "token_bucket",
		Capacity:     intPtr(20),
		RefillRate:   intPtr(10),
		RefillPeriod: intPtr(1),
	}

	result, err := limiter.TokenBucketLimiter(context.Background(), "test-key", algConfig, "new-hash")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.Allowed {
		t.Error("Expected request to be allowed with new config")
	}

	if result.Remaining != 19 { // New capacity (20) - 1 consumed
		t.Errorf("Expected 19 remaining tokens with new config, got %d", result.Remaining)
	}

	// Verify the state was updated with new config
	updatedState := mockStore.getStoredState("test-key")
	if updatedState.ConfigHash != "new-hash" {
		t.Errorf("Expected config hash to be updated to 'new-hash', got %s", updatedState.ConfigHash)
	}
}

func TestTokenBucketLimiter_MissingConfig(t *testing.T) {
	mockStore := newMockStore()
	limiter := NewRateLimiter(mockStore)

	testCases := []struct {
		name      string
		algConfig config.AlgorithmConfig
	}{
		{
			name: "missing capacity",
			algConfig: config.AlgorithmConfig{
				Algorithm:    "token_bucket",
				RefillRate:   intPtr(5),
				RefillPeriod: intPtr(1),
			},
		},
		{
			name: "missing refill rate",
			algConfig: config.AlgorithmConfig{
				Algorithm:    "token_bucket",
				Capacity:     intPtr(10),
				RefillPeriod: intPtr(1),
			},
		},
		{
			name: "missing refill period",
			algConfig: config.AlgorithmConfig{
				Algorithm:  "token_bucket",
				Capacity:   intPtr(10),
				RefillRate: intPtr(5),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := limiter.TokenBucketLimiter(context.Background(), "test-key", tc.algConfig, "hash123")
			if err == nil {
				t.Error("Expected error for missing config, got nil")
			}
		})
	}
}

// Helper function to create int pointers
func intPtr(i int) *int {
	return &i
}
