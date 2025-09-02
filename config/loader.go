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
