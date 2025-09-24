package config

import (
	"fmt"
	"path/filepath"
	"runtime"
)

func getConfigPath(file string) string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("[CONFIG] failed to get caller path")
	}
	dir := filepath.Dir(filename)
	return filepath.Join(dir, file)
}

func LoadConfig() (*Config, error) {
	toolCfg, err := loadToolConfig()
	useDefaults := err != nil || toolCfg.UseDefaultConfigs

	if useDefaults {
		fmt.Println("[CONFIG] Using default configuration values")
		return useDefaultConfigs(), nil
	}

	fmt.Println("[CONFIG] Loading configuration from files")
	return loadAllConfigs()
}

func loadToolConfig() (*toolConfig, error) {
	cfg, err := configLoader[toolConfig](getConfigPath("tool.yaml"))
	if err != nil {
		fmt.Printf("[CONFIG] couldn't load tool config: %v, using defaults\n", err)
		return &toolConfig{UseDefaultConfigs: false}, err
	}
	return cfg, nil
}

func useDefaultConfigs() *Config {
	loggerDefaults := getLoggerDefaults()
	proxyDefaults := getProxyDefaults()
	limiterDefaults := getLimiterDefaults()
	redisDefaults := getRedisDefaults()

	return &Config{
		Logger:  &loggerDefaults,
		Proxy:   &proxyDefaults,
		Limiter: &limiterDefaults,
		Redis:   &redisDefaults,
	}
}

func loadAllConfigs() (*Config, error) {
	loggerCfg, err := loadLoggerConfig()
	if err != nil {
		fmt.Printf("[CONFIG] couldn't load logger config: %v, using defaults\n", err)
		defaults := getLoggerDefaults()
		loggerCfg = &defaults
	}

	redisCfg, err := loadRedisConfig()
	if err != nil {
		fmt.Printf("[CONFIG] couldn't load redis config: %v, using defaults\n", err)
		defaults := getRedisDefaults()
		redisCfg = &defaults
	}

	proxyCfg, err := loadProxyConfig()
	if err != nil {
		fmt.Printf("[CONFIG] couldn't load proxy config: %v, using defaults\n", err)
		defaults := getProxyDefaults()
		proxyCfg = &defaults
	}

	limiterCfg, err := loadLimiterConfig()
	if err != nil {
		fmt.Printf("[CONFIG] couldn't load limiter config: %v, using defaults\n", err)
		defaults := getLimiterDefaults()
		limiterCfg = &defaults
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

	fmt.Println("[CONFIG] Logger configuration loaded")
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

	fmt.Println("[CONFIG] Redis configuration loaded")
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

	fmt.Println("[CONFIG] Proxy configuration loaded")
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

	fmt.Println("[CONFIG] Rate limiter configuration loaded")
	return cfg, nil
}
