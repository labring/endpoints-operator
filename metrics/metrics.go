package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	"time"
)

// MetricsInfo Metrics contains Prometheus metrics
type MetricsInfo struct {
	metrics map[string]prometheus.Collector
}

const (
	numCheckedKey           = "ep_num_checked"
	numCheckFailedKey       = "ep_num_check_failed"
	numCheckSuccessfulKey   = "ep_num_check_successful"
	checkDurationSecondsKey = "ep_check_duration_seconds"

	epLabel = "endpointName"
)

func NewMetricsInfo() *MetricsInfo {
	return &MetricsInfo{
		metrics: map[string]prometheus.Collector{
			numCheckedKey: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: numCheckedKey,
					Help: "Total number of check",
				},
				[]string{epLabel},
			),

			numCheckFailedKey: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: numCheckFailedKey,
					Help: "Total number of failed check",
				},
				[]string{epLabel},
			),

			numCheckSuccessfulKey: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: numCheckSuccessfulKey,
					Help: "Total number of successful check",
				},
				[]string{epLabel},
			),

			checkDurationSecondsKey: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name: checkDurationSecondsKey,
					Help: "Time taken to complete check, in seconds",
					Buckets: []float64{
						15.0,
						30.0,
						toSeconds(1 * time.Minute),
						toSeconds(5 * time.Minute),
						toSeconds(10 * time.Minute),
						toSeconds(15 * time.Minute),
						toSeconds(30 * time.Minute),
						toSeconds(1 * time.Hour),
						toSeconds(2 * time.Hour),
						toSeconds(3 * time.Hour),
						toSeconds(4 * time.Hour),
						toSeconds(5 * time.Hour),
						toSeconds(6 * time.Hour),
						toSeconds(7 * time.Hour),
						toSeconds(8 * time.Hour),
						toSeconds(9 * time.Hour),
						toSeconds(10 * time.Hour),
					},
				},
				[]string{epLabel},
			),
		},
	}
}

func (m *MetricsInfo) RegisterAllMetrics() {
	for _, pm := range m.metrics {
		crmetrics.Registry.MustRegister(pm)
	}
}

// RecordCheck updates the total number of checked.
func (m *MetricsInfo) RecordCheck(policy string) {
	if pm, ok := m.metrics[numCheckedKey].(*prometheus.CounterVec); ok {
		pm.WithLabelValues(policy).Inc()
	}
}

// RecordFailedCheck updates the total number of successful checked.
func (m *MetricsInfo) RecordFailedCheck(policy string) {
	if pm, ok := m.metrics[numCheckFailedKey].(*prometheus.CounterVec); ok {
		pm.WithLabelValues(policy).Inc()
	}
}

// RecordSuccessfulCheck updates the total number of successful checked.
func (m *MetricsInfo) RecordSuccessfulCheck(policy string) {
	if pm, ok := m.metrics[numCheckSuccessfulKey].(*prometheus.CounterVec); ok {
		pm.WithLabelValues(policy).Inc()
	}
}

// RecordCheckDuration records the number of seconds taken by a checked.
func (m *MetricsInfo) RecordCheckDuration(policy string, seconds float64) {
	if c, ok := m.metrics[checkDurationSecondsKey].(*prometheus.HistogramVec); ok {
		c.WithLabelValues(policy).Observe(seconds)
	}
}

func toSeconds(d time.Duration) float64 {
	return float64(d / time.Second)
}
