package tile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func ptr[T any](v T) *T { return &v }

func newRoute(ns, name string, hosts []string, parents []gwv1.ParentReference) *gwv1.HTTPRoute {
	hh := make([]gwv1.Hostname, len(hosts))
	for i, h := range hosts {
		hh[i] = gwv1.Hostname(h)
	}
	return &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec: gwv1.HTTPRouteSpec{
			Hostnames: hh,
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: parents,
			},
		},
	}
}

func parent(ns, name string) gwv1.ParentReference {
	return gwv1.ParentReference{
		Namespace: (*gwv1.Namespace)(ptr(gwv1.Namespace(ns))),
		Name:      gwv1.ObjectName(name),
	}
}

func TestDerive_BasicSingleHost(t *testing.T) {
	routes := []*gwv1.HTTPRoute{
		newRoute("media", "jellyfin",
			[]string{"jellyfin.example.com"},
			[]gwv1.ParentReference{parent("gateway", "ext")}),
	}
	gateways := map[GatewayKey]bool{{Namespace: "gateway", Name: "ext"}: true}

	got := Derive(routes, gateways)

	assert.Len(t, got, 1)
	assert.Equal(t, "media/jellyfin/jellyfin.example.com", got[0].ID)
	assert.Equal(t, "Jellyfin", got[0].Name)
	assert.Equal(t, "https://jellyfin.example.com", got[0].URL)
	assert.Equal(t, "jellyfin", got[0].Icon)
	assert.Equal(t, "media", got[0].Group)
	assert.Equal(t, SourceHTTPRoute, got[0].Source)
	assert.NotNil(t, got[0].K8s)
	assert.Equal(t, "media", got[0].K8s.Namespace)
	assert.Equal(t, "jellyfin", got[0].K8s.HTTPRouteName)
	assert.Equal(t, []GatewayRef{{Namespace: "gateway", Name: "ext"}},
		got[0].K8s.GatewayRefs)
}

func TestDerive_MultipleHostsProduceMultipleTiles(t *testing.T) {
	routes := []*gwv1.HTTPRoute{
		newRoute("media", "jellyfin",
			[]string{"jellyfin.example.com", "media.example.com"},
			[]gwv1.ParentReference{parent("gateway", "ext")}),
	}
	gateways := map[GatewayKey]bool{{Namespace: "gateway", Name: "ext"}: true}

	got := Derive(routes, gateways)

	assert.Len(t, got, 2)
}

func TestDerive_WildcardHostnameSkipped(t *testing.T) {
	routes := []*gwv1.HTTPRoute{
		newRoute("media", "x", []string{"*.example.com"},
			[]gwv1.ParentReference{parent("gateway", "ext")}),
	}
	gateways := map[GatewayKey]bool{{Namespace: "gateway", Name: "ext"}: true}

	got := Derive(routes, gateways)
	assert.Empty(t, got)
}

func TestDerive_EmptyHostnameSkipped(t *testing.T) {
	routes := []*gwv1.HTTPRoute{
		newRoute("media", "x", []string{""},
			[]gwv1.ParentReference{parent("gateway", "ext")}),
	}
	got := Derive(routes, map[GatewayKey]bool{{Namespace: "gateway", Name: "ext"}: true})
	assert.Empty(t, got)
}

func TestDerive_UnresolvedParentRefSkipped(t *testing.T) {
	routes := []*gwv1.HTTPRoute{
		newRoute("media", "x", []string{"a.example.com"},
			[]gwv1.ParentReference{parent("gateway", "missing")}),
	}
	got := Derive(routes, map[GatewayKey]bool{})
	assert.Empty(t, got)
}

func TestDerive_ParentRefWithoutNamespaceUsesRouteNamespace(t *testing.T) {
	r := newRoute("media", "x", []string{"a.example.com"}, nil)
	r.Spec.ParentRefs = []gwv1.ParentReference{{Name: gwv1.ObjectName("ext")}}
	gateways := map[GatewayKey]bool{{Namespace: "media", Name: "ext"}: true}

	got := Derive([]*gwv1.HTTPRoute{r}, gateways)
	assert.Len(t, got, 1)
	assert.Equal(t, []GatewayRef{{Namespace: "media", Name: "ext"}},
		got[0].K8s.GatewayRefs)
}

func TestDerive_Deterministic(t *testing.T) {
	routes := []*gwv1.HTTPRoute{
		newRoute("b", "z", []string{"z.example.com"},
			[]gwv1.ParentReference{parent("gateway", "ext")}),
		newRoute("a", "y", []string{"y.example.com"},
			[]gwv1.ParentReference{parent("gateway", "ext")}),
	}
	gateways := map[GatewayKey]bool{{Namespace: "gateway", Name: "ext"}: true}

	a := Derive(routes, gateways)
	b := Derive(routes, gateways)
	assert.Equal(t, a, b)
	assert.Equal(t, "a/y/y.example.com", a[0].ID, "sorted by id ascending")
}
