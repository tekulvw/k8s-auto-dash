# Version Sync and Non-Root UID Design

**Date:** 2026-05-24  
**Status:** Approved

## Problems

### 1. Chart version / image tag mismatch

`Chart.yaml` has `version` and `appVersion` hardcoded at `0.1.0`. The release workflow never updates them. The Helm image template defaults to `Chart.AppVersion` when `image.tag` is empty, so users installing with default values pull the wrong image tag (e.g., `0.1.0` instead of `0.2.0`).

### 2. `runAsNonRoot` with non-numeric user

`values.yaml` sets `runAsNonRoot: true` at the pod level but does not set `runAsUser`. The Dockerfile uses `USER nonroot:nonroot` (a named user, not a numeric UID). Kubernetes' admission controller cannot verify a named user is non-root at pod creation time and rejects the pod with:

> `container has runAsNonRoot and image has non-numeric user (nonroot), cannot verify user is non-root`

---

## Fixes

### Fix 1: Auto-sync chart version in release workflow

**File:** `.github/workflows/release.yaml`  
**Job:** `manifests`  
**Where:** New step inserted before `just manifests`

Strip the leading `v` from `github.ref_name` and rewrite both fields in `Chart.yaml` using `sed`:

```yaml
- name: Set chart version
  run: |
    BARE="${{ github.ref_name }}"
    BARE="${BARE#v}"
    sed -i "s/^version:.*/version: ${BARE}/" deploy/helm/k8s-auto-dash/Chart.yaml
    sed -i "s/^appVersion:.*/appVersion: \"${BARE}\"/" deploy/helm/k8s-auto-dash/Chart.yaml
```

This is a transient mutation in the CI workspace — `Chart.yaml` in the repo is not committed back. The git tag is the source of truth. The packaged `.tgz` and generated `install.yaml` (from `just manifests`) will both reflect the correct version.

`Chart.yaml` on disk stays at `0.1.0` as a placeholder; it is overwritten at release time.

### Fix 2: Add numeric `runAsUser` to container securityContext

**File:** `deploy/helm/k8s-auto-dash/values.yaml`  
**Field:** `securityContext` (container-level)

Add `runAsNonRoot: true` and `runAsUser: 65532` to the existing `securityContext` block:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  readOnlyRootFilesystem: true
```

`65532` is the stable, documented numeric UID of the `nonroot` user in Google's distroless images (`gcr.io/distroless/static-debian12:nonroot`). Setting it explicitly satisfies Kubernetes' admission check for `runAsNonRoot`.

`runAsNonRoot: true` is already set at the pod level via `podSecurityContext`; adding it at the container level too makes the container `securityContext` self-contained and explicit.

---

## What Is Not Changed

- `Chart.yaml` on disk — not committed back; CI overwrites at release time
- Dockerfile — no change needed; `USER nonroot:nonroot` remains
- `podSecurityContext` in `values.yaml` — unchanged
- Any other workflow jobs or templates
