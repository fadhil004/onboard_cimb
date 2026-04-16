package middleware

import (
	"net/http"
	"time"
	"transaction-service/helper"
	"transaction-service/pkg/metrics"
)

var NormalizePath = helper.NormalizePath

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		metrics.HTTPInFlight.Inc()
		defer metrics.HTTPInFlight.Dec()

		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		route := NormalizePath(r.URL.Path)
		statusClass := metrics.StatusClass(rw.statusCode)

		metrics.HTTPRequestTotal.WithLabelValues(r.Method, route, statusClass, http.StatusText(rw.statusCode)).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, route).Observe(duration)

		if statusClass == "5xx" {
			metrics.HTTPRequestErrors.WithLabelValues(r.Method, route).Inc()
		}
	})
}
