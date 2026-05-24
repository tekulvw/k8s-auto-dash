package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "github.com/tekulvw/k8s-auto-dash/api/v1alpha1"
	"github.com/tekulvw/k8s-auto-dash/internal/tile"
)

func TestState_MergedView(t *testing.T) {
	s := NewState()
	s.SetDerived([]tile.Tile{
		{ID: "media/jellyfin/jellyfin.example.com", Name: "Jellyfin", Group: "media"},
	})
	s.SetConfig(v1.DashboardConfigSpec{
		Groups: []v1.GroupSpec{{ID: "media", Name: "Media", Order: 0}},
		Tiles: []v1.TileOverride{{
			ID:   "media/jellyfin/jellyfin.example.com",
			Name: "Jelly",
		}},
		Bookmarks: []v1.Bookmark{{
			ID: "router", Name: "Router", URL: "https://r", Group: "infra",
		}},
	})
	s.SetStatus("media/jellyfin/jellyfin.example.com",
		tile.Status{State: tile.StateUp, StatusCode: 200})

	view := s.View()
	assert.Len(t, view.Groups, 1)
	assert.Len(t, view.Tiles, 2)

	byID := map[string]ViewTile{}
	for _, vt := range view.Tiles {
		byID[vt.ID] = vt
	}
	assert.Equal(t, "Jelly", byID["media/jellyfin/jellyfin.example.com"].Name)
	assert.Equal(t, tile.StateUp,
		byID["media/jellyfin/jellyfin.example.com"].Status.State)
	assert.Equal(t, tile.SourceBookmark, byID["router"].Source)
}
