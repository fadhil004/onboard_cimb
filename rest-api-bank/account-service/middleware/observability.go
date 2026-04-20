package middleware

import (
	"microservices-bank/account-service/helper"
	"microservices-bank/account-service/pkg/logger"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

var Tracer = otel.Tracer("account-service")

func Observability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, span := Tracer.Start(r.Context(), r.Method+" "+NormalizePath(r.URL.Path))
		defer span.End()

		next.ServeHTTP(w, r.WithContext(ctx))

		logger.Logger.Info("http request",
			zap.String("method", r.Method),
			zap.String("path", NormalizePath(r.URL.Path)),
			zap.Duration("duration", time.Since(start)),
			zap.String("trace_id", helper.GetTraceID(ctx)),
		)
	})
}
