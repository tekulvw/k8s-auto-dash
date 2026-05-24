package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/anomalyco/k8s-auto-dash/internal/testenv"
)

func TestStore_BootstrapCreatesDefault(t *testing.T) {
	te := testenv.Start(t)
	ctx := context.Background()

	s := NewStore(te.Client)
	require.NoError(t, s.Bootstrap(ctx))

	cfg, err := s.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "default", cfg.Name)
}

func TestStore_BootstrapIdempotent(t *testing.T) {
	te := testenv.Start(t)
	ctx := context.Background()
	s := NewStore(te.Client)

	require.NoError(t, s.Bootstrap(ctx))
	require.NoError(t, s.Bootstrap(ctx))

	cfg, err := s.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "default", cfg.Name)
}
