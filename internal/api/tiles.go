package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleTiles(w http.ResponseWriter, _ *http.Request) {
	view := s.state.View()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(view)
}
