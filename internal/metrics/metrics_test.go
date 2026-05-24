package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestMetrics_Exposed(t *testing.T) {
	reg := New()
	reg.DiscoveredTiles.Set(3)
	reg.ProbeTotal.WithLabelValues("up").Inc()
	reg.ProbeLatency.WithLabelValues("up").Observe(0.025)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	promhttp.HandlerFor(reg.Registry, promhttp.HandlerOpts{}).ServeHTTP(rec, req)

	body := rec.Body.String()
	require.Equal(t, 200, rec.Code)
	assert.True(t, strings.Contains(body, "kad_discovered_tiles 3"))
	assert.True(t, strings.Contains(body, `kad_probe_total{state="up"} 1`))
}
