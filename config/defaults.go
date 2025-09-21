package config

func intPtr(v int) *int { return &v }

func getProxyDefaults() ProxyConfig {
	return ProxyConfig{
		TargetUrl: "http://localhost:3000",
		ProxyPort: 8080,
	}
}

func getRedisDefaults() RedisConfig {
	return RedisConfig{
		Address:     "localhost:6379",
		Password:    "",
		DB:          0,
		PoolSize:    40,
		KeysTTL:     3600,
		CallTimeout: 500,
	}
}

func getLoggerDefaults() LoggerConfig {
	return LoggerConfig{
		Level:       "info",
		Environment: "development",
		OutputPath:  "stdout",
	}
}

func getLimiterDefaults() LimiterConfig {
	return LimiterConfig{
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
					TenantStrategy: &TenantStrategy{
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
