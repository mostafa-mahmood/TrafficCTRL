package main

import (
	"log"
	"net/url"

	"github.com/mostafa-mahmood/TrafficCTRL/internal/proxy"
)

func main() {

	targetUrl, err := url.Parse("http://localhost:5000")
	if err != nil {
		log.Fatal("fatal: invalid target url")
	}

	err = proxy.StartServer(3000, targetUrl)

	if err != nil {
		log.Fatal("fatal: error starting server")
	}
}
