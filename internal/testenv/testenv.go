// Package testenv provides a shared envtest bootstrap.
package testenv

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1 "github.com/anomalyco/k8s-auto-dash/api/v1alpha1"
)

type Env struct {
	Env    *envtest.Environment
	Cfg    *rest.Config
	Client client.Client
}

// repoRoot returns the absolute path to the repository root, computed
// from this source file's location. Works regardless of test cwd.
func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func Start(t *testing.T) *Env {
	t.Helper()
	env := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join(repoRoot(), "deploy", "crd"),
			filepath.Join(repoRoot(), "internal", "testenv", "crds"),
		},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := env.Start()
	require.NoError(t, err)
	t.Cleanup(func() { _ = env.Stop() })

	scheme := k8sruntime.NewScheme()
	require.NoError(t, v1.AddToScheme(scheme))
	require.NoError(t, gwv1.Install(scheme))

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	require.NoError(t, err)
	return &Env{Env: env, Cfg: cfg, Client: c}
}
