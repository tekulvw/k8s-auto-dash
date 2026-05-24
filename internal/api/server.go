package api

import (
	"context"
	"net/http"

	v1 "github.com/anomalyco/k8s-auto-dash/api/v1alpha1"
)

type ConfigMutator interface {
	Mutate(ctx context.Context, fn func(*v1.DashboardConfigSpec) error) error
}

type Server struct {
	state   *State
	mutator ConfigMutator
	bus     *Bus
	mux     *http.ServeMux
}

func NewServer(state *State) *Server {
	return NewServerFull(state, nil, nil)
}

func NewServerWith(state *State, m ConfigMutator) *Server {
	return NewServerFull(state, m, nil)
}

func NewServerFull(state *State, m ConfigMutator, b *Bus) *Server {
	s := &Server{state: state, mutator: m, bus: b, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/tiles", s.handleTiles)
	s.mux.HandleFunc("GET /api/events", s.handleEvents)
	s.mux.HandleFunc("PATCH /api/config", s.handlePatchConfig)
	s.mux.HandleFunc("PUT /api/config/groups", s.handlePutGroups)
	s.mux.HandleFunc("POST /api/config/bookmarks", s.handlePostBookmark)
	s.mux.HandleFunc("DELETE /api/config/bookmarks/{id}", s.handleDeleteBookmark)
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	s.mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
}
