package middleware

import (
	"context"
)

type ctxKey string

const (
	requestIDKey     ctxKey = "requestID"
	clientIPKey      ctxKey = "clientIP"
	endpointRuleKey  ctxKey = "endpointRule"
	tenantKeyKey     ctxKey = "tenantKey"
	requestLoggerKey ctxKey = "requestLogger"
	redisContextKey  ctxKey = "redisContext"
	bypassKey        ctxKey = "bypass"
)

func IsBypassEnabled(ctx context.Context) bool {
	if v := ctx.Value(bypassKey); v != nil {
		if bypass, ok := v.(bool); ok {
			return bypass
		}
	}
	return false
}
