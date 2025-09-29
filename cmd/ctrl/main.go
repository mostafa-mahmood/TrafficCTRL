package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/proxy"
	"go.uber.org/zap"
)

const trafficCtrlArt = `
 _____           __  __ _      ____ _____ ____  _     
|_   _| __ __ _ / _|/ _(_) ___ / ___|_   _|  _ \| |    
  | || '__/ _  | |_| |_| |/ __| |     | | | |_) | |    
  | || | | (_| |  _|  _| | (__| |___  | | |  _ <| |___ 
  |_||_|  \__,_|_| |_| |_|\___|\____| |_| |_| \_\_____|
`

func main() {
	fmt.Printf("%s \n", trafficCtrlArt)

	fmt.Printf("TrafficCTRL v%s starting...\n\n", "0.1.0")

	cfg, err := config.LoadConfigs()
	if err != nil {
		panic("failed to load configuration, terminating process: " + err.Error())
	}

	lgr, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		panic("failed to init logger, terminating process: " + err.Error())
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
		lgr.Fatal("redis connection failed, terminating process",
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
		lgr.Fatal("server failed unexpectedly", zap.Error(err))
		os.Exit(1)
	}
}
