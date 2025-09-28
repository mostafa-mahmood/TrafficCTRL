package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func getConfigPath(file string) string {
	configDir := os.Getenv("CONFIG_DIR")
	if configDir == "" {
		configDir = "./config"
	}
	return filepath.Join(configDir, file)
}

func LoadConfigs() (*Config, error) {
	loggerCfg, err := loadLoggerConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't load logger config: %v", err)
	}

	redisCfg, err := loadRedisConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't load redis config: %v", err)
	}

	proxyCfg, err := loadProxyConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't load proxy config: %v", err)
	}

	limiterCfg, err := loadLimiterConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't load limiter config: %v", err)
	}

	return &Config{
		Logger:  loggerCfg,
		Proxy:   proxyCfg,
		Limiter: limiterCfg,
		Redis:   redisCfg,
	}, nil
}

func loadLoggerConfig() (*LoggerConfig, error) {
	cfg, err := configLoader[LoggerConfig](getConfigPath("logger.yaml"))
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadRedisConfig() (*RedisConfig, error) {
	cfg, err := configLoader[RedisConfig](getConfigPath("redis.yaml"))
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadProxyConfig() (*ProxyConfig, error) {
	cfg, err := configLoader[ProxyConfig](getConfigPath("proxy.yaml"))
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadLimiterConfig() (*LimiterConfig, error) {
	cfg, err := configLoader[LimiterConfig](getConfigPath("limiter.yaml"))
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
