package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"go.uber.org/zap"
)

func RecoveryMiddleware(next http.Handler, fallBack http.Handler, lgr *logger.Logger) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				lgr.Error("error in middleware chain, panic recovered",
					zap.String("path", req.URL.Path),
					zap.String("method", req.Method),
					zap.String("host", req.Host),
					zap.Any("panic", rec),
					zap.ByteString("stack_trace", debug.Stack()))
				fallBack.ServeHTTP(res, req)
			}
		}()
		next.ServeHTTP(res, req)
	})
}
