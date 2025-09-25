package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/mostafa-mahmood/TrafficCTRL/config"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/limiter"
	"go.uber.org/zap"
)

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

func rejectBadReputationTenant(res http.ResponseWriter, reqLogger *requestLogger,
	reputation *limiter.Reputation, result *limiter.LimitResult) {

	reqLogger.Warn("server on high load, tenants with bad reputation are banned",
		zap.Float64("reputation_score", reputation.Score),
		zap.Int64("violations_count", reputation.ViolationCount),
		zap.Int64("reputation_ttl", reputation.TTL))

	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("X-RateLimit-Remaining", "0")

	res.WriteHeader(http.StatusTooManyRequests)

	body := map[string]interface{}{
		"error":            "server on high load, tenants with bad reputation are banned",
		"reputation_score": reputation.Score,
		"violations_count": reputation.ViolationCount,
		"retry_after":      result.RetryAfter.Seconds(),
	}
	_ = json.NewEncoder(res).Encode(body)
}
