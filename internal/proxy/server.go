package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/middleware"
	"github.com/mostafa-mahmood/TrafficCTRL/metrics"
	"go.uber.org/zap"
)

func StartServer(cfg *config.Config, lgr *logger.Logger, rateLimiter *limiter.RateLimiter) error {
	proxyAddr := net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", cfg.Proxy.ProxyPort))
	metricsAddr := net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", cfg.Proxy.MetricsPort))

	proxy, err := createProxy(cfg)
	if err != nil {
		return err
	}

	rootHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := config.WithConfigSnapshot(r.Context(), cfg)
		r = r.WithContext(ctx)

		var next http.Handler = proxy

		next = middleware.EndpointLimitMiddleware(next, rateLimiter)
		next = middleware.TenantLimitMiddleware(next, rateLimiter)
		next = middleware.GlobalLimitMiddleware(next, lgr, rateLimiter)
		next = middleware.DryRunMiddleware(next, rateLimiter)
		next = middleware.ClassifierMiddleware(next, lgr)
		next = middleware.MetadataMiddleware(next)
		next = middleware.RecoveryMiddleware(next, proxy, lgr)

		next.ServeHTTP(w, r)
	})

	proxyMux := http.NewServeMux()
	proxyMux.Handle("/", rootHandler)
	proxyServer := &http.Server{
		Addr:    proxyAddr,
		Handler: proxyMux,
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", metrics.Handler())
	metricsServer := &http.Server{
		Addr:    metricsAddr,
		Handler: metricsMux,
	}

	// Channel to catch errors from concurrent servers
	errChan := make(chan error, 2)

	lgr.Info("proxy server starting", zap.String("address", proxyAddr))
	go func() {
		if err := proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("proxy server failed: %w", err)
		}
	}()

	lgr.Info("metrics server starting", zap.String("address", metricsAddr))
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("metrics server failed: %w", err)
		}
	}()

	// --- SHUTDOWN BOTH SERVERS IN CASE ONE OF THEM FAILS ---
	err = <-errChan

	lgr.Error("critical server error received, initiating shutdown...", zap.Error(err))

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if shutdownErr := proxyServer.Shutdown(shutdownCtx); shutdownErr != nil {
		lgr.Warn("proxy server shutdown failed", zap.Error(shutdownErr))
	}
	if shutdownErr := metricsServer.Shutdown(shutdownCtx); shutdownErr != nil {
		lgr.Warn("metrics server shutdown failed", zap.Error(shutdownErr))
	}

	return err
}
