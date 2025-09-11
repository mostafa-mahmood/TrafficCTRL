package config

import (
	"fmt"
	"path/filepath"
	"runtime"
)

var (
	ToolConfigs    *ToolConfigsType
	LoggerConfigs  *LoggerConfigsType
	ProxyConfigs   *ProxyConfigsType
	LimiterConfigs *LimiterConfigsType
)

func getConfigPath(file string) string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("Failed to get caller path")
	}
	dir := filepath.Dir(filename)
	return filepath.Join(dir, file)
}

func getLoggerDefaults() LoggerConfigsType {
	return LoggerConfigsType{
		Level:       "info",
		Environment: "development",
		OutputPath:  "stdout",
	}
}

func getProxyDefaults() ProxyConfigsType {
	return ProxyConfigsType{
		TargetUrl: "http://localhost:3000",
		ProxyPort: 8080,
	}
}

func intPtr(v int) *int { return &v }

func getLimiterDefaults() LimiterConfigsType {
	return LimiterConfigsType{
		Global: Global{
			Enabled: true,
			AlgorithmConfig: AlgorithmConfig{
				Algorithm:    string(TokenBucket),
				Capacity:     intPtr(10000),
				RefillRate:   intPtr(10000),
				RefillPeriod: intPtr(1),
			},
		},
		PerTenant: PerTenant{
			Enabled: true,
			AlgorithmConfig: AlgorithmConfig{
				Algorithm:    string(TokenBucket),
				Capacity:     intPtr(20),
				RefillRate:   intPtr(20),
				RefillPeriod: intPtr(1),
			},
		},
		PerEndpoint: PerEndpoint{
			Rules: []EndpointRules{
				{
					Path: "*",
					TenantStrategy: &TenantStrategiesLimiterConfig{
						Type: string(TenantIP),
					},
					AlgorithmConfig: AlgorithmConfig{
						Algorithm:    string(TokenBucket),
						Capacity:     intPtr(10),
						RefillRate:   intPtr(10),
						RefillPeriod: intPtr(1),
					},
				},
			},
		},
	}
}

func InitConfigs() {
	loadToolConfig()

	if ToolConfigs.UseDefaultConfigs {
		fmt.Println("Tool config: using default values for all configurations")
		useDefaultConfigs()
	} else {
		fmt.Println("Tool config: loading configurations from files")
		loadAllConfigs()
	}
}

func loadToolConfig() {
	cfg, err := configLoader[ToolConfigsType](getConfigPath("tool.yaml"))
	if err != nil {
		fmt.Printf("Warning: couldn't load tool config, using defaults: %v\n", err)
		ToolConfigs = &ToolConfigsType{UseDefaultConfigs: false}
		return
	}
	ToolConfigs = cfg
}

func useDefaultConfigs() {
	loggerDefaults := getLoggerDefaults()
	LoggerConfigs = &loggerDefaults

	proxyDefaults := getProxyDefaults()
	ProxyConfigs = &proxyDefaults

	limiterDefaults := getLimiterDefaults()
	LimiterConfigs = &limiterDefaults
}

func loadAllConfigs() {
	loadLoggerConfigs()
	loadProxyConfigs()
	loadLimiterConfigs()
}

func loadLoggerConfigs() {
	cfg, err := configLoader[LoggerConfigsType](getConfigPath("logger.yaml"))
	if err != nil {
		fmt.Printf("Warning: couldn't load logger config, using defaults: %v\n", err)
		defaults := getLoggerDefaults()
		LoggerConfigs = &defaults
		return
	}

	if err := cfg.Validate(); err != nil {
		fmt.Printf("Warning: logger config validation failed, using defaults: %v\n", err)
		defaults := getLoggerDefaults()
		LoggerConfigs = &defaults
		return
	}

	LoggerConfigs = cfg
	fmt.Println("Successfully loaded logger configuration")
}

func loadProxyConfigs() {
	cfg, err := configLoader[ProxyConfigsType](getConfigPath("proxy.yaml"))
	if err != nil {
		fmt.Printf("Warning: couldn't load proxy config, using defaults: %v\n", err)
		defaults := getProxyDefaults()
		ProxyConfigs = &defaults
		return
	}

	if err := cfg.Validate(); err != nil {
		fmt.Printf("Warning: proxy config validation failed, using defaults: %v\n", err)
		defaults := getProxyDefaults()
		ProxyConfigs = &defaults
		return
	}

	ProxyConfigs = cfg
	fmt.Println("Successfully loaded proxy configuration")
}

func loadLimiterConfigs() {
	cfg, err := configLoader[LimiterConfigsType](getConfigPath("limiter.yaml"))
	if err != nil {
		fmt.Printf("Warning: couldn't load limiter configs, using defaults: %v\n", err)
		defaults := getLimiterDefaults()
		LimiterConfigs = &defaults
		return
	}

	if err := cfg.Validate(); err != nil {
		fmt.Printf("Warning: limiter config validation failed, using defaults: %v\n", err)
		defaults := getLimiterDefaults()
		LimiterConfigs = &defaults
		return
	}

	LimiterConfigs = cfg
	fmt.Println("Successfully loaded limiter configuration")
}
