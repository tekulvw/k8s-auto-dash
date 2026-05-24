package api

import (
	"net/http"
)

type Server struct {
	state *State
	mux   *http.ServeMux
}

func NewServer(state *State) *Server {
	s := &Server{state: state, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/tiles", s.handleTiles)
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})
	s.mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})
}
