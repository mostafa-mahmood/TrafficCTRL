package config

import (
	"fmt"
	"path/filepath"
	"runtime"
)

func getConfigPath(file string) string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("Failed to get caller path")
	}
	dir := filepath.Dir(filename)
	return filepath.Join(dir, file)
}

func LoadConfig() (*Config, error) {
	toolCfg, err := loadToolConfig()
	useDefaults := err != nil || toolCfg.UseDefaultConfigs

	if useDefaults {
		fmt.Println("Tool config: using default values for all configurations")
		return useDefaultConfigs(), nil
	}

	fmt.Println("Tool config: loading configurations from files")
	return loadAllConfigs()
}

func loadToolConfig() (*toolConfig, error) {
	cfg, err := configLoader[toolConfig](getConfigPath("tool.yaml"))
	if err != nil {
		fmt.Printf("Warning: couldn't load tool config, using defaults: %v\n", err)
		return &toolConfig{UseDefaultConfigs: false}, err
	}
	return cfg, nil
}

func useDefaultConfigs() *Config {
	loggerDefaults := getLoggerDefaults()
	proxyDefaults := getProxyDefaults()
	limiterDefaults := getLimiterDefaults()

	return &Config{
		Logger:  &loggerDefaults,
		Proxy:   &proxyDefaults,
		Limiter: &limiterDefaults,
	}
}

func loadAllConfigs() (*Config, error) {
	loggerCfg, err := loadLoggerConfig()
	if err != nil {
		fmt.Printf("Warning: couldn't load logger config, using defaults: %v\n", err)
		defaults := getLoggerDefaults()
		loggerCfg = &defaults
	}

	proxyCfg, err := loadProxyConfig()
	if err != nil {
		fmt.Printf("Warning: couldn't load proxy config, using defaults: %v\n", err)
		defaults := getProxyDefaults()
		proxyCfg = &defaults
	}

	limiterCfg, err := loadLimiterConfig()
	if err != nil {
		fmt.Printf("Warning: couldn't load limiter config, using defaults: %v\n", err)
		defaults := getLimiterDefaults()
		limiterCfg = &defaults
	}

	return &Config{
		Logger:  loggerCfg,
		Proxy:   proxyCfg,
		Limiter: limiterCfg,
	}, nil
}

func loadLoggerConfig() (*LoggerConfig, error) {
	cfg, err := configLoader[LoggerConfig](getConfigPath("logger.yaml"))
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("logger config validation failed: %w", err)
	}

	fmt.Println("Successfully loaded logger configuration")
	return cfg, nil
}

func loadProxyConfig() (*ProxyConfig, error) {
	cfg, err := configLoader[ProxyConfig](getConfigPath("proxy.yaml"))
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("proxy config validation failed: %w", err)
	}

	fmt.Println("Successfully loaded proxy configuration")
	return cfg, nil
}

func loadLimiterConfig() (*LimiterConfig, error) {
	cfg, err := configLoader[LimiterConfig](getConfigPath("limiter.yaml"))
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("limiter config validation failed: %w", err)
	}

	fmt.Println("Successfully loaded limiter configuration")
	return cfg, nil
}
