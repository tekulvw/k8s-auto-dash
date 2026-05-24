package api

import (
	"context"
	"net/http"

	v1 "github.com/anomalyco/k8s-auto-dash/api/v1alpha1"
)

// ConfigMutator is the narrow interface the API needs from config.Store.
type ConfigMutator interface {
	Mutate(ctx context.Context, fn func(*v1.DashboardConfigSpec) error) error
}

type Server struct {
	state   *State
	mutator ConfigMutator
	mux     *http.ServeMux
}

func NewServer(state *State) *Server {
	return NewServerWith(state, nil)
}

func NewServerWith(state *State, m ConfigMutator) *Server {
	s := &Server{state: state, mutator: m, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/tiles", s.handleTiles)
	s.mux.HandleFunc("PATCH /api/config", s.handlePatchConfig)
	s.mux.HandleFunc("PUT /api/config/groups", s.handlePutGroups)
	s.mux.HandleFunc("POST /api/config/bookmarks", s.handlePostBookmark)
	s.mux.HandleFunc("DELETE /api/config/bookmarks/{id}", s.handleDeleteBookmark)
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})
	s.mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})
}
