package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type E2ETestSuite struct {
	suite.Suite
	redisClient  *redis.Client
	rateLimiter  *limiter.RateLimiter
	backend      *httptest.Server
	proxyConfig  *config.Config
	logger       *logger.Logger
	cleanupFuncs []func()
}

func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

func (s *E2ETestSuite) SetupSuite() {
	s.setupRedis()

	s.setupBackend()

	s.setupConfig()

	s.setupLogger()

	s.rateLimiter = limiter.NewRateLimiter(s.redisClient)
}

func (s *E2ETestSuite) TearDownSuite() {
	for _, cleanup := range s.cleanupFuncs {
		cleanup()
	}
	s.redisClient.Close()
	s.backend.Close()
}

func (s *E2ETestSuite) setupRedis() {
	s.redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.redisClient.Ping(ctx).Err()
	require.NoError(s.T(), err, "Redis must be running for e2e tests")

	err = s.redisClient.FlushDB(ctx).Err()
	require.NoError(s.T(), err)

	s.cleanupFuncs = append(s.cleanupFuncs, func() {
		s.redisClient.FlushDB(context.Background())
	})
}

func (s *E2ETestSuite) setupBackend() {
	s.backend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"message":    "backend response",
			"path":       r.URL.Path,
			"method":     r.Method,
			"forwarded":  r.Header.Get("X-Forwarded-For"),
			"request_id": r.Header.Get("X-Request-ID"),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

func (s *E2ETestSuite) setupConfig() {
	s.proxyConfig = &config.Config{
		Proxy: &config.ProxyConfig{
			TargetUrl:   s.backend.URL,
			ProxyPort:   8080,
			MetricsPort: 8081,
			ServerName:  "TrafficCTRL-E2E-Test",
			DryRunMode:  false,
		},
		Redis: &config.RedisConfig{
			Address:  "localhost:6379",
			DB:       1,
			PoolSize: 10,
		},
		Logger: &config.LoggerConfig{
			Level:       "error",
			Environment: "test",
			OutputPath:  "stdout",
		},
		Limiter: &config.RateLimiterConfig{
			Global: config.Global{
				Enabled: true,
				AlgorithmConfig: config.AlgorithmConfig{
					Algorithm:    "token_bucket",
					Capacity:     intPtr(100),
					RefillRate:   intPtr(50),
					RefillPeriod: &config.Duration{Duration: time.Minute},
				},
			},
			PerTenant: config.PerTenant{
				Enabled: true,
				AlgorithmConfig: config.AlgorithmConfig{
					Algorithm:  "sliding_window",
					WindowSize: &config.Duration{Duration: time.Minute},
					Limit:      intPtr(30),
				},
			},
			PerEndpoint: config.PerEndpoint{
				Rules: []config.EndpointRule{
					{
						Path:    "/api/v1/login",
						Methods: []string{"POST"},
						TenantStrategy: &config.TenantStrategy{
							Type: "ip",
						},
						AlgorithmConfig: config.AlgorithmConfig{
							Algorithm:  "fixed_window",
							WindowSize: &config.Duration{Duration: time.Minute},
							Limit:      intPtr(5),
						},
					},
					{
						Path: "/api/v1/users/*",
						TenantStrategy: &config.TenantStrategy{
							Type: "header",
							Key:  "X-API-Key",
						},
						AlgorithmConfig: config.AlgorithmConfig{
							Algorithm:    "token_bucket",
							Capacity:     intPtr(20),
							RefillRate:   intPtr(10),
							RefillPeriod: &config.Duration{Duration: time.Minute},
						},
					},
					{
						Path:   "/health",
						Bypass: true,
					},
				},
			},
		},
	}
}

func (s *E2ETestSuite) setupLogger() {
	var err error
	s.logger, err = logger.NewLogger(s.proxyConfig.Logger)
	require.NoError(s.T(), err)
}

func (s *E2ETestSuite) newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

func intPtr(i int) *int {
	return &i
}
