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
	"github.com/mostafa-mahmood/TrafficCTRL/internal/shared"
	"go.uber.org/zap"
)

func RateLimiterMiddleware(next http.Handler, cfg *config.Config, lgr *logger.Logger,
	rateLimiter *limiter.RateLimiter) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		requestID := GetRequestID(req.Context())
		clientIP := GetClientIP(req.Context())

		reqLogger := newRequestLogger(lgr, req, requestID, clientIP)

		redisCtx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
		defer cancel()

		reqLogger.Debug("incoming request")

		endpointRule := shared.MapRequestToEndpointConfig(req, cfg.Limiter.PerEndpoint.Rules, lgr)
		if endpointRule == nil || endpointRule.Bypass {
			reqLogger.Warn("rate limiter bypassed, forwarding request to server")

			next.ServeHTTP(res, req)
			return
		}

		tenantKey, err := shared.ExtractTenantKey(req, endpointRule.TenantStrategy, lgr)
		if err != nil {
			reqLogger.Error("failed to extract tenant key, forwarding request to server", zap.Error(err))
			next.ServeHTTP(res, req)
			return
		}

		if cfg.Tool.DryRunMode {
			handleDryRunMode(res, req, next, cfg, rateLimiter, reqLogger, tenantKey, redisCtx, endpointRule)
			return
		}

		globalLimitResult, err := rateLimiter.CheckGlobalLimit(redisCtx, &cfg.Limiter.Global)
		if err != nil {
			reqLogger.Error("failed to enforce global limit", zap.Error(err))
			next.ServeHTTP(res, req)
			return
		}
		if !globalLimitResult.Allowed {
			rejectRequest(res, reqLogger, globalLimitResult, config.GlobalLevel)
			return
		}

		tenantLimitResult, err := rateLimiter.CheckTenantLimit(redisCtx, tenantKey, &cfg.Limiter.PerTenant)
		if err != nil {
			reqLogger.Error("failed to enforce tenant limit", zap.Error(err))
			next.ServeHTTP(res, req)
			return
		}
		if !tenantLimitResult.Allowed {
			rejectRequest(res, reqLogger, tenantLimitResult, config.PerTenantLevel)
			return
		}

		endpointLimitResult, err := rateLimiter.CheckEndpointLimit(redisCtx, tenantKey, endpointRule)
		if err != nil {
			reqLogger.Error("failed to enforce endpoint limit", zap.Error(err))
			next.ServeHTTP(res, req)
			return
		}
		if !endpointLimitResult.Allowed {
			rejectRequest(res, reqLogger, endpointLimitResult, config.PerEndpointLevel)
			return
		}

		reqLogger.Debug("rate limit check passed, request allowed",
			zap.Int("remaining_endpoint", int(endpointLimitResult.Remaining)),
			zap.Int("remaining_tenant", int(tenantLimitResult.Remaining)),
			zap.Int("remaining_global", int(globalLimitResult.Remaining)))

		next.ServeHTTP(res, req)
	})
}

func rejectRequest(res http.ResponseWriter, reqLogger *requestLogger, result *limiter.LimitResult,
	limitLevel config.LimitLevelType) {

	reqLogger.Warn("rate limit exceeded, request denied",
		zap.String("limit_level", string(limitLevel)),
		zap.Float64("retry_after", result.RetryAfter.Seconds()))

	res.Header().Set("Content-Type", "application/json")

	res.Header().Set("X-RateLimit-Remaining", "0")

	if result.RetryAfter > 0 {
		secs := int64(result.RetryAfter.Seconds())
		res.Header().Set("Retry-After", strconv.FormatInt(secs, 10))
	}

	res.WriteHeader(http.StatusTooManyRequests)

	body := map[string]interface{}{
		"error":       "rate limit exceeded",
		"limit_level": limitLevel,
		"remaining":   result.Remaining,
		"retry_after": result.RetryAfter.Seconds(),
	}
	_ = json.NewEncoder(res).Encode(body)
}

func handleDryRunMode(res http.ResponseWriter, req *http.Request, next http.Handler,
	cfg *config.Config, rateLimiter *limiter.RateLimiter,
	reqLogger *requestLogger, tenantKey string, redisCtx context.Context, endpointRule *config.EndpointRule) {

	globalLimitResult, globalLimitError := rateLimiter.CheckGlobalLimit(redisCtx, &cfg.Limiter.Global)
	if globalLimitError != nil {
		reqLogger.Error("failed to enforce global limit (dry run)", zap.Error(globalLimitError))
	}

	tenantLimitResult, tenantLimitError := rateLimiter.CheckTenantLimit(redisCtx, tenantKey, &cfg.Limiter.PerTenant)
	if tenantLimitError != nil {
		reqLogger.Error("failed to enforce tenant limit (dry run)", zap.Error(tenantLimitError))
	}

	endpointLimitResult, endpointLimitError := rateLimiter.CheckEndpointLimit(redisCtx, tenantKey, endpointRule)
	if endpointLimitError != nil {
		reqLogger.Error("failed to enforce endpoint limit (dry run)", zap.Error(endpointLimitError))
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
		reqLogger.Debug("all rate limit check passed, request would have been allowed (dry run)",
			zap.Int("remaining_endpoint", int(endpointLimitResult.Remaining)),
			zap.Int("remaining_tenant", int(tenantLimitResult.Remaining)),
			zap.Int("remaining_global", int(globalLimitResult.Remaining)))
	}

	next.ServeHTTP(res, req)
}

// requestLogger wraps the base logger with common request fields
type requestLogger struct {
	*logger.Logger
	baseFields []zap.Field
}

func newRequestLogger(lgr *logger.Logger, req *http.Request, requestID, clientIP string) *requestLogger {
	baseFields := []zap.Field{
		zap.String("request_id", requestID),
		zap.String("client_ip", clientIP),
		zap.String("path", req.URL.Path),
		zap.String("method", req.Method),
		zap.String("host", req.Host),
	}

	return &requestLogger{
		Logger:     lgr,
		baseFields: baseFields,
	}
}

func (rl *requestLogger) Debug(msg string, fields ...zap.Field) {
	rl.Logger.Debug(msg, append(rl.baseFields, fields...)...)
}

func (rl *requestLogger) Warn(msg string, fields ...zap.Field) {
	rl.Logger.Warn(msg, append(rl.baseFields, fields...)...)
}

func (rl *requestLogger) Error(msg string, fields ...zap.Field) {
	rl.Logger.Error(msg, append(rl.baseFields, fields...)...)
}
