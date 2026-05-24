GO ?= go
CONTROLLER_GEN ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.15.0

.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile=hack/boilerplate.go.txt paths=./api/...
	$(CONTROLLER_GEN) crd paths=./api/... output:crd:dir=deploy/crd

.PHONY: test
test:
	$(GO) test ./... -race -count=1

.PHONY: build
build:
	CGO_ENABLED=0 $(GO) build -o bin/k8s-auto-dash ./cmd/k8s-auto-dash

.PHONY: tidy
tidy:
	$(GO) mod tidy
