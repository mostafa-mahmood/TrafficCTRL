package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	TotalRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total number of requests received (Whole Server)",
		},
		[]string{"method", "endpoint"},
	)

	TotalBypassedRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "requests_bypassed",
			Help: "Total number of requests with no endpoint rule || bypass = true",
		},
	)

	RequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "requests_in_flight",
			Help: "Current number of requests being processed",
		},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Histogram of request latencies",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	AllowedRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "rate_limit_requests_allowed_total",
			Help: "Total number of requests allowed by rate limiter",
		},
	)

	DeniedRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_requests_denied_total",
			Help: "Total number of requests denied by rate limiter",
		},
		[]string{"level"},
	)

	ReputationDistribution = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "reputation_score_distribution",
			Help:    "Distribution of reputation scores for requests",
			Buckets: []float64{0.0, 0.25, 0.5, 0.75, 1.0},
		},
	)

	RedisErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_errors_total",
			Help: "Total number of Redis errors",
		},
	)

	GlobalLimitErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "global_limit_errors_total",
			Help: "Total number of errors in global limiter",
		},
	)

	TenantLimitErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tenant_limit_errors_total",
			Help: "Total number of errors in tenant limiter",
		},
	)

	EndpointLimitErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "endpoint_limit_errors_total",
			Help: "Total number of errors in endpoint limiter",
		},
	)

	PanicRecoveries = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "panic_recoveries_total",
			Help: "Total number of recovered panics",
		},
	)
)

func Init() {
	prometheus.MustRegister(
		TotalRequests,
		TotalBypassedRequests,
		RequestsInFlight,
		RequestDuration,
		AllowedRequests,
		DeniedRequests,
		ReputationDistribution,
		RedisErrors,
		GlobalLimitErrors,
		TenantLimitErrors,
		EndpointLimitErrors,
		PanicRecoveries,
	)
}

func Handler() http.Handler {
	return promhttp.Handler()
}
