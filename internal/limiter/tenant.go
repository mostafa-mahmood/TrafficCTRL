package limiter

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

func Extractor(req *http.Request, tenantStrategy string) (tenantKey string, err error) {
	switch tenantStrategy {
	case "ip":
		tenantKey, err := IPExtractor(req)
		if err != nil {
			return "", fmt.Errorf("error extracting IP: %v", err)
		}
		return tenantKey, nil
	case "header":

	}
}

func IPExtractor(req *http.Request) (string, error) {

	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri), nil
	}

	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])

		if ip != "" {
			return ip, nil
		}
	}

	host, _, err := net.SplitHostPort(req.RemoteAddr)

	if err != nil {
		return req.RemoteAddr, nil
	}

	return host, nil
}

func HeaderExtractor(req *http.Request, key string) (string, error) {
	value := req.Header.Get(key)

	if value == "" {
		return "", fmt.Errorf("header %s was not found", key)
	}

	return strings.TrimSpace(value), nil
}
