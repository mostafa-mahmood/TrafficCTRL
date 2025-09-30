package shared

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"unicode"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"go.uber.org/zap"
)

func ExtractTenantKey(req *http.Request, tenantRule *config.TenantStrategy,
	lgr *logger.Logger) (tenantKey string, err error) {
	if tenantRule == nil {
		lgr.Warn("tenant strategy is nil, falling back to IP",
			zap.String("request_id", req.Header.Get("X-Request-ID")),
			zap.String("path", req.URL.Path),
			zap.String("method", req.Method),
			zap.String("host", req.Host))
		return sanitizeRedisKey(ExtractIP(req)), nil
	}
	switch tenantRule.Type {
	case "ip":
		tenantKey = ExtractIP(req)
	case "header":
		tenantKey = extractFromHeader(req, tenantRule.Key)
	case "cookie":
		tenantKey = extractFromCookie(req, tenantRule.Key)
	case "query_parameter":
		tenantKey = extractFromParam(req, tenantRule.Key)
	default:
		return "", fmt.Errorf("unknown tenant strategy type: %s", tenantRule.Type)
	}

	if tenantKey == "" {
		lgr.Warn(
			"tenant key not found, falling back to IP",
			zap.String("request_id", req.Header.Get("X-Request-ID")),
			zap.String("strategy", tenantRule.Type),
			zap.String("key", tenantRule.Key),
			zap.String("path", req.URL.Path),
			zap.String("method", req.Method),
			zap.String("host", req.Host),
			zap.String("remoteAddr", req.RemoteAddr))
		return sanitizeRedisKey(ExtractIP(req)), nil
	}

	return sanitizeRedisKey(tenantKey), nil
}

func sanitizeRedisKey(input string) string {
	if input == "" {
		return input
	}

	cleaned := strings.Map(func(r rune) rune {
		if r <= 31 || r == 127 {
			return -1
		}
		if unicode.IsSpace(r) {
			return -1
		}
		switch r {
		case '-', '_', '.', ':', '@':
			return r
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return -1
	}, input)

	if len(cleaned) > 128 {
		cleaned = cleaned[:128]
	}

	return cleaned
}

func ExtractIP(req *http.Request) string {
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
