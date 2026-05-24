# Native ARM Runners Design

**Date:** 2026-05-24  
**Status:** Approved

## Problem

Both `ci.yaml` and `release.yaml` build `linux/arm64` images using QEMU emulation on `ubuntu-latest` (amd64) runners. QEMU-emulated Go builds are 5-20x slower than native and have caused Go GC corruption (documented in `justfile` line 49).

## Goal

Replace QEMU with native `ubuntu-24.04-arm` GitHub-hosted runners for `arm64` builds in both workflows.

## Approach

Use `docker/build-push-action` with `push-by-digest=true` per arch, then merge digests into a multi-arch manifest. This is the pattern recommended in Docker's official GitHub Actions documentation.

---

## `ci.yaml` Changes

### Before

Single `image` job on `ubuntu-latest` using QEMU to build both `linux/amd64` and `linux/arm64`. No push (`push: false`).

### After

The `image` job becomes a **matrix job** with two entries:

| matrix.platform | matrix.runner |
|---|---|
| `linux/amd64` | `ubuntu-latest` |
| `linux/arm64` | `ubuntu-24.04-arm` |

Each job:
- Checks out the repo
- Sets up Buildx (no QEMU step)
- Resolves the icons commit SHA
- Runs `docker/build-push-action` with `--load` (loads into local Docker daemon, no push)
- Validates the image was built successfully

`needs: [go, ui, helm]` stays on the matrix job.

`docker/setup-qemu-action` is removed entirely.

---

## `release.yaml` Changes

### Before

Single `image` job on `ubuntu-latest` using QEMU to build and push both arches to GHCR with version + latest tags.

### After

Three jobs:

#### `build` (matrix, parallel)

Same matrix as CI (`ubuntu-latest` + `ubuntu-24.04-arm`). Each job:
- Logs in to GHCR
- Resolves icons commit SHA
- Runs `docker/build-push-action` with:
  - `push-by-digest: true`
  - `push: true`
  - No tags (digest-only push)
- Outputs its digest via `$GITHUB_OUTPUT`

#### `merge` (needs: build)

Runs on `ubuntu-latest`. Collects both digests from the matrix job outputs and runs:

```
docker buildx imagetools create \
  --tag ghcr.io/${{ github.repository }}:${{ github.ref_name }} \
  --tag ghcr.io/${{ github.repository }}:latest \
  <amd64-digest> \
  <arm64-digest>
```

This produces the final multi-arch manifest with both version and latest tags.

#### `manifests` (needs: merge — was: needs: image)

Unchanged. Dependency updated from `image` to `merge`.

---

## What Is Not Changed

- `go`, `ui`, `helm` jobs in `ci.yaml` — unchanged
- `manifests` job in `release.yaml` — unchanged (only its `needs` is updated)
- Dockerfile — unchanged
- All other workflow triggers, permissions, and secrets — unchanged

---

## Key Details

- `docker/setup-qemu-action` is removed from both workflows
- `docker/setup-buildx-action` remains (needed for `buildx imagetools create` in `merge`)
- The icons commit resolution (`curl` to GitHub API) runs in each `build` matrix job — both arches need it as a `build-arg`, so it runs twice (once per runner)
- In `release.yaml`, GHCR login runs in both `build` (to push digests) and `merge` (to create manifest)
- Matrix digest passing: use `strategy.job-index` or a sanitized platform string as the matrix output key (e.g., `linux-amd64`, `linux-arm64`) to avoid YAML key collision with slashes
