package config

import (
	"fmt"
	"time"
)

type LimitLevelType string

const (
	GlobalLevel      LimitLevelType = "global"
	PerTenantLevel   LimitLevelType = "per_tenant"
	PerEndpointLevel LimitLevelType = "per_endpoint"
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

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw string
	if err := unmarshal(&raw); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", raw, err)
	}
	d.Duration = parsed
	return nil
}

type Config struct {
	Tool    *ToolConfig
	Proxy   *ProxyConfig
	Limiter *LimiterConfig
	Redis   *RedisConfig
	Logger  *LoggerConfig
}

type ProxyConfig struct {
	TargetUrl   string `yaml:"target_url"`
	ProxyPort   uint16 `yaml:"proxy_port"`
	ServerName  string `yaml:"server_name"`
	MetricsPort uint16 `yaml:"metrics_port"`
}

type LimiterConfig struct {
	Global      Global      `yaml:"global"`
	PerTenant   PerTenant   `yaml:"per_tenant"`
	PerEndpoint PerEndpoint `yaml:"per_endpoint"`
}

type RedisConfig struct {
	Address  string `yaml:"address"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

type LoggerConfig struct {
	Level       string `yaml:"level"`
	Environment string `yaml:"environment"`
	OutputPath  string `yaml:"output_path"`
}

type Global struct {
	Enabled         bool `yaml:"enabled"`
	AlgorithmConfig `yaml:",inline"`
}

type PerTenant struct {
	Enabled         bool `yaml:"enabled"`
	AlgorithmConfig `yaml:",inline"`
}

type PerEndpoint struct {
	Rules []EndpointRule `yaml:"rules"`
}

type TenantStrategy struct {
	Type string `yaml:"type" validate:"required"`
	Key  string `yaml:"key,omitempty"`
}

type EndpointRule struct {
	Path            string          `yaml:"path" validate:"required"`
	Methods         []string        `yaml:"methods,omitempty"`
	Bypass          bool            `yaml:"bypass,omitempty"`
	TenantStrategy  *TenantStrategy `yaml:"tenant_strategy,omitempty"`
	AlgorithmConfig `yaml:",inline"`
}

type AlgorithmConfig struct {
	Algorithm string `yaml:"algorithm" validate:"required"`

	Capacity     *int      `yaml:"capacity,omitempty"`
	RefillRate   *int      `yaml:"refill_rate,omitempty"`
	RefillPeriod *Duration `yaml:"refill_period,omitempty"`

	LeakRate   *int      `yaml:"leak_rate,omitempty"`
	LeakPeriod *Duration `yaml:"leak_period,omitempty"`

	WindowSize *Duration `yaml:"window_size,omitempty"`
	Limit      *int      `yaml:"limit,omitempty"`
}

type ToolConfig struct {
	UseDefaultConfigs bool `yaml:"use_default_configs"`
	DryRunMode        bool `yaml:"dry_run_mode"`
}
