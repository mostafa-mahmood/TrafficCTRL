package config

import (
	"fmt"
	"net/url"
	"strings"
)

func (a *AlgorithmConfig) validate() error {
	if a.Algorithm == "" {
		return fmt.Errorf("invalid limiter config (algorithm): field is required")
	}

	algorithm := AlgorithmType(a.Algorithm)

	switch algorithm {
	case TokenBucket:
		return a.validateTokenBucket()
	case LeakyBucket:
		return a.validateLeakyBucket()
	case FixedWindow:
		return a.validateFixedWindow()
	case SlidingWindow:
		return a.validateSlidingWindow()
	default:
		return fmt.Errorf("invalid limiter config (algorithm): unsupported algorithm: %s, must be one of [%s, %s, %s, %s]",
			a.Algorithm, TokenBucket, LeakyBucket, FixedWindow, SlidingWindow)
	}
}

func (a *AlgorithmConfig) validateTokenBucket() error {
	if a.Capacity == nil {
		return fmt.Errorf("invalid limiter config: capacity is required for token_bucket algorithm")
	}
	if *a.Capacity <= 0 {
		return fmt.Errorf("invalid limiter config: capacity must be positive, got: %d", *a.Capacity)
	}

	if a.RefillRate == nil {
		return fmt.Errorf("invalid limiter config: refill_rate is required for token_bucket algorithm")
	}
	if *a.RefillRate <= 0 {
		return fmt.Errorf("invalid limiter config: refill_rate must be positive, got: %d", *a.RefillRate)
	}

	if a.RefillPeriod == nil {
		return fmt.Errorf("invalid limiter config: refill_period is required for token_bucket algorithm")
	}
	if a.RefillPeriod.Duration <= 0 {
		return fmt.Errorf("invalid limiter config: refill_period must be positive, got: %d", *a.RefillPeriod)
	}

	return nil
}

func (a *AlgorithmConfig) validateLeakyBucket() error {
	if a.Capacity == nil {
		return fmt.Errorf("invalid limiter config: capacity is required for leaky_bucket algorithm")
	}
	if *a.Capacity <= 0 {
		return fmt.Errorf("invalid limiter config: capacity must be positive, got: %d", *a.Capacity)
	}

	if a.LeakRate == nil {
		return fmt.Errorf("invalid limiter config: leak_rate is required for leaky_bucket algorithm")
	}
	if *a.LeakRate <= 0 {
		return fmt.Errorf("invalid limiter config: leak_rate must be positive, got: %d", *a.LeakRate)
	}

	if a.LeakPeriod == nil {
		return fmt.Errorf("invalid limiter config: leak_period is required for leaky_bucket algorithm")
	}
	if a.LeakPeriod.Duration <= 0 {
		return fmt.Errorf("invalid limiter config: leak_period must be positive, got: %d", *a.LeakPeriod)
	}

	return nil
}

func (a *AlgorithmConfig) validateFixedWindow() error {
	if a.WindowSize == nil {
		return fmt.Errorf("invalid limiter config: window_size is required for fixed_window algorithm")
	}
	if a.WindowSize.Duration <= 0 {
		return fmt.Errorf("invalid limiter config: window_size must be positive, got: %d", *a.WindowSize)
	}

	if a.Limit == nil {
		return fmt.Errorf("invalid limiter config: limit is required for fixed_window algorithm")
	}
	if *a.Limit <= 0 {
		return fmt.Errorf("invalid limiter config: limit must be positive, got: %d", *a.Limit)
	}

	return nil
}

func (a *AlgorithmConfig) validateSlidingWindow() error {
	if a.WindowSize == nil {
		return fmt.Errorf("invalid limiter config: window_size is required for sliding_window algorithm")
	}
	if a.WindowSize.Duration <= 0 {
		return fmt.Errorf("invalid limiter config: window_size must be positive, got: %d", *a.WindowSize)
	}

	if a.Limit == nil {
		return fmt.Errorf("invalid limiter config: limit is required for sliding_window algorithm")
	}
	if *a.Limit <= 0 {
		return fmt.Errorf("invalid limiter config: limit must be positive, got: %d", *a.Limit)
	}

	return nil
}

func (t *TenantStrategy) validate() error {
	switch TenantStrategyType(t.Type) {
	case TenantIP:
		return nil
	case TenantHeader, TenantCookie, TenantQueryParameter:
		if t.Key == "" {
			return fmt.Errorf("invalid limiter config: key is required for tenant strategy type: %s", t.Type)
		}
		return nil
	default:
		return fmt.Errorf("invalid limiter config: unsupported tenant strategy type: %s, must be one of [%s, %s, %s, %s]",
			t.Type, TenantIP, TenantHeader, TenantCookie, TenantQueryParameter)
	}
}

func (e *EndpointRule) validate() error {
	if e.Path == "" {
		return fmt.Errorf("invalid limiter config: path is required for endpoint rule")
	}

	if e.Bypass {
		return nil
	}

	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "HEAD": true, "OPTIONS": true,
	}

	for _, method := range e.Methods {
		method = strings.ToUpper(method)
		if !validMethods[method] {
			return fmt.Errorf("invalid limiter config: invalid HTTP method %s", method)
		}
	}

	if e.TenantStrategy != nil {
		if err := e.TenantStrategy.validate(); err != nil {
			return fmt.Errorf("tenant strategy validation failed for path %s: %w", e.Path, err)
		}
	}

	if err := e.AlgorithmConfig.validate(); err != nil {
		return fmt.Errorf("algorithm config validation failed for path %s: %w", e.Path, err)
	}

	return nil
}

func (l *LoggerConfig) validate() error {
	validLevels := []string{"trace", "debug", "info", "warn", "error", "fatal"}

	valid := false
	for _, level := range validLevels {
		if l.Level == level {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid logger config (level): %s, must be one of %v", l.Level, validLevels)
	}

	if l.Environment != "development" && l.Environment != "production" {
		return fmt.Errorf("invalid logger config (environment): %s, must be %s or %s", l.Environment, "development", "production")
	}

	return nil
}

func (p *ProxyConfig) validate() error {
	if p.TargetUrl == "" {
		return fmt.Errorf("invalid proxy config (target_url): cannot be empty")
	}

	parsedURL, err := url.Parse(p.TargetUrl)
	if err != nil {
		return fmt.Errorf("invalid proxy config (target_url): invalid URL format: %v", err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("invalid proxy config (target_url): URL must include a scheme (http:// or https://)")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid proxy config (target_url): URL scheme must be http or https, got: %s", parsedURL.Scheme)
	}

	const minUserPort = 1024
	const maxPort = 65535

	if p.ProxyPort < minUserPort || p.ProxyPort > maxPort {
		return fmt.Errorf("invalid proxy config (proxy_port): must be between %d and %d, got %d",
			minUserPort, maxPort, p.ProxyPort)
	}

	if p.MetricsPort < minUserPort || p.MetricsPort > maxPort {
		return fmt.Errorf("invalid proxy config (metrics_port): must be between %d and %d, got %d",
			minUserPort, maxPort, p.MetricsPort)
	}

	if p.ProxyPort == p.MetricsPort {
		return fmt.Errorf("invalid proxy config: proxy_port and metrics_port cannot be the same (%d)", p.ProxyPort)
	}

	if p.ServerName == "" {
		return fmt.Errorf("invalid proxy config (server_name): cannot be empty")
	}

	return nil
}

func (l *LimiterConfig) validate() error {
	if l.Global.Enabled {
		if err := l.Global.AlgorithmConfig.validate(); err != nil {
			return fmt.Errorf("global limiter config validation failed: %w", err)
		}
	}

	if l.PerTenant.Enabled {
		if err := l.PerTenant.AlgorithmConfig.validate(); err != nil {
			return fmt.Errorf("per-tenant limiter config validation failed: %w", err)
		}
	}

	seenPaths := make(map[string]bool)
	for i, rule := range l.PerEndpoint.Rules {
		if err := rule.validate(); err != nil {
			return fmt.Errorf("per-endpoint rule %d validation failed: %w", i, err)
		}

		if seenPaths[rule.Path] {
			fmt.Printf("Warning: duplicate path found: %s\n", rule.Path)
		}
		seenPaths[rule.Path] = true
	}

	return nil
}

func (r *RedisConfig) validate() error {
	if r.Address == "" {
		return fmt.Errorf("invalid redis config: address cannot be empty")
	}

	if r.DB < 0 {
		return fmt.Errorf("invalid redis config: db must be >= 0, got %d", r.DB)
	}

	if r.PoolSize <= 0 {
		return fmt.Errorf("invalid redis config: pool_size must be > 0, got %d", r.PoolSize)
	}

	return nil
}
