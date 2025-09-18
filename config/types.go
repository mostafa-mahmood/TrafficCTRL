package config

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

type Config struct {
	Logger  *LoggerConfig
	Proxy   *ProxyConfig
	Limiter *LimiterConfig
}

type LoggerConfig struct {
	Level       string `yaml:"level"`
	Environment string `yaml:"environment"`
	OutputPath  string `yaml:"output_path"`
}

type ProxyConfig struct {
	TargetUrl string `yaml:"target_url"`
	ProxyPort uint16 `yaml:"proxy_port"`
}

type LimiterConfig struct {
	Global      Global      `yaml:"global"`
	PerTenant   PerTenant   `yaml:"per_tenant"`
	PerEndpoint PerEndpoint `yaml:"per_endpoint"`
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
	Rules []EndpointRules `yaml:"rules"`
}

type TenantStrategy struct {
	Type string `yaml:"type" validate:"required"`
	Key  string `yaml:"key,omitempty"`
}

type EndpointRules struct {
	Path            string          `yaml:"path" validate:"required"`
	Methods         []string        `yaml:"methods,omitempty"`
	Bypass          bool            `yaml:"bypass,omitempty"`
	TenantStrategy  *TenantStrategy `yaml:"tenant_strategy,omitempty"`
	AlgorithmConfig `yaml:",inline"`
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

type toolConfig struct {
	UseDefaultConfigs bool `yaml:"use_default_configs"`
}
