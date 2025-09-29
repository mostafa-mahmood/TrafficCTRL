package proxy

import (
	"fmt"
	"net"
	"net/http"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/middleware"
	"github.com/mostafa-mahmood/TrafficCTRL/metrics"
	"go.uber.org/zap"
)

func StartServer(cfg *config.Config, lgr *logger.Logger, rateLimiter *limiter.RateLimiter) error {
	lgr.Info("proxy server starting",
		zap.Uint16("port", cfg.Proxy.ProxyPort),
		zap.String("target_url", cfg.Proxy.TargetUrl))

	proxy, err := createProxy(cfg)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Inject config snapshot into context
		ctx := config.WithConfigSnapshot(r.Context(), cfg)
		r = r.WithContext(ctx)

		var h http.Handler = proxy

		h = middleware.EndpointLimitMiddleware(h, rateLimiter)
		h = middleware.TenantLimitMiddleware(h, rateLimiter)
		h = middleware.GlobalLimitMiddleware(h, lgr, rateLimiter)
		h = middleware.DryRunMiddleware(h, rateLimiter)
		h = middleware.ClassifierMiddleware(h, lgr)
		h = middleware.MetadataMiddleware(h)
		h = middleware.RecoveryMiddleware(h, proxy, lgr)

		h.ServeHTTP(w, r)
	})

	mux.Handle("/", handler)

	go func(lgr *logger.Logger, cfg *config.Config) error {
		mux := http.NewServeMux()
		mux.Handle("/metrics", metrics.Handler())

		lgr.Info("metrics server starting",
			zap.Uint16("port", cfg.Proxy.MetricsPort))

		address := net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", cfg.Proxy.MetricsPort))
		return http.ListenAndServe(address, mux)
	}(lgr, cfg)

	address := net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", cfg.Proxy.ProxyPort))
	return http.ListenAndServe(address, mux)
}
