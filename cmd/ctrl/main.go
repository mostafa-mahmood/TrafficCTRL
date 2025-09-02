package main

import (
	"log"
	"net/url"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/proxy"
)

func main() {
	config.InitConfigs()
	logger.InitLogger()

	targetUrl, err := url.Parse(config.ProxyConfigs.TargetUrl)
	if err != nil {
		log.Fatalf("couldn't parse target url: %v", err)
	}

	port := config.ProxyConfigs.ProxyPort

	err = proxy.StartServer(port, targetUrl)
	if err != nil {
		log.Fatalf("error on starting server: %v", err)
	}
}
