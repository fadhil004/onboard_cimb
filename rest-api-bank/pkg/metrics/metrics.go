package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	HTTPRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_total",
			Help: "Number of Total HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
		},
		[]string{"method", "endpoint"},
	)

	DBQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Duration of database queries in seconds",
		},
		[]string{"query"},
	)
)

func Init() {
	prometheus.MustRegister(HTTPRequestTotal)
	prometheus.MustRegister(HTTPRequestDuration)
	prometheus.MustRegister(DBQueryDuration)
}