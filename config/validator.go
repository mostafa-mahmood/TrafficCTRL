package config

import (
	"fmt"
	"strings"
)

type AlgorithmType string

const (
	TokenBucket   AlgorithmType = "token_bucket"
	LeakyBucket   AlgorithmType = "leaky_bucket"
	FixedWindow   AlgorithmType = "fixed_window"
	SlidingWindow AlgorithmType = "sliding_window"
)

type TenantStrategyType string

const (
	TenantIP             TenantStrategyType = "ip"
	TenantHeader         TenantStrategyType = "header"
	TenantCookie         TenantStrategyType = "cookie"
	TenantQueryParameter TenantStrategyType = "query_parameter"
)

type Validator interface {
	Validate() error
}

type AlgorithmConfig struct {
	Algorithm string `yaml:"algorithm" validate:"required"`

	Capacity     *int `yaml:"capacity,omitempty"`
	RefillRate   *int `yaml:"refill_rate,omitempty"`
	RefillPeriod *int `yaml:"refill_period,omitempty"`

	LeakRate   *int `yaml:"leak_rate,omitempty"`
	LeakPeriod *int `yaml:"leak_period,omitempty"`

	WindowSize *int `yaml:"window_size,omitempty"`
	Limit      *int `yaml:"limit,omitempty"`
}

func (a *AlgorithmConfig) Validate() error {
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
	if *a.RefillPeriod <= 0 {
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
	if *a.LeakPeriod <= 0 {
		return fmt.Errorf("invalid limiter config: leak_period must be positive, got: %d", *a.LeakPeriod)
	}

	return nil
}

func (a *AlgorithmConfig) validateFixedWindow() error {
	if a.WindowSize == nil {
		return fmt.Errorf("invalid limiter config: window_size is required for fixed_window algorithm")
	}
	if *a.WindowSize <= 0 {
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
	if *a.WindowSize <= 0 {
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

func (t *TenantStrategy) Validate() error {
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

func (e *EndpointRules) Validate() error {
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
		if err := e.TenantStrategy.Validate(); err != nil {
			return fmt.Errorf("tenant strategy validation failed for path %s: %w", e.Path, err)
		}
	}

	if err := e.AlgorithmConfig.Validate(); err != nil {
		return fmt.Errorf("algorithm config validation failed for path %s: %w", e.Path, err)
	}

	return nil
}

func (l *LoggerConfigsType) Validate() error {
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

func (p *ProxyConfigsType) Validate() error {
	if p.TargetUrl == "" {
		return fmt.Errorf("invalid proxy config (target_url): cannot be empty")
	}

	if p.ProxyPort == 0 {
		return fmt.Errorf("invalid proxy config (proxy_port): cannot be zero")
	}

	return nil
}

func (l *LimiterConfigsType) Validate() error {
	if l.Global.Enabled {
		if err := l.Global.AlgorithmConfig.Validate(); err != nil {
			return fmt.Errorf("global limiter config validation failed: %w", err)
		}
	}

	if l.PerTenant.Enabled {
		if err := l.PerTenant.AlgorithmConfig.Validate(); err != nil {
			return fmt.Errorf("per-tenant limiter config validation failed: %w", err)
		}
	}

	seenPaths := make(map[string]bool)
	for i, rule := range l.PerEndpoint.Rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("per-endpoint rule %d validation failed: %w", i, err)
		}

		if seenPaths[rule.Path] {
			fmt.Printf("Warning: duplicate path found: %s\n", rule.Path)
		}
		seenPaths[rule.Path] = true
	}

	return nil
}
