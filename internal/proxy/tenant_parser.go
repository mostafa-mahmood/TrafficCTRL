package proxy

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"go.uber.org/zap"
)

func ExtractTenantKey(req *http.Request, tenantRule *config.TenantStrategy) (tenantKey string, err error) {
	if tenantRule == nil {
		logger.Log.Warn("tenant rule is nil, falling back to IP")
		return extractIP(req), nil
	}
	switch tenantRule.Type {
	case "ip":
		tenantKey = extractIP(req)
	case "header":
		tenantKey = extractFromHeader(req, tenantRule.Key)
	case "cookie":
		tenantKey = extractFromCookie(req, tenantRule.Key)
	case "query_parameter":
		tenantKey = extractFromParam(req, tenantRule.Key)
	default:
		return "", fmt.Errorf("unknown tenant strategy type: %s, falling back to IP", tenantRule.Type)
	}

	if tenantKey == "" {
		logger.Log.Warn(
			"tenant key not found, falling back to IP",
			zap.String("strategy", tenantRule.Type),
			zap.String("key", tenantRule.Key),
			zap.String("remoteAddr", req.RemoteAddr),
		)
		return extractIP(req), nil
	}

	return tenantKey, nil
}

func extractIP(req *http.Request) string {
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		xff = strings.TrimSpace(xff)
		arr := strings.Split(xff, ",")
		if arr[0] != "" {
			return strings.TrimSpace(arr[0])
		}
	}

	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}

func extractFromHeader(req *http.Request, headerKey string) string {
	return strings.TrimSpace(req.Header.Get(headerKey))
}

func extractFromCookie(req *http.Request, cookieKey string) string {
	cookie, err := req.Cookie(cookieKey)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func extractFromParam(req *http.Request, paramKey string) string {
	return req.URL.Query().Get(paramKey)
}
