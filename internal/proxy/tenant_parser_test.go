package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
)

func init() {
	testLogger, _ := logger.NewLogger(logger.Config{
		Level:       "error",
		Environment: "test",
		OutputPath:  "",
	})
	logger.Log = testLogger
}

func TestExtractTenantKey(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func() *http.Request
		tenantRule   *config.TenantStrategy
		expected     string
		expectError  bool
	}{
		{
			name: "nil tenant rule falls back to IP",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			tenantRule:  nil,
			expected:    "192.168.1.1",
			expectError: false,
		},
		{
			name: "IP strategy extracts IP",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "10.0.0.1:9090"
				return req
			},
			tenantRule: &config.TenantStrategy{
				Type: "ip",
			},
			expected:    "10.0.0.1",
			expectError: false,
		},
		{
			name: "header strategy extracts from header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Client-ID", "client123")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			tenantRule: &config.TenantStrategy{
				Type: "header",
				Key:  "X-Client-ID",
			},
			expected:    "client123",
			expectError: false,
		},
		{
			name: "header strategy with missing header falls back to IP",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			tenantRule: &config.TenantStrategy{
				Type: "header",
				Key:  "X-Missing-Header",
			},
			expected:    "192.168.1.1",
			expectError: false,
		},
		{
			name: "cookie strategy extracts from cookie",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.AddCookie(&http.Cookie{
					Name:  "tenant_id",
					Value: "tenant456",
				})
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			tenantRule: &config.TenantStrategy{
				Type: "cookie",
				Key:  "tenant_id",
			},
			expected:    "tenant456",
			expectError: false,
		},
		{
			name: "cookie strategy with missing cookie falls back to IP",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			tenantRule: &config.TenantStrategy{
				Type: "cookie",
				Key:  "missing_cookie",
			},
			expected:    "192.168.1.1",
			expectError: false,
		},
		{
			name: "query parameter strategy extracts from URL param",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test?user_id=user789", nil)
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			tenantRule: &config.TenantStrategy{
				Type: "query_parameter",
				Key:  "user_id",
			},
			expected:    "user789",
			expectError: false,
		},
		{
			name: "query parameter strategy with missing param falls back to IP",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			tenantRule: &config.TenantStrategy{
				Type: "query_parameter",
				Key:  "missing_param",
			},
			expected:    "192.168.1.1",
			expectError: false,
		},
		{
			name: "unknown strategy type returns error",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			tenantRule: &config.TenantStrategy{
				Type: "unknown",
			},
			expected:    "",
			expectError: true,
		},
		{
			name: "header with whitespace is trimmed",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Client-ID", "  client123  ")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			tenantRule: &config.TenantStrategy{
				Type: "header",
				Key:  "X-Client-ID",
			},
			expected:    "client123",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			result, err := ExtractTenantKey(req, tt.tenantRule)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q but got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name     string
		setupReq func() *http.Request
		expected string
	}{
		{
			name: "X-Real-IP header takes precedence",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Real-IP", "203.0.113.1")
				req.Header.Set("X-Forwarded-For", "198.51.100.1")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "203.0.113.1",
		},
		{
			name: "X-Forwarded-For header when X-Real-IP is empty",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "198.51.100.1, 203.0.113.1")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "198.51.100.1",
		},
		{
			name: "X-Forwarded-For with whitespace is trimmed",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "  198.51.100.1  ,   203.0.113.1  ")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "198.51.100.1",
		},
		{
			name: "RemoteAddr when headers are empty",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "192.168.1.1",
		},
		{
			name: "IPv6 RemoteAddr",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "[2001:db8::1]:8080"
				return req
			},
			expected: "2001:db8::1", // Fixed IPv6 parsing without brackets
		},
		{
			name: "empty X-Forwarded-For falls back to RemoteAddr",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "192.168.1.1",
		},
		{
			name: "X-Forwarded-For with empty first value",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", ", 203.0.113.1")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractIP(tt.setupReq())
			if result != tt.expected {
				t.Errorf("expected %q but got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractFromHeader(t *testing.T) {
	tests := []struct {
		name      string
		headerKey string
		setupReq  func() *http.Request
		expected  string
	}{
		{
			name:      "existing header",
			headerKey: "X-API-Key",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-API-Key", "abc123")
				return req
			},
			expected: "abc123",
		},
		{
			name:      "missing header",
			headerKey: "X-Missing",
			setupReq: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			expected: "",
		},
		{
			name:      "header with whitespace",
			headerKey: "Authorization",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("Authorization", "  Bearer token123  ")
				return req
			},
			expected: "Bearer token123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFromHeader(tt.setupReq(), tt.headerKey)
			if result != tt.expected {
				t.Errorf("expected %q but got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractFromCookie(t *testing.T) {
	tests := []struct {
		name      string
		cookieKey string
		setupReq  func() *http.Request
		expected  string
	}{
		{
			name:      "existing cookie",
			cookieKey: "session_id",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: "sess123",
				})
				return req
			},
			expected: "sess123",
		},
		{
			name:      "missing cookie",
			cookieKey: "missing_cookie",
			setupReq: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			expected: "",
		},
		{
			name:      "multiple cookies, extract specific one",
			cookieKey: "user_id",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess123"})
				req.AddCookie(&http.Cookie{Name: "user_id", Value: "user456"})
				req.AddCookie(&http.Cookie{Name: "lang", Value: "en"})
				return req
			},
			expected: "user456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFromCookie(tt.setupReq(), tt.cookieKey)
			if result != tt.expected {
				t.Errorf("expected %q but got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractFromParam(t *testing.T) {
	tests := []struct {
		name     string
		paramKey string
		url      string
		expected string
	}{
		{
			name:     "existing parameter",
			paramKey: "user_id",
			url:      "/test?user_id=123&other=value",
			expected: "123",
		},
		{
			name:     "missing parameter",
			paramKey: "missing",
			url:      "/test?user_id=123",
			expected: "",
		},
		{
			name:     "empty parameter value",
			paramKey: "empty",
			url:      "/test?empty=&user_id=123",
			expected: "",
		},
		{
			name:     "URL encoded parameter",
			paramKey: "name",
			url:      "/test?name=John%20Doe",
			expected: "John Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			result := extractFromParam(req, tt.paramKey)
			if result != tt.expected {
				t.Errorf("expected %q but got %q", tt.expected, result)
			}
		})
	}
}
