package middleware

import (
	"context"
)

type ctxKey struct{}

var (
	RequestIDKey     = ctxKey{}
	ClientIPKey      = ctxKey{}
	EndpointRuleKey  = ctxKey{}
	TenantKeyKey     = ctxKey{}
	RequestLoggerKey = ctxKey{}
	RedisContextKey  = ctxKey{}
	BypassKey        = ctxKey{}
)

func IsBypassEnabled(ctx context.Context) bool {
	if v := ctx.Value(BypassKey); v != nil {
		if bypass, ok := v.(bool); ok {
			return bypass
		}
	}
	return false
}
