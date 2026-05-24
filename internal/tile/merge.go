package tile

import (
	"sort"

	v1 "github.com/tekulvw/k8s-auto-dash/api/v1alpha1"
)

// ApplyOverrides returns a copy of derived with overrides applied.
// Empty string and zero int fields in an override do NOT clobber the
// derived default — only set fields win. Pointer fields (Hidden) are
// always applied because their zero value is meaningful.
func ApplyOverrides(derived []Tile, overrides []v1.TileOverride) []Tile {
	idx := make(map[string]v1.TileOverride, len(overrides))
	for _, o := range overrides {
		idx[o.ID] = o
	}
	out := make([]Tile, len(derived))
	for i, t := range derived {
		out[i] = t
		o, ok := idx[t.ID]
		if !ok {
			continue
		}
		if o.Name != "" {
			out[i].Name = o.Name
		}
		if o.Description != "" {
			out[i].Description = o.Description
		}
		if o.Icon != "" {
			out[i].Icon = o.Icon
		}
		if o.Group != "" {
			out[i].Group = o.Group
		}
		if o.URL != "" {
			out[i].URL = o.URL
		}
		if o.Order != 0 {
			out[i].Order = o.Order
		}
		if o.InsecureSkipVerify != nil {
			out[i].InsecureSkipVerify = *o.InsecureSkipVerify
		}
		out[i].Hidden = o.Hidden
	}
	return out
}

// BookmarksToTiles converts bookmark specs into tiles for the merged view.
func BookmarksToTiles(bms []v1.Bookmark) []Tile {
	out := make([]Tile, len(bms))
	for i, b := range bms {
		out[i] = Tile{
			ID:     b.ID,
			Source: SourceBookmark,
			Name:   b.Name,
			URL:    b.URL,
			Icon:   b.Icon,
			Group:  b.Group,
			Order:  b.Order,
			Status: Status{State: StateUnknown},
		}
	}
	return out
}

// OrphanedOverrideIDs returns override IDs that no longer match any
// derived tile, so the UI can offer cleanup.
func OrphanedOverrideIDs(derived []Tile, overrides []v1.TileOverride) []string {
	have := make(map[string]bool, len(derived))
	for _, t := range derived {
		have[t.ID] = true
	}
	var orphans []string
	for _, o := range overrides {
		if !have[o.ID] {
			orphans = append(orphans, o.ID)
		}
	}
	sort.Strings(orphans)
	return orphans
}
