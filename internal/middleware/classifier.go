package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/shared"
	"github.com/mostafa-mahmood/TrafficCTRL/metrics"
	"go.uber.org/zap"
)

func ClassifierMiddleware(next http.Handler, lgr *logger.Logger) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		cfg := config.GetConfigSnapshot(req.Context())

		reqLogger := newRequestLogger(lgr, req, GetRequestID(req.Context()), GetClientIP(req.Context()))
		ctx := context.WithValue(req.Context(), requestLoggerKey, reqLogger)

		endpointRule := shared.MapRequestToEndpointConfig(req, cfg.Limiter.PerEndpoint.Rules, lgr)
		ctx = context.WithValue(ctx, endpointRuleKey, endpointRule)

		if endpointRule == nil || endpointRule.Bypass {
			reqLogger.Warn("rate limiter bypassed, forwarding request to server")
			ctx = context.WithValue(ctx, bypassKey, true)

			//===========================Metrics==============================
			metrics.TotalBypassedRequests.Inc()
			//===========================Metrics==============================

			next.ServeHTTP(res, req.WithContext(ctx))
			return
		}

		tenantKey, err := shared.ExtractTenantKey(req, endpointRule.TenantStrategy, lgr)
		if err != nil {
			reqLogger.Error("failed to extract tenant key, forwarding request to server {fail open}",
				zap.Error(err))
			ctx = context.WithValue(ctx, bypassKey, true)
			next.ServeHTTP(res, req.WithContext(ctx))
			return
		}
		ctx = context.WithValue(ctx, tenantKeyKey, tenantKey)

		redisCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		ctx = context.WithValue(ctx, redisContextKey, redisCtx)

		//======================Metrics===========================================
		track := metrics.TrackRequest(req.Method, endpointRule.Path)
		defer track()
		//======================Metrics===========================================

		next.ServeHTTP(res, req.WithContext(ctx))
	})
}

func GetEndpointRuleFromContext(ctx context.Context) *config.EndpointRule {
	if v := ctx.Value(endpointRuleKey); v != nil {
		if rule, ok := v.(*config.EndpointRule); ok {
			return rule
		}
	}
	return nil
}

func GetTenantKeyFromContext(ctx context.Context) string {
	if v := ctx.Value(tenantKeyKey); v != nil {
		if key, ok := v.(string); ok {
			return key
		}
	}
	return ""
}

func GetRequestLoggerFromContext(ctx context.Context) *requestLogger {
	if v := ctx.Value(requestLoggerKey); v != nil {
		if lgr, ok := v.(*requestLogger); ok {
			return lgr
		}
	}
	return nil
}

func GetRedisContextFromContext(ctx context.Context) context.Context {
	if v := ctx.Value(redisContextKey); v != nil {
		if redisCtx, ok := v.(context.Context); ok {
			return redisCtx
		}
	}
	return nil
}
