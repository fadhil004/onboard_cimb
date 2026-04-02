package middleware

import (
	"net/http"
	"rest-api-bank/helper"
	"rest-api-bank/pkg/logger"
	"time"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

var Tracer = otel.Tracer("rest-api-bank")

func Observability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, span := Tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
		defer span.End()

		next.ServeHTTP(w, r.WithContext(ctx))
		
		// traceID := span.SpanContext().TraceID().String()

		logger.Logger.Info("http request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Duration("duration", time.Since(start)),
			zap.String("trace_id", helper.GetTraceID(ctx)),
		)
	})
}