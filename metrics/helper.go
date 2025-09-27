package metrics

import (
	"time"
)

func TrackRequest(method string, endpoint string) func() {
	RequestsInFlight.Inc()

	start := time.Now()

	TotalRequests.WithLabelValues(method, endpoint).Inc()

	return func() {
		RequestsInFlight.Dec()

		duration := time.Since(start).Seconds()
		RequestDuration.WithLabelValues(method, endpoint).Observe(duration)
	}
}
