# Justfile, Repo Rename, Dep Updates, and linux/amd64 Build Fix

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert Makefile → justfile, rename repo from `tekulvw` → `tekulvw` everywhere, update all dependencies to latest compatible versions, and fix the linux/amd64 local build failure caused by QEMU memory corruption when emulating x86_64 on Apple Silicon.

**Architecture:** Four independent areas of change applied sequentially: (1) Go module rename via `go mod edit` + sed-style replacement across all `.go` files, (2) Makefile → justfile conversion, (3) dependency version bumps in `go.mod`, `package.json`, `Dockerfile`, and CI workflows, (4) local image target restricted to native arch only. All changes verified with the existing test suite (`make test` / `go test ./...`) and a local image build.

**Tech Stack:** Go 1.26, just 1.51+, pnpm 11, Docker buildx (Colima), Helm, GitHub Actions.

---

## File Map

| File | Change |
|------|--------|
| `go.mod` | Module path rename + dep version bumps |
| All `*.go` files (~35) | Import path rename |
| `Makefile` | Delete after justfile verified |
| `justfile` | New — 1:1 conversion of Makefile recipes |
| `Dockerfile` | No version changes needed (node:22, golang:1.26 are current) |
| `.github/workflows/ci.yaml` | pnpm version 9→11, helm v3.14.0→v4.2.0 |
| `.github/workflows/release.yaml` | No org references (uses `${{ github.repository }}`) |
| `deploy/helm/k8s-auto-dash/Chart.yaml` | `tekulvw` → `tekulvw` in home/sources |
| `deploy/helm/k8s-auto-dash/values.yaml` | image.repository org rename |
| `deploy/manifests/install.yaml` | image reference org rename |
| `README.md` | All `tekulvw` refs → `tekulvw` |
| `docs/superpowers/plans/*.md` | Historical plan docs — org rename |

---

## Task 1: Rename Go module path

**Files:**
- Modify: `go.mod` (line 1)
- Modify: all `*.go` files (import paths)

- [ ] **Step 1: Use `go mod edit` to rename the module**

```bash
go mod edit -module github.com/tekulvw/k8s-auto-dash
```

- [ ] **Step 2: Replace all import paths in Go source files**

```bash
find . -name '*.go' -not -path './.git/*' \
  | xargs sed -i '' 's|github.com/tekulvw/k8s-auto-dash|github.com/tekulvw/k8s-auto-dash|g'
```

- [ ] **Step 3: Verify the module line in go.mod is correct**

```bash
head -3 go.mod
```

Expected output:
```
module github.com/tekulvw/k8s-auto-dash

go 1.26.2
```

- [ ] **Step 4: Verify no tekulvw import paths remain in Go files**

```bash
grep -r "github.com/tekulvw" --include="*.go" .
```

Expected: no output.

- [ ] **Step 5: Run tests to confirm the rename didn't break anything**

```bash
go test ./... -count=1
```

Expected: all packages pass (envtest assets may be needed; if you see "KUBEBUILDER_ASSETS" errors run `make envtest-assets` first — or use `go build ./...` as a quick compile check).

- [ ] **Step 6: Commit**

```bash
git add go.mod $(git diff --name-only)
git commit -m "refactor: rename module to github.com/tekulvw/k8s-auto-dash"
```

---

## Task 2: Update non-Go references (helm, manifests, README, docs)

**Files:**
- Modify: `deploy/helm/k8s-auto-dash/Chart.yaml`
- Modify: `deploy/helm/k8s-auto-dash/values.yaml`
- Modify: `deploy/manifests/install.yaml`
- Modify: `README.md`
- Modify: `docs/superpowers/plans/2026-05-24-k8s-auto-dash-backend.md`
- Modify: `docs/superpowers/plans/2026-05-24-k8s-auto-dash-packaging.md`
- Modify: `docs/superpowers/plans/2026-05-24-k8s-auto-dash-frontend.md`

- [ ] **Step 1: Replace tekulvw in helm chart**

Edit `deploy/helm/k8s-auto-dash/Chart.yaml`:
```yaml
home: https://github.com/tekulvw/k8s-auto-dash
sources:
  - https://github.com/tekulvw/k8s-auto-dash
```

Edit `deploy/helm/k8s-auto-dash/values.yaml` line 4:
```yaml
  repository: ghcr.io/tekulvw/k8s-auto-dash
```

- [ ] **Step 2: Replace tekulvw in install.yaml**

```bash
sed -i '' 's|ghcr.io/tekulvw/k8s-auto-dash|ghcr.io/tekulvw/k8s-auto-dash|g' \
  deploy/manifests/install.yaml
```

- [ ] **Step 3: Replace tekulvw in README.md**

```bash
sed -i '' 's|tekulvw|tekulvw|g' README.md
```

Verify:
```bash
grep "tekulvw" README.md
```
Expected: no output.

- [ ] **Step 4: Replace tekulvw in historical plan docs**

```bash
sed -i '' 's|tekulvw|tekulvw|g' \
  docs/superpowers/plans/2026-05-24-k8s-auto-dash-backend.md \
  docs/superpowers/plans/2026-05-24-k8s-auto-dash-packaging.md \
  docs/superpowers/plans/2026-05-24-k8s-auto-dash-frontend.md
```

- [ ] **Step 5: Confirm no tekulvw references remain anywhere (excluding .git)**

```bash
grep -r "tekulvw" \
  --include="*.go" --include="*.yaml" --include="*.yml" \
  --include="*.json" --include="*.md" --include="Makefile" \
  --include="Dockerfile" --include="*.ts" --include="*.js" \
  --include="*.mjs" \
  . 2>&1 | grep -v "^Binary\|\.git/"
```

Expected: no output.

- [ ] **Step 6: Verify helm still lints**

```bash
helm lint deploy/helm/k8s-auto-dash
```

Expected: `1 chart(s) linted, 0 chart(s) failed`

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "refactor: rename org tekulvw → tekulvw in all non-Go files"
```

---

## Task 3: Convert Makefile to justfile

**Files:**
- Create: `justfile`
- Delete: `Makefile` (after justfile is verified)

The justfile is a direct translation of all Makefile targets. Key differences:
- `just` uses `{{variable}}` for interpolation (but `$()` shell substitution still works in recipe bodies)
- Variables use `:=` for assignment
- No `.PHONY` needed — all recipes are phony by default
- The `_BUILDX_COMMON` variable becomes a `_buildx_common` shell variable inside the recipes that share it, or we inline the flags
- `just` uses `--builder multiarch` for the local image target (required by Colima's docker-container driver)

- [ ] **Step 1: Create justfile**

Create `/Users/will/projects/k8s_auto_dashboard/justfile` with content:

```just
# Variables
go          := "go"
ui_dir      := "ui"
icons_commit := env_var_or_default("ICONS_COMMIT", "main")
image       := env_var_or_default("IMAGE", "ghcr.io/tekulvw/k8s-auto-dash")
tag         := env_var_or_default("TAG", "dev")
platforms   := env_var_or_default("PLATFORMS", "linux/amd64,linux/arm64")

controller_gen    := "go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.21.0"
envtest_k8s_ver   := env_var_or_default("ENVTEST_K8S_VERSION", "1.33.x")
envtest           := "go run sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.24.1"

# Default: list all recipes
default:
    @just --list

# Generate CRD manifests and DeepCopy methods
generate:
    {{controller_gen}} object:headerFile=hack/boilerplate.go.txt paths=./api/...
    {{controller_gen}} crd paths=./api/... output:crd:dir=deploy/crd

# Build the UI and copy artifacts into embed dirs
ui-build:
    cd {{ui_dir}} && pnpm install && ICONS_COMMIT={{icons_commit}} pnpm icons && pnpm run build
    rm -rf internal/assets/ui/* internal/assets/icons/*
    cp -r {{ui_dir}}/build/. internal/assets/ui/
    cp -r {{ui_dir}}/icons/. internal/assets/icons/

# Build UI then Go binary
build-all: ui-build build

# Download envtest binaries for the target k8s version
envtest-assets:
    @{{envtest}} use {{envtest_k8s_ver}} -p path > /dev/null

# Run Go tests (requires envtest assets)
test: envtest-assets
    KUBEBUILDER_ASSETS="$({{envtest}} use {{envtest_k8s_ver}} -p path)" \
      {{go}} test ./... -race -count=1

# Build Go binary only
build:
    CGO_ENABLED=0 {{go}} build -o bin/k8s-auto-dash ./cmd/k8s-auto-dash

# Tidy go.mod / go.sum
tidy:
    {{go}} mod tidy

# Build image for local dev (native arch only — linux/amd64 via QEMU on Apple Silicon
# triggers Go GC/QEMU memory corruption; use image-push or CI for multi-arch builds)
image:
    docker buildx build \
      --builder multiarch \
      --build-arg ICONS_COMMIT={{icons_commit}} \
      --build-arg VERSION={{tag}} \
      -t {{image}}:{{tag}} \
      --load \
      .

# Build and push multi-platform image
image-push:
    docker buildx build \
      --builder multiarch \
      --build-arg ICONS_COMMIT={{icons_commit}} \
      --build-arg VERSION={{tag}} \
      -t {{image}}:{{tag}} \
      --platform {{platforms}} \
      --push \
      .

# Sync generated CRD into helm chart crds/ dir
chart-sync: generate
    cp deploy/crd/k8s-auto-dash.io_dashboardconfigs.yaml \
       deploy/helm/k8s-auto-dash/crds/dashboardconfig.yaml

# Lint helm chart
chart-lint: chart-sync
    helm lint deploy/helm/k8s-auto-dash

# Render helm chart templates
chart-template: chart-sync
    helm template test deploy/helm/k8s-auto-dash

# Generate concatenated install.yaml from helm + CRDs
manifests: chart-sync
    mkdir -p deploy/manifests
    ( \
      echo "# Generated by 'just manifests'. Do not edit by hand."; \
      echo "# Source: deploy/helm/k8s-auto-dash"; \
      echo "---"; \
      cat deploy/crd/k8s-auto-dash.io_dashboardconfigs.yaml; \
      echo "---"; \
      helm template k8s-auto-dash deploy/helm/k8s-auto-dash \
        --namespace k8s-auto-dash \
        --set crd.install=false; \
    ) > deploy/manifests/install.yaml
```

- [ ] **Step 2: Verify justfile parses and lists recipes**

```bash
just --list
```

Expected output: a table listing `build`, `build-all`, `chart-lint`, `chart-sync`, `chart-template`, `envtest-assets`, `generate`, `image`, `image-push`, `manifests`, `test`, `tidy`, `ui-build`.

- [ ] **Step 3: Smoke-test the build recipe**

```bash
just build
```

Expected: `bin/k8s-auto-dash` binary is created with exit code 0.

- [ ] **Step 4: Smoke-test the tidy recipe**

```bash
just tidy
```

Expected: exits 0, no unexpected changes to go.mod/go.sum (run `git diff go.mod go.sum` to confirm).

- [ ] **Step 5: Remove the Makefile**

```bash
rm Makefile
```

- [ ] **Step 6: Commit**

```bash
git add justfile
git rm Makefile
git commit -m "build: replace Makefile with justfile"
```

---

## Task 4: Update Go dependency versions

**Files:**
- Modify: `go.mod`

Updates:
- `github.com/stretchr/testify` v1.11.1 → v1.11.1 (already latest — no change)
- `k8s.io/apimachinery` v0.36.1 → v0.36.1 (already latest stable — no change)
- `k8s.io/client-go` v0.36.0 → v0.36.1
- `sigs.k8s.io/gateway-api` v1.2.0 → v1.5.1
- `sigs.k8s.io/controller-runtime` v0.24.1 → v0.24.1 (already latest — no change)
- `go` toolchain directive: `1.26.2` → `1.26.3`

Note: `k8s.io/apimachinery`, `k8s.io/api`, `k8s.io/apiextensions-apiserver`, and `k8s.io/client-go` must be kept in sync (same minor version). Currently at v0.36.x — stay on v0.36.1 for all of them.

- [ ] **Step 1: Bump client-go to v0.36.1**

```bash
go get k8s.io/client-go@v0.36.1
```

- [ ] **Step 2: Bump gateway-api to v1.5.1**

```bash
go get sigs.k8s.io/gateway-api@v1.5.1
```

- [ ] **Step 3: Update the go toolchain directive**

Edit `go.mod` line 3 from `go 1.26.2` to `go 1.26.3`:
```
go 1.26.3
```

- [ ] **Step 4: Tidy**

```bash
go mod tidy
```

- [ ] **Step 5: Verify the module builds**

```bash
go build ./...
```

Expected: exits 0, no errors.

- [ ] **Step 6: Run tests**

```bash
KUBEBUILDER_ASSETS="$(go run sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.24.1 use 1.33.x -p path)" \
  go test ./... -race -count=1
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(go): bump client-go to v0.36.1, gateway-api to v1.5.1, toolchain to 1.26.3"
```

---

## Task 5: Update CI workflow dependency versions

**Files:**
- Modify: `.github/workflows/ci.yaml`
- Modify: `.github/workflows/release.yaml`

Updates:
- `pnpm/action-setup@v3` with `version: 9` → `version: 11`
- `azure/setup-helm@v4` with `version: 'v3.14.0'` → `version: 'v4.2.0'` (in both ci.yaml and release.yaml)

The `release.yaml` helm step is in the `manifests` job.

- [ ] **Step 1: Update pnpm version in ci.yaml**

In `.github/workflows/ci.yaml`, change:
```yaml
      - uses: pnpm/action-setup@v3
        with: { version: 9 }
```
to:
```yaml
      - uses: pnpm/action-setup@v3
        with: { version: 11 }
```

- [ ] **Step 2: Update helm version in ci.yaml**

In `.github/workflows/ci.yaml`, change:
```yaml
      - uses: azure/setup-helm@v4
        with: { version: 'v3.14.0' }
```
to:
```yaml
      - uses: azure/setup-helm@v4
        with: { version: 'v4.2.0' }
```

- [ ] **Step 3: Update helm version in release.yaml**

In `.github/workflows/release.yaml`, change:
```yaml
      - uses: azure/setup-helm@v4
        with: { version: 'v3.14.0' }
```
to:
```yaml
      - uses: azure/setup-helm@v4
        with: { version: 'v4.2.0' }
```

- [ ] **Step 4: Update ci.yaml to use `just` instead of `make`**

In `.github/workflows/ci.yaml`, change the Vet and Test steps in the `go` job:
```yaml
      - name: Vet
        run: go vet ./...
      - name: Test
        run: make test
```
to:
```yaml
      - name: Install just
        uses: extractions/setup-just@v2
      - name: Vet
        run: go vet ./...
      - name: Test
        run: just test
```

Also update the `helm` job:
```yaml
      - run: helm lint deploy/helm/k8s-auto-dash
      - run: helm template test deploy/helm/k8s-auto-dash > /dev/null
```
(No change needed here — these invoke `helm` directly, not `make`.)

And update the `manifests` job in `release.yaml` — change:
```yaml
      - name: Generate manifests
        run: make manifests
```
to:
```yaml
      - name: Install just
        uses: extractions/setup-just@v2
      - name: Generate manifests
        run: just manifests
```

- [ ] **Step 5: Verify helm still lints locally with Helm 4 syntax**

```bash
helm lint deploy/helm/k8s-auto-dash
```

Expected: `1 chart(s) linted, 0 chart(s) failed`

- [ ] **Step 6: Commit**

```bash
git add .github/workflows/ci.yaml .github/workflows/release.yaml
git commit -m "ci: bump pnpm to 11, helm to v4.2.0, switch make → just"
```

---

## Task 6: Update controller-gen version in justfile

**Files:**
- Modify: `justfile`

The `controller_gen` variable currently pins to `v0.17.0`. Latest is `v0.21.0`.

- [ ] **Step 1: Update controller_gen variable in justfile**

Change in `justfile`:
```just
controller_gen    := "go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.0"
```
to:
```just
controller_gen    := "go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.21.0"
```

(Note: this is already set to v0.21.0 in Task 3's justfile — this task is only needed if executing Task 3 used a different version. Skip if Task 3 already set v0.21.0.)

- [ ] **Step 2: Smoke-test generate**

```bash
just generate
```

Expected: exits 0, CRD yaml regenerated in `deploy/crd/`.

- [ ] **Step 3: Verify CRD output is unchanged**

```bash
git diff deploy/crd/
```

Expected: no diff (the generated output should be identical with the new controller-gen version for the same API types).

- [ ] **Step 4: Commit if there were any changes**

```bash
git add justfile deploy/crd/
git commit -m "chore: bump controller-gen to v0.21.0"
```

---

## Task 7: Fix linux/amd64 local image build

**Files:**
- Modify: `justfile` (already done in Task 3 — verify the `image` recipe is correct)

The root cause: when building `linux/amd64` via QEMU on Apple Silicon (arm64), Go's GC and `compress/flate`/`archive/zip` trigger QEMU memory-model races, causing `go mod download` to panic. This is a known QEMU limitation, not a Go or Alpine bug. CI is unaffected because it runs on native x86_64 hardware.

The fix: the local `image` recipe omits `--platform` (so buildx defaults to native `linux/arm64`) and retains `--load`. The multi-platform push stays in `image-push` and CI.

- [ ] **Step 1: Confirm the justfile image recipe has no --platform flag**

```bash
grep -A 10 "^image:" justfile
```

Expected output: no `--platform` line in the `image` recipe (only in `image-push`).

- [ ] **Step 2: Run a local image build**

```bash
just image
```

Expected: build completes successfully and the image `ghcr.io/tekulvw/k8s-auto-dash:dev` appears in `docker images`.

```bash
docker images ghcr.io/tekulvw/k8s-auto-dash:dev
```

Expected: one row with a recent `CREATED` timestamp.

- [ ] **Step 3: Confirm the image runs**

```bash
docker run --rm ghcr.io/tekulvw/k8s-auto-dash:dev --help 2>&1 | head -5
```

Expected: help text or usage output (not a crash).

- [ ] **Step 4: Commit**

```bash
git add justfile
git commit -m "fix: local image target builds native arch only (avoids QEMU/amd64 GC panic)"
```

---

## Task 8: Final verification

- [ ] **Step 1: Confirm zero tekulvw references remain in tracked files**

```bash
git grep "tekulvw" -- '*.go' '*.yaml' '*.yml' '*.md' '*.json' 'Dockerfile' 'justfile'
```

Expected: no output.

- [ ] **Step 2: Confirm build and tests pass end-to-end**

```bash
just build
just tidy
git diff go.mod go.sum
```

Expected: `bin/k8s-auto-dash` exists, `go.mod`/`go.sum` are clean.

- [ ] **Step 3: Confirm helm lint passes**

```bash
just chart-lint
```

Expected: `1 chart(s) linted, 0 chart(s) failed`

- [ ] **Step 4: Confirm just --list shows all expected recipes**

```bash
just --list
```

Expected: `build`, `build-all`, `chart-lint`, `chart-sync`, `chart-template`, `default`, `envtest-assets`, `generate`, `image`, `image-push`, `manifests`, `test`, `tidy`, `ui-build`.
