package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
)

type Logger struct {
	*zap.Logger
}

type Config struct {
	Level       string
	Environment string
	OutputPath  string
}

var Log = mustNewLogger(config.LoggerConfigs)

func mustNewLogger(cfg *config.LoggerConfigsType) *Logger {
	l, err := NewLogger(Config{
		Level:       cfg.Level,
		Environment: cfg.Environment,
		OutputPath:  cfg.OutputPath,
	})
	if err != nil {
		panic(err)
	}
	return l
}

func NewLogger(cfg Config) (*Logger, error) {
	var zapConfig zap.Config

	if cfg.Environment == "production" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}

	if level, err := zapcore.ParseLevel(cfg.Level); err == nil {
		zapConfig.Level.SetLevel(level)
	}

	if cfg.OutputPath != "" && cfg.OutputPath != "stdout" {
		zapConfig.OutputPaths = []string{cfg.OutputPath}
	}

	zapLogger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{zapLogger}, nil
}
