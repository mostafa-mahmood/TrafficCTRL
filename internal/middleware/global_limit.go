package middleware

import (
	"net/http"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"github.com/mostafa-mahmood/TrafficCTRL/metrics"
	"go.uber.org/zap"
)

func GlobalLimitMiddleware(next http.Handler, lgr *logger.Logger,
	rateLimiter *limiter.RateLimiter) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		ctx := req.Context()
		cfg := config.GetConfigFromContext(ctx)
		reqLogger := GetRequestLoggerFromContext(ctx)

		if IsBypassEnabled(ctx) {
			next.ServeHTTP(res, req)
			return
		}

		if !cfg.Limiter.Global.Enabled {
			next.ServeHTTP(res, req)
			return
		}

		redisCtx := GetRedisContextFromContext(ctx)
		tenantKey := GetTenantKeyFromContext(ctx)

		globalLimitResult, err := rateLimiter.CheckGlobalLimit(redisCtx, &cfg.Limiter.Global)
		if err != nil {
			reqLogger.Error("failed to enforce global limit", zap.Error(err))
			//============================Metrics============================
			metrics.GlobalLimitErrors.Inc()
			//===============================================================
			next.ServeHTTP(res, req)
			return
		}

		reputation, err := rateLimiter.GetTenantReputation(redisCtx, tenantKey)
		if err != nil {
			reqLogger.Error("failed to get tenant reputation")
			next.ServeHTTP(res, req)
			return
		}

		//=============================Metrics=============================
		metrics.ReputationDistribution.Observe(reputation.Score)
		//=================================================================

		if !globalLimitResult.Allowed {
			reqLogger.Debug("global limit is reached, server is on high load, applying reputation checks")

			if reputation.Score <= rateLimiter.GetReputationThreshold() {
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
