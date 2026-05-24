package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/anomalyco/k8s-auto-dash/internal/api"
	configstore "github.com/anomalyco/k8s-auto-dash/internal/config"
	"github.com/anomalyco/k8s-auto-dash/internal/discovery"
	"github.com/anomalyco/k8s-auto-dash/internal/testenv"
)

func TestE2E_DiscoveryToAPI(t *testing.T) {
	te := testenv.Start(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := configstore.NewStore(te.Client)
	require.NoError(t, store.Bootstrap(ctx))

	d, err := discovery.New(te.Cfg, discovery.Options{Debounce: 50 * time.Millisecond})
	require.NoError(t, err)
	go func() { _ = d.Run(ctx) }()

	state := api.NewState()
	go func() {
		for tiles := range d.Tiles() {
			state.SetDerived(tiles)
		}
	}()

	// Create the Kubernetes resources.
	ns := gwv1.Namespace("default")
	require.NoError(t, te.Client.Create(ctx, &gwv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "ext"},
		Spec: gwv1.GatewaySpec{GatewayClassName: "test",
			Listeners: []gwv1.Listener{{Name: "http", Port: 80, Protocol: gwv1.HTTPProtocolType}}},
	}))
	require.NoError(t, te.Client.Create(ctx, &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "app"},
		Spec: gwv1.HTTPRouteSpec{
			Hostnames: []gwv1.Hostname{"app.example.com"},
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{{Namespace: &ns, Name: "ext"}},
			},
		},
	}))

	srv := api.NewServer(state)

	// Poll /api/tiles until the tile appears.
	deadline := time.After(10 * time.Second)
	for {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec,
			httptest.NewRequest(http.MethodGet, "/api/tiles", nil))
		var v api.View
		_ = json.Unmarshal(rec.Body.Bytes(), &v)
		if len(v.Tiles) == 1 && v.Tiles[0].ID == "default/app/app.example.com" {
			assert.Equal(t, "App", v.Tiles[0].Name)
			return
		}
		select {
		case <-deadline:
			t.Fatalf("tile never appeared; last body=%s", rec.Body.String())
		case <-time.After(100 * time.Millisecond):
		}
	}
}
