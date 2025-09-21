package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/mostafa-mahmood/TrafficCTRL/internal/proxy"
)

type ctxKey string

const (
	requestIDKey ctxKey = "requestID"
	clientIPKey  ctxKey = "clientIP"
)

func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

func withClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey, ip)
}

func GetClientIP(ctx context.Context) string {
	if v := ctx.Value(clientIPKey); v != nil {
		if ip, ok := v.(string); ok {
			return ip
		}
	}
	return ""
}

func MetadataMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
			r.Header.Set("X-Request-ID", reqID)
		}

		clientIP := proxy.ExtractIP(r)
		if r.Header.Get("X-Real-IP") == "" {
			r.Header.Set("X-Real-IP", clientIP)
		}

		ctx := withRequestID(r.Context(), reqID)
		ctx = withClientIP(ctx, clientIP)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
