# k8s-auto-dash

Auto-discovering homelab dashboard for Kubernetes Gateway API.

- Watches `Gateway` and `HTTPRoute` resources cluster-wide.
- One tile per discovered hostname; grouped by namespace by default.
- HTTP health checks with up / degraded / down state.
- Curate layout, names, icons, groups in the browser; everything
  persists to a single `DashboardConfig` CRD (GitOps-friendly).
- Single Go binary, single container, ~30 MB. Multi-arch (amd64, arm64).

## Quick start

### Helm

```bash
helm install k8s-auto-dash oci://ghcr.io/anomalyco/charts/k8s-auto-dash \
  --namespace k8s-auto-dash --create-namespace
```

Then publish the dashboard onto your own gateway:

```bash
helm upgrade k8s-auto-dash oci://ghcr.io/anomalyco/charts/k8s-auto-dash \
  --reuse-values \
  --set httpRoute.enabled=true \
  --set httpRoute.hostname=dash.example.com \
  --set httpRoute.parentRef.name=external-gateway \
  --set httpRoute.parentRef.namespace=gateway
```

### kubectl

```bash
kubectl apply -f https://github.com/anomalyco/k8s-auto-dash/releases/latest/download/install.yaml
```

This creates a Namespace `k8s-auto-dash`, the CRD, RBAC, Deployment,
and Service. Expose it with your own HTTPRoute.

## Configuration

All UI-managed state lives in the cluster-scoped
`DashboardConfig/default`:

```bash
kubectl get dashboardconfig default -o yaml
```

You can also pre-seed configuration via Helm values — see
`deploy/helm/k8s-auto-dash/values.yaml`.

## Development

See `docs/superpowers/` for the design spec and implementation plans.

```bash
make test            # Go tests (envtest)
cd ui && pnpm test   # UI unit tests
make build-all       # static binary with embedded UI + icons
make image           # multi-arch container build
./hack/kind-cluster.sh   # end-to-end smoke in kind
```

## License

MIT
