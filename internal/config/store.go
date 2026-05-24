// Package config is the single point of access to the DashboardConfig
// custom resource. All reads and mutations of dashboard state flow
// through Store.
package config

import (
	"context"
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/anomalyco/k8s-auto-dash/api/v1alpha1"
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
