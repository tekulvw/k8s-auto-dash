package health

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tekulvw/k8s-auto-dash/internal/tile"
)

type recorder struct {
	mu sync.Mutex
	m  map[string]tile.Status
}

func (r *recorder) Set(id string, s tile.Status) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[id] = s
}

func (r *recorder) Get(id string) (tile.Status, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.m[id]
	return s, ok
}

func TestPool_ProbesAllProvidedTargets(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	rec := &recorder{m: map[string]tile.Status{}}
	p := NewPool(PoolOptions{
		Workers:  3,
		Interval: 50 * time.Millisecond,
		Prober:   NewProber(ProbeOptions{Timeout: time.Second}),
		OnResult: rec.Set,
	})

	targets := []Target{
		{ID: "a", URL: srv.URL},
		{ID: "b", URL: srv.URL},
		{ID: "c", URL: srv.URL},
	}
	p.Set(targets)

	stop := make(chan struct{})
	go p.Run(stop)

	// Wait for at least one tick.
	deadline := time.After(2 * time.Second)
loop:
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for probes")
		default:
			if _, ok := rec.Get("a"); ok {
				if _, ok := rec.Get("b"); ok {
					if _, ok := rec.Get("c"); ok {
						break loop
					}
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	close(stop)
	sA, _ := rec.Get("a")
	assert.Equal(t, tile.StateUp, sA.State)
}

func TestPool_SetReplacesTargets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	rec := &recorder{m: map[string]tile.Status{}}
	p := NewPool(PoolOptions{
		Workers:  2,
		Interval: 40 * time.Millisecond,
		Prober:   NewProber(ProbeOptions{Timeout: time.Second}),
		OnResult: rec.Set,
	})
	p.Set([]Target{{ID: "old", URL: srv.URL}})
	stop := make(chan struct{})
	go p.Run(stop)

	// Wait until "old" was probed.
	require := func() {
		for i := 0; i < 200; i++ {
			if _, ok := rec.Get("old"); ok {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("never probed old")
	}
	require()

	p.Set([]Target{{ID: "new", URL: srv.URL}})
	for i := 0; i < 200; i++ {
		if _, ok := rec.Get("new"); ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	close(stop)

	_, ok := rec.Get("new")
	assert.True(t, ok, "new target should have been probed")
}
