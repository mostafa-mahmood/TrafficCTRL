package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/middleware"
	"go.uber.org/zap"
)

func BootStrap(cfg *config.Config, lgr *logger.Logger, rateLimiter *limiter.RateLimiter) error {
	port := cfg.Proxy.ProxyPort
	stringTargetUrl := cfg.Proxy.TargetUrl
	targetUrl, err := url.Parse(stringTargetUrl)
	if err != nil {
		return fmt.Errorf("error parsing url, bootstrap failed: %v", err)
	}

	lgr.Info("TrafficCTRL Server Started", zap.Uint16("proxy_port", port),
		zap.String("target_url", stringTargetUrl))

	proxy := createProxy(targetUrl)

	mux := http.NewServeMux()

	middlewareChain := middleware.RecoveryMiddleware(
		middleware.MetadataMiddleware(
			middleware.RateLimiterMiddleware(
				proxy,
				cfg,
				lgr,
				rateLimiter,
			),
		),
		proxy,
		lgr,
	)

	mux.Handle("/", middlewareChain)

	address := net.JoinHostPort("", fmt.Sprintf("%d", port))
	return http.ListenAndServe(address, mux)
}
