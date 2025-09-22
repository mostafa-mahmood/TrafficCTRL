package main

import (
	"log"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/proxy"
)

func main() {
	configs, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("error loading configuration, terminated: %v", err)
	}

	logger, err := logger.NewLogger(configs.Logger)
	if err != nil {
		log.Fatalf("error initializing logger, terminated: %v", err)
	}

	redisClient := limiter.NewRedisClient(configs.Redis)

	rateLimiter := limiter.NewRateLimiter(redisClient)

	err = proxy.BootStrap(configs, logger, rateLimiter)
	if err != nil {
		log.Fatalf("error bootstraping server, terminated: %v", err)
	}
}
