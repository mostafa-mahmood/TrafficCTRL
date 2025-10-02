package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/proxy"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEnv struct {
	redisClient *redis.Client
	proxyURL    string
	cfg         *config.Config
	rateLimiter *limiter.RateLimiter
	cleanup     func()
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func setupTestEnvironment(t *testing.T) *testEnv {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Fatalf("Redis connection failed: %v. Make sure Redis is running on localhost:6379", err)
	}

	redisClient.FlushDB(ctx)

	backendURL := "http://localhost:5000"

	proxyPort, err := getFreePort()
	require.NoError(t, err, "Failed to get free proxy port")

	metricsPort, err := getFreePort()
	require.NoError(t, err, "Failed to get free metrics port")

	cfg := createTestConfig(backendURL, proxyPort, metricsPort)

	lgr, err := logger.NewLogger(cfg.Logger)
	require.NoError(t, err)

	rateLimiter := limiter.NewRateLimiter(redisClient)

	proxyAddr := fmt.Sprintf("localhost:%d", proxyPort)

	go func() {
		shutdownSignal := make(chan struct{})
		if err := proxy.StartServer(cfg, lgr, rateLimiter, shutdownSignal); err != nil && err != http.ErrServerClosed {
			t.Logf("Proxy server error: %v", err)
		}
	}()

	// Wait for proxy to be ready
	time.Sleep(500 * time.Millisecond)

	cleanup := func() {
		// Only cleanup Redis test data
		redisClient.FlushDB(context.Background())
		redisClient.Close()
	}

	return &testEnv{
		redisClient: redisClient,
		proxyURL:    "http://" + proxyAddr,
		cfg:         cfg,
		rateLimiter: rateLimiter,
		cleanup:     cleanup,
	}
}

func createTestConfig(targetURL string, proxyPort, metricsPort int) *config.Config {
	capacity100 := 100
	refillRate50 := 50
	refillPeriod10s := &config.Duration{Duration: 10 * time.Second}

	limit20 := 20
	windowSize10s := &config.Duration{Duration: 10 * time.Second}

	limit5 := 5
	windowSize5s := &config.Duration{Duration: 5 * time.Second}

	capacity10 := 10
	refillRate5 := 5
	refillPeriod5s := &config.Duration{Duration: 5 * time.Second}

	capacity15 := 15
	leakRate3 := 3
	leakPeriod5s := &config.Duration{Duration: 5 * time.Second}

	limit30 := 30

	limiterCfg := &config.RateLimiterConfig{
		Global: config.Global{
			Enabled: true,
			AlgorithmConfig: config.AlgorithmConfig{
				Algorithm:    "token_bucket",
				Capacity:     &capacity100,
				RefillRate:   &refillRate50,
				RefillPeriod: refillPeriod10s,
			},
		},
		PerTenant: config.PerTenant{
			Enabled: true,
			AlgorithmConfig: config.AlgorithmConfig{
				Algorithm:  "sliding_window",
				WindowSize: windowSize10s,
				Limit:      &limit20,
			},
		},
		PerEndpoint: config.PerEndpoint{
			Rules: []config.EndpointRule{
				{
					Path:    "/api/auth/login",
					Methods: []string{"POST"},
					TenantStrategy: &config.TenantStrategy{
						Type: "ip",
					},
					AlgorithmConfig: config.AlgorithmConfig{
						Algorithm:  "fixed_window",
						WindowSize: windowSize5s,
						Limit:      &limit5,
					},
				},
				{
					Path: "/api/users/*",
					TenantStrategy: &config.TenantStrategy{
						Type: "header",
						Key:  "Authorization",
					},
					AlgorithmConfig: config.AlgorithmConfig{
						Algorithm:    "token_bucket",
						Capacity:     &capacity10,
						RefillRate:   &refillRate5,
						RefillPeriod: refillPeriod5s,
					},
				},
				{
					Path:    "/api/upload",
					Methods: []string{"POST"},
					TenantStrategy: &config.TenantStrategy{
						Type: "header",
						Key:  "X-API-Key",
					},
					AlgorithmConfig: config.AlgorithmConfig{
						Algorithm:  "leaky_bucket",
						Capacity:   &capacity15,
						LeakRate:   &leakRate3,
						LeakPeriod: leakPeriod5s,
					},
				},
				{
					Path:   "/health",
					Bypass: true,
				},
				{
					Path: "*",
					TenantStrategy: &config.TenantStrategy{
						Type: "ip",
					},
					AlgorithmConfig: config.AlgorithmConfig{
						Algorithm:  "sliding_window",
						WindowSize: windowSize10s,
						Limit:      &limit30,
					},
				},
			},
		},
	}

	return &config.Config{
		Proxy: &config.ProxyConfig{
			TargetUrl:   targetURL,
			ProxyPort:   uint16(proxyPort),
			MetricsPort: uint16(metricsPort),
			ServerName:  "trafficctrl-test",
			DryRunMode:  false,
		},
		Limiter: limiterCfg,
		Redis: &config.RedisConfig{
			Address:  "localhost:6379",
			Password: "",
			DB:       15,
			PoolSize: 10,
		},
		Logger: &config.LoggerConfig{
			Level:       "error",
			Environment: "production",
		},
	}
}

func makeRequest(t *testing.T, method, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	require.NoError(t, err)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	return client.Do(req)
}

func TestBypassEndpoint(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	for i := 0; i < 100; i++ {
		resp, err := makeRequest(t, "GET", env.proxyURL+"/health", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	}
}

func TestIPBasedRateLimiting(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	successCount := 0
	deniedCount := 0

	for i := 0; i < 10; i++ {
		resp, err := makeRequest(t, "POST", env.proxyURL+"/api/auth/login", nil)
		require.NoError(t, err)

		if resp.StatusCode == http.StatusOK {
			successCount++
		} else if resp.StatusCode == http.StatusTooManyRequests {
			deniedCount++

			var body map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&body)
			assert.Equal(t, "rate limit exceeded", body["error"])
			assert.NotNil(t, body["retry_after"])
		}
		resp.Body.Close()
	}

	assert.Equal(t, 5, successCount, "Should allow exactly 5 requests")
	assert.Equal(t, 5, deniedCount, "Should deny 5 requests")
}

func TestHeaderBasedTenantStrategy(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	users := []string{"user1-token", "user2-token", "user3-token"}

	for _, userToken := range users {
		successCount := 0

		for i := 0; i < 12; i++ {
			resp, err := makeRequest(t, "GET", env.proxyURL+"/api/users/profile", map[string]string{
				"Authorization": userToken,
			})
			require.NoError(t, err)

			if resp.StatusCode == http.StatusOK {
				successCount++
			}
			resp.Body.Close()
			time.Sleep(10 * time.Millisecond)
		}

		assert.GreaterOrEqual(t, successCount, 10, "User %s should make at least 10 requests", userToken)
	}
}

func TestTenantIsolation(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	apiKey1 := "key-123"
	apiKey2 := "key-456"

	// User 1 exhausts their limit
	for i := 0; i < 15; i++ {
		resp, _ := makeRequest(t, "POST", env.proxyURL+"/api/upload", map[string]string{
			"X-API-Key": apiKey1,
		})
		resp.Body.Close()
	}

	successCount := 0
	for i := 0; i < 15; i++ {
		resp, err := makeRequest(t, "POST", env.proxyURL+"/api/upload", map[string]string{
			"X-API-Key": apiKey2,
		})
		require.NoError(t, err)

		if resp.StatusCode == http.StatusOK {
			successCount++
		}
		resp.Body.Close()
	}

	assert.Equal(t, 15, successCount, "User 2 should have independent limit")
}

func TestConcurrentRequests(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	var wg sync.WaitGroup
	numWorkers := 10
	requestsPerWorker := 5

	successCount := 0
	deniedCount := 0
	var mu sync.Mutex

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < requestsPerWorker; i++ {
				resp, err := makeRequest(t, "POST", env.proxyURL+"/api/auth/login", nil)
				if err != nil {
					continue
				}

				mu.Lock()
				if resp.StatusCode == http.StatusOK {
					successCount++
				} else if resp.StatusCode == http.StatusTooManyRequests {
					deniedCount++
				}
				mu.Unlock()

				resp.Body.Close()
				time.Sleep(10 * time.Millisecond)
			}
		}(w)
	}

	wg.Wait()

	assert.LessOrEqual(t, successCount, 6, "Should not exceed limit significantly")
	assert.Greater(t, deniedCount, 40, "Most requests should be denied")
}

func TestTokenBucketRefill(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	authToken := "refill-test-user"

	for i := 0; i < 10; i++ {
		resp, _ := makeRequest(t, "GET", env.proxyURL+"/api/users/data", map[string]string{
			"Authorization": authToken,
		})
		resp.Body.Close()
	}

	resp, _ := makeRequest(t, "GET", env.proxyURL+"/api/users/data", map[string]string{
		"Authorization": authToken,
	})
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	resp.Body.Close()

	time.Sleep(6 * time.Second)

	successCount := 0
	for i := 0; i < 7; i++ {
		resp, _ := makeRequest(t, "GET", env.proxyURL+"/api/users/data", map[string]string{
			"Authorization": authToken,
		})
		if resp.StatusCode == http.StatusOK {
			successCount++
		}
		resp.Body.Close()
	}

	assert.GreaterOrEqual(t, successCount, 5, "Should have refilled ~5 tokens")
}

func TestPerTenantLimitAcrossEndpoints(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	endpoints := []string{
		"/api/data",
		"/api/info",
		"/api/stats",
	}

	totalSuccess := 0
	totalDenied := 0

	for i := 0; i < 25; i++ {
		endpoint := endpoints[i%len(endpoints)]
		resp, _ := makeRequest(t, "GET", env.proxyURL+endpoint, nil)

		if resp.StatusCode == http.StatusOK {
			totalSuccess++
		} else if resp.StatusCode == http.StatusTooManyRequests {
			totalDenied++
		}
		resp.Body.Close()
		time.Sleep(50 * time.Millisecond)
	}

	assert.LessOrEqual(t, totalSuccess, 21, "Should respect per-tenant limit")
}
