package middleware

import (
	"context"
	"net/http"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"go.uber.org/zap"
)

func DryRunMiddleware(next http.Handler, cfg *config.Config, rateLimiter *limiter.RateLimiter) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		reqLogger := GetRequestLoggerFromContext(req.Context())

		if IsBypassEnabled(req.Context()) {
			next.ServeHTTP(res, req)
			return
		}

		if !cfg.Tool.DryRunMode {
			next.ServeHTTP(res, req)
			return
		}

		redisCtx := GetRedisContextFromContext(req.Context())
		tenantKey := GetTenantKeyFromContext(req.Context())
		endpointRule := GetEndpointRuleFromContext(req.Context())

		globalLimitResult, globalLimitError := rateLimiter.CheckGlobalLimit(redisCtx, &cfg.Limiter.Global)
		if globalLimitError != nil {
			reqLogger.Error("failed to check global limit (dry run)", zap.Error(globalLimitError))
		}

		tenantLimitResult, tenantLimitError := rateLimiter.CheckTenantLimit(redisCtx, tenantKey, &cfg.Limiter.PerTenant)
		if tenantLimitError != nil {
			reqLogger.Error("failed to check tenant limit (dry run)", zap.Error(tenantLimitError))
		}

		endpointLimitResult, endpointLimitError := rateLimiter.CheckEndpointLimit(redisCtx, tenantKey, endpointRule)
		if endpointLimitError != nil {
			reqLogger.Error("failed to check endpoint limit (dry run)", zap.Error(endpointLimitError))
		}

		if !globalLimitResult.Allowed {
			reqLogger.Warn("global limit would have been exceeded (dry run)",
				zap.Any("retry_after", globalLimitResult.RetryAfter.Seconds()))
		}
		if !tenantLimitResult.Allowed {
			reqLogger.Warn("tenant limit would have been exceeded (dry run)",
				zap.Any("retry_after", tenantLimitResult.RetryAfter.Seconds()))
		}
		if !endpointLimitResult.Allowed {
			reqLogger.Warn("endpoint limit would have been exceeded (dry run)",
				zap.Any("retry_after", endpointLimitResult.RetryAfter.Seconds()))
		}

		if endpointLimitResult.Allowed && tenantLimitResult.Allowed && globalLimitResult.Allowed {
			reqLogger.Debug("all rate limit checks passed (dry run)",
				zap.Int("remaining_endpoint", int(endpointLimitResult.Remaining)),
				zap.Int("remaining_tenant", int(tenantLimitResult.Remaining)),
				zap.Int("remaining_global", int(globalLimitResult.Remaining)))
		}
		ctx := context.WithValue(req.Context(), bypassKey, true)
		next.ServeHTTP(res, req.WithContext(ctx))
	})
}
