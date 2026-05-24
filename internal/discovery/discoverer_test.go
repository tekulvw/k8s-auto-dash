package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/anomalyco/k8s-auto-dash/internal/testenv"
)

func TestDiscoverer_EmitsTilesAfterRouteCreation(t *testing.T) {
	te := testenv.Start(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Pre-create a Gateway and a Namespace.
	require.NoError(t, te.Client.Create(ctx, corev1NS_("media")))
	require.NoError(t, te.Client.Create(ctx, corev1NS_("gateway")))
	gw := &gwv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "gateway", Name: "ext"},
		Spec: gwv1.GatewaySpec{
			GatewayClassName: "test",
			Listeners: []gwv1.Listener{{
				Name: "http", Port: 80, Protocol: gwv1.HTTPProtocolType,
			}},
		},
	}
	require.NoError(t, te.Client.Create(ctx, gw))

	d, err := New(te.Cfg, Options{Debounce: 50 * time.Millisecond})
	require.NoError(t, err)
	go func() { _ = d.Run(ctx) }()

	// Create an HTTPRoute after discoverer is running.
	ns := gwv1.Namespace("gateway")
	hr := &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "media", Name: "jellyfin"},
		Spec: gwv1.HTTPRouteSpec{
			Hostnames: []gwv1.Hostname{"jellyfin.example.com"},
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{{
					Namespace: &ns, Name: gwv1.ObjectName("ext"),
				}},
			},
		},
	}
	require.NoError(t, te.Client.Create(ctx, hr))

	// Poll the Tiles() channel for up to 10s, accepting any emission
	// that includes the expected tile. Earlier emissions (empty set
	// or set without the new route) are tolerated.
	deadline := time.After(10 * time.Second)
	for {
		select {
		case tiles := <-d.Tiles():
			for _, tt := range tiles {
				if tt.ID == "media/jellyfin/jellyfin.example.com" {
					return
				}
			}
		case <-deadline:
			t.Fatal("no tiles emitted with expected ID")
		}
	}
}

// assert is imported for potential future use in this test file.
var _ = assert.New
