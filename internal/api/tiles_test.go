package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/anomalyco/k8s-auto-dash/internal/tile"
)

func TestHandleTiles_ReturnsJSON(t *testing.T) {
	st := NewState()
	st.SetDerived([]tile.Tile{{ID: "a", Name: "A", Group: "g"}})
	srv := NewServer(st)

	req := httptest.NewRequest(http.MethodGet, "/api/tiles", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, 200, rec.Code)
	var v View
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v))
	assert.Len(t, v.Tiles, 1)
	assert.Equal(t, "A", v.Tiles[0].Name)
}
