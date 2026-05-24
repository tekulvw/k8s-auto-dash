# Native ARM Runners Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace QEMU-based cross-compilation with native `ubuntu-24.04-arm` runners in both `ci.yaml` and `release.yaml`.

**Architecture:** Each workflow's image build job becomes a matrix of two jobs (amd64 + arm64), each running on its native runner. In CI, each job loads the image locally to validate it builds. In release, each job pushes a digest-only image to GHCR, then a `merge` job combines them into a multi-arch manifest.

**Tech Stack:** GitHub Actions, `docker/build-push-action@v7`, `docker buildx imagetools create`

---

## File Structure

- Modify: `.github/workflows/ci.yaml` — replace `image` job with matrix build
- Modify: `.github/workflows/release.yaml` — replace `image` job with matrix `build` + `merge` jobs; update `manifests` dependency

---

### Task 1: Update `ci.yaml` image job to use native matrix runners

**Files:**
- Modify: `.github/workflows/ci.yaml`

- [ ] **Step 1: Replace the `image` job**

Open `.github/workflows/ci.yaml`. Replace the entire `image` job (lines 64–82) with the following:

```yaml
  image:
    runs-on: ${{ matrix.runner }}
    needs: [go, ui, helm]
    strategy:
      fail-fast: false
      matrix:
        include:
          - platform: linux/amd64
            runner: ubuntu-latest
          - platform: linux/arm64
            runner: ubuntu-24.04-arm
    steps:
      - uses: actions/checkout@v6
      - uses: docker/setup-buildx-action@v4
      - name: Resolve icons commit
        id: icons
        run: echo "sha=$(curl -fsS https://api.github.com/repos/homarr-labs/dashboard-icons/commits/main | jq -r .sha)" >> $GITHUB_OUTPUT
      - name: Build image (no push)
        uses: docker/build-push-action@v7
        with:
          context: .
          platforms: ${{ matrix.platform }}
          push: false
          load: true
          build-args: |
            ICONS_COMMIT=${{ steps.icons.outputs.sha }}
            VERSION=ci
```

Note: `docker/setup-qemu-action@v4` is removed. `load: true` replaces the no-push multi-arch build — it works because each job only builds one arch.

- [ ] **Step 2: Verify the file looks correct**

Run:
```bash
cat .github/workflows/ci.yaml
```

Confirm:
- No `setup-qemu-action` anywhere in the file
- `image` job has `runs-on: ${{ matrix.runner }}`
- `strategy.matrix` has two entries: `ubuntu-latest`/`linux/amd64` and `ubuntu-24.04-arm`/`linux/arm64`
- `load: true` is present
- `platforms: linux/amd64,linux/arm64` is gone (replaced by `${{ matrix.platform }}`)

- [ ] **Step 3: Validate YAML syntax**

Run:
```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yaml'))" && echo "YAML OK"
```

Expected: `YAML OK`

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yaml
git commit -m "ci: replace QEMU with native arm64 matrix runner in ci.yaml"
```

---

### Task 2: Update `release.yaml` image job to use native matrix runners with digest merge

**Files:**
- Modify: `.github/workflows/release.yaml`

- [ ] **Step 1: Replace the `image` job with `build` and `merge` jobs**

Open `.github/workflows/release.yaml`. Replace the entire `image` job (lines 12–41) with the following two jobs. Leave the `manifests` job untouched for now.

```yaml
  build:
    runs-on: ${{ matrix.runner }}
    strategy:
      fail-fast: false
      matrix:
        include:
          - platform: linux/amd64
            runner: ubuntu-latest
            platform_pair: linux-amd64
          - platform: linux/arm64
            runner: ubuntu-24.04-arm
            platform_pair: linux-arm64
    outputs:
      digest-linux-amd64: ${{ steps.push.outputs.digest }}
      digest-linux-arm64: ${{ steps.push.outputs.digest }}
    steps:
      - uses: actions/checkout@v6
      - uses: docker/setup-buildx-action@v4
      - uses: docker/login-action@v4
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Resolve icons commit
        id: icons
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          echo "sha=$(curl -fsS -H "Authorization: Bearer $GH_TOKEN" https://api.github.com/repos/homarr-labs/dashboard-icons/commits/main | jq -r .sha)" >> $GITHUB_OUTPUT
      - name: Build & push digest
        id: push
        uses: docker/build-push-action@v7
        with:
          context: .
          platforms: ${{ matrix.platform }}
          push: true
          outputs: type=image,name=ghcr.io/${{ github.repository }},push-by-digest=true,name-canonical=true
          build-args: |
            ICONS_COMMIT=${{ steps.icons.outputs.sha }}
            VERSION=${{ github.ref_name }}

  merge:
    runs-on: ubuntu-latest
    needs: [build]
    steps:
      - uses: docker/login-action@v4
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/setup-buildx-action@v4
      - name: Create multi-arch manifest
        run: |
          docker buildx imagetools create \
            --tag ghcr.io/${{ github.repository }}:${{ github.ref_name }} \
            --tag ghcr.io/${{ github.repository }}:latest \
            ${{ needs.build.outputs.digest-linux-amd64 }} \
            ${{ needs.build.outputs.digest-linux-arm64 }}
```

- [ ] **Step 2: Update `manifests` job dependency**

In the `manifests` job, change:
```yaml
    needs: [image]
```
to:
```yaml
    needs: [merge]
```

- [ ] **Step 3: Verify the full file looks correct**

Run:
```bash
cat .github/workflows/release.yaml
```

Confirm:
- No `setup-qemu-action` anywhere
- No `image` job exists (replaced by `build` and `merge`)
- `build` job has `runs-on: ${{ matrix.runner }}` and matrix with two entries
- `build` job has `outputs` block with `digest-linux-amd64` and `digest-linux-arm64`
- `build` step id `push` matches the `outputs` references
- `merge` job has `needs: [build]`
- `merge` job references `needs.build.outputs.digest-linux-amd64` and `digest-linux-arm64`
- `manifests` job has `needs: [merge]`

- [ ] **Step 4: Validate YAML syntax**

Run:
```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yaml'))" && echo "YAML OK"
```

Expected: `YAML OK`

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/release.yaml
git commit -m "ci: replace QEMU with native arm64 matrix runners and digest merge in release.yaml"
```

---

### Task 3: Verify with actionlint (optional but recommended)

**Files:** none modified

- [ ] **Step 1: Install actionlint if not present**

```bash
which actionlint || brew install actionlint
```

- [ ] **Step 2: Lint both workflow files**

```bash
actionlint .github/workflows/ci.yaml .github/workflows/release.yaml
```

Expected: no errors. If actionlint reports issues, fix them before proceeding.

Common issues to watch for:
- `outputs` on a matrix job: GitHub Actions does not natively fan-in matrix outputs. If actionlint flags the `build` job `outputs` block, see the note below.

**Note on matrix job outputs:** GitHub Actions matrix jobs can only expose a single output value per output name — the last matrix job to write wins. The `build` job above uses two separate output names (`digest-linux-amd64`, `digest-linux-arm64`) but both are written by the same step id (`push`). This works because each matrix job writes to `steps.push.outputs.digest`, and GitHub maps them to the output name defined at the job level. However, GitHub does not guarantee which matrix job's value lands in a given output if the key is the same. Since we use distinct output keys per `platform_pair`, this is safe — but only if each matrix job writes to its own key.

The approach above has a subtle issue: both matrix jobs write to `steps.push.outputs.digest`, but the job-level `outputs` map both keys to the same step output. GitHub will only keep the last-written value per key. To fix this correctly, use a `platform_pair` variable to distinguish:

Replace the `outputs` block in `build` with:

```yaml
    outputs:
      digest-linux-amd64: ${{ matrix.platform_pair == 'linux-amd64' && steps.push.outputs.digest || '' }}
      digest-linux-arm64: ${{ matrix.platform_pair == 'linux-arm64' && steps.push.outputs.digest || '' }}
```

This is still not reliable with the standard matrix output mechanism. The recommended production approach is to use a metadata action. Replace the `build` job's `outputs` and steps with:

```yaml
    outputs:
      digest: ${{ steps.push.outputs.digest }}
      platform_pair: ${{ matrix.platform_pair }}
```

Then in `merge`, collect digests via a separate step that reads from each matrix job using `fromJSON(needs.build.result)` — but GitHub Actions does not support per-matrix-job output access directly.

**Pragmatic solution:** Use `docker/metadata-action` to write digests to a file, upload as an artifact per matrix job, then download both in `merge`. This is the pattern from Docker's official docs. Update Task 2 steps to use this pattern:

In each `build` matrix job, after the push step, add:
```yaml
      - name: Export digest
        run: |
          mkdir -p /tmp/digests
          digest="${{ steps.push.outputs.digest }}"
          touch "/tmp/digests/${digest#sha256:}"
      - name: Upload digest
        uses: actions/upload-artifact@v4
        with:
          name: digests-${{ matrix.platform_pair }}
          path: /tmp/digests/*
          if-no-files-found: error
          retention-days: 1
```

Replace the `merge` job with:

```yaml
  merge:
    runs-on: ubuntu-latest
    needs: [build]
    steps:
      - name: Download digests
        uses: actions/download-artifact@v4
        with:
          path: /tmp/digests
          pattern: digests-*
          merge-multiple: true
      - uses: docker/login-action@v4
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/setup-buildx-action@v4
      - name: Create multi-arch manifest
        run: |
          docker buildx imagetools create \
            --tag ghcr.io/${{ github.repository }}:${{ github.ref_name }} \
            --tag ghcr.io/${{ github.repository }}:latest \
            $(printf 'ghcr.io/${{ github.repository }}@sha256:%s ' *)
        working-directory: /tmp/digests
```

And remove the `outputs` block from the `build` job entirely (it is no longer needed).

- [ ] **Step 3: Apply the artifact-based digest approach to `release.yaml`**

Update `.github/workflows/release.yaml` so the `build` job has no `outputs` block, includes the `Export digest` and `Upload digest` steps after `Build & push digest`, and the `merge` job uses `download-artifact` + the `printf` command above.

The final `release.yaml` should look like this in full:

```yaml
name: Release

on:
  push:
    tags: ['v*.*.*']

permissions:
  contents: write
  packages: write

jobs:
  build:
    runs-on: ${{ matrix.runner }}
    strategy:
      fail-fast: false
      matrix:
        include:
          - platform: linux/amd64
            runner: ubuntu-latest
            platform_pair: linux-amd64
          - platform: linux/arm64
            runner: ubuntu-24.04-arm
            platform_pair: linux-arm64
    steps:
      - uses: actions/checkout@v6
      - uses: docker/setup-buildx-action@v4
      - uses: docker/login-action@v4
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Resolve icons commit
        id: icons
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          echo "sha=$(curl -fsS -H "Authorization: Bearer $GH_TOKEN" https://api.github.com/repos/homarr-labs/dashboard-icons/commits/main | jq -r .sha)" >> $GITHUB_OUTPUT
      - name: Build & push digest
        id: push
        uses: docker/build-push-action@v7
        with:
          context: .
          platforms: ${{ matrix.platform }}
          push: true
          outputs: type=image,name=ghcr.io/${{ github.repository }},push-by-digest=true,name-canonical=true
          build-args: |
            ICONS_COMMIT=${{ steps.icons.outputs.sha }}
            VERSION=${{ github.ref_name }}
      - name: Export digest
        run: |
          mkdir -p /tmp/digests
          digest="${{ steps.push.outputs.digest }}"
          touch "/tmp/digests/${digest#sha256:}"
      - name: Upload digest
        uses: actions/upload-artifact@v4
        with:
          name: digests-${{ matrix.platform_pair }}
          path: /tmp/digests/*
          if-no-files-found: error
          retention-days: 1

  merge:
    runs-on: ubuntu-latest
    needs: [build]
    steps:
      - name: Download digests
        uses: actions/download-artifact@v4
        with:
          path: /tmp/digests
          pattern: digests-*
          merge-multiple: true
      - uses: docker/login-action@v4
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/setup-buildx-action@v4
      - name: Create multi-arch manifest
        run: |
          docker buildx imagetools create \
            --tag ghcr.io/${{ github.repository }}:${{ github.ref_name }} \
            --tag ghcr.io/${{ github.repository }}:latest \
            $(printf 'ghcr.io/${{ github.repository }}@sha256:%s ' *)
        working-directory: /tmp/digests

  manifests:
    runs-on: ubuntu-latest
    needs: [merge]
    steps:
      - uses: actions/checkout@v6
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

- [ ] **Step 4: Write the final `release.yaml`**

Replace `.github/workflows/release.yaml` entirely with the content above.

- [ ] **Step 5: Validate YAML syntax**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yaml'))" && echo "YAML OK"
```

Expected: `YAML OK`

- [ ] **Step 6: Commit**

```bash
git add .github/workflows/release.yaml
git commit -m "ci: use artifact-based digest collection for multi-arch manifest merge"
```
