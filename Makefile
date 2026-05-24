GO ?= go
UI_DIR ?= ui
ICONS_COMMIT ?= main
CONTROLLER_GEN ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.0

.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile=hack/boilerplate.go.txt paths=./api/...
	$(CONTROLLER_GEN) crd paths=./api/... output:crd:dir=deploy/crd

.PHONY: ui-build
ui-build:
	cd $(UI_DIR) && pnpm install && ICONS_COMMIT=$(ICONS_COMMIT) pnpm icons && pnpm run build
	rm -rf internal/assets/ui/* internal/assets/icons/*
	cp -r $(UI_DIR)/build/. internal/assets/ui/
	cp -r $(UI_DIR)/icons/. internal/assets/icons/

.PHONY: build-all
build-all: ui-build build

ENVTEST_K8S_VERSION ?= 1.30.x
ENVTEST ?= go run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: envtest-assets
envtest-assets:
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path > /dev/null

.PHONY: test
test: envtest-assets
	KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	  $(GO) test ./... -race -count=1

.PHONY: build
build:
	CGO_ENABLED=0 $(GO) build -o bin/k8s-auto-dash ./cmd/k8s-auto-dash

.PHONY: tidy
tidy:
	$(GO) mod tidy

IMAGE         ?= ghcr.io/anomalyco/k8s-auto-dash
TAG           ?= dev
PLATFORMS     ?= linux/amd64,linux/arm64

_BUILDX_COMMON = docker buildx build \
	  --build-arg ICONS_COMMIT=$(ICONS_COMMIT) \
	  --build-arg VERSION=$(TAG) \
	  -t $(IMAGE):$(TAG)

.PHONY: image
image:
	$(_BUILDX_COMMON) --load .

.PHONY: image-push
image-push:
	$(_BUILDX_COMMON) --platform $(PLATFORMS) --push .

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
