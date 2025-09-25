package proxy

import (
	"fmt"
	"net"
	"net/http"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/middleware"
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

	address := net.JoinHostPort("", fmt.Sprintf("%d", cfg.Proxy.ProxyPort))
	return http.ListenAndServe(address, mux)
}
