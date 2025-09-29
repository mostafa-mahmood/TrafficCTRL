package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	cfg, err := loadFromFile[LoggerConfig](getConfigPath("logger.yaml"))
	if err != nil {
		return nil, err
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.Level = level
	}
	if env := os.Getenv("LOG_ENVIRONMENT"); env != "" {
		cfg.Environment = env
	}
	if path := os.Getenv("LOG_OUTPUT_PATH"); path != "" {
		cfg.OutputPath = path
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadRedisConfig() (*RedisConfig, error) {
	cfg, err := loadFromFile[RedisConfig](getConfigPath("redis.yaml"))
	if err != nil {
		return nil, err
	}

	if address := os.Getenv("REDIS_ADDRESS"); address != "" {
		cfg.Address = address
	}
	if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		cfg.Password = password
	}
	if db, ok := parseIntEnv("REDIS_DB"); ok {
		cfg.DB = db
	}
	if poolSize, ok := parseIntEnv("REDIS_POOL_SIZE"); ok {
		cfg.PoolSize = poolSize
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadProxyConfig() (*ProxyConfig, error) {
	cfg, err := loadFromFile[ProxyConfig](getConfigPath("proxy.yaml"))
	if err != nil {
		return nil, err
	}

	if targetUrl := os.Getenv("TARGET_URL"); targetUrl != "" {
		cfg.TargetUrl = targetUrl
	}
	if port, ok := parsePortEnv("PROXY_PORT"); ok {
		cfg.ProxyPort = port
	}
	if port, ok := parsePortEnv("METRICS_PORT"); ok {
		cfg.MetricsPort = port
	}
	if dryRunStr := os.Getenv("DRY_RUN_MODE"); dryRunStr != "" {
		cfg.DryRunMode = dryRunStr == "true"
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadLimiterConfig() (*RateLimiterConfig, error) {
	cfg, err := loadFromFile[RateLimiterConfig](getConfigPath("limiter.yaml"))
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func parsePortEnv(envVar string) (uint16, bool) {
	if s := os.Getenv(envVar); s != "" {
		if port, err := strconv.ParseUint(s, 10, 16); err == nil {
			return uint16(port), true
		}
	}
	return 0, false
}

func parseIntEnv(envVar string) (int, bool) {
	if s := os.Getenv(envVar); s != "" {
		if i, err := strconv.Atoi(s); err == nil {
			return i, true
		}
	}
	return 0, false
}
