package db

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/redis/go-redis/v9"
)

// Integration test - requires Redis running
func TestRedisTokenBucketStore_Integration(t *testing.T) {
	// Skip if no Redis available
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use test database
	})

	// Test Redis connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Clean up test database
	defer func() {
		client.FlushDB(ctx)
		client.Close()
	}()

	store := NewRedisTokenBucketStore(client)
	testKey := "test:token:bucket"

	// Test GetState on non-existent key
	state, err := store.GetState(ctx, testKey)
	if err != nil {
		t.Fatalf("Expected no error for non-existent key, got %v", err)
	}
	if state != nil {
		t.Error("Expected nil state for non-existent key")
	}

	// Test UpdateState
	originalState := &limiter.TokenBucketState{
		Tokens:     50,
		LastRefill: time.Now().Truncate(time.Second), // Truncate for comparison
		ConfigHash: "test-hash-123",
	}

	err = store.UpdateState(ctx, testKey, originalState)
	if err != nil {
		t.Fatalf("Failed to update state: %v", err)
	}

	// Test GetState on existing key
	retrievedState, err := store.GetState(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	if retrievedState == nil {
		t.Fatal("Expected non-nil state")
	}

	// Compare states
	if retrievedState.Tokens != originalState.Tokens {
		t.Errorf("Expected tokens %d, got %d", originalState.Tokens, retrievedState.Tokens)
	}
	if retrievedState.ConfigHash != originalState.ConfigHash {
		t.Errorf("Expected config hash %s, got %s", originalState.ConfigHash, retrievedState.ConfigHash)
	}
	if !retrievedState.LastRefill.Equal(originalState.LastRefill) {
		t.Errorf("Expected last refill %v, got %v", originalState.LastRefill, retrievedState.LastRefill)
	}

	// Test UpdateState overwrites existing data
	updatedState := &limiter.TokenBucketState{
		Tokens:     25,
		LastRefill: time.Now().Add(-time.Hour).Truncate(time.Second),
		ConfigHash: "new-hash-456",
	}

	err = store.UpdateState(ctx, testKey, updatedState)
	if err != nil {
		t.Fatalf("Failed to update state: %v", err)
	}

	finalState, err := store.GetState(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to get updated state: %v", err)
	}

	if finalState.Tokens != updatedState.Tokens {
		t.Errorf("Expected updated tokens %d, got %d", updatedState.Tokens, finalState.Tokens)
	}
	if finalState.ConfigHash != updatedState.ConfigHash {
		t.Errorf("Expected updated config hash %s, got %s", updatedState.ConfigHash, finalState.ConfigHash)
	}
}

// Unit test without Redis - tests error handling
func TestRedisTokenBucketStore_ErrorHandling(t *testing.T) {
	// Create client with invalid address to simulate connection errors
	client := redis.NewClient(&redis.Options{
		Addr:        "localhost:9999", // Non-existent Redis
		DialTimeout: 100 * time.Millisecond,
	})
	defer client.Close()

	store := NewRedisTokenBucketStore(client)
	ctx := context.Background()

	// Test GetState with connection error
	_, err := store.GetState(ctx, "test-key")
	if err == nil {
		t.Error("Expected error when Redis is unavailable")
	}

	// Test UpdateState with connection error
	state := &limiter.TokenBucketState{
		Tokens:     10,
		LastRefill: time.Now(),
		ConfigHash: "hash",
	}
	err = store.UpdateState(ctx, "test-key", state)
	if err == nil {
		t.Error("Expected error when Redis is unavailable")
	}
}

func TestRedisTokenBucketStore_JSONSerialization(t *testing.T) {
	// Test that our state struct serializes/deserializes correctly
	originalState := &limiter.TokenBucketState{
		Tokens:     42,
		LastRefill: time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC),
		ConfigHash: "test-hash-with-special-chars-!@#$%",
	}

	data, err := json.Marshal(originalState)
	if err != nil {
		t.Fatalf("Failed to marshal state: %v", err)
	}

	var deserializedState limiter.TokenBucketState
	err = json.Unmarshal(data, &deserializedState)
	if err != nil {
		t.Fatalf("Failed to unmarshal state: %v", err)
	}

	if deserializedState.Tokens != originalState.Tokens {
		t.Errorf("Tokens mismatch: expected %d, got %d", originalState.Tokens, deserializedState.Tokens)
	}
	if deserializedState.ConfigHash != originalState.ConfigHash {
		t.Errorf("ConfigHash mismatch: expected %s, got %s", originalState.ConfigHash, deserializedState.ConfigHash)
	}
	if !deserializedState.LastRefill.Equal(originalState.LastRefill) {
		t.Errorf("LastRefill mismatch: expected %v, got %v", originalState.LastRefill, deserializedState.LastRefill)
	}
}
