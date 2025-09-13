package proxy

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
)

func TestMapRequestToEndpointConfig(t *testing.T) {
	rules := []config.EndpointRules{
		{
			Path:    "/api/auth/login",
			Methods: []string{"POST"},
			TenantStrategy: &config.TenantStrategy{
				Type: "ip",
			},
			AlgorithmConfig: config.AlgorithmConfig{
				Algorithm: "fixed_window",
			},
		},
		{
			Path:    "/api/auth/register",
			Methods: []string{"POST"},
			TenantStrategy: &config.TenantStrategy{
				Type: "ip",
			},
			AlgorithmConfig: config.AlgorithmConfig{
				Algorithm: "fixed_window",
			},
		},
		{
			Path: "/api/v1/*",
			TenantStrategy: &config.TenantStrategy{
				Type: "jwt",
				Key:  "sub",
			},
			AlgorithmConfig: config.AlgorithmConfig{
				Algorithm: "token_bucket",
			},
		},
		{
			Path:    "/api/uploads/*",
			Methods: []string{"POST", "PUT"},
			TenantStrategy: &config.TenantStrategy{
				Type: "header",
				Key:  "x-api-key",
			},
			AlgorithmConfig: config.AlgorithmConfig{
				Algorithm: "leaky_bucket",
			},
		},
		{
			Path:   "/health",
			Bypass: true,
		},
		{
			Path: "*",
			TenantStrategy: &config.TenantStrategy{
				Type: "header",
				Key:  "x-client-id",
			},
			AlgorithmConfig: config.AlgorithmConfig{
				Algorithm: "sliding_window",
			},
		},
	}

	tests := []struct {
		name         string
		method       string
		path         string
		expectedPath string // Path of the rule we expect to match
	}{
		{
			name:         "exact match - login POST",
			method:       "POST",
			path:         "/api/auth/login",
			expectedPath: "/api/auth/login",
		},
		{
			name:         "exact match - register POST",
			method:       "POST",
			path:         "/api/auth/register",
			expectedPath: "/api/auth/register",
		},
		{
			name:         "method mismatch - login GET falls to catch-all",
			method:       "GET",
			path:         "/api/auth/login",
			expectedPath: "*",
		},
		{
			name:         "wildcard match - api v1",
			method:       "GET",
			path:         "/api/v1/users",
			expectedPath: "/api/v1/*",
		},
		{
			name:         "wildcard match - api v1 nested",
			method:       "POST",
			path:         "/api/v1/users/123/posts",
			expectedPath: "/api/v1/*",
		},
		{
			name:         "wildcard with method match - uploads POST",
			method:       "POST",
			path:         "/api/uploads/file",
			expectedPath: "/api/uploads/*",
		},
		{
			name:         "wildcard with method mismatch - uploads GET falls to catch-all",
			method:       "GET",
			path:         "/api/uploads/file",
			expectedPath: "*",
		},
		{
			name:         "bypass rule - health",
			method:       "GET",
			path:         "/health",
			expectedPath: "/health",
		},
		{
			name:         "catch-all wildcard",
			method:       "GET",
			path:         "/some/random/path",
			expectedPath: "*",
		},
		{
			name:         "trailing slash normalization",
			method:       "POST",
			path:         "/api/auth/login/",
			expectedPath: "/api/auth/login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock request
			req := &http.Request{
				Method: tt.method,
				URL: &url.URL{
					Path: tt.path,
				},
			}

			result := MapRequestToEndpointConfig(*req, rules)

			if result == nil {
				t.Errorf("Expected to match rule with path %s but got nil", tt.expectedPath)
				return
			}

			if result.Path != tt.expectedPath {
				t.Errorf("Expected path %s, got %s", tt.expectedPath, result.Path)
			}
		})
	}
}

func TestPathMatches(t *testing.T) {
	tests := []struct {
		name        string
		rulePath    string
		requestPath string
		expected    bool
	}{
		{"exact match", "/api/users", "/api/users", true},
		{"trailing slash request", "/api/users", "/api/users/", true},
		{"trailing slash rule", "/api/users/", "/api/users", true},
		{"wildcard match", "/api/*", "/api/users", true},
		{"wildcard nested match", "/api/*", "/api/users/123", true},
		{"wildcard no match", "/api/*", "/admin/users", false},
		{"catch all", "*", "/any/path", true},
		{"no match", "/api/users", "/api/posts", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pathMatches(tt.rulePath, tt.requestPath)
			if result != tt.expected {
				t.Errorf("pathMatches(%s, %s) = %v, want %v", tt.rulePath, tt.requestPath, result, tt.expected)
			}
		})
	}
}

func TestMethodMatches(t *testing.T) {
	tests := []struct {
		name          string
		ruleMethods   []string
		requestMethod string
		expected      bool
	}{
		{"empty methods list", []string{}, "GET", true},
		{"exact match", []string{"POST"}, "POST", true},
		{"case insensitive match", []string{"post"}, "POST", true},
		{"multiple methods match", []string{"GET", "POST"}, "POST", true},
		{"no match", []string{"POST"}, "GET", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := methodMatches(tt.ruleMethods, tt.requestMethod)
			if result != tt.expected {
				t.Errorf("methodMatches(%v, %s) = %v, want %v", tt.ruleMethods, tt.requestMethod, result, tt.expected)
			}
		})
	}
}
