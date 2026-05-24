# Version Sync and Non-Root UID Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix two independent bugs: (1) Helm chart version/appVersion not synced to the git tag at release time, causing image-not-found errors; (2) pod rejected at admission because `runAsNonRoot: true` is set but the container user is a named string rather than a numeric UID.

**Architecture:** Task 1 adds a `sed` step to `release.yaml`'s `manifests` job that rewrites `Chart.yaml` before packaging. Task 2 adds `runAsUser: 65532` and `runAsNonRoot: true` to `values.yaml`'s container-level `securityContext`. Both are single-file changes with no dependencies on each other.

**Tech Stack:** GitHub Actions (bash/sed), Helm, Kubernetes securityContext

---

## File Structure

- Modify: `.github/workflows/release.yaml` — add version-sync step to `manifests` job
- Modify: `deploy/helm/k8s-auto-dash/values.yaml` — add `runAsUser` and `runAsNonRoot` to container `securityContext`

---

### Task 1: Sync chart version to git tag in release workflow

**Files:**
- Modify: `.github/workflows/release.yaml`

- [ ] **Step 1: Insert the version-sync step into the `manifests` job**

Open `.github/workflows/release.yaml`. In the `manifests` job, insert a new step between the `actions/checkout@v6` step (line 90) and the `just manifests` step (line 97–98). The full `manifests` job after the change should look like this:

```yaml
  manifests:
    runs-on: ubuntu-latest
    needs: [merge]
    steps:
      - uses: actions/checkout@v6
      - name: Set chart version
        run: |
          BARE="${{ github.ref_name }}"
          BARE="${BARE#v}"
          sed -i "s/^version:.*/version: ${BARE}/" deploy/helm/k8s-auto-dash/Chart.yaml
          sed -i "s/^appVersion:.*/appVersion: \"${BARE}\"/" deploy/helm/k8s-auto-dash/Chart.yaml
      - uses: azure/setup-helm@v5
        with: { version: 'v4.2.0' }
      - uses: actions/setup-go@v6
        with: { go-version: '1.26' }
      - name: Install just
        uses: extractions/setup-just@v4
      - name: Generate manifests
        run: just manifests
      - name: Package chart
        run: helm package deploy/helm/k8s-auto-dash --destination dist/
      - name: Log in to GHCR
        uses: docker/login-action@v4
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Push chart to OCI registry
        run: helm push dist/*.tgz oci://ghcr.io/${{ github.repository }}/charts
      - name: Create GitHub Release
        uses: softprops/action-gh-release@v3
        with:
          files: |
            deploy/manifests/install.yaml
            dist/*.tgz
          generate_release_notes: true
```

The `Set chart version` step:
- Strips a leading `v` from the tag (`v0.2.0` → `0.2.0`; `0.2.0` → `0.2.0`)
- Rewrites `version:` and `appVersion:` in `Chart.yaml` in-place with `sed`
- Runs before `just manifests` and `helm package` so both pick up the correct version
- Does NOT commit `Chart.yaml` back — the mutation is transient in the CI workspace

- [ ] **Step 2: Validate YAML syntax**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yaml'))" && echo "YAML OK"
```

Expected: `YAML OK`

- [ ] **Step 3: Verify the sed patterns work correctly**

Run this locally to confirm the patterns match the current `Chart.yaml`:

```bash
grep -E "^version:|^appVersion:" deploy/helm/k8s-auto-dash/Chart.yaml
```

Expected output:
```
version: 0.1.0
appVersion: "0.1.0"
```

Then dry-run the sed against a copy:

```bash
cp deploy/helm/k8s-auto-dash/Chart.yaml /tmp/Chart.yaml.bak
sed -i "s/^version:.*/version: 0.2.0/" /tmp/Chart.yaml.bak
sed -i "s/^appVersion:.*/appVersion: \"0.2.0\"/" /tmp/Chart.yaml.bak
grep -E "^version:|^appVersion:" /tmp/Chart.yaml.bak
```

Expected output:
```
version: 0.2.0
appVersion: "0.2.0"
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/release.yaml
git commit -m "ci: sync chart version and appVersion to git tag at release time"
```

---

### Task 2: Add numeric runAsUser to container securityContext

**Files:**
- Modify: `deploy/helm/k8s-auto-dash/values.yaml`

- [ ] **Step 1: Update the `securityContext` block in `values.yaml`**

Open `deploy/helm/k8s-auto-dash/values.yaml`. Replace the existing `securityContext` block (lines 47–51):

```yaml
securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  readOnlyRootFilesystem: true
```

With:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  readOnlyRootFilesystem: true
```

`65532` is the stable numeric UID of the `nonroot` user in `gcr.io/distroless/static-debian12:nonroot`. Setting it explicitly satisfies Kubernetes' admission check for `runAsNonRoot: true` — without a numeric UID, kubelet cannot verify the user is non-root at pod creation time and rejects the pod.

`runAsNonRoot: true` is already set at the pod level via `podSecurityContext`; adding it at the container level makes the container `securityContext` self-contained.

- [ ] **Step 2: Verify the change**

```bash
grep -A6 "^securityContext:" deploy/helm/k8s-auto-dash/values.yaml
```

Expected output:
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  readOnlyRootFilesystem: true
```

- [ ] **Step 3: Validate the chart still renders without error**

```bash
helm template test deploy/helm/k8s-auto-dash | grep -A12 "securityContext:"
```

Expected: The container-level `securityContext` in the rendered Deployment should include `runAsUser: 65532` and `runAsNonRoot: true`.

- [ ] **Step 4: Commit**

```bash
git add deploy/helm/k8s-auto-dash/values.yaml
git commit -m "fix: add numeric runAsUser 65532 to satisfy runAsNonRoot admission check"
```
