package main

import (
	"fmt"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("failed to load configurations: %v", err)
	}

	fmt.Println(cfg.Limiter.PerTenant.Algorithm)
}
