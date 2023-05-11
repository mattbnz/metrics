package prom

import (
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"mattb.nz/web/metrics/metrics"
)

// Implements prometheus Collector interface to export event data
type Collector struct {
}

// Descriptors for our exports
var (
	// Per Monitor Stats
	mEvents = prometheus.NewDesc(
		"events_total",
		"Number of events",
		[]string{"event", "site"}, nil,
	)
)

func (c Collector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

// Helper to export a counter metric
func (c Collector) emitCounter(val uint, ts time.Time, desc *prometheus.Desc, ch chan<- prometheus.Metric, labels ...string) {
	m, err := prometheus.NewConstMetric(
		desc, prometheus.CounterValue, float64(val), labels...,
	)
	if err != nil {
		log.Printf("Failed to export %v for %v: %v", *desc, labels, err)
		return
	}
	if ts.IsZero() {
		ts = time.Now()
	}
	ch <- prometheus.NewMetricWithTimestamp(ts, m)
}

// Helper to export a map of counters
func (c Collector) emitCounterMap(m collectMap, ts time.Time, desc *prometheus.Desc, ch chan<- prometheus.Metric) {
	/*for k, v := range m {
		c.emitCounter(k, v, ts, desc, ch)
	}*/
}

// Helper to export a gauge metric
func (c Collector) emitGauge(label string, val float64, ts time.Time, desc *prometheus.Desc, ch chan<- prometheus.Metric) {
	m, err := prometheus.NewConstMetric(
		desc, prometheus.GaugeValue, val, label,
	)
	if err != nil {
		log.Printf("Failed to export %v for %s: %v", *desc, label, err)
		return
	}
	if ts.IsZero() {
		ts = time.Now()
	}
	ch <- prometheus.NewMetricWithTimestamp(ts, m)
}

type collectMap map[string]uint

func (m collectMap) Inc(key string) {
	if key != "" {
		m[key] += 1
	}
}

func (c Collector) Collect(ch chan<- prometheus.Metric) {
	for site, data := range metrics.Sites {
		for event, count := range data.EventCount {
			c.emitCounter(count, time.Now(), mEvents, ch, string(event), site)
		}
	}
}
