package middleware

import (
	"net/http"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"go.uber.org/zap"
)

func EndpointLimitMiddleware(next http.Handler, rateLimiter *limiter.RateLimiter) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		reqLogger := GetRequestLoggerFromContext(req.Context())

		if IsBypassEnabled(req.Context()) {
			next.ServeHTTP(res, req)
			return
		}

		redisCtx := GetRedisContextFromContext(req.Context())
		tenantKey := GetTenantKeyFromContext(req.Context())
		endpointRule := GetEndpointRuleFromContext(req.Context())

		endpointLimitResult, err := rateLimiter.CheckEndpointLimit(redisCtx, tenantKey, endpointRule)
		if err != nil {
			reqLogger.Error("failed to enforce endpoint limit", zap.Error(err))
			next.ServeHTTP(res, req)
			return
		}

		if !endpointLimitResult.Allowed {
			_, err := rateLimiter.UpdateReputation(redisCtx, tenantKey, true)
			if err != nil {
				reqLogger.Error("failed to update reputation", zap.Error(err))
			}
			rejectRequest(res, reqLogger, endpointLimitResult, config.PerEndpointLevel)
			return
		}

		reqLogger.Debug("endpoint rate limit check passed",
			zap.Int("remaining_endpoint", int(endpointLimitResult.Remaining)))

		_, err = rateLimiter.UpdateReputation(redisCtx, tenantKey, false)
		if err != nil {
			reqLogger.Error("failed to update reputation", zap.Error(err))
		}
		next.ServeHTTP(res, req)
	})
}
