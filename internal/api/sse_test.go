package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSE_DeliversEvents(t *testing.T) {
	bus := NewBus()
	srv := httptest.NewServer(newSSEHandler(bus))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/events", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	time.Sleep(50 * time.Millisecond)
	bus.Publish(Event{Type: "tile-added", Data: map[string]any{"id": "x"}})

	br := bufio.NewReader(resp.Body)
	got, err := readSSEEvent(br, 2*time.Second)
	require.NoError(t, err)
	assert.Contains(t, got, "event: tile-added")
	assert.Contains(t, got, `"id":"x"`)
}

func readSSEEvent(br *bufio.Reader, timeout time.Duration) (string, error) {
	type result struct {
		s   string
		err error
	}
	ch := make(chan result, 1)
	go func() {
		var sb strings.Builder
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				ch <- result{sb.String(), err}
				return
			}
			sb.WriteString(line)
			if line == "\n" {
				ch <- result{sb.String(), nil}
				return
			}
		}
	}()
	select {
	case r := <-ch:
		return r.s, r.err
	case <-time.After(timeout):
		return "", context.DeadlineExceeded
	}
}

func newSSEHandler(b *Bus) http.Handler {
	s := &Server{state: NewState(), bus: b, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /events", s.handleEvents)
	return s.mux
}
