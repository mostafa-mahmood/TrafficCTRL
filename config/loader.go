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
	Global      `yaml:"global"`
	PerTenant   `yaml:"per_tenant"`
	PerEndpoint `yaml:"per_endpoint"`
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
