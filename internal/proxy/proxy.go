package proxy

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func createProxy(targetUrl *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(targetUrl)

	// Director is responsable for editing the request headers
	originalDirector := proxy.Director

	proxy.Director = func(req *http.Request) {
		addForwardedHost(req)
		addForwardedPort(req)
		addForwardedProto(req)
		addForwardedServer(req)
		originalDirector(req)
	}

	return proxy
}

// X-Forwarded-Proto: <preserve protocol the client originally used>
func addForwardedProto(req *http.Request) {
	if header := req.Header.Get("X-Forwarded-Proto"); header != "" {
		return
	}
	// doesn't handel tls connections so it would always be http in case nginx absence
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

func addForwardedServer(req *http.Request) {
	serverName := "TrafficCTRL"
	req.Header.Set("X-Forwarded-Server", serverName)
}
