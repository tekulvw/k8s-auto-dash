// Package discovery watches Gateway/HTTPRoute resources and emits
// derived tile sets on Tiles().
package discovery

import (
	"context"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/tekulvw/k8s-auto-dash/internal/tile"
)

var backendGVR = schema.GroupVersionResource{
	Group:    "gateway.envoyproxy.io",
	Version:  "v1alpha1",
	Resource: "backends",
}

type Options struct {
	Debounce time.Duration // default 250ms
}

type Discoverer struct {
	cfg     *rest.Config
	opts    Options
	cache   cache.Cache
	client  client.Client
	scheme  *runtime.Scheme
	out     chan []tile.Tile
	dirty   chan struct{}
	startMu sync.Once
}

func New(cfg *rest.Config, opts Options) (*Discoverer, error) {
	if opts.Debounce == 0 {
		opts.Debounce = 250 * time.Millisecond
	}
	s := runtime.NewScheme()
	if err := gwv1.Install(s); err != nil {
		return nil, err
	}
	c, err := cache.New(cfg, cache.Options{Scheme: s})
	if err != nil {
		return nil, err
	}
	// A non-caching client is used only for fetching Backend resources
	// (gateway.envoyproxy.io/v1alpha1), which are not registered in the
	// scheme and are fetched as unstructured on each compute.
	cl, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}
	return &Discoverer{
		cfg:    cfg,
		opts:   opts,
		cache:  c,
		client: cl,
		scheme: s,
		out:    make(chan []tile.Tile, 4),
		dirty:  make(chan struct{}, 1),
	}, nil
}

func (d *Discoverer) Tiles() <-chan []tile.Tile { return d.out }

// Run blocks until ctx is cancelled.
func (d *Discoverer) Run(ctx context.Context) error {
	hrInf, err := d.cache.GetInformer(ctx, &gwv1.HTTPRoute{})
	if err != nil {
		return err
	}
	gwInf, err := d.cache.GetInformer(ctx, &gwv1.Gateway{})
	if err != nil {
		return err
	}
	mark := func(_ any) { d.markDirty() }
	updateMark := func(_, _ any) { d.markDirty() }
	for _, inf := range []cache.Informer{hrInf, gwInf} {
		if _, err := inf.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
			AddFunc:    mark,
			UpdateFunc: updateMark,
			DeleteFunc: mark,
		}); err != nil {
			return err
		}
	}

	go func() { _ = d.cache.Start(ctx) }()
	if !d.cache.WaitForCacheSync(ctx) {
		return ctx.Err()
	}

	// Force an initial emit.
	d.markDirty()

	debounced := time.NewTimer(time.Hour)
	debounced.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-d.dirty:
			if !debounced.Stop() {
				select {
				case <-debounced.C:
				default:
				}
			}
			debounced.Reset(d.opts.Debounce)
		case <-debounced.C:
			tiles, err := d.compute(ctx)
			if err == nil {
				select {
				case d.out <- tiles:
				default:
					// drop if downstream is slow; latest-wins is fine.
				}
			}
		}
	}
}

func (d *Discoverer) markDirty() {
	select {
	case d.dirty <- struct{}{}:
	default:
	}
}

func (d *Discoverer) compute(ctx context.Context) ([]tile.Tile, error) {
	var hrs gwv1.HTTPRouteList
	if err := d.cache.List(ctx, &hrs); err != nil {
		return nil, err
	}
	var gws gwv1.GatewayList
	if err := d.cache.List(ctx, &gws); err != nil {
		return nil, err
	}

	gwSet := make(map[tile.GatewayKey]bool, len(gws.Items))
	for i := range gws.Items {
		gw := &gws.Items[i]
		gwSet[tile.GatewayKey{Namespace: gw.Namespace, Name: gw.Name}] = true
	}
	routes := make([]*gwv1.HTTPRoute, len(hrs.Items))
	for i := range hrs.Items {
		routes[i] = &hrs.Items[i]
	}

	backends := d.fetchBackends(ctx)
	return tile.Derive(routes, gwSet, backends), nil
}

// fetchBackends lists all Envoy Gateway Backend resources across namespaces
// and builds a BackendMap keyed by "namespace/name". Errors are logged and
// a best-effort (possibly empty) map is returned so tile derivation can
// still proceed.
func (d *Discoverer) fetchBackends(ctx context.Context) tile.BackendMap {
	var list unstructured.UnstructuredList
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   backendGVR.Group,
		Version: backendGVR.Version,
		Kind:    "BackendList",
	})
	if err := d.client.List(ctx, &list); err != nil {
		// Backend CRD may not be installed; treat as empty rather than fatal.
		return nil
	}

	result := make(tile.BackendMap, len(list.Items))
	for _, item := range list.Items {
		ep, ok := extractFirstIPEndpoint(item)
		if !ok {
			continue
		}
		key := item.GetNamespace() + "/" + item.GetName()
		result[key] = ep
	}
	return result
}

// extractFirstIPEndpoint pulls the first spec.endpoints[].ip endpoint out of
// an unstructured Backend object.
func extractFirstIPEndpoint(obj unstructured.Unstructured) (tile.BackendEndpoint, bool) {
	endpoints, ok, _ := unstructured.NestedSlice(obj.Object, "spec", "endpoints")
	if !ok || len(endpoints) == 0 {
		return tile.BackendEndpoint{}, false
	}
	ep, ok := endpoints[0].(map[string]any)
	if !ok {
		return tile.BackendEndpoint{}, false
	}
	ip, ok := ep["ip"].(map[string]any)
	if !ok {
		return tile.BackendEndpoint{}, false
	}
	address, _, _ := unstructured.NestedString(ip, "address")
	port, _, _ := unstructured.NestedInt64(ip, "port")
	if address == "" || port == 0 {
		return tile.BackendEndpoint{}, false
	}
	return tile.BackendEndpoint{Address: address, Port: int32(port)}, true
}

// (compile-time guard: client.Object referenced)
var _ client.Object = (*gwv1.HTTPRoute)(nil)
