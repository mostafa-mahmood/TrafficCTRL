package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/proxy"
	"go.uber.org/zap"
)

func RateLimiterMiddleware(next http.Handler, cfg config.Config, lgr *logger.Logger) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		rateLimiter := limiter.NewRateLimiter()

		redisCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		endpointRule := proxy.MapRequestToEndpointConfig(req, cfg.Limiter.PerEndpoint.Rules, lgr)
		if endpointRule == nil || endpointRule.Bypass {
			next.ServeHTTP(res, req) // pass to proxy middleware which will forward request
			return
		}

		tenantKey, err := proxy.ExtractTenantKey(req, endpointRule.TenantStrategy, lgr)
		if err != nil {
			lgr.Warn("failed to extract tenant key", zap.Error(err),
				zap.String("requestID", GetRequestID(req.Context())))
		}

		var limitResult *limiter.LimitResult

		limitResult, err = rateLimiter.CheckGlobalLimit(redisCtx, cfg.Limiter.Global)
		if err != nil {
			lgr.Error("failed to apply global limit", zap.Error(err),
				zap.String("requestID", GetRequestID(req.Context())))
		}
		if !limitResult.Allowed {
			rejectRequest(res, limitResult)
			return
		}

		limitResult, err = rateLimiter.CheckTenantLimit(redisCtx, tenantKey, cfg.Limiter.PerTenant)
		if err != nil {
			lgr.Error("failed to apply global limit", zap.Error(err),
				zap.String("requestID", GetRequestID(req.Context())))
		}
		if !limitResult.Allowed {
			rejectRequest(res, limitResult)
			return
		}

		limitResult, err = rateLimiter.CheckEndpointLimit(redisCtx, tenantKey, *endpointRule)
		if err != nil {
			lgr.Error("failed to apply global limit", zap.Error(err),
				zap.String("requestID", GetRequestID(req.Context())))
		}
		if !limitResult.Allowed {
			rejectRequest(res, limitResult)
			return
		}

		next.ServeHTTP(res, req)
	})
}

func rejectRequest(w http.ResponseWriter, result *limiter.LimitResult) {
	w.Header().Set("Content-Type", "application/json")

	w.Header().Set("X-RateLimit-Remaining", "0")

	if result.RetryAfter > 0 {
		secs := int64(result.RetryAfter.Seconds())
		w.Header().Set("Retry-After", strconv.FormatInt(secs, 10))
	}

	w.WriteHeader(http.StatusTooManyRequests)

	body := map[string]interface{}{
		"error":       "rate limit exceeded",
		"remaining":   0,
		"retry_after": result.RetryAfter.Seconds(),
	}
	_ = json.NewEncoder(w).Encode(body)
}
