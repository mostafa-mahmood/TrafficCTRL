package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

func loadFromFile[T any](path string) (*T, error) {
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
