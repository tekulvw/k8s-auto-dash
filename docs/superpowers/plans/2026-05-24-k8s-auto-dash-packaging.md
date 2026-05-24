# k8s-auto-dash Packaging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a multi-arch container image, a Helm chart, and concatenated raw manifests so the project can be deployed via `helm install` or `kubectl apply -f`. Add a GitHub Actions release workflow.

**Architecture:** Multi-stage Dockerfile builds the SvelteKit UI, fetches the bundled icon snapshot, then builds a static Go binary into a `distroless/static` final image. Helm chart (in `deploy/helm/k8s-auto-dash/`) renders CRD + RBAC + Deployment + Service + optional HTTPRoute + optional inline DashboardConfig. `deploy/manifests/install.yaml` is produced by `helm template` for the kubectl-only crowd.

**Tech Stack:**
- Docker BuildKit with multi-platform support (amd64, arm64)
- Helm 3.x
- GitHub Actions
- GHCR for image hosting

**Prerequisites:** Backend plan complete through Task 21. Frontend plan complete through Task 20. Repo builds end-to-end via `make build-all`.

---

## File Structure

```
.
├── Dockerfile
├── .dockerignore
├── deploy/
│   ├── helm/
│   │   └── k8s-auto-dash/
│   │       ├── Chart.yaml
│   │       ├── values.yaml
│   │       ├── values.schema.json
│   │       ├── templates/
│   │       │   ├── _helpers.tpl
│   │       │   ├── serviceaccount.yaml
│   │       │   ├── clusterrole.yaml
│   │       │   ├── clusterrolebinding.yaml
│   │       │   ├── deployment.yaml
│   │       │   ├── service.yaml
│   │       │   ├── httproute.yaml          # optional
│   │       │   ├── dashboardconfig.yaml    # optional inline CR
│   │       │   └── NOTES.txt
│   │       └── crds/
│   │           └── dashboardconfig.yaml     # copied from deploy/crd/
│   └── manifests/
│       └── install.yaml                     # generated, committed
├── .github/
│   └── workflows/
│       ├── ci.yaml
│       └── release.yaml
└── CHANGELOG.md
```

---

## Phase A: Container image

### Task 1: Dockerfile and .dockerignore

**Files:**
- Create: `Dockerfile`, `.dockerignore`

- [ ] **Step 1: Write `.dockerignore`**

```
.git
.github
docs
**/node_modules
**/.svelte-kit
ui/build
ui/icons
internal/assets/ui/*
internal/assets/icons/*
!internal/assets/ui/.gitkeep
!internal/assets/icons/.gitkeep
bin
*.test
coverage.out
.superpowers
```

- [ ] **Step 2: Write `Dockerfile`**

```dockerfile
# syntax=docker/dockerfile:1.7

# ---- Stage 1: UI build (Node) ---------------------------------------
FROM node:20-alpine AS ui
WORKDIR /ui

# Install dependencies first for better cache hits.
COPY ui/package.json ui/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile

COPY ui/ ./
ARG ICONS_COMMIT
RUN test -n "$ICONS_COMMIT" || (echo "ICONS_COMMIT build-arg required" && exit 1)
RUN ICONS_COMMIT=$ICONS_COMMIT pnpm icons \
 && pnpm run build

# ---- Stage 2: Go build ----------------------------------------------
FROM golang:1.22-alpine AS go
WORKDIR /src

# Dependencies (cache layer).
COPY go.mod go.sum ./
RUN go mod download

# Source.
COPY . .

# Pull UI artifacts into the embed source dirs.
RUN rm -rf internal/assets/ui/* internal/assets/icons/* \
 && mkdir -p internal/assets/ui internal/assets/icons
COPY --from=ui /ui/build/.  internal/assets/ui/
COPY --from=ui /ui/icons/.  internal/assets/icons/

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/k8s-auto-dash \
      ./cmd/k8s-auto-dash

# ---- Stage 3: Final -------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=go /out/k8s-auto-dash /k8s-auto-dash
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/k8s-auto-dash"]
```

- [ ] **Step 3: Build locally (single-arch) to verify**

```bash
docker build \
  --build-arg ICONS_COMMIT=$(curl -fsS https://api.github.com/repos/selfhst/dashboard-icons/commits/main | jq -r .sha) \
  --build-arg VERSION=dev \
  -t k8s-auto-dash:dev .
```
Expected: image builds successfully.

- [ ] **Step 4: Smoke run**

```bash
docker run --rm -p 8080:8080 k8s-auto-dash:dev --addr :8080 &
sleep 1
curl -fsS http://localhost:8080/healthz && echo OK
docker kill $(docker ps -q --filter ancestor=k8s-auto-dash:dev)
```

Expected: `OK` (note: the container will fail to reach a kubeconfig and exit; `/healthz` should respond before exit). If the binary exits immediately due to no kubeconfig, that's expected here — for a real smoke test that exercises k8s connectivity, use the live cluster test in Task 6.

- [ ] **Step 5: Commit**

```bash
git add Dockerfile .dockerignore
git commit -m "feat: multi-stage Dockerfile (Node UI + Go binary + distroless)"
```

---

### Task 2: Multi-arch image build via Buildx

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Add Makefile targets**

Append to `Makefile`:

```make
IMAGE         ?= ghcr.io/anomalyco/k8s-auto-dash
TAG           ?= dev
PLATFORMS     ?= linux/amd64,linux/arm64
ICONS_COMMIT  ?= main

.PHONY: image
image:
	docker buildx build \
	  --platform $(PLATFORMS) \
	  --build-arg ICONS_COMMIT=$(ICONS_COMMIT) \
	  --build-arg VERSION=$(TAG) \
	  -t $(IMAGE):$(TAG) \
	  .

.PHONY: image-push
image-push:
	docker buildx build \
	  --platform $(PLATFORMS) \
	  --build-arg ICONS_COMMIT=$(ICONS_COMMIT) \
	  --build-arg VERSION=$(TAG) \
	  -t $(IMAGE):$(TAG) \
	  --push \
	  .
```

- [ ] **Step 2: Verify the multi-arch build works**

```bash
make image TAG=dev ICONS_COMMIT=$(curl -fsS https://api.github.com/repos/selfhst/dashboard-icons/commits/main | jq -r .sha)
```

Expected: build completes for both `linux/amd64` and `linux/arm64`. (Requires `docker buildx create --use` once if not already configured.)

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "build: multi-arch image targets via buildx"
```

---

## Phase B: Helm chart

### Task 3: Chart scaffold

**Files:**
- Create: `deploy/helm/k8s-auto-dash/Chart.yaml`
- Create: `deploy/helm/k8s-auto-dash/values.yaml`
- Create: `deploy/helm/k8s-auto-dash/templates/_helpers.tpl`
- Create: `deploy/helm/k8s-auto-dash/templates/NOTES.txt`

- [ ] **Step 1: Write `Chart.yaml`**

```yaml
apiVersion: v2
name: k8s-auto-dash
description: Auto-discovering dashboard for Kubernetes Gateway API HTTPRoutes
type: application
version: 0.1.0
appVersion: "0.1.0"
home: https://github.com/anomalyco/k8s-auto-dash
sources:
  - https://github.com/anomalyco/k8s-auto-dash
keywords:
  - dashboard
  - gateway-api
  - homelab
kubeVersion: ">=1.27.0-0"
```

- [ ] **Step 2: Write `values.yaml`**

```yaml
# Default values for k8s-auto-dash.

image:
  repository: ghcr.io/anomalyco/k8s-auto-dash
  tag: ""                       # defaults to .Chart.AppVersion
  pullPolicy: IfNotPresent

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""

# Single replica is supported; multiple replicas are not.
replicaCount: 1

serviceAccount:
  create: true
  name: ""
  annotations: {}

rbac:
  create: true

crd:
  # Install the DashboardConfig CRD as part of this release.
  # Disable when CRDs are managed out-of-band (Argo CD apps-of-apps,
  # kustomize, separate Helm chart, etc.).
  install: true

service:
  type: ClusterIP
  port: 80          # external port; container always listens on 8080

resources:
  requests:
    cpu: 50m
    memory: 64Mi
  limits:
    cpu: 500m
    memory: 256Mi

podAnnotations: {}
podLabels: {}
podSecurityContext:
  runAsNonRoot: true
  seccompProfile:
    type: RuntimeDefault
securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  readOnlyRootFilesystem: true

nodeSelector: {}
tolerations: []
affinity: {}

# Optional HTTPRoute publishing the dashboard onto your own gateway.
httpRoute:
  enabled: false
  hostname: ""
  parentRef:
    name: ""
    namespace: ""
    sectionName: ""

# Optional inline DashboardConfig. When set, the chart renders a
# DashboardConfig/default resource so curation lives in your values.yaml.
# Leave commented to let the controller bootstrap an empty one.
config: {}
#   settings:
#     title: "Homelab"
#     theme: dark
#     healthCheck:
#       intervalSeconds: 60
```

- [ ] **Step 3: Write `templates/_helpers.tpl`**

```yaml
{{/* Common name and label helpers. */}}

{{- define "k8s-auto-dash.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "k8s-auto-dash.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "k8s-auto-dash.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "k8s-auto-dash.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "k8s-auto-dash.labels" -}}
app.kubernetes.io/name: {{ include "k8s-auto-dash.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{- define "k8s-auto-dash.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k8s-auto-dash.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "k8s-auto-dash.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}
```

- [ ] **Step 4: Write `templates/NOTES.txt`**

```
{{ include "k8s-auto-dash.fullname" . }} has been installed.

The dashboard is available inside the cluster at:
  http://{{ include "k8s-auto-dash.fullname" . }}.{{ .Release.Namespace }}.svc:{{ .Values.service.port }}

{{- if .Values.httpRoute.enabled }}
External URL (via your Gateway): https://{{ .Values.httpRoute.hostname }}
{{- else }}
Expose it by creating an HTTPRoute pointing at the Service above, or
set `httpRoute.enabled: true` in values.yaml.
{{- end }}

The controller will auto-create a DashboardConfig named "default" if
one does not exist:
  kubectl get dashboardconfig default -o yaml
```

- [ ] **Step 5: Lint**

```bash
helm lint deploy/helm/k8s-auto-dash
```
Expected: `0 chart(s) failed`.

- [ ] **Step 6: Commit**

```bash
git add deploy/helm/k8s-auto-dash/
git commit -m "feat(helm): chart scaffold with helpers and values"
```

---

### Task 4: RBAC, ServiceAccount, Deployment, Service templates

**Files:**
- Create: `deploy/helm/k8s-auto-dash/templates/serviceaccount.yaml`
- Create: `deploy/helm/k8s-auto-dash/templates/clusterrole.yaml`
- Create: `deploy/helm/k8s-auto-dash/templates/clusterrolebinding.yaml`
- Create: `deploy/helm/k8s-auto-dash/templates/deployment.yaml`
- Create: `deploy/helm/k8s-auto-dash/templates/service.yaml`

- [ ] **Step 1: Write `templates/serviceaccount.yaml`**

```yaml
{{- if .Values.serviceAccount.create }}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "k8s-auto-dash.serviceAccountName" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "k8s-auto-dash.labels" . | nindent 4 }}
  {{- with .Values.serviceAccount.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- end }}
```

- [ ] **Step 2: Write `templates/clusterrole.yaml`**

```yaml
{{- if .Values.rbac.create }}
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ include "k8s-auto-dash.fullname" . }}
  labels:
    {{- include "k8s-auto-dash.labels" . | nindent 4 }}
rules:
  - apiGroups: ["gateway.networking.k8s.io"]
    resources: ["gateways", "httproutes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["k8s-auto-dash.io"]
    resources: ["dashboardconfigs"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: ["k8s-auto-dash.io"]
    resources: ["dashboardconfigs/status"]
    verbs: ["get", "update", "patch"]
{{- end }}
```

- [ ] **Step 3: Write `templates/clusterrolebinding.yaml`**

```yaml
{{- if .Values.rbac.create }}
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ include "k8s-auto-dash.fullname" . }}
  labels:
    {{- include "k8s-auto-dash.labels" . | nindent 4 }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: {{ include "k8s-auto-dash.fullname" . }}
subjects:
  - kind: ServiceAccount
    name: {{ include "k8s-auto-dash.serviceAccountName" . }}
    namespace: {{ .Release.Namespace }}
{{- end }}
```

- [ ] **Step 4: Write `templates/deployment.yaml`**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "k8s-auto-dash.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "k8s-auto-dash.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  strategy:
    type: Recreate                # single-replica; no rolling updates
  selector:
    matchLabels:
      {{- include "k8s-auto-dash.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "k8s-auto-dash.selectorLabels" . | nindent 8 }}
        {{- with .Values.podLabels }}{{ toYaml . | nindent 8 }}{{- end }}
      {{- with .Values.podAnnotations }}
      annotations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
    spec:
      serviceAccountName: {{ include "k8s-auto-dash.serviceAccountName" . }}
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
        - name: k8s-auto-dash
          image: "{{ include "k8s-auto-dash.image" . }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          args: ["--addr=:8080"]
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
          readinessProbe:
            httpGet: { path: /readyz, port: http }
            periodSeconds: 5
          livenessProbe:
            httpGet: { path: /healthz, port: http }
            periodSeconds: 30
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
```

- [ ] **Step 5: Write `templates/service.yaml`**

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "k8s-auto-dash.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "k8s-auto-dash.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - name: http
      port: {{ .Values.service.port }}
      targetPort: http
      protocol: TCP
  selector:
    {{- include "k8s-auto-dash.selectorLabels" . | nindent 4 }}
```

- [ ] **Step 6: Lint and template**

```bash
helm lint deploy/helm/k8s-auto-dash
helm template test deploy/helm/k8s-auto-dash | head -80
```

Expected: lint passes; template output shows ClusterRole, ClusterRoleBinding, ServiceAccount, Deployment, Service.

- [ ] **Step 7: Commit**

```bash
git add deploy/helm/k8s-auto-dash/templates/
git commit -m "feat(helm): RBAC, ServiceAccount, Deployment, Service templates"
```

---

### Task 5: Optional HTTPRoute, inline DashboardConfig, and CRD

**Files:**
- Create: `deploy/helm/k8s-auto-dash/templates/httproute.yaml`
- Create: `deploy/helm/k8s-auto-dash/templates/dashboardconfig.yaml`
- Create: `deploy/helm/k8s-auto-dash/crds/dashboardconfig.yaml`
- Modify: `Makefile`

- [ ] **Step 1: Write `templates/httproute.yaml`**

```yaml
{{- if .Values.httpRoute.enabled }}
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: {{ include "k8s-auto-dash.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "k8s-auto-dash.labels" . | nindent 4 }}
spec:
  hostnames:
    - {{ required ".Values.httpRoute.hostname is required" .Values.httpRoute.hostname | quote }}
  parentRefs:
    - name: {{ required ".Values.httpRoute.parentRef.name is required" .Values.httpRoute.parentRef.name | quote }}
      {{- with .Values.httpRoute.parentRef.namespace }}
      namespace: {{ . | quote }}
      {{- end }}
      {{- with .Values.httpRoute.parentRef.sectionName }}
      sectionName: {{ . | quote }}
      {{- end }}
  rules:
    - backendRefs:
        - name: {{ include "k8s-auto-dash.fullname" . }}
          port: {{ .Values.service.port }}
{{- end }}
```

- [ ] **Step 2: Write `templates/dashboardconfig.yaml`**

```yaml
{{- if .Values.config }}
{{- if not (empty .Values.config) }}
apiVersion: k8s-auto-dash.io/v1alpha1
kind: DashboardConfig
metadata:
  name: default
  labels:
    {{- include "k8s-auto-dash.labels" . | nindent 4 }}
spec:
  {{- toYaml .Values.config | nindent 2 }}
{{- end }}
{{- end }}
```

- [ ] **Step 3: Copy the CRD into the chart's crds/ directory**

Helm installs anything in `chart/crds/` once on install and does not template it. The CRD source of truth lives in `deploy/crd/`; the chart's copy is kept in sync by a Makefile target.

```bash
mkdir -p deploy/helm/k8s-auto-dash/crds
cp deploy/crd/k8s-auto-dash.io_dashboardconfigs.yaml \
   deploy/helm/k8s-auto-dash/crds/dashboardconfig.yaml
```

- [ ] **Step 4: Add Makefile target for chart sync**

Append to `Makefile`:

```make
.PHONY: chart-sync
chart-sync: generate
	cp deploy/crd/k8s-auto-dash.io_dashboardconfigs.yaml \
	   deploy/helm/k8s-auto-dash/crds/dashboardconfig.yaml

.PHONY: chart-lint
chart-lint: chart-sync
	helm lint deploy/helm/k8s-auto-dash

.PHONY: chart-template
chart-template: chart-sync
	helm template test deploy/helm/k8s-auto-dash
```

Also wire the `crd.install: false` toggle in `values.yaml`: Helm doesn't allow conditional `crds/`, so the recommended workaround is documented but not enforced — when `crd.install=false`, users should remove the `crds/` directory from their chart copy or use a `--skip-crds` flag.

Update `templates/NOTES.txt` to mention `--skip-crds` if appropriate:

Append:

```
{{- if not .Values.crd.install }}
NOTE: To skip CRD installation, pass --skip-crds to `helm install`.
{{- end }}
```

- [ ] **Step 5: Lint with `config` set**

```bash
helm template test deploy/helm/k8s-auto-dash \
  --set 'config.settings.title=Test' \
  --set 'httpRoute.enabled=true' \
  --set 'httpRoute.hostname=dash.example.com' \
  --set 'httpRoute.parentRef.name=ext' \
  --set 'httpRoute.parentRef.namespace=gateway' \
  | grep -E '^kind:' | sort -u
```

Expected output includes: `ClusterRole`, `ClusterRoleBinding`, `DashboardConfig`, `Deployment`, `HTTPRoute`, `Service`, `ServiceAccount`.

- [ ] **Step 6: Commit**

```bash
git add deploy/helm/k8s-auto-dash/templates/ deploy/helm/k8s-auto-dash/crds/ Makefile
git commit -m "feat(helm): optional HTTPRoute, inline DashboardConfig, CRD bundling"
```

---

## Phase C: Raw manifests + live verification

### Task 6: Generate concatenated install.yaml

**Files:**
- Create: `deploy/manifests/install.yaml` (generated)
- Modify: `Makefile`

- [ ] **Step 1: Add Makefile target**

```make
.PHONY: manifests
manifests: chart-sync
	@mkdir -p deploy/manifests
	@( \
	  echo "# Generated by 'make manifests'. Do not edit by hand."; \
	  echo "# Source: deploy/helm/k8s-auto-dash"; \
	  echo "---"; \
	  cat deploy/crd/k8s-auto-dash.io_dashboardconfigs.yaml; \
	  echo "---"; \
	  helm template k8s-auto-dash deploy/helm/k8s-auto-dash \
	    --namespace k8s-auto-dash \
	    --set crd.install=false; \
	) > deploy/manifests/install.yaml
```

- [ ] **Step 2: Generate**

```bash
make manifests
```

- [ ] **Step 3: Inspect**

```bash
grep '^kind:' deploy/manifests/install.yaml | sort | uniq -c
```

Expected: `CustomResourceDefinition`, `ServiceAccount`, `ClusterRole`, `ClusterRoleBinding`, `Deployment`, `Service` each appearing once.

- [ ] **Step 4: Commit**

```bash
git add Makefile deploy/manifests/
git commit -m "feat(manifests): generate concatenated install.yaml"
```

---

### Task 7: Live install smoke against a kind cluster

**Files:**
- Create: `hack/kind-cluster.sh`, `hack/kind-config.yaml`

This task wires up a reproducible local install test you can run end-to-end. Skip in CI if kind isn't available.

- [ ] **Step 1: Write `hack/kind-config.yaml`**

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
```

- [ ] **Step 2: Write `hack/kind-cluster.sh`**

```bash
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
  --build-arg ICONS_COMMIT="$(curl -fsS https://api.github.com/repos/selfhst/dashboard-icons/commits/main | jq -r .sha)" \
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
```

- [ ] **Step 3: Make executable**

```bash
chmod +x hack/kind-cluster.sh
```

- [ ] **Step 4: Run it (requires kind + docker + helm + jq + kubectl)**

```bash
./hack/kind-cluster.sh
```

Expected: the controller comes up, `/healthz` returns 200, `/api/tiles` returns `{"groups":[],"tiles":[]}` because no HTTPRoutes are deployed yet.

- [ ] **Step 5: Add one HTTPRoute and verify discovery**

```bash
kubectl create namespace demo
kubectl apply -n demo -f - <<'EOF'
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata: { name: ext }
spec:
  gatewayClassName: dummy
  listeners:
    - name: http
      port: 80
      protocol: HTTP
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata: { name: hello }
spec:
  hostnames: ["hello.example.com"]
  parentRefs:
    - name: ext
EOF
sleep 2
curl -fsS http://localhost:18080/api/tiles | jq '.tiles | length'
```

Expected: `1`.

- [ ] **Step 6: Commit**

```bash
git add hack/
git commit -m "test: kind cluster smoke script"
```

---

## Phase D: CI/CD

### Task 8: GitHub Actions CI workflow

**Files:**
- Create: `.github/workflows/ci.yaml`

- [ ] **Step 1: Write `.github/workflows/ci.yaml`**

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true
      - name: Vet
        run: go vet ./...
      - name: Test
        run: make test

  ui:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: ui
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v3
        with: { version: 9 }
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'pnpm'
          cache-dependency-path: ui/pnpm-lock.yaml
      - run: pnpm install --frozen-lockfile
      - name: Unit tests
        run: pnpm vitest run
      - name: Resolve icons commit
        id: icons
        run: echo "sha=$(curl -fsS https://api.github.com/repos/selfhst/dashboard-icons/commits/main | jq -r .sha)" >> $GITHUB_OUTPUT
      - name: Fetch icons
        env:
          ICONS_COMMIT: ${{ steps.icons.outputs.sha }}
        run: pnpm icons
      - name: Build
        run: pnpm run build
      - name: Playwright
        run: |
          pnpm exec playwright install --with-deps chromium
          pnpm exec playwright test

  helm:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: azure/setup-helm@v4
        with: { version: 'v3.14.0' }
      - run: helm lint deploy/helm/k8s-auto-dash
      - run: helm template test deploy/helm/k8s-auto-dash > /dev/null

  image:
    runs-on: ubuntu-latest
    needs: [go, ui, helm]
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - name: Resolve icons commit
        id: icons
        run: echo "sha=$(curl -fsS https://api.github.com/repos/selfhst/dashboard-icons/commits/main | jq -r .sha)" >> $GITHUB_OUTPUT
      - name: Build image (no push)
        uses: docker/build-push-action@v6
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: false
          build-args: |
            ICONS_COMMIT=${{ steps.icons.outputs.sha }}
            VERSION=ci
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/ci.yaml
git commit -m "ci: lint, test, and build on every push and PR"
```

---

### Task 9: GitHub Actions release workflow

**Files:**
- Create: `.github/workflows/release.yaml`
- Create: `CHANGELOG.md`

- [ ] **Step 1: Write `.github/workflows/release.yaml`**

```yaml
name: Release

on:
  push:
    tags: ['v*.*.*']

permissions:
  contents: write
  packages: write

jobs:
  image:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-qemu-action@v3
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Resolve icons commit
        id: icons
        run: echo "sha=$(curl -fsS https://api.github.com/repos/selfhst/dashboard-icons/commits/main | jq -r .sha)" >> $GITHUB_OUTPUT
      - name: Build & push image
        uses: docker/build-push-action@v6
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ghcr.io/${{ github.repository }}:${{ github.ref_name }}
            ghcr.io/${{ github.repository }}:latest
          build-args: |
            ICONS_COMMIT=${{ steps.icons.outputs.sha }}
            VERSION=${{ github.ref_name }}

  manifests:
    runs-on: ubuntu-latest
    needs: [image]
    steps:
      - uses: actions/checkout@v4
      - uses: azure/setup-helm@v4
        with: { version: 'v3.14.0' }
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - name: Generate manifests
        run: make manifests
      - name: Package chart
        run: helm package deploy/helm/k8s-auto-dash --destination dist/
      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: |
            deploy/manifests/install.yaml
            dist/*.tgz
          generate_release_notes: true
```

- [ ] **Step 2: Write a minimal `CHANGELOG.md`**

```markdown
# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [0.1.0] - first release
- Initial release: Gateway API HTTPRoute auto-discovery, dashboard SPA,
  CRD-backed curation, Helm chart, multi-arch image.
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yaml CHANGELOG.md
git commit -m "ci: release workflow publishing image and chart artifacts"
```

---

## Phase E: README and final assembly

### Task 10: README and getting-started docs

**Files:**
- Create: `README.md`

- [ ] **Step 1: Write `README.md`**

```markdown
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
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: README with quick-start"
```

---

## Done criteria

- `docker build` (or `make image`) produces a multi-arch image that
  starts and serves `/healthz`.
- `helm lint deploy/helm/k8s-auto-dash` is clean.
- `helm template ...` renders all expected kinds (CRD, ServiceAccount,
  ClusterRole, ClusterRoleBinding, Deployment, Service, optionally
  HTTPRoute and DashboardConfig).
- `deploy/manifests/install.yaml` exists and is up-to-date.
- `./hack/kind-cluster.sh` succeeds end-to-end on a fresh kind cluster
  with the Gateway API CRDs.
- CI workflow runs Go tests, UI tests, Playwright, helm lint, and
  multi-arch image build on every PR.
- Tagging `vX.Y.Z` publishes the image to GHCR and attaches the chart
  tarball + `install.yaml` to a GitHub Release.

## Coverage check (self-review against the design spec)

- ✅ §"Container" multi-stage, distroless, multi-arch — Tasks 1, 2.
- ✅ §"Helm chart" values, RBAC, optional HTTPRoute, inline config — Tasks 3, 4, 5.
- ✅ §"Raw manifests" `install.yaml` generated from chart — Task 6.
- ✅ §"RBAC" ClusterRole + ClusterRoleBinding (CR is cluster-scoped) — Task 4.
- ✅ §"Config bootstrapping" — covered by the backend; the chart only
  optionally pre-seeds the CR via `.Values.config`.
- ⏭ §"Observability" already handled in the backend plan.
- 🔶 The chart's `crd.install: false` toggle is documented but not
  enforced via templating (Helm's `crds/` directory is unconditional);
  users disable installation with `--skip-crds`. Acceptable for v1.

## Notes for the implementing engineer

- **Task ordering:** 1 → 2 → (3, 4 in series) → 5 → 6 → 7. CI (8) and
  release (9) can come after the chart works end-to-end. README (10)
  is last.
- **Single replica:** the Deployment uses `strategy: Recreate`. Do not
  switch to `RollingUpdate` without first adding leader election to
  the backend — the controller assumes exclusive write access to its
  CR.
- **Repository owner placeholder:** `ghcr.io/anomalyco/...` appears in
  several files. If publishing under a different org, do a single
  search-and-replace before tagging.
- **`ICONS_COMMIT`:** every build path (Makefile, Dockerfile, CI,
  release) resolves this at build time. To pin reproducibly, set it
  to a literal SHA in `Makefile` instead of `main`.
