package shared

import (
	"net/http"
	"strings"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
)

func MapRequestToEndpointConfig(req *http.Request, rules []config.EndpointRule,
	lgr *logger.Logger) *config.EndpointRule {
	requestPath := req.URL.Path
	requestMethod := req.Method

	for _, rule := range rules {
		if !pathMatches(rule.Path, requestPath) {
			continue
		}

		if !methodMatches(rule.Methods, requestMethod) {
			continue
		}

		return &rule
	}

	return nil
}

// ensures path starts with / and handles trailing slashes consistently
func normalizePath(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Remove trailing slash unless it's the root path "/"
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	return path
}

func pathMatches(rulePath, requestPath string) bool {
	normalizedRulePath := normalizePath(rulePath)
	normalizedRequestPath := normalizePath(requestPath)

	if normalizedRulePath == "*" {
		return true
	}

	if normalizedRulePath == normalizedRequestPath {
		return true
	}

	// Handle wildcard prefix match
	if strings.HasSuffix(normalizedRulePath, "/*") {
		prefix := strings.TrimSuffix(normalizedRulePath, "/*")
		return strings.HasPrefix(normalizedRequestPath, prefix+"/") ||
			normalizedRequestPath == prefix
	}

	return false
}

func methodMatches(ruleMethods []string, requestMethod string) bool {
	if len(ruleMethods) == 0 {
		return true
	}

	for _, method := range ruleMethods {
		if strings.EqualFold(method, requestMethod) {
			return true
		}
	}

	return false
}
