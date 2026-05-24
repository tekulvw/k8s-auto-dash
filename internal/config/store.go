// Package config is the single point of access to the DashboardConfig
// custom resource. All reads and mutations of dashboard state flow
// through Store.
package config

import (
	"context"
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/tekulvw/k8s-auto-dash/api/v1alpha1"
)

// SingletonName is the only DashboardConfig the controller reads.
const SingletonName = "default"

type Store struct {
	c client.Client
}

func NewStore(c client.Client) *Store { return &Store{c: c} }

// Bootstrap ensures a DashboardConfig named "default" exists.
func (s *Store) Bootstrap(ctx context.Context) error {
	cfg := &v1.DashboardConfig{}
	err := s.c.Get(ctx, types.NamespacedName{Name: SingletonName}, cfg)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	cfg = &v1.DashboardConfig{}
	cfg.Name = SingletonName
	if err := s.c.Create(ctx, cfg); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// Get returns the current DashboardConfig.
func (s *Store) Get(ctx context.Context) (*v1.DashboardConfig, error) {
	cfg := &v1.DashboardConfig{}
	if err := s.c.Get(ctx, types.NamespacedName{Name: SingletonName}, cfg); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, errors.New("DashboardConfig/default not found; call Bootstrap first")
		}
		return nil, err
	}
	return cfg, nil
}

// Mutate applies mutateFn to a fresh copy of spec and writes it back.
// Retries up to maxRetries times on Conflict errors (optimistic
// concurrency).
func (s *Store) Mutate(ctx context.Context, mutateFn func(*v1.DashboardConfigSpec) error) error {
	const maxRetries = 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		cfg, err := s.Get(ctx)
		if err != nil {
			return err
		}
		if err := mutateFn(&cfg.Spec); err != nil {
			return err
		}
		if err := s.c.Update(ctx, cfg); err != nil {
			if apierrors.IsConflict(err) {
				continue
			}
			return err
		}
		return nil
	}
	return errors.New("config update conflict after retries")
}

// Watch returns a channel that receives the latest DashboardConfig
// whenever it changes (including external kubectl edits). The channel
// is closed when ctx is cancelled. Delivery is best-effort: if a
// consumer is slow, intermediate updates are dropped.
func (s *Store) Watch(ctx context.Context, cfg *rest.Config) (<-chan *v1.DashboardConfig, error) {
	out := make(chan *v1.DashboardConfig, 8)

	c, err := cacheNew(ctx, cfg)
	if err != nil {
		return nil, err
	}
	inf, err := c.GetInformer(ctx, &v1.DashboardConfig{})
	if err != nil {
		return nil, err
	}

	emit := func(obj any) {
		dc, ok := obj.(*v1.DashboardConfig)
		if !ok || dc.Name != SingletonName {
			return
		}
		select {
		case out <- dc.DeepCopy():
		default:
		}
	}
	if _, err := inf.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, n any) { emit(n) },
	}); err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		close(out)
	}()
	return out, nil
}
