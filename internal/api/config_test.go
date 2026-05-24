package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "github.com/anomalyco/k8s-auto-dash/api/v1alpha1"
)

// fakeMutator implements ConfigMutator for handler tests.
type fakeMutator struct {
	spec v1.DashboardConfigSpec
}

func (f *fakeMutator) Mutate(_ context.Context, fn func(*v1.DashboardConfigSpec) error) error {
	return fn(&f.spec)
}

func TestPatchConfig_AppliesPartialMerge(t *testing.T) {
	m := &fakeMutator{}
	srv := NewServerWith(NewState(), m)

	body := strings.NewReader(`{
		"settings": {"title": "Home"},
		"tiles": [{"id":"a/b/c","name":"X"}]
	}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/config", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, 200, rec.Code)
	assert.Equal(t, "Home", m.spec.Settings.Title)
	require.Len(t, m.spec.Tiles, 1)
	assert.Equal(t, "X", m.spec.Tiles[0].Name)
}

func TestPatchConfig_TilesArrayUpsertsByID(t *testing.T) {
	m := &fakeMutator{spec: v1.DashboardConfigSpec{
		Tiles: []v1.TileOverride{{ID: "x", Name: "old"}},
	}}
	srv := NewServerWith(NewState(), m)

	body := strings.NewReader(`{"tiles":[{"id":"x","name":"new"},{"id":"y","name":"Y"}]}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/config", body)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, 200, rec.Code)
	require.Len(t, m.spec.Tiles, 2)

	byID := map[string]string{}
	for _, t := range m.spec.Tiles {
		byID[t.ID] = t.Name
	}
	assert.Equal(t, "new", byID["x"])
	assert.Equal(t, "Y", byID["y"])
}

func TestPostBookmark_Append(t *testing.T) {
	m := &fakeMutator{}
	srv := NewServerWith(NewState(), m)

	body := strings.NewReader(`{"id":"r","name":"Router","url":"https://r"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/config/bookmarks", body)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, 201, rec.Code)
	require.Len(t, m.spec.Bookmarks, 1)
	assert.Equal(t, "Router", m.spec.Bookmarks[0].Name)
}

func TestDeleteBookmark(t *testing.T) {
	m := &fakeMutator{spec: v1.DashboardConfigSpec{
		Bookmarks: []v1.Bookmark{{ID: "r", Name: "Router"}, {ID: "s", Name: "Switch"}},
	}}
	srv := NewServerWith(NewState(), m)
	req := httptest.NewRequest(http.MethodDelete, "/api/config/bookmarks/r", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, 204, rec.Code)
	require.Len(t, m.spec.Bookmarks, 1)
	assert.Equal(t, "s", m.spec.Bookmarks[0].ID)
}

func TestPutGroups_Replaces(t *testing.T) {
	m := &fakeMutator{spec: v1.DashboardConfigSpec{
		Groups: []v1.GroupSpec{{ID: "old", Name: "Old"}},
	}}
	srv := NewServerWith(NewState(), m)

	body := strings.NewReader(`[{"id":"a","name":"A","order":0},{"id":"b","name":"B","order":1}]`)
	req := httptest.NewRequest(http.MethodPut, "/api/config/groups", body)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, 200, rec.Code)
	require.Len(t, m.spec.Groups, 2)
	assert.Equal(t, "a", m.spec.Groups[0].ID)
}

var _ = json.Marshal // silence unused
