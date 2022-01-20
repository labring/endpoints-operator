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
	numCepsKey              = "cep_num_cpes"
	numCheckedKey           = "cep_num_checked"
	numCheckFailedKey       = "cep_num_check_failed"
	numCheckSuccessfulKey   = "cep_num_check_successful"
	checkDurationSecondsKey = "cep_check_duration_seconds"

	cepLabel   = "clusterEndpointName"
	nameSpaces = "nameSpacesName"
)

func NewMetricsInfo() *MetricsInfo {
	return &MetricsInfo{
		metrics: map[string]prometheus.Collector{
			numCepsKey: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: numCepsKey,
					Help: "Total number of ceps",
				},
				[]string{"totalCeps", nameSpaces},
			),

			numCheckedKey: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: numCheckedKey,
					Help: "Total number of check",
				},
				[]string{cepLabel, nameSpaces},
			),

			numCheckFailedKey: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: numCheckFailedKey,
					Help: "Total number of failed check",
				},
				[]string{cepLabel, nameSpaces},
			),

			numCheckSuccessfulKey: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: numCheckSuccessfulKey,
					Help: "Total number of successful check",
				},
				[]string{cepLabel, nameSpaces},
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
				[]string{cepLabel, nameSpaces},
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
func (m *MetricsInfo) RecordCheck(epname, ns string) {

	if pm, ok := m.metrics[numCheckedKey].(*prometheus.CounterVec); ok {
		pm.WithLabelValues(epname, ns).Inc()
	}
}

// RecordFailedCheck updates the total number of successful checked.
func (m *MetricsInfo) RecordFailedCheck(epname, ns string) {
	if pm, ok := m.metrics[numCheckFailedKey].(*prometheus.CounterVec); ok {
		pm.WithLabelValues(epname, ns).Inc()
	}
}

// RecordSuccessfulCheck updates the total number of successful checked.
func (m *MetricsInfo) RecordSuccessfulCheck(epname, ns string) {
	if pm, ok := m.metrics[numCheckSuccessfulKey].(*prometheus.CounterVec); ok {
		pm.WithLabelValues(epname, ns).Inc()
	}
}

// RecordCheckDuration records the number of seconds taken by a checked.
func (m *MetricsInfo) RecordCheckDuration(epname, ns string, seconds float64) {
	if c, ok := m.metrics[checkDurationSecondsKey].(*prometheus.HistogramVec); ok {
		c.WithLabelValues(epname, ns).Observe(seconds)
	}
}

func toSeconds(d time.Duration) float64 {
	return float64(d / time.Second)
}
