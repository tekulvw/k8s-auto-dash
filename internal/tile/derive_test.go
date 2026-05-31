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

func withServiceBackend(r *gwv1.HTTPRoute, svcName string, port int) *gwv1.HTTPRoute {
	r.Spec.Rules = []gwv1.HTTPRouteRule{{
		BackendRefs: []gwv1.HTTPBackendRef{{
			BackendRef: gwv1.BackendRef{
				BackendObjectReference: gwv1.BackendObjectReference{
					Name: gwv1.ObjectName(svcName),
					Port: ptr(gwv1.PortNumber(port)),
				},
			},
		}},
	}}
	return r
}

func withServiceBackendInNS(r *gwv1.HTTPRoute, svcName, ns string, port int) *gwv1.HTTPRoute {
	r.Spec.Rules = []gwv1.HTTPRouteRule{{
		BackendRefs: []gwv1.HTTPBackendRef{{
			BackendRef: gwv1.BackendRef{
				BackendObjectReference: gwv1.BackendObjectReference{
					Name:      gwv1.ObjectName(svcName),
					Namespace: ptr(gwv1.Namespace(ns)),
					Port:      ptr(gwv1.PortNumber(port)),
				},
			},
		}},
	}}
	return r
}

func withBackendRef(r *gwv1.HTTPRoute, backendName string, port int) *gwv1.HTTPRoute {
	r.Spec.Rules = []gwv1.HTTPRouteRule{{
		BackendRefs: []gwv1.HTTPBackendRef{{
			BackendRef: gwv1.BackendRef{
				BackendObjectReference: gwv1.BackendObjectReference{
					Kind: ptr(gwv1.Kind("Backend")),
					Name: gwv1.ObjectName(backendName),
					Port: ptr(gwv1.PortNumber(port)),
				},
			},
		}},
	}}
	return r
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

	got := Derive(routes, gateways, nil)

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

	got := Derive(routes, gateways, nil)

	assert.Len(t, got, 2)
}

func TestDerive_WildcardHostnameSkipped(t *testing.T) {
	routes := []*gwv1.HTTPRoute{
		newRoute("media", "x", []string{"*.example.com"},
			[]gwv1.ParentReference{parent("gateway", "ext")}),
	}
	gateways := map[GatewayKey]bool{{Namespace: "gateway", Name: "ext"}: true}

	got := Derive(routes, gateways, nil)
	assert.Empty(t, got)
}

func TestDerive_EmptyHostnameSkipped(t *testing.T) {
	routes := []*gwv1.HTTPRoute{
		newRoute("media", "x", []string{""},
			[]gwv1.ParentReference{parent("gateway", "ext")}),
	}
	got := Derive(routes, map[GatewayKey]bool{{Namespace: "gateway", Name: "ext"}: true}, nil)
	assert.Empty(t, got)
}

func TestDerive_UnresolvedParentRefSkipped(t *testing.T) {
	routes := []*gwv1.HTTPRoute{
		newRoute("media", "x", []string{"a.example.com"},
			[]gwv1.ParentReference{parent("gateway", "missing")}),
	}
	got := Derive(routes, map[GatewayKey]bool{}, nil)
	assert.Empty(t, got)
}

func TestDerive_ParentRefWithoutNamespaceUsesRouteNamespace(t *testing.T) {
	r := newRoute("media", "x", []string{"a.example.com"}, nil)
	r.Spec.ParentRefs = []gwv1.ParentReference{{Name: gwv1.ObjectName("ext")}}
	gateways := map[GatewayKey]bool{{Namespace: "media", Name: "ext"}: true}

	got := Derive([]*gwv1.HTTPRoute{r}, gateways, nil)
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

	a := Derive(routes, gateways, nil)
	b := Derive(routes, gateways, nil)
	assert.Equal(t, a, b)
	assert.Equal(t, "a/y/y.example.com", a[0].ID, "sorted by id ascending")
}

func TestDerive_ServiceBackendSetsHealthCheckURL(t *testing.T) {
	r := withServiceBackend(
		newRoute("media", "jellyfin", []string{"jellyfin.vpn.example.com"},
			[]gwv1.ParentReference{parent("gateway", "ext")}),
		"jellyfin", 8096,
	)
	gateways := map[GatewayKey]bool{{Namespace: "gateway", Name: "ext"}: true}

	got := Derive([]*gwv1.HTTPRoute{r}, gateways, nil)

	assert.Len(t, got, 1)
	assert.Equal(t, "http://jellyfin.media.svc.cluster.local:8096", got[0].HealthCheckURL)
	assert.Equal(t, "https://jellyfin.vpn.example.com", got[0].URL, "display URL unchanged")
}

func TestDerive_ServiceBackendCrossNamespace(t *testing.T) {
	r := withServiceBackendInNS(
		newRoute("gateway", "ha", []string{"home.vpn.example.com"},
			[]gwv1.ParentReference{parent("gateway", "ext")}),
		"home-assistant", "homeassistant", 8123,
	)
	gateways := map[GatewayKey]bool{{Namespace: "gateway", Name: "ext"}: true}

	got := Derive([]*gwv1.HTTPRoute{r}, gateways, nil)

	assert.Len(t, got, 1)
	assert.Equal(t, "http://home-assistant.homeassistant.svc.cluster.local:8123", got[0].HealthCheckURL)
}

func TestDerive_BackendRefResolvesToIPEndpoint(t *testing.T) {
	r := withBackendRef(
		newRoute("nas", "nas-console", []string{"nas.vpn.example.com"},
			[]gwv1.ParentReference{parent("gateway", "ext")}),
		"nas-console", 9001,
	)
	gateways := map[GatewayKey]bool{{Namespace: "gateway", Name: "ext"}: true}
	backends := BackendMap{
		"nas/nas-console": {Address: "192.168.1.204", Port: 5000},
	}

	got := Derive([]*gwv1.HTTPRoute{r}, gateways, backends)

	assert.Len(t, got, 1)
	assert.Equal(t, "http://192.168.1.204:5000", got[0].HealthCheckURL)
}

func TestDerive_BackendRefMissingFromMapLeavesHealthCheckURLEmpty(t *testing.T) {
	r := withBackendRef(
		newRoute("nas", "nas-console", []string{"nas.vpn.example.com"},
			[]gwv1.ParentReference{parent("gateway", "ext")}),
		"nas-console", 9001,
	)
	gateways := map[GatewayKey]bool{{Namespace: "gateway", Name: "ext"}: true}

	got := Derive([]*gwv1.HTTPRoute{r}, gateways, BackendMap{})

	assert.Len(t, got, 1)
	assert.Empty(t, got[0].HealthCheckURL)
}

func TestDerive_NoBackendRefsLeavesHealthCheckURLEmpty(t *testing.T) {
	r := newRoute("media", "x", []string{"x.example.com"},
		[]gwv1.ParentReference{parent("gateway", "ext")})
	gateways := map[GatewayKey]bool{{Namespace: "gateway", Name: "ext"}: true}

	got := Derive([]*gwv1.HTTPRoute{r}, gateways, nil)

	assert.Len(t, got, 1)
	assert.Empty(t, got[0].HealthCheckURL)
}
