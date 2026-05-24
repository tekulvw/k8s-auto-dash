package tile

import (
	"sort"
	"strings"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// GatewayKey identifies a Gateway resource for parentRef resolution.
type GatewayKey struct {
	Namespace string
	Name      string
}

// Derive turns HTTPRoutes + a set of known Gateways into the canonical
// set of derived tiles (before override merging). Output is sorted by
// tile ID for deterministic results.
func Derive(routes []*gwv1.HTTPRoute, gateways map[GatewayKey]bool) []Tile {
	var tiles []Tile
	for _, hr := range routes {
		refs := resolveParents(hr, gateways)
		if len(refs) == 0 {
			continue
		}
		for _, h := range hr.Spec.Hostnames {
			host := string(h)
			if host == "" || strings.Contains(host, "*") {
				continue
			}
			tiles = append(tiles, Tile{
				ID:     ComputeID(hr.Namespace, hr.Name, host),
				Source: SourceHTTPRoute,
				Name:   DeriveName(host),
				URL:    DeriveURL(host),
				Icon:   DeriveIconSlug(host),
				Group:  hr.Namespace,
				Status: Status{State: StateUnknown},
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
