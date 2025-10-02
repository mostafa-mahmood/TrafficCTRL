package middleware

import "context"

type ctxKey string

const (
	RequestIDKey     ctxKey = "requestID"
	ClientIPKey      ctxKey = "clientIP"
	EndpointRuleKey  ctxKey = "endpointRule"
	TenantKeyKey     ctxKey = "tenantKey"
	RequestLoggerKey ctxKey = "requestLogger"
	RedisContextKey  ctxKey = "redisContext"
	BypassKey        ctxKey = "bypass"
)

func IsBypassEnabled(ctx context.Context) bool {
	if v := ctx.Value(BypassKey); v != nil {
		if bypass, ok := v.(bool); ok {
			return bypass
		}
	}
	return false
}
