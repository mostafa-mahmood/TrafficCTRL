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

func RateLimiterMiddleware(next http.Handler, cfg config.Config, lgr *logger.Logger,
	rateLimiter *limiter.RateLimiter) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		redisCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		requestID := GetRequestID(req.Context())
		clientIP := GetClientIP(req.Context())

		endpointRule := proxy.MapRequestToEndpointConfig(req, cfg.Limiter.PerEndpoint.Rules, lgr)
		if endpointRule == nil || endpointRule.Bypass {
			lgr.Info("rate limiter bypassed, forwarding request to server",
				zap.String("requestID", requestID),
				zap.String("clientIP", clientIP))

			next.ServeHTTP(res, req) // pass to proxy middleware which will forward request
			return
		}

		tenantKey, err := proxy.ExtractTenantKey(req, endpointRule.TenantStrategy, lgr)
		if err != nil {
			lgr.Error("failed to extract tenant key, forwarding request to server",
				zap.String("requestID", requestID),
				zap.Error(err))

			next.ServeHTTP(res, req)
			return
		}

		var limitResult *limiter.LimitResult

		limitResult, err = rateLimiter.CheckGlobalLimit(redisCtx, cfg.Limiter.Global)
		if err != nil {
			lgr.Error("failed to enforce global limit", zap.String("requestID", requestID),
				zap.String("clientIP", clientIP), zap.Error(err))
		}
		if !limitResult.Allowed {
			rejectRequest(res, lgr, limitResult, config.GlobalLevel, requestID, clientIP)
			return
		}

		limitResult, err = rateLimiter.CheckTenantLimit(redisCtx, tenantKey, cfg.Limiter.PerTenant)
		if err != nil {
			lgr.Error("failed to enforce tenant limit", zap.String("requestID", requestID),
				zap.String("clientIP", clientIP), zap.Error(err))
		}
		if !limitResult.Allowed {
			rejectRequest(res, lgr, limitResult, config.PerTenantLevel, requestID, clientIP)
			return
		}

		limitResult, err = rateLimiter.CheckEndpointLimit(redisCtx, tenantKey, *endpointRule)
		if err != nil {
			lgr.Error("failed to apply endpoint limit", zap.String("requestID", requestID),
				zap.String("clientIP", clientIP), zap.Error(err))
		}
		if !limitResult.Allowed {
			rejectRequest(res, lgr, limitResult, config.PerEndpointLevel, requestID, clientIP)
			return
		}

		lgr.Info("request allowed", zap.String("request_id", requestID), zap.String("client_ip", clientIP),
			zap.Bool("allowed", limitResult.Allowed), zap.Int("remaining", int(limitResult.Remaining)),
			zap.Float64("retry_after", limitResult.RetryAfter.Seconds()))

		next.ServeHTTP(res, req)
	})
}

func rejectRequest(w http.ResponseWriter, lgr *logger.Logger, result *limiter.LimitResult,
	limitLevel config.LimitLevelType, requestID string, clientIP string) {

	lgr.Info("request denied", zap.String("limit_level", string(limitLevel)),
		zap.String("request_id", requestID), zap.String("client_ip", clientIP),
		zap.Bool("allowed", result.Allowed), zap.Int("remaining", int(result.Remaining)),
		zap.Float64("retry_after", result.RetryAfter.Seconds()))

	w.Header().Set("Content-Type", "application/json")

	w.Header().Set("X-RateLimit-Remaining", "0")

	if result.RetryAfter > 0 {
		secs := int64(result.RetryAfter.Seconds())
		w.Header().Set("Retry-After", strconv.FormatInt(secs, 10))
	}

	w.WriteHeader(http.StatusTooManyRequests)

	body := map[string]interface{}{
		"error":       "rate limit exceeded",
		"limit_level": limitLevel,
		"remaining":   result.Remaining,
		"retry_after": result.RetryAfter.Seconds(),
	}
	_ = json.NewEncoder(w).Encode(body)
}
