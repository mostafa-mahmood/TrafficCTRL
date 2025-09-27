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

	var handler http.Handler = proxy

	handler = middleware.EndpointLimitMiddleware(
		handler,
		rateLimiter,
	)

	handler = middleware.TenantLimitMiddleware(
		handler,
		cfg,
		rateLimiter,
	)

	handler = middleware.GlobalLimitMiddleware(
		handler,
		cfg,
		lgr,
		rateLimiter,
	)

	handler = middleware.DryRunMiddleware(
		handler,
		cfg,
		rateLimiter,
	)

	handler = middleware.ClassifierMiddleware(
		handler,
		cfg,
		lgr,
	)

	handler = middleware.MetadataMiddleware(
		handler,
	)

	middlewareChain := middleware.RecoveryMiddleware(
		handler,
		proxy,
		lgr,
	)

	mux.Handle("/", middlewareChain)

	go func(lgr *logger.Logger, cfg *config.Config) error {
		mux := http.NewServeMux()
		mux.Handle("/metrics", metrics.Handler())

		lgr.Info("metrics server starting",
			zap.Uint16("port", cfg.Proxy.MetricsPort))

		address := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", cfg.Proxy.MetricsPort))
		return http.ListenAndServe(address, mux)
	}(lgr, cfg)

	address := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", cfg.Proxy.ProxyPort))
	return http.ListenAndServe(address, mux)
}
