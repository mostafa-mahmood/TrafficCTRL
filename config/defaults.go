package config

import "time"

func intPtr(v int) *int { return &v }

func getProxyDefaults() ProxyConfig {
	return ProxyConfig{
		TargetUrl: "http://localhost:3000",
		ProxyPort: 8080,
	}
}

func getRedisDefaults() RedisConfig {
	return RedisConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
		PoolSize: 40,
	}
}

func getLoggerDefaults() LoggerConfig {
	return LoggerConfig{
		Level:       "info",
		Environment: "production",
		OutputPath:  "stdout",
	}
}

func durationPtr(d time.Duration) *Duration {
	return &Duration{Duration: d}
}

func getLimiterDefaults() LimiterConfig {
	return LimiterConfig{
		Global: Global{
			Enabled: true,
			AlgorithmConfig: AlgorithmConfig{
				Algorithm:    string(TokenBucket),
				Capacity:     intPtr(10000),
				RefillRate:   intPtr(10000),
				RefillPeriod: durationPtr(1 * time.Minute),
			},
		},
		PerTenant: PerTenant{
			Enabled: true,
			AlgorithmConfig: AlgorithmConfig{
				Algorithm:    string(TokenBucket),
				Capacity:     intPtr(20),
				RefillRate:   intPtr(20),
				RefillPeriod: durationPtr(1 * time.Minute),
			},
		},
		PerEndpoint: PerEndpoint{
			Rules: []EndpointRule{
				{
					Path: "*",
					TenantStrategy: &TenantStrategy{
						Type: string(TenantIP),
					},
					AlgorithmConfig: AlgorithmConfig{
						Algorithm:    string(TokenBucket),
						Capacity:     intPtr(10),
						RefillRate:   intPtr(10),
						RefillPeriod: durationPtr(1 * time.Minute),
					},
				},
			},
		},
	}
}
