package api

import (
	"encoding/json"
	"net/http"

	v1 "github.com/tekulvw/k8s-auto-dash/api/v1alpha1"
)

func (s *Server) requireMutator(w http.ResponseWriter) bool {
	if s.mutator == nil {
		http.Error(w, "config mutator not configured", 501)
		return false
	}
	return true
}

// partialSpec mirrors DashboardConfigSpec so we can detect which top-
// level fields were present in the request body (presence-aware merge
// for arrays; nested struct merge for settings).
type partialSpec struct {
	Settings  *v1.Settings        `json:"settings,omitempty"`
	Groups    *[]v1.GroupSpec     `json:"groups,omitempty"`
	Tiles     *[]v1.TileOverride  `json:"tiles,omitempty"`
	Bookmarks *[]v1.Bookmark      `json:"bookmarks,omitempty"`
}

func (s *Server) handlePatchConfig(w http.ResponseWriter, r *http.Request) {
	if !s.requireMutator(w) {
		return
	}
	var p partialSpec
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	err := s.mutator.Mutate(r.Context(), func(spec *v1.DashboardConfigSpec) error {
		if p.Settings != nil {
			spec.Settings = *p.Settings
		}
		if p.Groups != nil {
			spec.Groups = *p.Groups
		}
		if p.Tiles != nil {
			spec.Tiles = upsertTiles(spec.Tiles, *p.Tiles)
		}
		if p.Bookmarks != nil {
			spec.Bookmarks = *p.Bookmarks
		}
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(200)
}

func upsertTiles(existing, incoming []v1.TileOverride) []v1.TileOverride {
	idx := make(map[string]int, len(existing))
	for i, t := range existing {
		idx[t.ID] = i
	}
	for _, t := range incoming {
		if i, ok := idx[t.ID]; ok {
			existing[i] = t
		} else {
			existing = append(existing, t)
			idx[t.ID] = len(existing) - 1
		}
	}
	return existing
}

func (s *Server) handlePutGroups(w http.ResponseWriter, r *http.Request) {
	if !s.requireMutator(w) {
		return
	}
	var groups []v1.GroupSpec
	if err := json.NewDecoder(r.Body).Decode(&groups); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	err := s.mutator.Mutate(r.Context(), func(spec *v1.DashboardConfigSpec) error {
		spec.Groups = groups
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(200)
}

func (s *Server) handlePostBookmark(w http.ResponseWriter, r *http.Request) {
	if !s.requireMutator(w) {
		return
	}
	var b v1.Bookmark
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if b.ID == "" {
		http.Error(w, "bookmark id required", 400)
		return
	}
	err := s.mutator.Mutate(r.Context(), func(spec *v1.DashboardConfigSpec) error {
		spec.Bookmarks = append(spec.Bookmarks, b)
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(201)
}

func (s *Server) handleDeleteBookmark(w http.ResponseWriter, r *http.Request) {
	if !s.requireMutator(w) {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id required", 400)
		return
	}
	err := s.mutator.Mutate(r.Context(), func(spec *v1.DashboardConfigSpec) error {
		out := spec.Bookmarks[:0]
		for _, b := range spec.Bookmarks {
			if b.ID != id {
				out = append(out, b)
			}
		}
		spec.Bookmarks = out
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(204)
}
