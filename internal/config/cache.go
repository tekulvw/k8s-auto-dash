package config

import (
	"context"

	"k8s.io/client-go/rest"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
)

// cacheNew constructs and starts a controller-runtime cache scoped to
// our CRD scheme. It blocks until the cache has synced or ctx ends.
func cacheNew(ctx context.Context, cfg *rest.Config) (ctrlcache.Cache, error) {
	c, err := ctrlcache.New(cfg, ctrlcache.Options{Scheme: newScheme()})
	if err != nil {
		return nil, err
	}
	go func() { _ = c.Start(ctx) }()
	c.WaitForCacheSync(ctx)
	return c, nil
}
