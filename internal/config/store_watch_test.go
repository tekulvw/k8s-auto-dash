package config

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "github.com/anomalyco/k8s-auto-dash/api/v1alpha1"
	"github.com/anomalyco/k8s-auto-dash/internal/testenv"
)

func TestStore_WatchEmitsOnChange(t *testing.T) {
	te := testenv.Start(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := NewStore(te.Client)
	require.NoError(t, s.Bootstrap(ctx))

	ch, err := s.Watch(ctx, te.Cfg)
	require.NoError(t, err)

	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = s.Mutate(ctx, func(spec *v1.DashboardConfigSpec) error {
			spec.Settings.Title = "Changed"
			return nil
		})
	}()

	select {
	case cfg := <-ch:
		assert.Equal(t, "Changed", cfg.Spec.Settings.Title)
	case <-time.After(5 * time.Second):
		t.Fatal("no watch event")
	}
}
