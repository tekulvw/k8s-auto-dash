package config

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "github.com/tekulvw/k8s-auto-dash/api/v1alpha1"
	"github.com/tekulvw/k8s-auto-dash/internal/testenv"
)

func TestStore_MutateSpec(t *testing.T) {
	te := testenv.Start(t)
	ctx := context.Background()
	s := NewStore(te.Client)
	require.NoError(t, s.Bootstrap(ctx))

	err := s.Mutate(ctx, func(spec *v1.DashboardConfigSpec) error {
		spec.Settings.Title = "Home"
		spec.Tiles = append(spec.Tiles, v1.TileOverride{ID: "a/b/c", Name: "X"})
		return nil
	})
	require.NoError(t, err)

	cfg, err := s.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Home", cfg.Spec.Settings.Title)
	assert.Equal(t, "X", cfg.Spec.Tiles[0].Name)
}

func TestStore_MutateRetriesOnConflict(t *testing.T) {
	te := testenv.Start(t)
	ctx := context.Background()
	s := NewStore(te.Client)
	require.NoError(t, s.Bootstrap(ctx))

	// Two concurrent mutators; both should succeed thanks to retry.
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.Mutate(ctx, func(spec *v1.DashboardConfigSpec) error {
				spec.Groups = append(spec.Groups,
					v1.GroupSpec{ID: id(i), Name: id(i), Order: i})
				return nil
			})
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	cfg, err := s.Get(ctx)
	require.NoError(t, err)
	assert.Len(t, cfg.Spec.Groups, 2)
}

func id(i int) string {
	if i == 0 {
		return "a"
	}
	return "b"
}
