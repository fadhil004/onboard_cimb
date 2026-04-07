package middleware

import (
	"net/http"
	"time"

	"rest-api-bank/pkg/metrics"
)

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

		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()

		metrics.HTTPRequestTotal.
				WithLabelValues(r.Method, r.URL.Path, http.StatusText(rw.statusCode)).
				Inc()

		metrics.HTTPRequestDuration.
				WithLabelValues(r.Method, r.URL.Path).
				Observe(duration)
	})
}