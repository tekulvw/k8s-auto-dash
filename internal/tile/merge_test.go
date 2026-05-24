package tile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "github.com/tekulvw/k8s-auto-dash/api/v1alpha1"
)

func TestApplyOverrides_FieldsWin(t *testing.T) {
	derived := []Tile{{
		ID:    "media/jellyfin/jellyfin.example.com",
		Name:  "Jellyfin",
		URL:   "https://jellyfin.example.com",
		Icon:  "jellyfin",
		Group: "media",
	}}
	overrides := []v1.TileOverride{{
		ID:    "media/jellyfin/jellyfin.example.com",
		Name:  "Jelly",
		Icon:  "plex",
		Group: "stream",
		Order: 3,
	}}

	got := ApplyOverrides(derived, overrides)
	assert.Equal(t, "Jelly", got[0].Name)
	assert.Equal(t, "plex", got[0].Icon)
	assert.Equal(t, "stream", got[0].Group)
	assert.Equal(t, 3, got[0].Order)
	assert.Equal(t, "https://jellyfin.example.com", got[0].URL,
		"unset override does not clobber default")
}

func TestApplyOverrides_HiddenFlag(t *testing.T) {
	derived := []Tile{{ID: "x/y/z"}}
	overrides := []v1.TileOverride{{ID: "x/y/z", Hidden: true}}
	got := ApplyOverrides(derived, overrides)
	assert.True(t, got[0].Hidden)
}

func TestApplyOverrides_OrphanedOverrideIgnored(t *testing.T) {
	derived := []Tile{{ID: "x/y/z"}}
	overrides := []v1.TileOverride{{ID: "gone/route/host", Name: "x"}}
	got := ApplyOverrides(derived, overrides)
	assert.Len(t, got, 1)
	assert.Equal(t, "x/y/z", got[0].ID)
}

func TestBookmarksToTiles(t *testing.T) {
	bms := []v1.Bookmark{{
		ID: "router", Name: "Router", URL: "https://192.168.1.1",
		Icon: "ubiquiti", Group: "infra", Order: 9,
	}}
	got := BookmarksToTiles(bms)
	assert.Len(t, got, 1)
	assert.Equal(t, SourceBookmark, got[0].Source)
	assert.Equal(t, "router", got[0].ID)
	assert.Equal(t, "https://192.168.1.1", got[0].URL)
	assert.Nil(t, got[0].K8s)
}

func TestOrphanedOverrideIDs(t *testing.T) {
	derived := []Tile{{ID: "a"}, {ID: "b"}}
	overrides := []v1.TileOverride{{ID: "a"}, {ID: "ghost"}}
	assert.Equal(t, []string{"ghost"}, OrphanedOverrideIDs(derived, overrides))
}
