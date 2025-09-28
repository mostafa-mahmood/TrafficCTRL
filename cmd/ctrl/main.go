package main

import (
	"context"
	"os"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/proxy"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadConfigs()
	if err != nil {
		panic("failed to load configuration: " + err.Error())
	}

	lgr, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		panic("failed to init logger: " + err.Error())
	}
	defer func() {
		_ = lgr.Sync() // flush buffered logs
	}()

	redisClient := limiter.NewRedisClient(cfg.Redis)
	defer func() {
		if err := redisClient.Close(); err != nil {
			lgr.Warn("failed to close redis client", zap.Error(err))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rateLimiter := limiter.NewRateLimiter(redisClient)

	if err := rateLimiter.Ping(ctx); err != nil {
		lgr.Fatal("redis connection failed",
			zap.Error(err),
			zap.String("address", cfg.Redis.Address),
			zap.Int("db", cfg.Redis.DB))
		os.Exit(1)
	}

	lgr.Info("redis connection established",
		zap.String("address", cfg.Redis.Address),
		zap.Int("db", cfg.Redis.DB))

	err = proxy.StartServer(cfg, lgr, rateLimiter)
	if err != nil {
		lgr.Fatal("proxy server failed unexpectedly", zap.Error(err))
		os.Exit(1)
	}
}
