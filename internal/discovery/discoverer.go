// Package discovery watches Gateway/HTTPRoute resources and emits
// derived tile sets on Tiles().
package discovery

import (
	"context"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/tekulvw/k8s-auto-dash/internal/tile"
)

type Options struct {
	Debounce time.Duration // default 250ms
}

type Discoverer struct {
	cfg     *rest.Config
	opts    Options
	cache   cache.Cache
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
	return &Discoverer{
		cfg:    cfg,
		opts:   opts,
		cache:  c,
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
	return tile.Derive(routes, gwSet), nil
}

// (compile-time guard: client.Object referenced)
var _ client.Object = (*gwv1.HTTPRoute)(nil)
