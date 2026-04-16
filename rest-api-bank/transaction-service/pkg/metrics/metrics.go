package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	HTTPRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_request_total", Help: "Total HTTP requests"},
		[]string{"method", "route", "status_class", "status_code"},
	)
	HTTPRequestErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_request_errors_total", Help: "Total HTTP request errors"},
		[]string{"method", "route"},
	)
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "HTTP request duration"},
		[]string{"method", "route"},
	)
	HTTPInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{Name: "http_in_flight_requests", Help: "Current in-flight requests"},
	)
	TransferTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "transfer_total", Help: "Total transfer success"},
	)
	TransferFailed = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "transfer_failed_total", Help: "Total transfer failed"},
	)
	TransferAmount = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "transfer_amount",
			Help:    "Transfer amount distribution",
			Buckets: []float64{1000, 10000, 100000, 1000000, 10000000},
		},
	)
)

func Init() {
	prometheus.MustRegister(
		HTTPRequestTotal, HTTPRequestErrors, HTTPRequestDuration, HTTPInFlight,
		TransferTotal, TransferFailed, TransferAmount,
	)
}

func StatusClass(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "unknown"
	}
}
