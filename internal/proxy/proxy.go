package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
)

// injects standard X-Forwarded-* headers for downstream services.
func createProxy(cfg *config.Config) (*httputil.ReverseProxy, error) {
	targetURL, err := url.Parse(cfg.Proxy.TargetUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid target url: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director

	proxy.Director = func(req *http.Request) {
		addForwardedHost(req)
		addForwardedPort(req)
		addForwardedProto(req)
		addForwardedServer(req, cfg.Proxy.ServerName)
		originalDirector(req)
	}

	return proxy, nil
}

// X-Forwarded-Proto: <preserve protocol the client originally used>
func addForwardedProto(req *http.Request) {
	if header := req.Header.Get("X-Forwarded-Proto"); header != "" {
		return
	}
	// Since TrafficCTRL does not terminate TLS, assume "http" unless behind a TLS terminator (e.g., nginx).
	req.Header.Set("X-Forwarded-Proto", "http")
}

// X-Forwarded-Host: <preserve original Host header client used>
func addForwardedHost(req *http.Request) {
	if header := req.Header.Get("X-Forwarded-Host"); header != "" {
		return
	}

	req.Header.Set("X-Forwarded-Host", req.Host)
}

// X-Forwarded-Port: <preserve original Port header client connected to>
func addForwardedPort(req *http.Request) {
	if existing := req.Header.Get("X-Forwarded-Port"); existing != "" {
		return
	}

	// Extract port from the original host
	_, port, err := net.SplitHostPort(req.Host)
	if err != nil {
		// No explicit port, determine from X-Forwarded-Proto or assume defaults
		if proto := req.Header.Get("X-Forwarded-Proto"); proto == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	req.Header.Set("X-Forwarded-Port", port)
}

func addForwardedServer(req *http.Request, serverName string) {
	req.Header.Set("X-Forwarded-Server", serverName)
}
