package middleware

import (
	"net/http"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"go.uber.org/zap"
)

func GlobalLimitMiddleware(next http.Handler, cfg *config.Config, lgr *logger.Logger,
	rateLimiter *limiter.RateLimiter) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		reqLogger := GetRequestLoggerFromContext(req.Context())

		if IsBypassEnabled(req.Context()) {
			next.ServeHTTP(res, req)
			return
		}

		if !cfg.Limiter.Global.Enabled {
			next.ServeHTTP(res, req)
			return
		}

		redisCtx := GetRedisContextFromContext(req.Context())
		tenantKey := GetTenantKeyFromContext(req.Context())

		globalLimitResult, err := rateLimiter.CheckGlobalLimit(redisCtx, &cfg.Limiter.Global)
		if err != nil {
			reqLogger.Error("failed to enforce global limit", zap.Error(err))
			next.ServeHTTP(res, req)
			return
		}

		reputation, err := rateLimiter.GetTenantReputation(redisCtx, tenantKey)
		if err != nil {
			reqLogger.Error("failed to get tenant reputation")
			next.ServeHTTP(res, req)
			return
		}

		if !globalLimitResult.Allowed {
			lgr.Warn("global limit is reached, server is on high load, applying reputation checks")

			if reputation.Score <= 0.3 {
				rejectBadReputationTenant(res, reqLogger, reputation, globalLimitResult)
				return
			} else {
				reqLogger.Debug("reputation check passed",
					zap.Float64("reputation_score", reputation.Score),
					zap.Int64("good_requests", reputation.GoodRequests),
					zap.Int64("reputation_ttl", reputation.TTL))
			}
		}

		next.ServeHTTP(res, req)
	})
}
