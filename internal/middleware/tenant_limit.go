package middleware

import (
	"net/http"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/metrics"
	"go.uber.org/zap"
)

func TenantLimitMiddleware(next http.Handler, cfg *config.Config, rateLimiter *limiter.RateLimiter) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		reqLogger := GetRequestLoggerFromContext(req.Context())

		if IsBypassEnabled(req.Context()) {
			next.ServeHTTP(res, req)
			return
		}

		if !cfg.Limiter.PerTenant.Enabled {
			next.ServeHTTP(res, req)
			return
		}

		redisCtx := GetRedisContextFromContext(req.Context())
		tenantKey := GetTenantKeyFromContext(req.Context())
		tenantLimitResult, err := rateLimiter.CheckTenantLimit(redisCtx, tenantKey, &cfg.Limiter.PerTenant)
		if err != nil {
			reqLogger.Error("failed to enforce tenant limit", zap.Error(err))
			//============================Metrics============================
			metrics.TenantLimitErrors.Inc()
			//===============================================================
			next.ServeHTTP(res, req)
			return
		}

		if !tenantLimitResult.Allowed {
			_, err := rateLimiter.UpdateReputation(redisCtx, tenantKey, true)
			if err != nil {
				reqLogger.Error("failed to update reputation", zap.Error(err))
			}
			rejectRequest(res, reqLogger, tenantLimitResult, config.PerTenantLevel)
			return
		}

		reqLogger.Debug("tenant rate limit check passed",
			zap.Int("remaining_tenant", int(tenantLimitResult.Remaining)))

		_, err = rateLimiter.UpdateReputation(redisCtx, tenantKey, false)
		if err != nil {
			reqLogger.Error("failed to update reputation", zap.Error(err))
		}

		next.ServeHTTP(res, req)
	})
}
