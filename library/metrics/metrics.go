// Copyright © 2022 The sealyun Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

type Metrics interface {
	Registry()
	CounterVec(name ActionName) (*prometheus.CounterVec, error)
	GaugeVec(name ActionName) (*prometheus.GaugeVec, error)
	HistogramVec(name ActionName) (*prometheus.HistogramVec, error)
}

type ActionName string

type PromCounterVecActions []PromCounterVecAction

func (proms PromCounterVecActions) Get(name ActionName) *prometheus.CounterVec {
	for _, metric := range proms {
		if metric.Name == name {
			return &metric.Counter
		}
	}
	return nil
}

type PromGaugeVecActions []PromGaugeVecAction

func (proms PromGaugeVecActions) Get(name ActionName) *prometheus.GaugeVec {
	for _, metric := range proms {
		if metric.Name == name {
			return &metric.Gauge
		}
	}
	return nil
}

type PromHistogramVecActions []PromHistogramVecAction

func (proms PromHistogramVecActions) Get(name ActionName) *prometheus.HistogramVec {
	for _, metric := range proms {
		if metric.Name == name {
			return &metric.Histogram
		}
	}
	return nil
}

type PromCounterVecAction struct {
	Name    ActionName
	Counter prometheus.CounterVec
}

type PromGaugeVecAction struct {
	Name  ActionName
	Gauge prometheus.GaugeVec
}

type PromHistogramVecAction struct {
	Name      ActionName
	Histogram prometheus.HistogramVec
}

type IMetrics struct {
	Counters   PromCounterVecActions
	Gauges     PromGaugeVecActions
	Histograms PromHistogramVecActions
}

func (m *IMetrics) Registry() {
	for _, c := range m.Counters {
		crmetrics.Registry.MustRegister(c.Counter)
	}
	for _, g := range m.Gauges {
		crmetrics.Registry.MustRegister(g.Gauge)
	}
	for _, h := range m.Histograms {
		crmetrics.Registry.MustRegister(h.Histogram)
	}
}

func (m *IMetrics) CounterVec(name ActionName) (*prometheus.CounterVec, error) {
	counter := m.Counters.Get(name)
	if counter == nil {
		return nil, fmt.Errorf("not found counter name")
	}
	return counter, nil
}
func (m *IMetrics) GaugeVec(name ActionName) (*prometheus.GaugeVec, error) {
	gauge := m.Gauges.Get(name)
	if gauge == nil {
		return nil, fmt.Errorf("not found gauge name")
	}
	return gauge, nil
}
func (m *IMetrics) HistogramVec(name ActionName) (*prometheus.HistogramVec, error) {
	histogram := m.Histograms.Get(name)
	if histogram == nil {
		return nil, fmt.Errorf("not found histogram name")
	}
	return histogram, nil
}
