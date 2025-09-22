package middleware

import (
	"net/http"

	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"go.uber.org/zap"
)

func RecoveryMiddleware(next http.Handler, fallBack http.Handler, lgr *logger.Logger) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				lgr.Error("fatal error in middleware chain, panic recovered",
					zap.Any("panic", rec),
				)
				fallBack.ServeHTTP(res, req)
			}
		}()
		next.ServeHTTP(res, req)
	})
}
