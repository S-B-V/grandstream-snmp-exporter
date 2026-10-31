package exporter

import (
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// CachedGatherer decouples the expensive SNMP collection from HTTP scrapes.
// Returned metric families are immutable once published.
type CachedGatherer struct {
	mu       sync.RWMutex
	families []*dto.MetricFamily
}

func (c *CachedGatherer) Gather() ([]*dto.MetricFamily, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.families, nil
}

func (c *CachedGatherer) publish(families []*dto.MetricFamily) {
	c.mu.Lock()
	c.families = families
	c.mu.Unlock()
}

// RunPolling executes immediately, then waits interval after the previous
// collection finishes. No overlapping SNMP requests occur in this process.
// A failed AP client-table walk never replaces the last good snapshot.
func RunPolling(source prometheus.Gatherer, cache *CachedGatherer, interval time.Duration, clientPollOK func() bool, deviceIP string, metrics *PollMetrics) {
	if interval < time.Second {
		interval = time.Second
	}
	go func() {
		for {
			started := time.Now()
			families, err := source.Gather()
			success := err == nil && (clientPollOK == nil || clientPollOK())
			metrics.Duration.Set(time.Since(started).Seconds())
			if success {
				cache.publish(families)
				metrics.Success.Set(1)
				metrics.LastSuccess.Set(float64(time.Now().Unix()))
			} else {
				metrics.Success.Set(0)
				log.Printf("SNMP polling failed for %s: gather_error=%v client_table_ok=%t; preserving previous snapshot", deviceIP, err, clientPollOK == nil || clientPollOK())
			}
			time.Sleep(interval)
		}
	}()
}

type PollMetrics struct {
	Success     prometheus.Gauge
	LastSuccess prometheus.Gauge
	Duration    prometheus.Gauge
}

func NewPollMetrics(deviceIP string) (*PollMetrics, prometheus.Gatherer) {
	labels := prometheus.Labels{"device_ip": deviceIP}
	m := &PollMetrics{
		Success:     prometheus.NewGauge(prometheus.GaugeOpts{Name: "grandstream_poll_success", Help: "Whether the latest complete SNMP collection succeeded", ConstLabels: labels}),
		LastSuccess: prometheus.NewGauge(prometheus.GaugeOpts{Name: "grandstream_poll_last_success_timestamp_seconds", Help: "Unix timestamp of latest successful SNMP collection", ConstLabels: labels}),
		Duration:    prometheus.NewGauge(prometheus.GaugeOpts{Name: "grandstream_poll_duration_seconds", Help: "Duration of latest SNMP collection", ConstLabels: labels}),
	}
	r := prometheus.NewRegistry()
	r.MustRegister(m.Success, m.LastSuccess, m.Duration)
	return m, r
}
