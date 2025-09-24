package main

import (
	"context"
	"log"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/proxy"
	"go.uber.org/zap"
)

func main() {
	configs, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("error loading configuration, server terminated: %v", err)
	}

	logger, err := logger.NewLogger(configs.Logger)
	if err != nil {
		log.Fatalf("error initializing logger, server terminated: %v", err)
	}

	redisClient := limiter.NewRedisClient(configs.Redis)

	rateLimiter := limiter.NewRateLimiter(redisClient)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rateLimiter.Ping(ctx); err != nil {
		log.Fatalf("error connecting to redis, server terminated: %v", err)
	} else {
		logger.Info("connected to redis successfully",
			zap.String("addr", configs.Redis.Address),
			zap.Int("db", configs.Redis.DB),
		)
	}

	err = proxy.BootStrap(configs, logger, rateLimiter)
	if err != nil {
		log.Fatalf("error bootstraping server, server terminated: %v", err)
	}
}
