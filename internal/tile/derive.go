package tile

import (
	"fmt"
	"sort"
	"strings"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// BackendEndpoint holds a resolved address for a non-Service backend
// (e.g. an Envoy Gateway Backend resource pointing at an external host).
type BackendEndpoint struct {
	Address string
	Port    int32
}

// BackendMap maps "namespace/name" to a resolved BackendEndpoint.
// Pass nil to skip Backend resolution.
type BackendMap map[string]BackendEndpoint

// GatewayKey identifies a Gateway resource for parentRef resolution.
type GatewayKey struct {
	Namespace string
	Name      string
}

// Derive turns HTTPRoutes + a set of known Gateways into the canonical
// set of derived tiles (before override merging). Output is sorted by
// tile ID for deterministic results.
//
// backends maps "namespace/name" to a resolved BackendEndpoint for
// Envoy Gateway Backend resources; pass nil if not needed.
func Derive(routes []*gwv1.HTTPRoute, gateways map[GatewayKey]bool, backends BackendMap) []Tile {
	var tiles []Tile
	for _, hr := range routes {
		refs := resolveParents(hr, gateways)
		if len(refs) == 0 {
			continue
		}
		healthURL := deriveHealthCheckURL(hr, backends)
		for _, h := range hr.Spec.Hostnames {
			host := string(h)
			if host == "" || strings.Contains(host, "*") {
				continue
			}
			tiles = append(tiles, Tile{
				ID:             ComputeID(hr.Namespace, hr.Name, host),
				Source:         SourceHTTPRoute,
				Name:           DeriveName(host),
				URL:            DeriveURL(host),
				Icon:           DeriveIconSlug(host),
				Group:          hr.Namespace,
				Status:         Status{State: StateUnknown},
				HealthCheckURL: healthURL,
				K8s: &K8sInfo{
					Namespace:     hr.Namespace,
					HTTPRouteName: hr.Name,
					GatewayRefs:   refs,
				},
			})
		}
	}
	sort.Slice(tiles, func(i, j int) bool { return tiles[i].ID < tiles[j].ID })
	return tiles
}

// deriveHealthCheckURL returns the internal URL to use when probing the
// route's backend. For Service backends it constructs a cluster-local
// svc.cluster.local URL; for Envoy Gateway Backend resources it looks up
// the resolved IP endpoint. Returns "" if no backend can be resolved.
func deriveHealthCheckURL(hr *gwv1.HTTPRoute, backends BackendMap) string {
	for _, rule := range hr.Spec.Rules {
		for _, ref := range rule.BackendRefs {
			kind := "Service"
			if ref.Kind != nil && string(*ref.Kind) != "" {
				kind = string(*ref.Kind)
			}
			ns := hr.Namespace
			if ref.Namespace != nil && string(*ref.Namespace) != "" {
				ns = string(*ref.Namespace)
			}
			name := string(ref.Name)

			switch kind {
			case "Service":
				port := ""
				if ref.Port != nil {
					port = fmt.Sprintf(":%d", *ref.Port)
				}
				return "http://" + name + "." + ns + ".svc.cluster.local" + port
			case "Backend":
				key := ns + "/" + name
				if ep, ok := backends[key]; ok {
					return fmt.Sprintf("http://%s:%d", ep.Address, ep.Port)
				}
			}
		}
	}
	return ""
}

func resolveParents(hr *gwv1.HTTPRoute, gateways map[GatewayKey]bool) []GatewayRef {
	var out []GatewayRef
	for _, p := range hr.Spec.ParentRefs {
		ns := hr.Namespace
		if p.Namespace != nil && *p.Namespace != "" {
			ns = string(*p.Namespace)
		}
		key := GatewayKey{Namespace: ns, Name: string(p.Name)}
		if gateways[key] {
			out = append(out, GatewayRef{Namespace: ns, Name: string(p.Name)})
		}
	}
	return out
}
