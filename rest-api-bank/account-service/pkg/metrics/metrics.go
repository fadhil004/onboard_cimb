package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	HTTPRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "route", "status_class", "status_code"},
	)

	HTTPRequestErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_errors_total",
			Help: "Total HTTP request errors",
		},
		[]string{"method", "route"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration",
		},
		[]string{"method", "route"},
	)

	HTTPInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_in_flight_requests",
			Help: "Current in-flight requests",
		},
	)

	DBQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	DBQueryErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_query_errors_total",
			Help: "Database query errors",
		},
		[]string{"operation"},
	)
)

func Init() {
	prometheus.MustRegister(
		HTTPRequestTotal,
		HTTPRequestErrors,
		HTTPRequestDuration,
		HTTPInFlight,
		DBQueryDuration,
		DBQueryErrors,
	)
}

func StatusClass(code int) string {
	switch {
	case code >= 100 && code < 200:
		return "1xx"
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "unknown"
	}
}
