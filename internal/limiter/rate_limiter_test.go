package limiter

import (
	"context"
	"testing"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
)

func TestRateLimiter_CheckGlobalLimit_Disabled(t *testing.T) {
	mockStore := newMockStore()
	limiter := NewRateLimiter(mockStore)

	globalConfig := config.Global{
		Enabled: false,
		AlgorithmConfig: config.AlgorithmConfig{
			Algorithm:    "token_bucket",
			Capacity:     intPtr(10),
			RefillRate:   intPtr(5),
			RefillPeriod: intPtr(1),
		},
	}

	result, err := limiter.CheckGlobalLimit(context.Background(), globalConfig)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.Allowed {
		t.Error("Expected request to be allowed when global limit is disabled")
	}
}

func TestRateLimiter_CheckGlobalLimit_Enabled(t *testing.T) {
	mockStore := newMockStore()
	limiter := NewRateLimiter(mockStore)

	globalConfig := config.Global{
		Enabled: true,
		AlgorithmConfig: config.AlgorithmConfig{
			Algorithm:    "token_bucket",
			Capacity:     intPtr(10),
			RefillRate:   intPtr(5),
			RefillPeriod: intPtr(1),
		},
	}

	result, err := limiter.CheckGlobalLimit(context.Background(), globalConfig)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.Allowed {
		t.Error("Expected first request to be allowed")
	}

	if result.Remaining != 9 {
		t.Errorf("Expected 9 remaining tokens, got %d", result.Remaining)
	}
}

func TestRateLimiter_generateConfigHash(t *testing.T) {
	mockStore := newMockStore()
	limiter := NewRateLimiter(mockStore)

	testCases := []struct {
		name        string
		config1     config.AlgorithmConfig
		config2     config.AlgorithmConfig
		shouldMatch bool
	}{
		{
			name: "identical token bucket configs",
			config1: config.AlgorithmConfig{
				Algorithm:    "token_bucket",
				Capacity:     intPtr(10),
				RefillRate:   intPtr(5),
				RefillPeriod: intPtr(1),
			},
			config2: config.AlgorithmConfig{
				Algorithm:    "token_bucket",
				Capacity:     intPtr(10),
				RefillRate:   intPtr(5),
				RefillPeriod: intPtr(1),
			},
			shouldMatch: true,
		},
		{
			name: "different token bucket capacity",
			config1: config.AlgorithmConfig{
				Algorithm:    "token_bucket",
				Capacity:     intPtr(10),
				RefillRate:   intPtr(5),
				RefillPeriod: intPtr(1),
			},
			config2: config.AlgorithmConfig{
				Algorithm:    "token_bucket",
				Capacity:     intPtr(20),
				RefillRate:   intPtr(5),
				RefillPeriod: intPtr(1),
			},
			shouldMatch: false,
		},
		{
			name: "different algorithms",
			config1: config.AlgorithmConfig{
				Algorithm:    "token_bucket",
				Capacity:     intPtr(10),
				RefillRate:   intPtr(5),
				RefillPeriod: intPtr(1),
			},
			config2: config.AlgorithmConfig{
				Algorithm:  "fixed_window",
				WindowSize: intPtr(60),
				Limit:      intPtr(100),
			},
			shouldMatch: false,
		},
		{
			name: "nil vs zero values",
			config1: config.AlgorithmConfig{
				Algorithm:    "token_bucket",
				Capacity:     nil,
				RefillRate:   intPtr(5),
				RefillPeriod: intPtr(1),
			},
			config2: config.AlgorithmConfig{
				Algorithm:    "token_bucket",
				Capacity:     intPtr(0),
				RefillRate:   intPtr(5),
				RefillPeriod: intPtr(1),
			},
			shouldMatch: true, // Both resolve to 0 via safeIntValue
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash1 := limiter.generateConfigHash(tc.config1)
			hash2 := limiter.generateConfigHash(tc.config2)

			if tc.shouldMatch && hash1 != hash2 {
				t.Errorf("Expected hashes to match, got %s and %s", hash1, hash2)
			}
			if !tc.shouldMatch && hash1 == hash2 {
				t.Errorf("Expected hashes to be different, both got %s", hash1)
			}
		})
	}
}

func TestRateLimiter_checkLimit_UnsupportedAlgorithm(t *testing.T) {
	mockStore := newMockStore()
	limiter := NewRateLimiter(mockStore)

	algConfig := config.AlgorithmConfig{
		Algorithm: "unsupported_algo",
	}

	_, err := limiter.checkLimit(context.Background(), "test-key", algConfig, "hash123")

	if err == nil {
		t.Error("Expected error for unsupported algorithm")
	}

	expectedError := "unknown algorithm: unsupported_algo"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestSafeIntValue(t *testing.T) {
	testCases := []struct {
		name     string
		input    *int
		expected int
	}{
		{
			name:     "nil pointer",
			input:    nil,
			expected: 0,
		},
		{
			name:     "valid pointer",
			input:    intPtr(42),
			expected: 42,
		},
		{
			name:     "zero value pointer",
			input:    intPtr(0),
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := safeIntValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}
