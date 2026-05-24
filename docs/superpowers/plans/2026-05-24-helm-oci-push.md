# Helm OCI Push Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `helm push` step to the `manifests` job in `.github/workflows/release.yaml` so the Helm chart is also published as an OCI artifact to `ghcr.io/<repo>/charts/k8s-auto-dash` on every versioned tag release.

**Architecture:** The existing `manifests` job already installs Helm and packages the chart to `dist/*.tgz`. We add a `docker/login-action` step (authenticated via `GITHUB_TOKEN`, same pattern as the `image` job) followed by a `helm push` step that pushes the packaged chart to the OCI path `oci://ghcr.io/${{ github.repository }}/charts`. No new jobs, no new secrets, no new tools required.

**Tech Stack:** GitHub Actions, Helm v4.2.0 (already pinned), ghcr.io OCI registry, `docker/login-action@v3`

---

### Task 1: Add OCI login and helm push steps to the manifests job

**Files:**
- Modify: `.github/workflows/release.yaml`

This is a pure YAML edit — no code to test with unit tests. Verification is done by inspecting the diff and confirming YAML validity.

- [ ] **Step 1: Read the current workflow file**

```bash
cat .github/workflows/release.yaml
```

Expected output: the 60-line file ending with `generate_release_notes: true`.

- [ ] **Step 2: Add the login step before the helm push**

In `.github/workflows/release.yaml`, after the `- name: Package chart` step and before `- name: Create GitHub Release`, insert the following two steps:

```yaml
      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Push chart to OCI registry
        run: helm push dist/*.tgz oci://ghcr.io/${{ github.repository }}/charts
```

The final `manifests` job steps section should look like this (complete, for reference):

```yaml
  manifests:
    runs-on: ubuntu-latest
    needs: [image]
    steps:
      - uses: actions/checkout@v4
      - uses: azure/setup-helm@v4
        with: { version: 'v4.2.0' }
      - uses: actions/setup-go@v5
        with: { go-version: '1.26' }
      - name: Install just
        uses: extractions/setup-just@v2
      - name: Generate manifests
        run: just manifests
      - name: Package chart
        run: helm package deploy/helm/k8s-auto-dash --destination dist/
      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Push chart to OCI registry
        run: helm push dist/*.tgz oci://ghcr.io/${{ github.repository }}/charts
      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: |
            deploy/manifests/install.yaml
            dist/*.tgz
          generate_release_notes: true
```

- [ ] **Step 3: Validate the YAML is well-formed**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yaml'))" && echo "YAML OK"
```

Expected output: `YAML OK`

- [ ] **Step 4: Verify the diff looks correct**

```bash
git diff .github/workflows/release.yaml
```

Expected: two new steps added between `Package chart` and `Create GitHub Release`. No other lines changed.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/release.yaml
git commit -m "ci: push helm chart to ghcr.io OCI registry on release"
```
