package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type LoggerConfigsType struct {
	Level       string `yaml:"level"`
	Environment string `yaml:"environment"`
	OutputPath  string `yaml:"output_path"`
}

func loadLoggerConfigs() *LoggerConfigsType {
	path := "config/logger.yaml"

	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("couldn't open file: %s, err: %v", path, err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)

	cfg := &LoggerConfigsType{}
	err = decoder.Decode(cfg)
	if err != nil {
		log.Fatalf("couldn't read file: %s, err: %v", path, err)
	}

	return cfg
}

var LoggerConfigs *LoggerConfigsType = loadLoggerConfigs()
