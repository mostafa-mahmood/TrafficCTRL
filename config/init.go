package config

import (
	"fmt"
)

var (
	ToolConfigs   *ToolConfigsType
	LoggerConfigs *LoggerConfigsType
	ProxyConfigs  *ProxyConfigsType
)

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

func getToolDefaults() ToolConfigsType {
	return ToolConfigsType{
		UseDefaultConfigs: false,
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
	cfg, err := configLoader[ToolConfigsType]("config/tool.yaml")
	if err != nil {
		fmt.Printf("Warning: couldn't load tool config, using defaults: %v\n", err)
		ToolConfigs = &ToolConfigsType{UseDefaultConfigs: false}
		return
	}
	ToolConfigs = cfg
}

func useDefaultConfigs() {
	LoggerConfigs = &LoggerConfigsType{
		Level:       getLoggerDefaults().Level,
		Environment: getLoggerDefaults().Environment,
		OutputPath:  getLoggerDefaults().OutputPath,
	}

	ProxyConfigs = &ProxyConfigsType{
		TargetUrl: getProxyDefaults().TargetUrl,
		ProxyPort: getProxyDefaults().ProxyPort,
	}
}

func loadAllConfigs() {
	loadLoggerConfigsWithFallback()
	loadProxyConfigsWithFallback()
}

func loadLoggerConfigsWithFallback() {
	cfg, err := configLoader[LoggerConfigsType]("config/logger.yaml")
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

func loadProxyConfigsWithFallback() {
	cfg, err := configLoader[ProxyConfigsType]("config/proxy.yaml")
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
