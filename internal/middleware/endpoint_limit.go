package middleware

import (
	"net/http"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/metrics"
	"go.uber.org/zap"
)

func EndpointLimitMiddleware(next http.Handler, rateLimiter *limiter.RateLimiter) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		ctx := req.Context()
		reqLogger := GetRequestLoggerFromContext(ctx)

		if IsBypassEnabled(ctx) {
			next.ServeHTTP(res, req)
			return
		}

		redisCtx := GetRedisContextFromContext(ctx)
		tenantKey := GetTenantKeyFromContext(ctx)
		endpointRule := GetEndpointRuleFromContext(ctx)

		endpointLimitResult, err := rateLimiter.CheckEndpointLimit(redisCtx, tenantKey, endpointRule)
		if err != nil {
			reqLogger.Error("failed to enforce endpoint limit", zap.Error(err))
			//============================Metrics============================
			metrics.EndpointLimitErrors.Inc()
			//===============================================================
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
			zap.Int64("remaining_endpoint", endpointLimitResult.Remaining))

		//==========================Metrics==================================
		metrics.AllowedRequests.Inc()
		//==========================Metrics==================================
		_, err = rateLimiter.UpdateReputation(redisCtx, tenantKey, false)
		if err != nil {
			reqLogger.Error("failed to update reputation", zap.Error(err))
		}
		next.ServeHTTP(res, req)
	})
}
