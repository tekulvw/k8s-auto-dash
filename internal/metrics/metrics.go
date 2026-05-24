package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Registry struct {
	Registry        *prometheus.Registry
	DiscoveredTiles prometheus.Gauge
	ProbeTotal      *prometheus.CounterVec
	ProbeLatency    *prometheus.HistogramVec
	ConfigWrites    prometheus.Counter
	SSEClients      prometheus.Gauge
}

func New() *Registry {
	r := prometheus.NewRegistry()
	m := &Registry{
		Registry: r,
		DiscoveredTiles: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "kad_discovered_tiles",
			Help: "Number of tiles currently discovered.",
		}),
		ProbeTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "kad_probe_total",
			Help: "Health-check probes by resulting state.",
		}, []string{"state"}),
		ProbeLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "kad_probe_latency_seconds",
			Help:    "Health-check probe latency.",
			Buckets: prometheus.DefBuckets,
		}, []string{"state"}),
		ConfigWrites: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "kad_config_writes_total",
			Help: "DashboardConfig writes.",
		}),
		SSEClients: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "kad_sse_clients",
			Help: "Connected SSE subscribers.",
		}),
	}
	r.MustRegister(m.DiscoveredTiles, m.ProbeTotal, m.ProbeLatency, m.ConfigWrites, m.SSEClients)
	return m
}
