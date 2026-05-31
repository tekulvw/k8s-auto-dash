package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1 "github.com/tekulvw/k8s-auto-dash/api/v1alpha1"
	"github.com/tekulvw/k8s-auto-dash/internal/api"
	configstore "github.com/tekulvw/k8s-auto-dash/internal/config"
	"github.com/tekulvw/k8s-auto-dash/internal/discovery"
	"github.com/tekulvw/k8s-auto-dash/internal/health"
	"github.com/tekulvw/k8s-auto-dash/internal/metrics"
	"github.com/tekulvw/k8s-auto-dash/internal/tile"
)

var version = "dev"

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	ctrllog.SetLogger(logr.FromSlogHandler(logger.Handler()))
	slog.Info("starting", "version", version)

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.GetConfig()
	must(err)

	scheme := newScheme()
	cli, err := client.New(cfg, client.Options{Scheme: scheme})
	must(err)

	store := configstore.NewStore(cli)
	must(store.Bootstrap(ctx))

	state := api.NewState()
	bus := api.NewBus()
	mreg := metrics.New()

	// Load initial config state (Watch only emits on Update events, not initial sync).
	if initial, err := store.Get(ctx); err == nil {
		state.SetConfig(initial.Spec)
	}

	// Discoverer
	d, err := discovery.New(cfg, discovery.Options{})
	must(err)
	go func() { _ = d.Run(ctx) }()

	// Health pool
	prober := health.NewProber(health.ProbeOptions{
		Timeout:   5 * time.Second,
		UserAgent: "k8s-auto-dash/" + version + " (health-check)",
	})
	pool := health.NewPool(health.PoolOptions{
		Workers:  5,
		Interval: 60 * time.Second,
		Prober:   prober,
		OnResult: func(id string, s tile.Status) {
			state.SetStatus(id, s)
			mreg.ProbeTotal.WithLabelValues(string(s.State)).Inc()
			mreg.ProbeLatency.WithLabelValues(string(s.State)).
				Observe(float64(s.LatencyMs) / 1000.0)
			bus.Publish(api.Event{Type: "status-changed", Data: map[string]any{
				"id": id, "status": s,
			}})
		},
	})
	go pool.Run(ctx.Done())

	// Config watch (only emits on Update events; initial state was loaded above).
	cfgCh, err := store.Watch(ctx, cfg)
	must(err)

	// Reconcile loop: pump derived tiles, config updates, and refresh
	// pool targets in one place.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case derived := <-d.Tiles():
				state.SetDerived(derived)
				mreg.DiscoveredTiles.Set(float64(len(derived)))
				pool.Set(targetsFromState(state))
				bus.Publish(api.Event{Type: "tile-updated",
					Data: map[string]any{"reason": "discovery"}})
			case c := <-cfgCh:
				if c == nil {
					return
				}
				state.SetConfig(c.Spec)
				pool.Set(targetsFromState(state))
				bus.Publish(api.Event{Type: "config-changed",
					Data: map[string]any{"source": "kubectl"}})
			}
		}
	}()

	srv := api.NewServerFull(state, &mutatorAdapter{store: store, m: mreg}, bus, mreg)

	httpSrv := &http.Server{Addr: *addr, Handler: srv.Handler()}
	go func() {
		slog.Info("listening", "addr", *addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = httpSrv.Shutdown(shutCtx)
}

// mutatorAdapter wires config.Store into api.ConfigMutator and bumps
// the metrics counter on every successful write.
type mutatorAdapter struct {
	store *configstore.Store
	m     *metrics.Registry
}

func (a *mutatorAdapter) Mutate(ctx context.Context,
	fn func(*v1.DashboardConfigSpec) error) error {
	if err := a.store.Mutate(ctx, fn); err != nil {
		return err
	}
	a.m.ConfigWrites.Inc()
	return nil
}

func targetsFromState(s *api.State) []health.Target {
	view := s.View()
	out := make([]health.Target, 0, len(view.Tiles))
	for _, t := range view.Tiles {
		if t.URL == "" {
			continue
		}
		probeURL := t.URL
		if t.HealthCheckURL != "" {
			probeURL = t.HealthCheckURL
		}
		out = append(out, health.Target{
			ID: t.ID, URL: probeURL, InsecureSkipVerify: t.InsecureSkipVerify,
		})
	}
	return out
}

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	must(v1.AddToScheme(s))
	must(gwv1.Install(s))
	return s
}

func must(err error) {
	if err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}
