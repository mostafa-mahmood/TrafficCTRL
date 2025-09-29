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

		ctx := req.Context()
		cfg := config.GetConfigFromContext(ctx)

		reqLogger := newRequestLogger(lgr, req, GetRequestID(ctx), GetClientIP(ctx))
		ctx = setRequestLogger(ctx, reqLogger)

		endpointRule := shared.MapRequestToEndpointConfig(req, cfg.Limiter.PerEndpoint.Rules, lgr)
		ctx = setEndpointRule(ctx, endpointRule)

		if endpointRule == nil || endpointRule.Bypass {
			reqLogger.Warn("rate limiter bypassed (no rule matched or bypass flag set), forwarding request to server")
			ctx = setBypass(ctx, true)

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
			ctx = setBypass(ctx, true)
			next.ServeHTTP(res, req.WithContext(ctx))
			return
		}
		ctx = setTenantKey(ctx, tenantKey)

		redisCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		ctx = setRedisContext(ctx, redisCtx)

		//======================Metrics===========================================
		track := metrics.TrackRequest(req.Method, endpointRule.Path)
		defer track()
		//======================Metrics===========================================

		next.ServeHTTP(res, req.WithContext(ctx))
	})
}

func setEndpointRule(ctx context.Context, rule *config.EndpointRule) context.Context {
	return context.WithValue(ctx, EndpointRuleKey, rule)
}

func setTenantKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, TenantKeyKey, key)
}

func setRequestLogger(ctx context.Context, lgr *requestLogger) context.Context {
	return context.WithValue(ctx, RequestLoggerKey, lgr)
}

func setRedisContext(ctx context.Context, redisCtx context.Context) context.Context {
	return context.WithValue(ctx, RedisContextKey, redisCtx)
}

func setBypass(ctx context.Context, bypass bool) context.Context {
	return context.WithValue(ctx, BypassKey, bypass)
}

func GetEndpointRuleFromContext(ctx context.Context) *config.EndpointRule {
	if v := ctx.Value(EndpointRuleKey); v != nil {
		if rule, ok := v.(*config.EndpointRule); ok {
			return rule
		}
	}
	return nil
}

func GetTenantKeyFromContext(ctx context.Context) string {
	if v := ctx.Value(TenantKeyKey); v != nil {
		if key, ok := v.(string); ok {
			return key
		}
	}
	return ""
}

func GetRequestLoggerFromContext(ctx context.Context) *requestLogger {
	if v := ctx.Value(RequestLoggerKey); v != nil {
		if lgr, ok := v.(*requestLogger); ok {
			return lgr
		}
	}
	return nil
}

func GetRedisContextFromContext(ctx context.Context) context.Context {
	if v := ctx.Value(RedisContextKey); v != nil {
		if redisCtx, ok := v.(context.Context); ok {
			return redisCtx
		}
	}
	return nil
}
