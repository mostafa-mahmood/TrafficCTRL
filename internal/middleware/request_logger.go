package middleware

import (
	"net/http"

	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"go.uber.org/zap"
)

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
