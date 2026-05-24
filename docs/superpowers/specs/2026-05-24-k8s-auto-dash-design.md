# k8s-auto-dash — Design

Status: Approved (brainstorming)
Date: 2026-05-24

A Kubernetes-native homelab dashboard that auto-discovers Gateway API
`Gateway` and `HTTPRoute` resources, renders one tile per discovered
hostname, and lets the user manually curate layout, naming, icons, and
visibility — with all customization persisted to a single CRD in the
cluster.

Working name: `k8s-auto-dash`. Binary, image, Helm chart, and CRD API
group all use this slug.

## Goals

- Zero-config first run: deploy the chart, visit the URL, see every
  hostname routed by the cluster's Gateway API.
- Manual curation: drag-and-drop reorder, rename, re-icon, regroup, and
  hide individual tiles without leaving the browser.
- GitOps-friendly: all curation state lives in a CRD that can be diffed,
  versioned, and applied with `kubectl`.
- Low operational cost: single Go binary, single container, single
  replica, ~30 MB image, no external database.

## Non-Goals (v1)

- Live service widgets (Sonarr queue, Plex now-playing, etc.) — defer to
  v2.
- Custom wallpapers / background images.
- Authentication or multi-user editing (assume the dashboard sits behind
  an upstream auth proxy such as authentik or oauth2-proxy).
- Uptime history, status timeline, alerting.
- Filter query language (a search bar + hide-tile UI are sufficient).
- High availability (a homelab dashboard does not need multiple
  replicas).

## Users & Operating Assumptions

- Homelab operators running a Kubernetes cluster with at least one
  Gateway API implementation installed (Cilium, Envoy Gateway, Istio,
  etc.).
- Dashboard is reached over the same Gateway, optionally behind a
  cluster-level auth proxy.
- Single-replica deployment is acceptable. State lives in a cluster
  CRD, not on a PVC.

## Architecture

Single Go binary, single container, single Kubernetes Deployment with
one replica.

```
                  ┌────────────────────────────────────────────┐
                  │  k8s-auto-dash pod                          │
                  │                                            │
  k8s API ─watch──┤  Discoverer  ──►  in-memory route cache    │
                  │                       │                    │
                  │                       ▼                    │
  HTTP probes ◄───┤  Health checker ──►  status cache          │
                  │                       │                    │
                  │                       ▼                    │
  Browser ◄──SSE──┤  HTTP API (Go)  ◄────┘                    │
                  │   GET    /api/tiles                        │
                  │   PATCH  /api/config                       │
                  │   GET    /api/events    (SSE)              │
                  │                                            │
                  │  Config store ──►  DashboardConfig CR      │
                  │                                            │
                  │  Static assets (embedded SvelteKit build)  │
                  └────────────────────────────────────────────┘
```

### Components

**Discoverer.** Uses `client-go` informers to watch
`gateway.networking.k8s.io/v1` `Gateway` and `HTTPRoute` cluster-wide,
plus its own `DashboardConfig` CR. Maintains an in-memory map keyed by
`(namespace, route-name, hostname)`. Recomputes the tile set on every
event, debounced 250 ms, and emits add/update/remove deltas on an
internal channel.

**Health checker.** Worker pool (default 5 goroutines, configurable)
that periodically probes every discovered hostname. Default interval
60 s with ±10% jitter. Uses `HEAD` first; on 405/501 retries once with
`GET`. Timeout default 5 s. TLS verified by default; a global
`insecureSkipVerify` flag plus per-tile override allows self-signed
setups. Identifies itself with `User-Agent: k8s-auto-dash/<version>
(health-check)`. Hidden tiles are still probed so toggling visibility
is cheap.

**HTTP API + embedded UI.** `net/http` server. Serves the embedded
SvelteKit SPA via `embed.FS`. Exposes a small JSON API and an SSE
stream for live tile/status updates.

**Config writer.** Single chokepoint for mutations. Reads current CR,
merges patch in memory, performs an optimistic-concurrency `Update`
using `resourceVersion`. Retries once on conflict, then surfaces the
error to the client.

### Why this shape

- Cluster-wide watch streams (no polling) match the answers given
  during brainstorming: k8s-native persistence, no external database,
  unauthenticated UI behind an upstream proxy.
- Single replica means no leader election, no cache coordination, no
  distributed state. State that must survive restarts lives in the
  CRD.
- A CRD (rather than a ConfigMap) gives schema validation, typed
  client-go bindings, status subresource for diagnostics, and a
  natural `kubectl get / -o yaml` workflow for GitOps.

## Data Model

### CRD: `DashboardConfig` (cluster-scoped, group `k8s-auto-dash.io/v1alpha1`)

The controller reads exactly one instance, named `default`. If absent
on startup, it is created empty.

```yaml
apiVersion: k8s-auto-dash.io/v1alpha1
kind: DashboardConfig
metadata:
  name: default
spec:
  settings:
    title: "Homelab"
    theme: dark           # dark | light | auto
    healthCheck:
      enabled: true
      intervalSeconds: 60
      timeoutSeconds: 5
      insecureSkipVerify: false
    discovery:
      namespaceSelector: {}   # labelSelector; empty = cluster-wide
      gatewayClassNames: []   # empty = all classes

  groups:
    - id: media             # stable slug
      name: "Media"         # editable display label
      order: 0
    - id: infra
      name: "Infrastructure"
      order: 1

  tiles:                    # only entries with non-default overrides
    - id: "media/jellyfin/jellyfin.example.com"
      hidden: false
      name: "Jellyfin"
      description: "Media server"
      icon: "jellyfin"      # selfhst/dashboard-icons slug, or http(s) URL
      group: media          # group id; defaults to namespace
      order: 0
      url: "https://jellyfin.example.com"   # defaults to https://<hostname>
      insecureSkipVerify: false

  bookmarks:                # manual non-k8s tiles
    - id: "router"
      name: "Router"
      url: "https://192.168.1.1"
      icon: "ubiquiti"
      group: infra
      order: 99

status:
  discoveredTiles: 47
  lastReconciled: "2026-05-24T10:00:00Z"
  conditions: []
```

### Tile ID

`<namespace>/<httproute-name>/<hostname>`. Used as the stable lookup
key for overrides. If the HTTPRoute is renamed or its hostname list
changes, the override becomes orphaned; orphaned overrides are
surfaced in a diagnostics UI panel with a one-click clean-up.

### Tile derivation algorithm

Runs on every relevant informer event, debounced 250 ms:

```
tiles = []
for each HTTPRoute hr:
    for each hostname h in hr.spec.hostnames:
        if h is empty or contains "*":      continue   # not tiled
        gatewayRefs = resolve hr.spec.parentRefs to known Gateways
        if gatewayRefs is empty:            continue
        id = f"{hr.namespace}/{hr.name}/{h}"
        tiles.append(RouteInfo{id, hostname:h, hr, gatewayRefs})
```

Skipped routes (wildcards, unresolved parentRefs) are not silently
dropped — they appear in a "skipped routes" diagnostics panel so the
operator can see why a route they expected isn't showing up.

Duplicate hostnames across HTTPRoutes are kept as separate tiles. The
operator hides one if undesired; the dashboard does not attempt to
guess which is canonical.

### Auto-derivation defaults (when no override exists)

- `name` — first hostname segment, label-cased
  (`jellyfin.example.com` → "Jellyfin").
- `icon` — lowercase first hostname segment, matched against the
  bundled `dashboard-icons` slug list at startup; falls back to a
  generic globe icon.
- `group` — HTTPRoute namespace; a default group is auto-created the
  first time a namespace is seen.
- `url` — `https://<hostname>`.

### Merged tile view served to the UI

```json
{
  "groups": [
    {"id":"media","name":"Media","order":0}
  ],
  "tiles": [
    {
      "id": "media/jellyfin/jellyfin.example.com",
      "source": "httproute",
      "name": "Jellyfin",
      "url": "https://jellyfin.example.com",
      "icon": "jellyfin",
      "description": "Media server",
      "group": "media",
      "order": 0,
      "hidden": false,
      "status": {
        "state": "up",
        "statusCode": 200,
        "latencyMs": 42,
        "checkedAt": "2026-05-24T10:00:00Z",
        "error": ""
      },
      "k8s": {
        "namespace": "media",
        "httpRouteName": "jellyfin",
        "gatewayRefs": [
          {"namespace":"gateway","name":"external"}
        ]
      }
    }
  ]
}
```

`source: "bookmark"` tiles carry the same shape with `k8s: null`.

## HTTP API

- `GET /api/tiles` — merged tile + group + status view.
- `GET /api/events` — SSE stream:
  - `tile-added`     `{ tile }`
  - `tile-updated`   `{ id, fields }`
  - `tile-removed`   `{ id }`
  - `status-changed` `{ id, status }`
  - `config-changed` `{ source: "kubectl" }`
- `PATCH /api/config` — partial merge of `spec`. Body is a partial
  `DashboardConfig.spec`; only the fields present are applied.
- `PUT /api/config/groups` — replace groups list (used by group
  reorder).
- `POST /api/config/bookmarks` — add a bookmark.
- `DELETE /api/config/bookmarks/{id}` — remove a bookmark.
- `GET /healthz`, `GET /readyz` — liveness/readiness.
- `GET /metrics` — Prometheus metrics.

All mutations go through the `ConfigWriter` described above.

`config-changed` is emitted whenever the controller observes its own
CR being modified by an outside actor (e.g., `kubectl apply`). The
client refetches and shows a toast.

## Health Checks

States: `up`, `degraded`, `down`, `unknown`.

- `up`        — final response 200–399.
- `degraded`  — reachable but final response is 4xx/5xx.
- `down`      — network error or timeout.
- `unknown`   — never checked yet, or health checks disabled.

No history is kept in v1; only the latest snapshot per tile.

## UI

### Page structure

Single-page app, dark theme by default, light/dark/auto toggle.
Layout: header with title, search box, settings cog, theme toggle, and
edit-mode pencil. Below: grouped sections of wide info cards (icon +
name + hostname + status dot) — confirmed visual style during
brainstorming.

Mobile-responsive: groups stack, cards fall back to one column on
narrow viewports.

### View vs. edit mode

A pencil in the header toggles edit mode. There is no separate edit
API; the same `PATCH /api/config` powers everything.

Edit-mode additions:

- Drag handles on tiles and group headers.
- `⋯` menu per tile: Edit, Hide, Move to group.
- A right-side drawer listing all hidden tiles with one-click restore.
- "+ Add bookmark" affordance per group.
- An "Unsorted" group at the top containing any discovered tile that
  has not yet been assigned a group via the UI (initial group is the
  namespace, so this is rare in practice).

### Drag-and-drop

Library: `svelte-dnd-action` (or equivalent — must be small and
dependency-light).

- Drag within a group updates `tile.order`.
- Drag across groups updates `tile.group` and `tile.order`.
- Drag a group header updates `groups[*].order`.
- The client applies the change optimistically and sends a single
  `PATCH /api/config`. On API error, it rolls back and shows a toast.

### Tile editor

Side panel triggered by `⋯ → Edit`. Fields:

- Name, URL, icon (typeahead against bundled icon slugs with live
  preview; "custom URL" tab for arbitrary image URLs), description,
  group, insecure-TLS toggle.
- Read-only info: namespace, HTTPRoute name, Gateway parent refs, all
  hostnames on the route.
- "Reset to auto" wipes the override entry.

### Search & filtering

- Header search box does live client-side fuzzy filtering on name,
  hostname, and description. All data is already in the browser.
- Per-tile hide via `⋯ → Hide` (sets `hidden: true`).
- Per-group "hide all" via the group `⋯` menu.

## Deployment

### Container

- Multi-stage Dockerfile producing a single static binary in a
  `distroless/static` (or `scratch`) final image, ~25–30 MB.
- Built for `linux/amd64` and `linux/arm64`.
- Published to GHCR as `ghcr.io/<owner>/k8s-auto-dash:vX.Y.Z` and
  `:latest`.

### Helm chart

Located at `deploy/helm/k8s-auto-dash/`. Representative values:

```yaml
image:
  repository: ghcr.io/<owner>/k8s-auto-dash
  tag: ""                  # defaults to chart appVersion
serviceAccount:
  create: true
rbac:
  create: true
crd:
  install: true            # disable when CRDs managed externally
service:
  type: ClusterIP
httpRoute:                 # optional: chart can publish its own route
  enabled: false
  hostname: ""
  parentRef:
    name: ""
    namespace: ""
config:                    # optional inline DashboardConfig spec
  settings:
    title: "Homelab"
```

### Raw manifests

`deploy/manifests/install.yaml` is a concatenated CRD + RBAC +
Deployment + Service generated by `helm template`. Supports the
`kubectl apply -f https://...` workflow common in homelabs.

### RBAC

```yaml
ClusterRole (read-only, cluster-wide):
  - apiGroups: ["gateway.networking.k8s.io"]
    resources: ["gateways","httproutes"]
    verbs: ["get","list","watch"]

ClusterRole (own CR):
  - apiGroups: ["k8s-auto-dash.io"]
    resources: ["dashboardconfigs"]
    verbs: ["get","list","watch","update","patch","create"]
  - apiGroups: ["k8s-auto-dash.io"]
    resources: ["dashboardconfigs/status"]
    verbs: ["get","update","patch"]
```

The CR is cluster-scoped; both bindings are `ClusterRoleBinding`.

### Config bootstrap

On startup: if no `DashboardConfig/default` exists, the controller
creates an empty one. Idempotent.

### Observability

- `/healthz`, `/readyz` endpoints.
- Prometheus metrics at `/metrics`: discovered tile count, health
  probe success/failure counters, probe latency histogram, CR write
  count, SSE client count.
- Structured JSON logs via `log/slog`.

## Testing Strategy

- Unit tests: tile derivation, override merging, status
  classification, JSON-patch merge, optimistic-concurrency retry.
- Integration tests with `envtest` (controller-runtime's local API
  server): real Gateway API CRDs installed, controller exercised
  end-to-end, assertions on `/api/tiles` output.
- A small Playwright smoke test in CI loading the SPA against a
  mocked API to catch SSR/embed regressions.

## Repository Layout (proposed)

```
.
├── cmd/k8s-auto-dash/         # main()
├── internal/
│   ├── discoverer/           # informers + tile derivation
│   ├── health/               # probe worker pool
│   ├── api/                  # net/http handlers + SSE
│   ├── config/               # CR read/write + merge
│   └── server/               # wiring
├── api/v1alpha1/             # CRD types + zz_generated_deepcopy.go
├── ui/                       # SvelteKit static build source
├── ui/build/                 # embed.FS target (gitignored)
├── deploy/
│   ├── helm/k8s-auto-dash/
│   └── manifests/install.yaml
├── docs/
└── Dockerfile
```

## Icons

A snapshot of [`selfhst/dashboard-icons`](https://github.com/selfhst/dashboard-icons)
is bundled into the binary via `embed.FS`. Predictable, offline-friendly,
no runtime egress required. The build pipeline pins a specific commit
of the icons repo and vendors it under `ui/icons/` before the Go build.
Image size impact (~20–40 MB of SVG/PNG depending on which subset we
ship) is acceptable for a homelab tool.

A "custom URL" option in the tile editor lets users point at any
external image if a service isn't in the bundled set.

## Roadmap Beyond v1

- Per-service widgets with secrets.
- Uptime history and per-tile status timeline.
- Custom wallpapers / theming beyond light/dark.
- Filter expressions (`namespace=media status!=up`).
- Multi-cluster discovery.
