package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type ToolConfigsType struct {
	UseDefaultConfigs bool `yaml:"use_default_configs"`
}

type LoggerConfigsType struct {
	Level       string `yaml:"level"`
	Environment string `yaml:"environment"`
	OutputPath  string `yaml:"output_path"`
}

type ProxyConfigsType struct {
	TargetUrl string `yaml:"target_url"`
	ProxyPort uint16 `yaml:"proxy_port"`
}

type LimiterConfigsType struct {
	GlobalLimiterConfig      `yaml:"global"`
	PerTenantLimiterConfig   `yaml:"per_tenant"`
	PerEndpointLimiterConfig `yaml:"per_endpoint"`
}

type GlobalLimiterConfig struct {
	Enabled         bool `yaml:"enabled"`
	AlgorithmConfig `yaml:",inline"`
}

type PerTenantLimiterConfig struct {
	Enabled         bool `yaml:"enabled"`
	AlgorithmConfig `yaml:",inline"`
}

type PerEndpointLimiterConfig struct {
	Rules []EndpointRulesLimiterConfig `yaml:"rules"`
}

type TenantStrategiesLimiterConfig struct {
	Type string `yaml:"type" validate:"required"`
	Key  string `yaml:"key,omitempty"`
}

type EndpointRulesLimiterConfig struct {
	Path            string                         `yaml:"path" validate:"required"`
	Methods         []string                       `yaml:"methods,omitempty"`
	Bypass          bool                           `yaml:"bypass,omitempty"`
	TenantStrategy  *TenantStrategiesLimiterConfig `yaml:"tenant_strategy,omitempty"`
	AlgorithmConfig `yaml:",inline"`
}

func configLoader[T any](path string) (*T, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't open file %s: %w", path, err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)

	cfg := new(T)

	err = decoder.Decode(cfg)
	if err != nil {
		return nil, fmt.Errorf("couldn't decode file %s: %w", path, err)
	}

	return cfg, nil
}
