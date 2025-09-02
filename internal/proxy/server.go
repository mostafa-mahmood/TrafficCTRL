package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"go.uber.org/zap"
)

func StartServer(port uint16, targetUrl *url.URL) error {
	logger.Log.Info("TrafficCTRL started succesfully", zap.Uint16("proxy_port", port),
		zap.String("target_url", targetUrl.String()))

	proxy := createProxy(targetUrl)

	mux := http.NewServeMux()

	mux.Handle("/", rateLimitingMiddleware(proxy))

	address := net.JoinHostPort("", fmt.Sprintf("%d", port))
	return http.ListenAndServe(address, mux)
}

func rateLimitingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		tenant := parseTenant(req)

		allowed, _ := limiter.FixedWindowLimiter(tenant)

		if !allowed {
			res.WriteHeader(http.StatusTooManyRequests)
			res.Write([]byte("Too Many Requests"))
			return
		}

		next.ServeHTTP(res, req)
	})
}

func parseTenant(*http.Request) string {
	return ""
}
