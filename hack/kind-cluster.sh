#!/usr/bin/env bash
set -euo pipefail

CLUSTER=${CLUSTER:-k8s-auto-dash}
IMAGE=${IMAGE:-k8s-auto-dash:dev}

if ! kind get clusters | grep -qx "$CLUSTER"; then
  kind create cluster --name "$CLUSTER" --config hack/kind-config.yaml
fi

# Install Gateway API standard CRDs.
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/standard-install.yaml

# Build the dev image locally (single arch is fine for kind).
docker build \
  --build-arg ICONS_COMMIT="$(curl -fsS https://api.github.com/repos/homarr-labs/dashboard-icons/commits/main | jq -r .sha)" \
  --build-arg VERSION=dev \
  -t "$IMAGE" .
kind load docker-image "$IMAGE" --name "$CLUSTER"

helm upgrade --install k8s-auto-dash deploy/helm/k8s-auto-dash \
  --namespace k8s-auto-dash --create-namespace \
  --set image.repository=k8s-auto-dash \
  --set image.tag=dev \
  --set image.pullPolicy=Never

kubectl -n k8s-auto-dash rollout status deploy/k8s-auto-dash --timeout=2m

# Smoke: port-forward and check /healthz and /api/tiles.
kubectl -n k8s-auto-dash port-forward svc/k8s-auto-dash 18080:80 >/dev/null 2>&1 &
PFPID=$!
trap "kill $PFPID 2>/dev/null || true" EXIT
sleep 2
curl -fsS http://localhost:18080/healthz
echo ""
curl -fsS http://localhost:18080/api/tiles | jq .
