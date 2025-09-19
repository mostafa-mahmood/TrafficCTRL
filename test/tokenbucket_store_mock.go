package test

import (
	"context"
	"sync"
	"time"
)

// TokenBucketState - duplicate the struct to avoid import cycle
type TokenBucketState struct {
	Tokens     int64     `json:"tokens"`
	LastRefill time.Time `json:"last_refill"`
	ConfigHash string    `json:"config_hash"`
}

// MockTokenBucketStore implements the interface without importing limiter package
type MockTokenBucketStore struct {
	mu     sync.RWMutex
	states map[string]*TokenBucketState

	// For testing error scenarios
	GetError    error
	UpdateError error
}

func NewMockTokenBucketStore() *MockTokenBucketStore {
	return &MockTokenBucketStore{
		states: make(map[string]*TokenBucketState),
	}
}

func (m *MockTokenBucketStore) GetState(ctx context.Context, key string) (*TokenBucketState, error) {
	if m.GetError != nil {
		return nil, m.GetError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[key]
	if !exists {
		return nil, nil
	}

	// Return a copy
	stateCopy := *state
	return &stateCopy, nil
}

func (m *MockTokenBucketStore) UpdateState(ctx context.Context, key string, state *TokenBucketState) error {
	if m.UpdateError != nil {
		return m.UpdateError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Store a copy
	stateCopy := *state
	m.states[key] = &stateCopy
	return nil
}

// Helper methods
func (m *MockTokenBucketStore) SetState(key string, state *TokenBucketState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[key] = state
}

func (m *MockTokenBucketStore) GetStoredState(key string) *TokenBucketState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.states[key]
}

func (m *MockTokenBucketStore) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states = make(map[string]*TokenBucketState)
}
