package config

import (
	"testing"
	"time"
)

func ptr(v int) *int { return &v }
func dPtr(d time.Duration) *Duration {
	return &Duration{Duration: d}
}

func TestAlgorithmConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    AlgorithmConfig
		shouldErr bool
	}{
		{
			name: "valid token bucket",
			config: AlgorithmConfig{
				Algorithm:    "token_bucket",
				Capacity:     ptr(100),
				RefillRate:   ptr(10),
				RefillPeriod: dPtr(60),
			},
			shouldErr: false,
		},
		{
			name: "token bucket missing capacity",
			config: AlgorithmConfig{
				Algorithm:    "token_bucket",
				RefillRate:   ptr(10),
				RefillPeriod: dPtr(60),
			},
			shouldErr: true,
		},
		{
			name: "token bucket zero capacity",
			config: AlgorithmConfig{
				Algorithm:    "token_bucket",
				Capacity:     ptr(0),
				RefillRate:   ptr(10),
				RefillPeriod: dPtr(60),
			},
			shouldErr: true,
		},
		{
			name: "token bucket negative refill rate",
			config: AlgorithmConfig{
				Algorithm:    "token_bucket",
				Capacity:     ptr(100),
				RefillRate:   ptr(-5),
				RefillPeriod: dPtr(60),
			},
			shouldErr: true,
		},

		{
			name: "valid leaky bucket",
			config: AlgorithmConfig{
				Algorithm:  "leaky_bucket",
				Capacity:   ptr(200),
				LeakRate:   ptr(5),
				LeakPeriod: dPtr(30),
			},
			shouldErr: false,
		},
		{
			name: "leaky bucket missing leak rate",
			config: AlgorithmConfig{
				Algorithm:  "leaky_bucket",
				Capacity:   ptr(200),
				LeakPeriod: dPtr(30),
			},
			shouldErr: true,
		},

		{
			name: "valid fixed window",
			config: AlgorithmConfig{
				Algorithm:  "fixed_window",
				WindowSize: dPtr(60),
				Limit:      ptr(100),
			},
			shouldErr: false,
		},
		{
			name: "fixed window missing limit",
			config: AlgorithmConfig{
				Algorithm:  "fixed_window",
				WindowSize: dPtr(60),
			},
			shouldErr: true,
		},

		{
			name: "valid sliding window",
			config: AlgorithmConfig{
				Algorithm:  "sliding_window",
				WindowSize: dPtr(60),
				Limit:      ptr(50),
			},
			shouldErr: false,
		},
		{
			name: "sliding window zero window size",
			config: AlgorithmConfig{
				Algorithm:  "sliding_window",
				WindowSize: dPtr(0),
				Limit:      ptr(50),
			},
			shouldErr: true,
		},

		{
			name: "empty algorithm",
			config: AlgorithmConfig{
				Algorithm: "",
			},
			shouldErr: true,
		},
		{
			name: "invalid algorithm",
			config: AlgorithmConfig{
				Algorithm: "invalid_algorithm",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestTenantStrategy_Validate(t *testing.T) {
	tests := []struct {
		name      string
		strategy  TenantStrategy
		shouldErr bool
	}{
		{
			name: "valid ip strategy",
			strategy: TenantStrategy{
				Type: "ip",
			},
			shouldErr: false,
		},
		{
			name: "valid header strategy with key",
			strategy: TenantStrategy{
				Type: "header",
				Key:  "Authorization",
			},
			shouldErr: false,
		},
		{
			name: "valid cookie strategy with key",
			strategy: TenantStrategy{
				Type: "cookie",
				Key:  "session_id",
			},
			shouldErr: false,
		},
		{
			name: "valid query parameter strategy with key",
			strategy: TenantStrategy{
				Type: "query_parameter",
				Key:  "api_key",
			},
			shouldErr: false,
		},
		{
			name: "header strategy missing key",
			strategy: TenantStrategy{
				Type: "header",
				Key:  "",
			},
			shouldErr: true,
		},
		{
			name: "cookie strategy missing key",
			strategy: TenantStrategy{
				Type: "cookie",
			},
			shouldErr: true,
		},
		{
			name: "query parameter strategy missing key",
			strategy: TenantStrategy{
				Type: "query_parameter",
			},
			shouldErr: true,
		},
		{
			name: "invalid strategy type",
			strategy: TenantStrategy{
				Type: "invalid_type",
			},
			shouldErr: true,
		},
		{
			name: "empty strategy type",
			strategy: TenantStrategy{
				Type: "",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.strategy.validate()
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestProxyConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    ProxyConfig
		shouldErr bool
	}{
		{
			name: "valid http URL",
			config: ProxyConfig{
				TargetUrl: "http://localhost:3000",
				ProxyPort: 8080,
			},
			shouldErr: false,
		},
		{
			name: "valid https URL",
			config: ProxyConfig{
				TargetUrl: "https://api.example.com",
				ProxyPort: 443,
			},
			shouldErr: false,
		},
		{
			name: "valid URL with path",
			config: ProxyConfig{
				TargetUrl: "http://localhost:3000/api/v1",
				ProxyPort: 8080,
			},
			shouldErr: false,
		},
		{
			name: "empty target URL",
			config: ProxyConfig{
				TargetUrl: "",
				ProxyPort: 8080,
			},
			shouldErr: true,
		},
		{
			name: "URL without scheme",
			config: ProxyConfig{
				TargetUrl: "localhost:3000",
				ProxyPort: 8080,
			},
			shouldErr: true,
		},
		{
			name: "invalid scheme",
			config: ProxyConfig{
				TargetUrl: "ftp://localhost:3000",
				ProxyPort: 8080,
			},
			shouldErr: true,
		},
		{
			name: "malformed URL",
			config: ProxyConfig{
				TargetUrl: "http://[invalid",
				ProxyPort: 8080,
			},
			shouldErr: true,
		},
		{
			name: "zero proxy port",
			config: ProxyConfig{
				TargetUrl: "http://localhost:3000",
				ProxyPort: 0,
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestEndpointRule_Validate(t *testing.T) {
	tests := []struct {
		name      string
		rule      EndpointRule
		shouldErr bool
	}{
		{
			name: "valid bypass rule",
			rule: EndpointRule{
				Path:   "/health",
				Bypass: true,
			},
			shouldErr: false,
		},
		{
			name: "valid rule with methods",
			rule: EndpointRule{
				Path:    "/api/auth/login",
				Methods: []string{"POST", "put"}, // Test mixed case
				TenantStrategy: &TenantStrategy{
					Type: "ip",
				},
				AlgorithmConfig: AlgorithmConfig{
					Algorithm:  "fixed_window",
					WindowSize: dPtr(60),
					Limit:      ptr(5),
				},
			},
			shouldErr: false,
		},
		{
			name: "empty path",
			rule: EndpointRule{
				Path: "",
				AlgorithmConfig: AlgorithmConfig{
					Algorithm:  "fixed_window",
					WindowSize: dPtr(60),
					Limit:      ptr(5),
				},
			},
			shouldErr: true,
		},
		{
			name: "invalid HTTP method",
			rule: EndpointRule{
				Path:    "/api/test",
				Methods: []string{"INVALID_METHOD"},
				AlgorithmConfig: AlgorithmConfig{
					Algorithm:  "fixed_window",
					WindowSize: dPtr(60),
					Limit:      ptr(5),
				},
			},
			shouldErr: true,
		},
		{
			name: "invalid tenant strategy",
			rule: EndpointRule{
				Path: "/api/test",
				TenantStrategy: &TenantStrategy{
					Type: "header",
					// Missing key
				},
				AlgorithmConfig: AlgorithmConfig{
					Algorithm:  "fixed_window",
					WindowSize: dPtr(60),
					Limit:      ptr(5),
				},
			},
			shouldErr: true,
		},
		{
			name: "invalid algorithm config",
			rule: EndpointRule{
				Path: "/api/test",
				TenantStrategy: &TenantStrategy{
					Type: "ip",
				},
				AlgorithmConfig: AlgorithmConfig{
					Algorithm: "token_bucket",
					// Missing required fields
				},
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.validate()
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestLoggerConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    LoggerConfig
		shouldErr bool
	}{
		{
			name: "valid development config",
			config: LoggerConfig{
				Level:       "debug",
				Environment: "development",
				OutputPath:  "stdout",
			},
			shouldErr: false,
		},
		{
			name: "valid production config",
			config: LoggerConfig{
				Level:       "info",
				Environment: "production",
				OutputPath:  "/var/log/trafficctrl.log",
			},
			shouldErr: false,
		},
		{
			name: "all valid log levels",
			config: LoggerConfig{
				Level:       "trace",
				Environment: "development",
				OutputPath:  "stdout",
			},
			shouldErr: false,
		},
		{
			name: "invalid log level",
			config: LoggerConfig{
				Level:       "invalid_level",
				Environment: "development",
				OutputPath:  "stdout",
			},
			shouldErr: true,
		},
		{
			name: "invalid environment",
			config: LoggerConfig{
				Level:       "info",
				Environment: "staging", // Not supported
				OutputPath:  "stdout",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestRedisConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    RedisConfig
		shouldErr bool
	}{
		{
			name: "valid config",
			config: RedisConfig{
				Address:  "localhost:6379",
				Password: "",
				DB:       0,
				PoolSize: 40,
			},
			shouldErr: false,
		},
		{
			name: "valid config with auth",
			config: RedisConfig{
				Address:  "redis.example.com:6379",
				Password: "secret",
				DB:       1,
				PoolSize: 100,
			},
			shouldErr: false,
		},
		{
			name: "empty address",
			config: RedisConfig{
				Address:  "",
				Password: "",
				DB:       0,
				PoolSize: 40,
			},
			shouldErr: true,
		},
		{
			name: "negative DB",
			config: RedisConfig{
				Address:  "localhost:6379",
				Password: "",
				DB:       -1,
				PoolSize: 40,
			},
			shouldErr: true,
		},
		{
			name: "zero pool size",
			config: RedisConfig{
				Address:  "localhost:6379",
				Password: "",
				DB:       0,
				PoolSize: 0,
			},
			shouldErr: true,
		},
		{
			name: "negative pool size",
			config: RedisConfig{
				Address:  "localhost:6379",
				Password: "",
				DB:       0,
				PoolSize: -5,
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestConfigLoader_Integration(t *testing.T) {
	// Test that default configs are valid
	t.Run("default configs are valid", func(t *testing.T) {
		proxyDefaults := getProxyDefaults()
		if err := proxyDefaults.validate(); err != nil {
			t.Errorf("default proxy config is invalid: %v", err)
		}

		redisDefaults := getRedisDefaults()
		if err := redisDefaults.validate(); err != nil {
			t.Errorf("default redis config is invalid: %v", err)
		}

		loggerDefaults := getLoggerDefaults()
		if err := loggerDefaults.validate(); err != nil {
			t.Errorf("default logger config is invalid: %v", err)
		}

		limiterDefaults := getLimiterDefaults()
		if err := limiterDefaults.validate(); err != nil {
			t.Errorf("default limiter config is invalid: %v", err)
		}
	})
}
