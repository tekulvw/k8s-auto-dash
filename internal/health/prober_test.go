package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anomalyco/k8s-auto-dash/internal/tile"
	"github.com/stretchr/testify/assert"
)

func TestProbe_HEAD200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	p := NewProber(ProbeOptions{Timeout: 2 * time.Second})
	s := p.Probe(srv.URL, false)
	assert.Equal(t, tile.StateUp, s.State)
	assert.Equal(t, 200, s.StatusCode)
	assert.GreaterOrEqual(t, s.LatencyMs, int64(0))
}

func TestProbe_HEAD405FallsBackToGET(t *testing.T) {
	gets := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(405)
			return
		}
		gets++
		w.WriteHeader(200)
	}))
	defer srv.Close()

	p := NewProber(ProbeOptions{Timeout: 2 * time.Second})
	s := p.Probe(srv.URL, false)
	assert.Equal(t, tile.StateUp, s.State)
	assert.Equal(t, 1, gets)
}

func TestProbe_NetworkErrorDown(t *testing.T) {
	p := NewProber(ProbeOptions{Timeout: 200 * time.Millisecond})
	s := p.Probe("http://127.0.0.1:1", false) // closed port
	assert.Equal(t, tile.StateDown, s.State)
	assert.NotEmpty(t, s.Error)
}

func TestProbe_500Degraded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	p := NewProber(ProbeOptions{Timeout: 2 * time.Second})
	s := p.Probe(srv.URL, false)
	assert.Equal(t, tile.StateDegraded, s.State)
	assert.Equal(t, 500, s.StatusCode)
}

func TestProbe_UserAgentSet(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	NewProber(ProbeOptions{Timeout: time.Second, UserAgent: "k8s-auto-dash/test (health-check)"}).
		Probe(srv.URL, false)
	assert.Equal(t, "k8s-auto-dash/test (health-check)", ua)
}
