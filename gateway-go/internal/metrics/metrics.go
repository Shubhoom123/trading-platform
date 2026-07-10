// Package metrics holds the gateway's Prometheus instruments and the /metrics
// handler. Collectors are grouped in a struct (not package globals) so they can
// be threaded explicitly through the server and consumer and swapped in tests.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	reg *prometheus.Registry

	// Live WebSocket connections currently held open.
	ActiveWS prometheus.Gauge
	// Fills consumed from Kafka and fanned out to clients.
	FillsBroadcast prometheus.Counter
	// Book/quote cache outcomes.
	CacheHits   prometheus.Counter
	CacheMisses prometheus.Counter
}

func New() *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		reg: reg,
		ActiveWS: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "gateway_active_ws_connections",
			Help: "Currently open WebSocket connections",
		}),
		FillsBroadcast: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "gateway_fills_broadcast_total",
			Help: "Fills consumed from Kafka and broadcast to clients",
		}),
		CacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "gateway_book_cache_hits_total",
			Help: "Book/quote reads served from the Redis cache",
		}),
		CacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "gateway_book_cache_misses_total",
			Help: "Book/quote reads that fell back to the engine",
		}),
	}
	reg.MustRegister(m.ActiveWS, m.FillsBroadcast, m.CacheHits, m.CacheMisses)
	// Standard process + Go runtime metrics.
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	return m
}

// Handler serves the metrics in the Prometheus text exposition format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}
