package api

import (
	"sort"
	"sync"

	v1 "github.com/tekulvw/k8s-auto-dash/api/v1alpha1"
	"github.com/tekulvw/k8s-auto-dash/internal/tile"
)

// ViewTile is the wire shape returned from /api/tiles. It mirrors
// tile.Tile but is a separate type so we can evolve the API without
// changing internal structs.
type ViewTile = tile.Tile

type ViewGroup = tile.Group

type View struct {
	Groups []ViewGroup `json:"groups"`
	Tiles  []ViewTile  `json:"tiles"`
}

type State struct {
	mu       sync.RWMutex
	derived  []tile.Tile
	spec     v1.DashboardConfigSpec
	statuses map[string]tile.Status
}

func NewState() *State {
	return &State{statuses: map[string]tile.Status{}}
}

func (s *State) SetDerived(d []tile.Tile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.derived = append([]tile.Tile(nil), d...)
}

func (s *State) SetConfig(spec v1.DashboardConfigSpec) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.spec = *spec.DeepCopy()
}

func (s *State) SetStatus(id string, st tile.Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses[id] = st
}

func (s *State) View() View {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tiles := tile.ApplyOverrides(s.derived, s.spec.Tiles)
	tiles = append(tiles, tile.BookmarksToTiles(s.spec.Bookmarks)...)
	for i := range tiles {
		if st, ok := s.statuses[tiles[i].ID]; ok {
			tiles[i].Status = st
		}
	}

	// Group output: prefer explicit GroupSpec entries; auto-add any
	// referenced group that has no spec (so tiles always have a home).
	groups := make([]ViewGroup, 0, len(s.spec.Groups))
	seen := map[string]bool{}
	for _, g := range s.spec.Groups {
		groups = append(groups, ViewGroup{ID: g.ID, Name: g.Name, Order: g.Order})
		seen[g.ID] = true
	}
	for _, t := range tiles {
		if t.Source != tile.SourceBookmark && t.Group != "" && !seen[t.Group] {
			groups = append(groups, ViewGroup{ID: t.Group, Name: t.Group, Order: len(groups)})
			seen[t.Group] = true
		}
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Order != groups[j].Order {
			return groups[i].Order < groups[j].Order
		}
		return groups[i].ID < groups[j].ID
	})

	return View{Groups: groups, Tiles: tiles}
}
