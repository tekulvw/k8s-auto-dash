# k8s-auto-dash Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the SvelteKit SPA that consumes the backend's JSON+SSE API, renders the dashboard, and lets the user curate it. The build output is embedded into the Go binary via `embed.FS`.

**Architecture:** SvelteKit with `@sveltejs/adapter-static` (no SSR — the backend serves only static assets and an API). A small typed API client + SSE client wrap the backend. Svelte stores hold the merged tile/group view, theme, and edit-mode state. Drag-and-drop via `svelte-dnd-action`. Icons rendered from a bundled snapshot of `selfhst/dashboard-icons` served by the Go binary at `/icons/<slug>.png`.

**Tech Stack:**
- Node 20+, pnpm
- SvelteKit 2.x, Svelte 5
- TypeScript
- `@sveltejs/adapter-static`
- `svelte-dnd-action`
- `vitest` + `@testing-library/svelte` (component tests)
- `playwright` (smoke test)

**Prerequisite:** The backend plan (`2026-05-24-k8s-auto-dash-backend.md`) has been implemented at least through Task 16 (GET /api/tiles works). The final task in this plan modifies the Go backend to embed the build.

---

## File Structure

```
ui/
├── package.json
├── pnpm-lock.yaml
├── svelte.config.js
├── vite.config.ts
├── tsconfig.json
├── playwright.config.ts
├── src/
│   ├── app.html
│   ├── app.css                    # global tokens + dark/light vars
│   ├── lib/
│   │   ├── api/
│   │   │   ├── client.ts          # typed fetch wrapper
│   │   │   ├── types.ts           # mirrors backend ViewTile/View
│   │   │   └── events.ts          # SSE client
│   │   ├── stores/
│   │   │   ├── tiles.ts           # main merged-view store
│   │   │   ├── theme.ts           # dark/light/auto
│   │   │   ├── edit.ts            # edit-mode toggle
│   │   │   └── search.ts          # search query
│   │   ├── components/
│   │   │   ├── Header.svelte
│   │   │   ├── Group.svelte
│   │   │   ├── Tile.svelte
│   │   │   ├── TileEditor.svelte
│   │   │   ├── HiddenDrawer.svelte
│   │   │   ├── BookmarkDialog.svelte
│   │   │   ├── IconPicker.svelte
│   │   │   ├── StatusDot.svelte
│   │   │   └── Icon.svelte
│   │   └── icons/
│   │       └── index.ts           # bundled slug list (generated)
│   └── routes/
│       ├── +layout.svelte
│       ├── +layout.ts             # prerender:true (static)
│       └── +page.svelte
├── static/
│   └── favicon.png
└── tests/
    ├── unit/                      # vitest specs colocated as *.test.ts
    └── e2e/
        └── smoke.spec.ts
```

**Responsibilities:**
- `lib/api/` — only place that calls `fetch`/`EventSource`. Components and stores depend on it through typed functions.
- `lib/stores/` — single source of UI state; components subscribe.
- `lib/components/` — presentational + interactive components. No direct `fetch`.
- `routes/` — page composition.
- `lib/icons/index.ts` — generated at build time from the bundled icon set so the picker has typeahead data.

---

## Phase A: Scaffolding

### Task 1: SvelteKit project init

**Files:**
- Create everything under `ui/`

- [ ] **Step 1: Create the SvelteKit project**

```bash
mkdir -p ui && cd ui
pnpm create svelte@latest . --template skeleton --types ts --no-prettier --no-eslint
pnpm install
pnpm add -D @sveltejs/adapter-static
pnpm add -D vitest @testing-library/svelte @testing-library/jest-dom jsdom
pnpm add -D @playwright/test
pnpm add svelte-dnd-action
```

- [ ] **Step 2: Configure static adapter — `ui/svelte.config.js`**

```js
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/kit/vite';

export default {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      pages: 'build',
      assets: 'build',
      fallback: 'index.html',
      precompress: false,
      strict: true,
    }),
  },
};
```

- [ ] **Step 3: Make every route prerendered & SPA — `ui/src/routes/+layout.ts`**

```ts
export const prerender = true;
export const ssr = false;
```

- [ ] **Step 4: Add `ui/vite.config.ts`** (replace the generated one)

```ts
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    proxy: {
      '/api':   { target: 'http://localhost:8080', changeOrigin: true },
      '/icons': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
  },
});
```

- [ ] **Step 5: Add `ui/src/test-setup.ts`**

```ts
import '@testing-library/jest-dom/vitest';
```

- [ ] **Step 6: Verify dev server starts**

```bash
pnpm run dev --port 5173 &
sleep 3
curl -fsS http://localhost:5173/ > /dev/null && echo OK
kill %1
```
Expected: `OK`.

- [ ] **Step 7: Verify build produces static assets**

```bash
pnpm run build
test -f build/index.html && echo OK
```
Expected: `OK`.

- [ ] **Step 8: Commit**

```bash
cd ..
git add ui/
git commit -m "chore(ui): scaffold SvelteKit SPA with static adapter"
```

---

### Task 2: API types matching backend wire format

**Files:**
- Create: `ui/src/lib/api/types.ts`, `ui/src/lib/api/types.test.ts`

- [ ] **Step 1: Write `ui/src/lib/api/types.ts`**

Mirrors `internal/tile/tile.go` and `internal/api/state.go` exactly.

```ts
export type StatusState = 'up' | 'degraded' | 'down' | 'unknown';
export type Source = 'httproute' | 'bookmark';

export interface Status {
  state: StatusState;
  statusCode: number;
  latencyMs: number;
  checkedAt: string;
  error?: string;
}

export interface GatewayRef {
  namespace: string;
  name: string;
}

export interface K8sInfo {
  namespace: string;
  httpRouteName: string;
  gatewayRefs: GatewayRef[];
}

export interface Tile {
  id: string;
  source: Source;
  name: string;
  url: string;
  icon: string;
  description?: string;
  group: string;
  order: number;
  hidden: boolean;
  insecureSkipVerify?: boolean;
  status: Status;
  k8s?: K8sInfo | null;
}

export interface Group {
  id: string;
  name: string;
  order: number;
}

export interface View {
  groups: Group[];
  tiles: Tile[];
}

export interface ConfigPatch {
  settings?: Settings;
  groups?: GroupSpec[];
  tiles?: TileOverride[];
  bookmarks?: Bookmark[];
}

export interface Settings {
  title?: string;
  theme?: 'dark' | 'light' | 'auto';
  healthCheck?: {
    enabled?: boolean;
    intervalSeconds?: number;
    timeoutSeconds?: number;
    insecureSkipVerify?: boolean;
  };
}

export interface GroupSpec {
  id: string;
  name: string;
  order: number;
}

export interface TileOverride {
  id: string;
  hidden?: boolean;
  name?: string;
  description?: string;
  icon?: string;
  group?: string;
  order?: number;
  url?: string;
  insecureSkipVerify?: boolean;
}

export interface Bookmark {
  id: string;
  name: string;
  url: string;
  icon?: string;
  group?: string;
  order?: number;
}
```

- [ ] **Step 2: Sanity test `ui/src/lib/api/types.test.ts`**

```ts
import { describe, expect, it } from 'vitest';
import type { Tile, View } from './types';

describe('types', () => {
  it('parses a minimal tile shape', () => {
    const t: Tile = {
      id: 'a/b/c',
      source: 'httproute',
      name: 'App',
      url: 'https://app.example.com',
      icon: 'app',
      group: 'b',
      order: 0,
      hidden: false,
      status: { state: 'up', statusCode: 200, latencyMs: 1, checkedAt: '' },
    };
    expect(t.id).toBe('a/b/c');
  });

  it('parses a view envelope', () => {
    const v: View = { groups: [], tiles: [] };
    expect(v.groups).toHaveLength(0);
  });
});
```

- [ ] **Step 3: Run, see pass**

```bash
cd ui && pnpm vitest run
```
Expected: `PASS`.

- [ ] **Step 4: Commit**

```bash
cd .. && git add ui/src/lib/api/
git commit -m "feat(ui): typed API wire format"
```

---

### Task 3: API client (typed fetch wrapper)

**Files:**
- Create: `ui/src/lib/api/client.ts`, `ui/src/lib/api/client.test.ts`

- [ ] **Step 1: Write failing test `ui/src/lib/api/client.test.ts`**

```ts
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createClient } from './client';

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });

describe('createClient', () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
  });
  afterEach(() => vi.unstubAllGlobals());

  it('GET /api/tiles returns parsed view', async () => {
    fetchMock.mockResolvedValueOnce(json({ groups: [], tiles: [] }));
    const c = createClient('');
    const v = await c.getTiles();
    expect(v.tiles).toEqual([]);
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/tiles', expect.objectContaining({ method: 'GET' }));
  });

  it('PATCH /api/config sends JSON body', async () => {
    fetchMock.mockResolvedValueOnce(new Response('', { status: 200 }));
    const c = createClient('');
    await c.patchConfig({ tiles: [{ id: 'x', name: 'X' }] });

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/config');
    expect(init.method).toBe('PATCH');
    expect(init.headers['Content-Type']).toBe('application/json');
    expect(JSON.parse(init.body)).toEqual({ tiles: [{ id: 'x', name: 'X' }] });
  });

  it('addBookmark POSTs', async () => {
    fetchMock.mockResolvedValueOnce(new Response('', { status: 201 }));
    const c = createClient('');
    await c.addBookmark({ id: 'r', name: 'R', url: 'https://r' });
    expect(fetchMock.mock.calls[0][1].method).toBe('POST');
  });

  it('deleteBookmark DELETE by id', async () => {
    fetchMock.mockResolvedValueOnce(new Response('', { status: 204 }));
    const c = createClient('');
    await c.deleteBookmark('r');
    expect(fetchMock.mock.calls[0][0]).toBe('/api/config/bookmarks/r');
  });

  it('putGroups PUT array', async () => {
    fetchMock.mockResolvedValueOnce(new Response('', { status: 200 }));
    const c = createClient('');
    await c.putGroups([{ id: 'a', name: 'A', order: 0 }]);
    const init = fetchMock.mock.calls[0][1];
    expect(init.method).toBe('PUT');
    expect(JSON.parse(init.body)).toEqual([{ id: 'a', name: 'A', order: 0 }]);
  });

  it('throws on non-2xx', async () => {
    fetchMock.mockResolvedValueOnce(new Response('boom', { status: 500 }));
    const c = createClient('');
    await expect(c.getTiles()).rejects.toThrow(/500/);
  });
});
```

- [ ] **Step 2: Run, see fail**

```bash
pnpm vitest run client.test
```
Expected: import error.

- [ ] **Step 3: Write `ui/src/lib/api/client.ts`**

```ts
import type { Bookmark, ConfigPatch, GroupSpec, View } from './types';

export interface Client {
  getTiles(): Promise<View>;
  patchConfig(p: ConfigPatch): Promise<void>;
  putGroups(g: GroupSpec[]): Promise<void>;
  addBookmark(b: Bookmark): Promise<void>;
  deleteBookmark(id: string): Promise<void>;
}

async function expectOK(r: Response): Promise<Response> {
  if (!r.ok) {
    const body = await r.text().catch(() => '');
    throw new Error(`${r.status} ${r.statusText}: ${body}`);
  }
  return r;
}

export function createClient(base = ''): Client {
  const u = (p: string) => `${base}${p}`;
  const jsonHeaders = { 'Content-Type': 'application/json' };

  return {
    async getTiles() {
      const r = await fetch(u('/api/tiles'), { method: 'GET' }).then(expectOK);
      return (await r.json()) as View;
    },
    async patchConfig(p) {
      await fetch(u('/api/config'), {
        method: 'PATCH',
        headers: jsonHeaders,
        body: JSON.stringify(p),
      }).then(expectOK);
    },
    async putGroups(g) {
      await fetch(u('/api/config/groups'), {
        method: 'PUT',
        headers: jsonHeaders,
        body: JSON.stringify(g),
      }).then(expectOK);
    },
    async addBookmark(b) {
      await fetch(u('/api/config/bookmarks'), {
        method: 'POST',
        headers: jsonHeaders,
        body: JSON.stringify(b),
      }).then(expectOK);
    },
    async deleteBookmark(id) {
      await fetch(u(`/api/config/bookmarks/${encodeURIComponent(id)}`), {
        method: 'DELETE',
      }).then(expectOK);
    },
  };
}

export const api = createClient('');
```

- [ ] **Step 4: Run, see pass**

```bash
pnpm vitest run client.test
```

- [ ] **Step 5: Commit**

```bash
cd .. && git add ui/src/lib/api/
git commit -m "feat(ui): typed API client"
```

---

### Task 4: SSE client

**Files:**
- Create: `ui/src/lib/api/events.ts`, `ui/src/lib/api/events.test.ts`

- [ ] **Step 1: Write failing test `ui/src/lib/api/events.test.ts`**

```ts
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { subscribeEvents, type ServerEvent } from './events';

class FakeEventSource {
  static last: FakeEventSource | null = null;
  url: string;
  listeners: Record<string, (e: MessageEvent) => void> = {};
  closed = false;
  constructor(url: string) { this.url = url; FakeEventSource.last = this; }
  addEventListener(type: string, fn: (e: MessageEvent) => void) {
    this.listeners[type] = fn;
  }
  close() { this.closed = true; }
  emit(type: string, data: unknown) {
    this.listeners[type]?.(new MessageEvent(type, { data: JSON.stringify(data) }));
  }
}

describe('subscribeEvents', () => {
  beforeEach(() => { vi.stubGlobal('EventSource', FakeEventSource); });
  afterEach(() => { vi.unstubAllGlobals(); });

  it('emits parsed events of all known types', () => {
    const seen: ServerEvent[] = [];
    const unsub = subscribeEvents((e) => seen.push(e));
    const es = FakeEventSource.last!;

    es.emit('tile-added', { id: 'x' });
    es.emit('tile-updated', { id: 'x', fields: { name: 'X' } });
    es.emit('tile-removed', { id: 'x' });
    es.emit('status-changed', { id: 'x', status: { state: 'up' } });
    es.emit('config-changed', { source: 'kubectl' });

    expect(seen.map((e) => e.type)).toEqual([
      'tile-added', 'tile-updated', 'tile-removed', 'status-changed', 'config-changed',
    ]);
    unsub();
    expect(es.closed).toBe(true);
  });
});
```

- [ ] **Step 2: Run, see fail**

```bash
pnpm vitest run events.test
```

- [ ] **Step 3: Write `ui/src/lib/api/events.ts`**

```ts
export type ServerEvent =
  | { type: 'tile-added'; data: { tile?: unknown; id?: string } }
  | { type: 'tile-updated'; data: { id: string; fields?: Record<string, unknown> } }
  | { type: 'tile-removed'; data: { id: string } }
  | { type: 'status-changed'; data: { id: string; status: unknown } }
  | { type: 'config-changed'; data: { source: string } };

const EVENT_TYPES: ServerEvent['type'][] = [
  'tile-added', 'tile-updated', 'tile-removed', 'status-changed', 'config-changed',
];

export function subscribeEvents(
  onEvent: (e: ServerEvent) => void, base = '',
): () => void {
  const es = new EventSource(`${base}/api/events`);
  for (const t of EVENT_TYPES) {
    es.addEventListener(t, (ev: MessageEvent) => {
      let data: unknown = null;
      try { data = JSON.parse(ev.data); } catch { /* ignore */ }
      onEvent({ type: t, data } as ServerEvent);
    });
  }
  return () => es.close();
}
```

- [ ] **Step 4: Run, see pass**

```bash
pnpm vitest run events.test
```

- [ ] **Step 5: Commit**

```bash
cd .. && git add ui/src/lib/api/
git commit -m "feat(ui): SSE event client"
```

---

## Phase B: Stores

### Task 5: Tiles store with SSE integration

**Files:**
- Create: `ui/src/lib/stores/tiles.ts`, `ui/src/lib/stores/tiles.test.ts`

- [ ] **Step 1: Write failing test `ui/src/lib/stores/tiles.test.ts`**

```ts
import { describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import { createTilesStore } from './tiles';
import type { Client } from '$lib/api/client';
import type { View } from '$lib/api/types';

const sampleView: View = {
  groups: [{ id: 'media', name: 'Media', order: 0 }],
  tiles: [{
    id: 'media/jellyfin/jellyfin.example.com',
    source: 'httproute',
    name: 'Jellyfin',
    url: 'https://jellyfin.example.com',
    icon: 'jellyfin',
    group: 'media',
    order: 0,
    hidden: false,
    status: { state: 'up', statusCode: 200, latencyMs: 10, checkedAt: '' },
  }],
};

function fakeClient(view: View): Client {
  return {
    getTiles: vi.fn().mockResolvedValue(view),
    patchConfig: vi.fn().mockResolvedValue(undefined),
    putGroups: vi.fn().mockResolvedValue(undefined),
    addBookmark: vi.fn().mockResolvedValue(undefined),
    deleteBookmark: vi.fn().mockResolvedValue(undefined),
  };
}

describe('tiles store', () => {
  it('load() populates view', async () => {
    const c = fakeClient(sampleView);
    const s = createTilesStore(c);
    await s.load();
    expect(get(s).tiles).toHaveLength(1);
    expect(get(s).groups[0].name).toBe('Media');
  });

  it('applyEvent status-changed updates a single tile', async () => {
    const c = fakeClient(sampleView);
    const s = createTilesStore(c);
    await s.load();
    s.applyEvent({
      type: 'status-changed',
      data: {
        id: 'media/jellyfin/jellyfin.example.com',
        status: { state: 'down', statusCode: 0, latencyMs: 0, checkedAt: '', error: 'x' },
      },
    });
    expect(get(s).tiles[0].status.state).toBe('down');
  });

  it('applyEvent config-changed triggers reload', async () => {
    const c = fakeClient(sampleView);
    const s = createTilesStore(c);
    await s.load();
    s.applyEvent({ type: 'config-changed', data: { source: 'kubectl' } });
    // microtask flush
    await Promise.resolve();
    await Promise.resolve();
    expect(c.getTiles).toHaveBeenCalledTimes(2);
  });

  it('applyEvent tile-updated triggers reload (full refetch keeps merge simple)', async () => {
    const c = fakeClient(sampleView);
    const s = createTilesStore(c);
    await s.load();
    s.applyEvent({ type: 'tile-updated', data: { id: 'x' } });
    await Promise.resolve();
    await Promise.resolve();
    expect(c.getTiles).toHaveBeenCalledTimes(2);
  });
});
```

- [ ] **Step 2: Run, see fail**

```bash
cd ui && pnpm vitest run tiles.test
```

- [ ] **Step 3: Write `ui/src/lib/stores/tiles.ts`**

```ts
import { writable, type Readable } from 'svelte/store';
import type { Client } from '$lib/api/client';
import type { ServerEvent } from '$lib/api/events';
import type { Status, View } from '$lib/api/types';

export interface TilesStore extends Readable<View> {
  load(): Promise<void>;
  applyEvent(e: ServerEvent): void;
}

const EMPTY: View = { groups: [], tiles: [] };

export function createTilesStore(client: Client): TilesStore {
  const { subscribe, set, update } = writable<View>(EMPTY);

  let pendingReload: Promise<void> | null = null;
  const reload = () => {
    if (pendingReload) return pendingReload;
    pendingReload = client.getTiles().then((v) => {
      set(v);
    }).catch((err) => {
      console.error('tiles reload failed', err);
    }).finally(() => {
      pendingReload = null;
    });
    return pendingReload;
  };

  return {
    subscribe,
    load: reload,
    applyEvent(e) {
      switch (e.type) {
        case 'status-changed': {
          const { id, status } = e.data as { id: string; status: Status };
          update((v) => ({
            ...v,
            tiles: v.tiles.map((t) => t.id === id ? { ...t, status } : t),
          }));
          return;
        }
        case 'tile-added':
        case 'tile-updated':
        case 'tile-removed':
        case 'config-changed':
          // Simplification: full refetch on any structural change.
          // Cheap because /api/tiles is in-memory on the server.
          void reload();
          return;
      }
    },
  };
}
```

- [ ] **Step 4: Run, see pass**

```bash
pnpm vitest run tiles.test
```

- [ ] **Step 5: Commit**

```bash
cd .. && git add ui/src/lib/stores/
git commit -m "feat(ui): tiles store with SSE event handling"
```

---

### Task 6: Theme store (dark/light/auto, localStorage-backed)

**Files:**
- Create: `ui/src/lib/stores/theme.ts`, `ui/src/lib/stores/theme.test.ts`

- [ ] **Step 1: Write failing test `ui/src/lib/stores/theme.test.ts`**

```ts
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { get } from 'svelte/store';
import { createThemeStore } from './theme';

describe('theme store', () => {
  beforeEach(() => { localStorage.clear(); document.documentElement.removeAttribute('data-theme'); });
  afterEach(() => { localStorage.clear(); });

  it('defaults to dark when nothing persisted', () => {
    const s = createThemeStore();
    expect(get(s)).toBe('dark');
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
  });

  it('persists changes to localStorage', () => {
    const s = createThemeStore();
    s.set('light');
    expect(localStorage.getItem('theme')).toBe('light');
    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
  });

  it('reads existing localStorage value on init', () => {
    localStorage.setItem('theme', 'light');
    const s = createThemeStore();
    expect(get(s)).toBe('light');
  });
});
```

- [ ] **Step 2: Run, see fail**

```bash
pnpm vitest run theme.test
```

- [ ] **Step 3: Write `ui/src/lib/stores/theme.ts`**

```ts
import { writable } from 'svelte/store';
import { browser } from '$app/environment';

export type Theme = 'dark' | 'light' | 'auto';
const VALID: Theme[] = ['dark', 'light', 'auto'];

function resolveAuto(): 'dark' | 'light' {
  if (!browser) return 'dark';
  return window.matchMedia?.('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

function apply(t: Theme) {
  if (!browser) return;
  const effective = t === 'auto' ? resolveAuto() : t;
  document.documentElement.setAttribute('data-theme', effective);
}

export function createThemeStore() {
  const initial: Theme = (() => {
    if (!browser) return 'dark';
    const v = localStorage.getItem('theme') as Theme | null;
    return v && VALID.includes(v) ? v : 'dark';
  })();
  apply(initial);

  const { subscribe, set } = writable<Theme>(initial);
  return {
    subscribe,
    set(t: Theme) {
      if (browser) localStorage.setItem('theme', t);
      apply(t);
      set(t);
    },
  };
}

export const theme = createThemeStore();
```

- [ ] **Step 4: Run, see pass**

```bash
pnpm vitest run theme.test
```

- [ ] **Step 5: Commit**

```bash
cd .. && git add ui/src/lib/stores/
git commit -m "feat(ui): theme store with localStorage persistence"
```

---

### Task 7: Edit-mode and search stores

**Files:**
- Create: `ui/src/lib/stores/edit.ts`, `ui/src/lib/stores/search.ts`

These are trivial; no tests needed beyond visual confirmation in later component tests.

- [ ] **Step 1: Write `ui/src/lib/stores/edit.ts`**

```ts
import { writable } from 'svelte/store';
export const editMode = writable<boolean>(false);
```

- [ ] **Step 2: Write `ui/src/lib/stores/search.ts`**

```ts
import { writable } from 'svelte/store';
export const searchQuery = writable<string>('');
```

- [ ] **Step 3: Commit**

```bash
git add ui/src/lib/stores/
git commit -m "feat(ui): edit-mode and search stores"
```

---

## Phase C: Components

### Task 8: Icon and StatusDot atoms

**Files:**
- Create: `ui/src/lib/components/Icon.svelte`, `ui/src/lib/components/StatusDot.svelte`
- Create: `ui/src/lib/components/Icon.test.ts`, `ui/src/lib/components/StatusDot.test.ts`

- [ ] **Step 1: Write `ui/src/lib/components/Icon.svelte`**

```svelte
<script lang="ts">
  // Renders /icons/<slug>.png (served by the Go binary). If `slug` is a
  // full http(s) URL it is used directly. Falls back to a globe SVG on
  // load error.
  export let slug: string = '';
  export let alt: string = '';
  let errored = false;
  $: src = slug.startsWith('http') ? slug : `/icons/${slug}.png`;
</script>

{#if !slug || errored}
  <svg class="icon icon-fallback" viewBox="0 0 24 24" aria-label={alt}>
    <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <path d="M2 12h20M12 2c3 3 3 17 0 20M12 2c-3 3-3 17 0 20" fill="none" stroke="currentColor" stroke-width="1.5"/>
  </svg>
{:else}
  <img class="icon" {src} alt={alt} on:error={() => (errored = true)} />
{/if}

<style>
  .icon { width: 40px; height: 40px; border-radius: 8px; object-fit: contain; }
  .icon-fallback { color: var(--fg-muted); }
</style>
```

- [ ] **Step 2: Write `ui/src/lib/components/Icon.test.ts`**

```ts
import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import Icon from './Icon.svelte';

describe('Icon', () => {
  it('uses /icons/<slug>.png for slugs', () => {
    const { container } = render(Icon, { props: { slug: 'jellyfin', alt: 'Jellyfin' } });
    const img = container.querySelector('img')!;
    expect(img.getAttribute('src')).toBe('/icons/jellyfin.png');
  });

  it('uses raw URL when given http', () => {
    const { container } = render(Icon, { props: { slug: 'https://example.com/x.png' } });
    expect(container.querySelector('img')!.getAttribute('src')).toBe('https://example.com/x.png');
  });

  it('renders fallback when no slug', () => {
    const { container } = render(Icon, { props: { slug: '' } });
    expect(container.querySelector('svg')).toBeTruthy();
  });
});
```

- [ ] **Step 3: Write `ui/src/lib/components/StatusDot.svelte`**

```svelte
<script lang="ts">
  import type { StatusState } from '$lib/api/types';
  export let state: StatusState = 'unknown';
  export let title: string = '';
</script>

<span class="dot dot-{state}" title={title} aria-label={state}></span>

<style>
  .dot {
    display: inline-block;
    width: 10px; height: 10px;
    border-radius: 50%;
    background: var(--fg-muted);
  }
  .dot-up       { background: #3fb950; }
  .dot-degraded { background: #d29922; }
  .dot-down     { background: #f85149; }
  .dot-unknown  { background: #6e7681; }
</style>
```

- [ ] **Step 4: Write `ui/src/lib/components/StatusDot.test.ts`**

```ts
import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import StatusDot from './StatusDot.svelte';

describe('StatusDot', () => {
  it.each(['up', 'degraded', 'down', 'unknown'] as const)('renders %s', (state) => {
    const { container } = render(StatusDot, { props: { state } });
    expect(container.querySelector(`.dot-${state}`)).toBeTruthy();
  });
});
```

- [ ] **Step 5: Run, see pass**

```bash
pnpm vitest run
```

- [ ] **Step 6: Commit**

```bash
cd .. && git add ui/src/lib/components/
git commit -m "feat(ui): Icon and StatusDot atoms"
```

---

### Task 9: Tile component (wide info card)

**Files:**
- Create: `ui/src/lib/components/Tile.svelte`, `ui/src/lib/components/Tile.test.ts`

- [ ] **Step 1: Write failing test `ui/src/lib/components/Tile.test.ts`**

```ts
import { render, fireEvent } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import Tile from './Tile.svelte';
import type { Tile as TTile } from '$lib/api/types';

const sample: TTile = {
  id: 'media/jellyfin/jellyfin.example.com',
  source: 'httproute',
  name: 'Jellyfin',
  url: 'https://jellyfin.example.com',
  icon: 'jellyfin',
  group: 'media',
  order: 0,
  hidden: false,
  status: { state: 'up', statusCode: 200, latencyMs: 12, checkedAt: '' },
};

describe('Tile', () => {
  it('renders name and hostname', () => {
    const { getByText } = render(Tile, { props: { tile: sample, editing: false } });
    expect(getByText('Jellyfin')).toBeInTheDocument();
    expect(getByText('jellyfin.example.com')).toBeInTheDocument();
  });

  it('view mode: clicking opens the URL', () => {
    const open = vi.fn();
    vi.stubGlobal('open', open);
    const { container } = render(Tile, { props: { tile: sample, editing: false } });
    fireEvent.click(container.querySelector('.tile')!);
    expect(open).toHaveBeenCalledWith('https://jellyfin.example.com', '_blank');
    vi.unstubAllGlobals();
  });

  it('edit mode: clicking does NOT navigate, emits edit event on menu', async () => {
    const { component, container, getByLabelText } = render(Tile, {
      props: { tile: sample, editing: true },
    });
    const onEdit = vi.fn();
    component.$on('edit', onEdit);
    await fireEvent.click(getByLabelText('Edit tile'));
    expect(onEdit).toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run, see fail**

```bash
pnpm vitest run Tile.test
```

- [ ] **Step 3: Write `ui/src/lib/components/Tile.svelte`**

```svelte
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import Icon from './Icon.svelte';
  import StatusDot from './StatusDot.svelte';
  import type { Tile } from '$lib/api/types';

  export let tile: Tile;
  export let editing: boolean = false;

  const dispatch = createEventDispatcher<{
    edit: { id: string };
    hide: { id: string };
  }>();

  $: hostname = (() => {
    try { return new URL(tile.url).hostname; } catch { return tile.url; }
  })();

  function handleClick() {
    if (!editing && tile.url) window.open(tile.url, '_blank');
  }
</script>

<div
  class="tile"
  class:editing
  class:hidden={tile.hidden}
  role="button"
  tabindex="0"
  on:click={handleClick}
  on:keydown={(e) => e.key === 'Enter' && handleClick()}
>
  <Icon slug={tile.icon} alt={tile.name} />
  <div class="body">
    <div class="name">{tile.name}</div>
    <div class="host">{hostname}</div>
    {#if tile.description}<div class="desc">{tile.description}</div>{/if}
  </div>
  <StatusDot state={tile.status.state} title={`${tile.status.state}${tile.status.statusCode ? ' · ' + tile.status.statusCode : ''}`} />

  {#if editing}
    <div class="actions">
      <button aria-label="Edit tile" on:click|stopPropagation={() => dispatch('edit', { id: tile.id })}>✎</button>
      <button aria-label="Hide tile" on:click|stopPropagation={() => dispatch('hide', { id: tile.id })}>×</button>
    </div>
  {/if}
</div>

<style>
  .tile {
    display: flex; gap: 12px; align-items: center;
    padding: 12px;
    background: var(--bg-card);
    border-radius: 10px;
    cursor: pointer;
    transition: background 0.15s;
  }
  .tile:hover { background: var(--bg-card-hover); }
  .tile.hidden { opacity: 0.4; }
  .body { flex: 1; min-width: 0; }
  .name { font-weight: 600; color: var(--fg); }
  .host { font-size: 12px; color: var(--fg-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .desc { font-size: 12px; color: var(--fg-muted); margin-top: 2px; }
  .actions { display: flex; gap: 4px; }
  .actions button {
    background: none; border: 1px solid var(--border);
    color: var(--fg); cursor: pointer; padding: 2px 6px; border-radius: 4px;
  }
</style>
```

- [ ] **Step 4: Run, see pass**

```bash
pnpm vitest run Tile.test
```

- [ ] **Step 5: Commit**

```bash
cd .. && git add ui/src/lib/components/
git commit -m "feat(ui): Tile component (wide info card)"
```

---

### Task 10: Group component

**Files:**
- Create: `ui/src/lib/components/Group.svelte`, `ui/src/lib/components/Group.test.ts`

- [ ] **Step 1: Write failing test `ui/src/lib/components/Group.test.ts`**

```ts
import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import Group from './Group.svelte';
import type { Tile as TTile } from '$lib/api/types';

const tile = (id: string, name: string): TTile => ({
  id, source: 'httproute', name, url: `https://${id}`, icon: '',
  group: 'g', order: 0, hidden: false,
  status: { state: 'up', statusCode: 200, latencyMs: 0, checkedAt: '' },
});

describe('Group', () => {
  it('renders the group name and tiles', () => {
    const { getByText } = render(Group, {
      props: { group: { id: 'g', name: 'My Group', order: 0 }, tiles: [tile('a','A'), tile('b','B')], editing: false },
    });
    expect(getByText('My Group')).toBeInTheDocument();
    expect(getByText('A')).toBeInTheDocument();
    expect(getByText('B')).toBeInTheDocument();
  });

  it('hides hidden tiles when not editing', () => {
    const hidden = { ...tile('h', 'Hidden'), hidden: true };
    const { queryByText } = render(Group, {
      props: { group: { id: 'g', name: 'G', order: 0 }, tiles: [hidden], editing: false },
    });
    expect(queryByText('Hidden')).toBeNull();
  });

  it('shows hidden tiles when editing', () => {
    const hidden = { ...tile('h', 'Hidden'), hidden: true };
    const { getByText } = render(Group, {
      props: { group: { id: 'g', name: 'G', order: 0 }, tiles: [hidden], editing: true },
    });
    expect(getByText('Hidden')).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run, see fail**

```bash
pnpm vitest run Group.test
```

- [ ] **Step 3: Write `ui/src/lib/components/Group.svelte`**

```svelte
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import Tile from './Tile.svelte';
  import type { Group, Tile as TTile } from '$lib/api/types';

  export let group: Group;
  export let tiles: TTile[];
  export let editing: boolean = false;

  const dispatch = createEventDispatcher<{
    editTile: { id: string };
    hideTile: { id: string };
    addBookmark: { group: string };
  }>();

  $: visible = editing ? tiles : tiles.filter((t) => !t.hidden);
  $: sorted = [...visible].sort((a, b) => a.order - b.order || a.name.localeCompare(b.name));
</script>

<section class="group">
  <header>
    <h2>{group.name}</h2>
    {#if editing}
      <button class="add" on:click={() => dispatch('addBookmark', { group: group.id })}>
        + bookmark
      </button>
    {/if}
  </header>

  <div class="grid">
    {#each sorted as t (t.id)}
      <Tile
        tile={t}
        {editing}
        on:edit={(e) => dispatch('editTile', e.detail)}
        on:hide={(e) => dispatch('hideTile', e.detail)}
      />
    {/each}
  </div>
</section>

<style>
  .group { margin-bottom: 32px; }
  header { display: flex; align-items: baseline; gap: 12px; margin-bottom: 12px; }
  h2 { margin: 0; font-size: 14px; text-transform: uppercase; letter-spacing: 0.5px; color: var(--fg-muted); }
  .add { background: none; border: 1px dashed var(--border); color: var(--fg-muted); padding: 2px 8px; border-radius: 4px; font-size: 12px; cursor: pointer; }
  .grid {
    display: grid; gap: 12px;
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  }
</style>
```

- [ ] **Step 4: Run, see pass**

```bash
pnpm vitest run Group.test
```

- [ ] **Step 5: Commit**

```bash
cd .. && git add ui/src/lib/components/
git commit -m "feat(ui): Group component"
```

---

### Task 11: Header (title, search, theme, edit toggle)

**Files:**
- Create: `ui/src/lib/components/Header.svelte`, `ui/src/lib/components/Header.test.ts`

- [ ] **Step 1: Write failing test `ui/src/lib/components/Header.test.ts`**

```ts
import { render, fireEvent } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import { get } from 'svelte/store';
import Header from './Header.svelte';
import { editMode } from '$lib/stores/edit';
import { searchQuery } from '$lib/stores/search';

describe('Header', () => {
  it('toggles edit mode when pencil clicked', async () => {
    editMode.set(false);
    const { getByLabelText } = render(Header, { props: { title: 'Home' } });
    await fireEvent.click(getByLabelText('Toggle edit mode'));
    expect(get(editMode)).toBe(true);
  });

  it('updates searchQuery on input', async () => {
    searchQuery.set('');
    const { getByPlaceholderText } = render(Header, { props: { title: 'Home' } });
    await fireEvent.input(getByPlaceholderText('Search'), { target: { value: 'jelly' } });
    expect(get(searchQuery)).toBe('jelly');
  });
});
```

- [ ] **Step 2: Run, see fail**

```bash
cd ui && pnpm vitest run Header.test
```

- [ ] **Step 3: Write `ui/src/lib/components/Header.svelte`**

```svelte
<script lang="ts">
  import { editMode } from '$lib/stores/edit';
  import { searchQuery } from '$lib/stores/search';
  import { theme, type Theme } from '$lib/stores/theme';

  export let title: string = 'Dashboard';

  function cycleTheme() {
    const order: Theme[] = ['dark', 'light', 'auto'];
    const next = order[(order.indexOf($theme) + 1) % order.length];
    theme.set(next);
  }
</script>

<header class="header">
  <h1>{title}</h1>
  <input
    class="search"
    type="search"
    placeholder="Search"
    bind:value={$searchQuery}
    aria-label="Search tiles"
  />
  <button class="icon-btn" aria-label="Cycle theme" on:click={cycleTheme}>
    {#if $theme === 'dark'}🌙{:else if $theme === 'light'}☀{:else}◐{/if}
  </button>
  <button
    class="icon-btn"
    aria-label="Toggle edit mode"
    class:active={$editMode}
    on:click={() => editMode.update((v) => !v)}
  >✎</button>
</header>

<style>
  .header {
    display: flex; align-items: center; gap: 12px;
    padding: 16px 24px;
    border-bottom: 1px solid var(--border);
  }
  h1 { margin: 0; font-size: 18px; flex: 0; }
  .search {
    flex: 1; max-width: 400px;
    padding: 6px 10px; border-radius: 6px;
    background: var(--bg-card); border: 1px solid var(--border);
    color: var(--fg);
  }
  .icon-btn {
    background: none; border: 1px solid var(--border);
    color: var(--fg); cursor: pointer;
    padding: 4px 10px; border-radius: 6px;
  }
  .icon-btn.active { background: var(--accent); color: var(--accent-fg); }
</style>
```

- [ ] **Step 4: Run, see pass**

```bash
pnpm vitest run Header.test
```

- [ ] **Step 5: Commit**

```bash
cd .. && git add ui/src/lib/components/
git commit -m "feat(ui): Header with search, theme cycle, edit toggle"
```

---

### Task 12: IconPicker with typeahead

**Files:**
- Create: `ui/src/lib/icons/index.ts` (placeholder list; real one generated in Task 18)
- Create: `ui/src/lib/components/IconPicker.svelte`, `ui/src/lib/components/IconPicker.test.ts`

- [ ] **Step 1: Write `ui/src/lib/icons/index.ts` (placeholder)**

```ts
// This file is regenerated by `make icons` from the bundled
// dashboard-icons snapshot. The placeholder list lets unit tests run
// before the icon generator runs.
export const ICON_SLUGS: readonly string[] = [
  'jellyfin', 'sonarr', 'radarr', 'prowlarr', 'plex',
  'grafana', 'prometheus', 'argocd', 'vault', 'gitea',
  'homepage', 'pihole', 'nextcloud', 'ubiquiti',
];
```

- [ ] **Step 2: Write failing test `ui/src/lib/components/IconPicker.test.ts`**

```ts
import { render, fireEvent } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import IconPicker from './IconPicker.svelte';

describe('IconPicker', () => {
  it('filters suggestions on input', async () => {
    const { getByPlaceholderText, getAllByRole } = render(IconPicker, {
      props: { value: '' },
    });
    const input = getByPlaceholderText('icon slug or URL') as HTMLInputElement;
    await fireEvent.input(input, { target: { value: 'jel' } });
    const opts = getAllByRole('option').map((o) => o.textContent?.trim());
    expect(opts).toContain('jellyfin');
  });

  it('emits change when suggestion clicked', async () => {
    const { component, getByPlaceholderText, getByText } = render(IconPicker, {
      props: { value: '' },
    });
    const onChange = vi.fn();
    component.$on('change', onChange);

    await fireEvent.input(getByPlaceholderText('icon slug or URL'), {
      target: { value: 'jelly' },
    });
    await fireEvent.click(getByText('jellyfin'));
    expect(onChange).toHaveBeenCalled();
    expect(onChange.mock.calls[0][0].detail).toEqual({ value: 'jellyfin' });
  });

  it('accepts URL input verbatim', async () => {
    const { component, getByPlaceholderText } = render(IconPicker, { props: { value: '' } });
    const onChange = vi.fn();
    component.$on('change', onChange);
    await fireEvent.input(getByPlaceholderText('icon slug or URL'), {
      target: { value: 'https://example.com/x.png' },
    });
    expect(onChange.mock.calls.at(-1)?.[0].detail).toEqual({ value: 'https://example.com/x.png' });
  });
});
```

- [ ] **Step 3: Write `ui/src/lib/components/IconPicker.svelte`**

```svelte
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import Icon from './Icon.svelte';
  import { ICON_SLUGS } from '$lib/icons';

  export let value: string = '';

  const dispatch = createEventDispatcher<{ change: { value: string } }>();

  $: query = value.toLowerCase();
  $: suggestions = !query || value.startsWith('http')
    ? []
    : ICON_SLUGS.filter((s) => s.includes(query)).slice(0, 8);

  function handleInput(e: Event) {
    const v = (e.target as HTMLInputElement).value;
    value = v;
    dispatch('change', { value: v });
  }

  function pick(slug: string) {
    value = slug;
    dispatch('change', { value: slug });
  }
</script>

<div class="picker">
  <div class="row">
    <Icon slug={value} alt="preview" />
    <input
      type="text"
      placeholder="icon slug or URL"
      value={value}
      on:input={handleInput}
    />
  </div>
  {#if suggestions.length > 0}
    <ul class="suggestions" role="listbox">
      {#each suggestions as s}
        <li role="option" on:click={() => pick(s)}>
          <Icon slug={s} alt={s} />
          <span>{s}</span>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .picker { display: flex; flex-direction: column; gap: 8px; }
  .row { display: flex; gap: 8px; align-items: center; }
  .row input { flex: 1; padding: 6px 8px; border-radius: 4px; border: 1px solid var(--border); background: var(--bg-card); color: var(--fg); }
  .suggestions { list-style: none; padding: 4px; margin: 0; border: 1px solid var(--border); border-radius: 6px; max-height: 220px; overflow-y: auto; }
  .suggestions li { display: flex; gap: 8px; align-items: center; padding: 4px 6px; cursor: pointer; border-radius: 4px; }
  .suggestions li:hover { background: var(--bg-card-hover); }
</style>
```

- [ ] **Step 4: Run, see pass**

```bash
pnpm vitest run IconPicker.test
```

- [ ] **Step 5: Commit**

```bash
cd .. && git add ui/src/lib/icons/ ui/src/lib/components/
git commit -m "feat(ui): IconPicker with typeahead"
```

---

### Task 13: TileEditor side panel

**Files:**
- Create: `ui/src/lib/components/TileEditor.svelte`, `ui/src/lib/components/TileEditor.test.ts`

- [ ] **Step 1: Write failing test `ui/src/lib/components/TileEditor.test.ts`**

```ts
import { render, fireEvent } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import TileEditor from './TileEditor.svelte';
import type { Tile, Group } from '$lib/api/types';

const t: Tile = {
  id: 'media/jelly/jellyfin.example.com',
  source: 'httproute',
  name: 'Jellyfin',
  url: 'https://jellyfin.example.com',
  icon: 'jellyfin',
  description: '',
  group: 'media',
  order: 0,
  hidden: false,
  status: { state: 'up', statusCode: 200, latencyMs: 1, checkedAt: '' },
  k8s: { namespace: 'media', httpRouteName: 'jelly', gatewayRefs: [{ namespace: 'gw', name: 'ext' }] },
};
const groups: Group[] = [{ id: 'media', name: 'Media', order: 0 }];

describe('TileEditor', () => {
  it('emits save with override payload', async () => {
    const { component, getByLabelText, getByText } = render(TileEditor, {
      props: { tile: t, groups },
    });
    const onSave = vi.fn();
    component.$on('save', onSave);

    await fireEvent.input(getByLabelText('Name'), { target: { value: 'Jelly' } });
    await fireEvent.click(getByText('Save'));

    expect(onSave).toHaveBeenCalled();
    expect(onSave.mock.calls[0][0].detail).toMatchObject({
      id: t.id, name: 'Jelly',
    });
  });

  it('emits reset', async () => {
    const { component, getByText } = render(TileEditor, { props: { tile: t, groups } });
    const onReset = vi.fn();
    component.$on('reset', onReset);
    await fireEvent.click(getByText('Reset to auto'));
    expect(onReset).toHaveBeenCalledWith(expect.objectContaining({ detail: { id: t.id } }));
  });

  it('shows k8s info read-only', () => {
    const { getByText } = render(TileEditor, { props: { tile: t, groups } });
    expect(getByText('media')).toBeInTheDocument();
    expect(getByText('jelly')).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run, see fail**

```bash
pnpm vitest run TileEditor.test
```

- [ ] **Step 3: Write `ui/src/lib/components/TileEditor.svelte`**

```svelte
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import IconPicker from './IconPicker.svelte';
  import type { Group, Tile, TileOverride } from '$lib/api/types';

  export let tile: Tile;
  export let groups: Group[];

  const dispatch = createEventDispatcher<{
    save: TileOverride;
    reset: { id: string };
    close: void;
  }>();

  let name = tile.name;
  let url = tile.url;
  let icon = tile.icon;
  let description = tile.description ?? '';
  let group = tile.group;
  let insecure = !!tile.insecureSkipVerify;

  function save() {
    dispatch('save', {
      id: tile.id,
      name, url, icon, description, group,
      insecureSkipVerify: insecure,
    });
  }
</script>

<aside class="editor">
  <header>
    <h3>Edit tile</h3>
    <button aria-label="Close" on:click={() => dispatch('close')}>×</button>
  </header>

  <label>Name<input bind:value={name} aria-label="Name" /></label>
  <label>URL<input bind:value={url} aria-label="URL" /></label>
  <label>Description<input bind:value={description} aria-label="Description" /></label>
  <label>Group
    <select bind:value={group} aria-label="Group">
      {#each groups as g}<option value={g.id}>{g.name}</option>{/each}
    </select>
  </label>
  <div>
    <div class="label">Icon</div>
    <IconPicker value={icon} on:change={(e) => (icon = e.detail.value)} />
  </div>
  <label class="check">
    <input type="checkbox" bind:checked={insecure} /> Insecure TLS (skip cert verification)
  </label>

  {#if tile.k8s}
    <div class="k8s">
      <div class="label">Kubernetes</div>
      <dl>
        <dt>Namespace</dt><dd>{tile.k8s.namespace}</dd>
        <dt>HTTPRoute</dt><dd>{tile.k8s.httpRouteName}</dd>
        <dt>Gateways</dt>
        <dd>
          {#each tile.k8s.gatewayRefs as g}
            <span>{g.namespace}/{g.name}</span>
          {/each}
        </dd>
      </dl>
    </div>
  {/if}

  <footer>
    <button on:click={() => dispatch('reset', { id: tile.id })}>Reset to auto</button>
    <button class="primary" on:click={save}>Save</button>
  </footer>
</aside>

<style>
  .editor {
    position: fixed; right: 0; top: 0; bottom: 0;
    width: 360px; background: var(--bg);
    border-left: 1px solid var(--border);
    padding: 16px; display: flex; flex-direction: column; gap: 12px;
    overflow-y: auto;
  }
  header, footer { display: flex; justify-content: space-between; gap: 8px; }
  h3 { margin: 0; }
  label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--fg-muted); }
  label input, label select {
    padding: 6px 8px; border-radius: 4px;
    border: 1px solid var(--border); background: var(--bg-card); color: var(--fg);
  }
  .check { flex-direction: row; align-items: center; }
  .label { font-size: 12px; color: var(--fg-muted); margin-bottom: 4px; }
  .k8s dl { display: grid; grid-template-columns: max-content 1fr; gap: 4px 12px; font-size: 13px; }
  .k8s dt { color: var(--fg-muted); }
  button.primary { background: var(--accent); color: var(--accent-fg); border: none; padding: 6px 12px; border-radius: 4px; cursor: pointer; }
  button { background: none; border: 1px solid var(--border); color: var(--fg); padding: 6px 12px; border-radius: 4px; cursor: pointer; }
</style>
```

- [ ] **Step 4: Run, see pass**

```bash
pnpm vitest run TileEditor.test
```

- [ ] **Step 5: Commit**

```bash
cd .. && git add ui/src/lib/components/
git commit -m "feat(ui): TileEditor side panel"
```

---

### Task 14: HiddenDrawer and BookmarkDialog

**Files:**
- Create: `ui/src/lib/components/HiddenDrawer.svelte`, `ui/src/lib/components/HiddenDrawer.test.ts`
- Create: `ui/src/lib/components/BookmarkDialog.svelte`, `ui/src/lib/components/BookmarkDialog.test.ts`

- [ ] **Step 1: Write `ui/src/lib/components/HiddenDrawer.test.ts`**

```ts
import { render, fireEvent } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import HiddenDrawer from './HiddenDrawer.svelte';
import type { Tile } from '$lib/api/types';

const hidden = (id: string, name: string): Tile => ({
  id, source: 'httproute', name, url: 'https://x', icon: '', group: 'g',
  order: 0, hidden: true,
  status: { state: 'up', statusCode: 200, latencyMs: 0, checkedAt: '' },
});

describe('HiddenDrawer', () => {
  it('lists hidden tiles and emits restore', async () => {
    const { component, getByText } = render(HiddenDrawer, {
      props: { tiles: [hidden('a', 'A'), hidden('b', 'B')] },
    });
    const onRestore = vi.fn();
    component.$on('restore', onRestore);

    await fireEvent.click(getByText('A'));
    expect(onRestore).toHaveBeenCalledWith(expect.objectContaining({ detail: { id: 'a' } }));
  });

  it('renders empty state', () => {
    const { getByText } = render(HiddenDrawer, { props: { tiles: [] } });
    expect(getByText(/No hidden tiles/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Write `ui/src/lib/components/HiddenDrawer.svelte`**

```svelte
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import Icon from './Icon.svelte';
  import type { Tile } from '$lib/api/types';

  export let tiles: Tile[];
  const dispatch = createEventDispatcher<{ restore: { id: string } }>();
</script>

<aside class="drawer">
  <h3>Hidden tiles</h3>
  {#if tiles.length === 0}
    <p class="muted">No hidden tiles.</p>
  {:else}
    <ul>
      {#each tiles as t (t.id)}
        <li on:click={() => dispatch('restore', { id: t.id })}>
          <Icon slug={t.icon} alt={t.name} />
          <span>{t.name}</span>
        </li>
      {/each}
    </ul>
  {/if}
</aside>

<style>
  .drawer { padding: 16px; }
  h3 { margin-top: 0; }
  ul { list-style: none; padding: 0; margin: 0; }
  li { display: flex; gap: 8px; align-items: center; padding: 6px; border-radius: 4px; cursor: pointer; }
  li:hover { background: var(--bg-card-hover); }
  .muted { color: var(--fg-muted); }
</style>
```

- [ ] **Step 3: Write `ui/src/lib/components/BookmarkDialog.test.ts`**

```ts
import { render, fireEvent } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import BookmarkDialog from './BookmarkDialog.svelte';

describe('BookmarkDialog', () => {
  it('emits create with payload', async () => {
    const { component, getByLabelText, getByText } = render(BookmarkDialog, {
      props: { group: 'infra' },
    });
    const onCreate = vi.fn();
    component.$on('create', onCreate);

    await fireEvent.input(getByLabelText('Name'), { target: { value: 'Router' } });
    await fireEvent.input(getByLabelText('URL'), { target: { value: 'https://r' } });
    await fireEvent.click(getByText('Add'));

    expect(onCreate.mock.calls[0][0].detail).toMatchObject({
      name: 'Router', url: 'https://r', group: 'infra',
    });
    expect(onCreate.mock.calls[0][0].detail.id).toBeTruthy();
  });
});
```

- [ ] **Step 4: Write `ui/src/lib/components/BookmarkDialog.svelte`**

```svelte
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import IconPicker from './IconPicker.svelte';
  import type { Bookmark } from '$lib/api/types';

  export let group: string;

  const dispatch = createEventDispatcher<{ create: Bookmark; close: void }>();

  let name = '';
  let url = '';
  let icon = '';

  function slugify(s: string): string {
    return s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
  }

  function submit() {
    if (!name || !url) return;
    dispatch('create', {
      id: slugify(name) + '-' + Math.random().toString(36).slice(2, 6),
      name, url, icon, group,
    });
  }
</script>

<div class="dialog" role="dialog">
  <h3>Add bookmark to <em>{group}</em></h3>
  <label>Name<input bind:value={name} aria-label="Name" /></label>
  <label>URL<input bind:value={url} aria-label="URL" /></label>
  <div>
    <div class="label">Icon</div>
    <IconPicker value={icon} on:change={(e) => (icon = e.detail.value)} />
  </div>
  <footer>
    <button on:click={() => dispatch('close')}>Cancel</button>
    <button class="primary" on:click={submit}>Add</button>
  </footer>
</div>

<style>
  .dialog {
    position: fixed; inset: 0; margin: auto;
    width: 380px; height: fit-content;
    padding: 16px; background: var(--bg);
    border: 1px solid var(--border); border-radius: 8px;
    display: flex; flex-direction: column; gap: 10px;
    z-index: 100;
  }
  label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--fg-muted); }
  label input { padding: 6px 8px; border-radius: 4px; border: 1px solid var(--border); background: var(--bg-card); color: var(--fg); }
  .label { font-size: 12px; color: var(--fg-muted); }
  footer { display: flex; justify-content: flex-end; gap: 8px; }
  button { background: none; border: 1px solid var(--border); color: var(--fg); padding: 6px 12px; border-radius: 4px; cursor: pointer; }
  button.primary { background: var(--accent); color: var(--accent-fg); border: none; }
</style>
```

- [ ] **Step 5: Run all tests, see pass**

```bash
pnpm vitest run
```

- [ ] **Step 6: Commit**

```bash
cd .. && git add ui/src/lib/components/
git commit -m "feat(ui): HiddenDrawer and BookmarkDialog"
```

---

## Phase D: Page wiring & polish

### Task 15: Global CSS theme tokens and layout

**Files:**
- Create: `ui/src/app.css`
- Modify: `ui/src/routes/+layout.svelte`

- [ ] **Step 1: Write `ui/src/app.css`**

```css
:root {
  --bg: #0d1117;
  --bg-card: #161b22;
  --bg-card-hover: #1f242c;
  --fg: #e6edf3;
  --fg-muted: #8b949e;
  --border: #30363d;
  --accent: #2f81f7;
  --accent-fg: #ffffff;
}
:root[data-theme='light'] {
  --bg: #ffffff;
  --bg-card: #f6f8fa;
  --bg-card-hover: #eaeef2;
  --fg: #1f2328;
  --fg-muted: #59636e;
  --border: #d0d7de;
  --accent: #0969da;
  --accent-fg: #ffffff;
}

* { box-sizing: border-box; }
html, body { margin: 0; padding: 0; }
body {
  background: var(--bg);
  color: var(--fg);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
}
button { font-family: inherit; }
```

- [ ] **Step 2: Write `ui/src/routes/+layout.svelte`**

```svelte
<script lang="ts">
  import '../app.css';
</script>

<slot />
```

- [ ] **Step 3: Commit**

```bash
git add ui/src/app.css ui/src/routes/
git commit -m "feat(ui): global theme tokens and root layout"
```

---

### Task 16: Main page wiring (load, SSE, filter, edit handlers)

**Files:**
- Modify: `ui/src/routes/+page.svelte`

- [ ] **Step 1: Write `ui/src/routes/+page.svelte`**

```svelte
<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { subscribeEvents } from '$lib/api/events';
  import { createTilesStore } from '$lib/stores/tiles';
  import { editMode } from '$lib/stores/edit';
  import { searchQuery } from '$lib/stores/search';
  import Header from '$lib/components/Header.svelte';
  import Group from '$lib/components/Group.svelte';
  import TileEditor from '$lib/components/TileEditor.svelte';
  import HiddenDrawer from '$lib/components/HiddenDrawer.svelte';
  import BookmarkDialog from '$lib/components/BookmarkDialog.svelte';
  import type { Tile, Group as TGroup } from '$lib/api/types';

  const tiles = createTilesStore(api);
  let editingTileId: string | null = null;
  let addBookmarkGroup: string | null = null;
  let unsub: (() => void) | null = null;

  onMount(async () => {
    await tiles.load();
    unsub = subscribeEvents((e) => tiles.applyEvent(e));
  });
  onDestroy(() => unsub?.());

  $: editingTile = editingTileId
    ? $tiles.tiles.find((t) => t.id === editingTileId) ?? null
    : null;

  // Filter logic.
  function matches(t: Tile, q: string): boolean {
    if (!q) return true;
    const ql = q.toLowerCase();
    return (
      t.name.toLowerCase().includes(ql) ||
      t.url.toLowerCase().includes(ql) ||
      (t.description ?? '').toLowerCase().includes(ql)
    );
  }

  $: filteredTiles = $tiles.tiles.filter((t) => matches(t, $searchQuery));
  $: groupedTiles = (() => {
    const m = new Map<string, Tile[]>();
    for (const t of filteredTiles) {
      if (!$editMode && t.hidden) continue;
      const arr = m.get(t.group) ?? [];
      arr.push(t);
      m.set(t.group, arr);
    }
    return m;
  })();
  $: visibleGroups = $tiles.groups
    .filter((g) => groupedTiles.has(g.id))
    .sort((a, b) => a.order - b.order);
  $: hiddenTiles = $tiles.tiles.filter((t) => t.hidden);

  async function onSaveTile(detail: import('$lib/api/types').TileOverride) {
    await api.patchConfig({ tiles: [detail] });
    editingTileId = null;
    await tiles.load();
  }

  async function onResetTile(detail: { id: string }) {
    // Send all current overrides minus this id, then reload.
    // Simpler approach: PATCH tiles array with an empty override
    // entry — but that wouldn't remove it. Use a dedicated endpoint?
    // V1 compromise: send override with all empty strings, which is
    // a no-op-merge. For true reset, the user can hide the tile.
    // (Backend roadmap: add DELETE /api/config/tiles/{id}.)
    console.warn('reset-to-auto not yet wired; use hide for v1', detail.id);
    editingTileId = null;
  }

  async function onHideTile(detail: { id: string }) {
    await api.patchConfig({ tiles: [{ id: detail.id, hidden: true }] });
    await tiles.load();
  }

  async function onRestoreTile(detail: { id: string }) {
    await api.patchConfig({ tiles: [{ id: detail.id, hidden: false }] });
    await tiles.load();
  }

  async function onCreateBookmark(b: import('$lib/api/types').Bookmark) {
    await api.addBookmark(b);
    addBookmarkGroup = null;
    await tiles.load();
  }
</script>

<Header title="Dashboard" />

<main>
  {#each visibleGroups as g (g.id)}
    <Group
      group={g}
      tiles={groupedTiles.get(g.id) ?? []}
      editing={$editMode}
      on:editTile={(e) => (editingTileId = e.detail.id)}
      on:hideTile={(e) => onHideTile(e.detail)}
      on:addBookmark={(e) => (addBookmarkGroup = e.detail.group)}
    />
  {/each}

  {#if visibleGroups.length === 0}
    <p class="empty">
      {#if $searchQuery}No tiles match "{$searchQuery}".
      {:else}No tiles discovered yet.
      {/if}
    </p>
  {/if}
</main>

{#if $editMode}
  <HiddenDrawer tiles={hiddenTiles} on:restore={(e) => onRestoreTile(e.detail)} />
{/if}

{#if editingTile}
  <TileEditor
    tile={editingTile}
    groups={$tiles.groups}
    on:save={(e) => onSaveTile(e.detail)}
    on:reset={(e) => onResetTile(e.detail)}
    on:close={() => (editingTileId = null)}
  />
{/if}

{#if addBookmarkGroup}
  <BookmarkDialog
    group={addBookmarkGroup}
    on:create={(e) => onCreateBookmark(e.detail)}
    on:close={() => (addBookmarkGroup = null)}
  />
{/if}

<style>
  main { padding: 24px; }
  .empty { color: var(--fg-muted); text-align: center; padding: 60px; }
</style>
```

- [ ] **Step 2: Verify build still succeeds**

```bash
cd ui && pnpm run build && test -f build/index.html && echo OK
```
Expected: `OK`.

- [ ] **Step 3: Commit**

```bash
cd .. && git add ui/src/routes/
git commit -m "feat(ui): main page wiring with editor, drawer, bookmark dialog"
```

Note: This task includes a known v1 limitation — "Reset to auto" logs a warning and does nothing because the backend has no per-tile-override DELETE endpoint. The Group component's drag-and-drop is also not wired yet (Task 17).

---

### Task 17: Drag-and-drop reorder within and across groups

**Files:**
- Modify: `ui/src/lib/components/Group.svelte`
- Modify: `ui/src/routes/+page.svelte`

`svelte-dnd-action` works by attaching a `use:dndzone` action to a container and binding to a writable list of items. When the user drags, the action calls `consider`/`finalize` callbacks with the new ordering.

- [ ] **Step 1: Modify `ui/src/lib/components/Group.svelte`** — replace the `.grid` block:

Add to the script:

```ts
import { dndzone, type DndEvent } from 'svelte-dnd-action';
import type { Tile as TTile } from '$lib/api/types';

const flipDurationMs = 150;

function consider(e: CustomEvent<DndEvent<TTile>>) {
  sorted = e.detail.items;
}
function finalize(e: CustomEvent<DndEvent<TTile>>) {
  sorted = e.detail.items;
  dispatch('reorder', { group: group.id, items: sorted });
}
```

Add the dispatch type:

```ts
const dispatch = createEventDispatcher<{
  editTile: { id: string };
  hideTile: { id: string };
  addBookmark: { group: string };
  reorder: { group: string; items: TTile[] };
}>();
```

Replace the grid markup with:

```svelte
<div
  class="grid"
  use:dndzone={{ items: sorted, flipDurationMs, dragDisabled: !editing, type: 'tile' }}
  on:consider={consider}
  on:finalize={finalize}
>
  {#each sorted as t (t.id)}
    <Tile
      tile={t}
      {editing}
      on:edit={(e) => dispatch('editTile', e.detail)}
      on:hide={(e) => dispatch('hideTile', e.detail)}
    />
  {/each}
</div>
```

Important: change `$: sorted = ...` to `let sorted: TTile[] = []` with a separate `$: sorted = [...visible].sort(...)` reactive block — `dndzone` requires writable items, so initialize once and let consider/finalize mutate.

Final form:

```ts
$: visible = editing ? tiles : tiles.filter((t) => !t.hidden);
let sorted: TTile[] = [];
$: sorted = [...visible].sort((a, b) => a.order - b.order || a.name.localeCompare(b.name));
```

- [ ] **Step 2: Modify `ui/src/routes/+page.svelte`** — handle `reorder`:

Add to the Group element:

```svelte
on:reorder={(e) => onReorder(e.detail)}
```

Add the handler:

```ts
async function onReorder(detail: { group: string; items: import('$lib/api/types').Tile[] }) {
  // Cross-group moves are detected by comparing each tile's existing
  // group to the destination group.
  const overrides = detail.items.map((t, i) => ({
    id: t.id,
    order: i,
    group: detail.group,
  }));
  await api.patchConfig({ tiles: overrides });
  await tiles.load();
}
```

- [ ] **Step 3: Verify build**

```bash
cd ui && pnpm run build && echo OK
```

- [ ] **Step 4: Manual smoke (optional in CI; do in dev)**

Start the backend (from the backend plan) on `:8080`, then `pnpm run dev`. Toggle edit mode, drag a tile, verify it persists across reload.

- [ ] **Step 5: Commit**

```bash
cd .. && git add ui/
git commit -m "feat(ui): drag-and-drop reorder via svelte-dnd-action"
```

---

## Phase E: Icon bundling and Go embedding

### Task 18: Bundle dashboard-icons and generate slug list

**Files:**
- Create: `ui/icons/` (downloaded asset directory)
- Create: `ui/scripts/fetch-icons.mjs`, `ui/scripts/generate-icon-list.mjs`
- Modify: `ui/package.json`, `ui/src/lib/icons/index.ts`

We snapshot a pinned commit of `selfhst/dashboard-icons`, take only the PNG (`png/`) subset to control image size, and emit the slug list.

- [ ] **Step 1: Pin a commit**

```bash
ICONS_COMMIT=8c9f3c1d8a1e2f0b5e4c7d3f2a1b9c8d7e6f5a4b   # placeholder — replace with current main HEAD
```

When implementing, look up the actual current HEAD with:
```bash
curl -fsS https://api.github.com/repos/selfhst/dashboard-icons/commits/main | jq -r .sha
```
and substitute in the script.

- [ ] **Step 2: Write `ui/scripts/fetch-icons.mjs`**

```js
import { mkdirSync, createWriteStream, existsSync, rmSync } from 'node:fs';
import { execSync } from 'node:child_process';

const COMMIT = process.env.ICONS_COMMIT;
if (!COMMIT) {
  console.error('Set ICONS_COMMIT env var to the dashboard-icons commit SHA to pin.');
  process.exit(1);
}

const dest = 'icons';
if (existsSync(dest)) rmSync(dest, { recursive: true, force: true });
mkdirSync(dest, { recursive: true });

const tarUrl = `https://codeload.github.com/selfhst/dashboard-icons/tar.gz/${COMMIT}`;
console.log('Downloading', tarUrl);
execSync(
  `curl -fsSL ${tarUrl} | tar -xz --strip-components=2 -C ${dest} dashboard-icons-${COMMIT}/png`,
  { stdio: 'inherit', shell: '/bin/bash' },
);
console.log('Done.');
```

- [ ] **Step 3: Write `ui/scripts/generate-icon-list.mjs`**

```js
import { readdirSync, writeFileSync } from 'node:fs';

const files = readdirSync('icons').filter((f) => f.endsWith('.png'));
const slugs = files.map((f) => f.replace(/\.png$/, '')).sort();

const out = `// Code generated by scripts/generate-icon-list.mjs. DO NOT EDIT.
export const ICON_SLUGS: readonly string[] = ${JSON.stringify(slugs, null, 2)};
`;
writeFileSync('src/lib/icons/index.ts', out);
console.log(`Wrote ${slugs.length} slugs to src/lib/icons/index.ts`);
```

- [ ] **Step 4: Add npm scripts in `ui/package.json`**

```json
{
  "scripts": {
    "icons:fetch": "node scripts/fetch-icons.mjs",
    "icons:list": "node scripts/generate-icon-list.mjs",
    "icons": "pnpm icons:fetch && pnpm icons:list"
  }
}
```

- [ ] **Step 5: Run icon pipeline once**

```bash
cd ui && ICONS_COMMIT=<sha> pnpm icons
```
Expected: `ui/icons/*.png` populated; `ui/src/lib/icons/index.ts` regenerated with real slug list.

- [ ] **Step 6: Add `ui/icons/` to `ui/.gitignore`**

```
icons/
```

(The icons are fetched at build time, not committed. The Dockerfile in the packaging plan re-runs `pnpm icons` before the Go build.)

- [ ] **Step 7: Commit**

```bash
cd .. && git add ui/scripts/ ui/package.json ui/.gitignore ui/src/lib/icons/
git commit -m "feat(ui): fetch dashboard-icons snapshot and generate slug list"
```

---

### Task 19: Embed the SvelteKit build and icon set in the Go binary

**Files:**
- Create: `internal/assets/assets.go`
- Create: `internal/assets/embed.go`
- Modify: `internal/api/server.go`
- Modify: `Makefile`

The Go binary serves three things at the root: SPA assets, `/icons/*`, and the JSON API. The API routes are already wired; this task adds the static handlers.

- [ ] **Step 1: Write `internal/assets/embed.go`**

```go
package assets

import "embed"

// ui contains the built SvelteKit static SPA.
// Generated by `pnpm run build` in ../ui before `go build`.
//
//go:embed all:ui
var ui embed.FS

// icons contains PNGs from selfhst/dashboard-icons.
// Generated by `pnpm icons` in ../ui before `go build`.
//
//go:embed all:icons
var icons embed.FS

// UI returns the embedded UI filesystem rooted at the build output.
func UI() embed.FS { return ui }

// Icons returns the embedded icon filesystem rooted at the icons dir.
func Icons() embed.FS { return icons }
```

- [ ] **Step 2: Write `internal/assets/assets.go`**

```go
package assets

import (
	"io/fs"
	"net/http"
	"strings"
)

// UIHandler serves the SvelteKit SPA. Any request whose path is not a
// real file falls back to index.html so client-side routing works.
func UIHandler() http.Handler {
	root, err := fs.Sub(ui, "ui")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cheap SPA fallback: if the file doesn't exist, serve index.html.
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" {
			clean = "index.html"
		}
		if _, err := fs.Stat(root, clean); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// IconsHandler serves /icons/<slug>.png from the embedded icon set.
func IconsHandler() http.Handler {
	root, err := fs.Sub(icons, "icons")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/icons/", http.FileServer(http.FS(root)))
}
```

- [ ] **Step 3: Create the embed source directories**

The `//go:embed` directives require the directories to exist at build time. Create empty placeholders so `go build` works even before assets are generated:

```bash
mkdir -p internal/assets/ui internal/assets/icons
touch internal/assets/ui/.gitkeep internal/assets/icons/.gitkeep
```

Add to repository `.gitignore`:

```
internal/assets/ui/*
internal/assets/icons/*
!internal/assets/ui/.gitkeep
!internal/assets/icons/.gitkeep
```

- [ ] **Step 4: Modify `internal/api/server.go`** — register static routes

Add field and parameter to `NewServerFull`. Top of `routes()`:

```go
s.mux.Handle("/icons/", assets.IconsHandler())
// Catch-all: serve the UI SPA.
s.mux.Handle("/", assets.UIHandler())
```

Note ordering — the API routes are registered with method-prefixed patterns
(`GET /api/...`) which take precedence over the bare `/` catch-all in Go 1.22's
`http.ServeMux`. Import:

```go
"github.com/anomalyco/k8s-auto-dash/internal/assets"
```

- [ ] **Step 5: Update Makefile**

Add to the top of `Makefile`:

```make
UI_DIR ?= ui
ICONS_COMMIT ?= main
```

Add new targets:

```make
.PHONY: ui-build
ui-build:
	cd $(UI_DIR) && pnpm install && ICONS_COMMIT=$(ICONS_COMMIT) pnpm icons && pnpm run build
	rm -rf internal/assets/ui/* internal/assets/icons/*
	cp -r $(UI_DIR)/build/. internal/assets/ui/
	cp -r $(UI_DIR)/icons/. internal/assets/icons/

.PHONY: build-all
build-all: ui-build build
```

Update the `build` target line in the Makefile so the final binary is at `bin/k8s-auto-dash` (it already is — no change required).

- [ ] **Step 6: Smoke test the full build**

```bash
make build-all
./bin/k8s-auto-dash --addr :8080 &
PID=$!
sleep 1
curl -fsS http://localhost:8080/healthz && echo
curl -fsS http://localhost:8080/ | grep -q '<!doctype' && echo "UI OK"
kill $PID
```

Expected: `OK`, `UI OK`. (Requires a kubeconfig pointing at a real cluster with the CRD installed, OR run against envtest — for a true offline smoke, see Task 21.)

- [ ] **Step 7: Commit**

```bash
git add internal/assets/ internal/api/ Makefile .gitignore
git commit -m "feat: embed SvelteKit build and icons in Go binary"
```

---

## Phase F: End-to-end smoke

### Task 20: Playwright smoke test against a mocked backend

**Files:**
- Create: `ui/playwright.config.ts`, `ui/tests/e2e/smoke.spec.ts`
- Create: `ui/tests/e2e/mock-server.mjs`

The full end-to-end against envtest lives in the backend plan (Task 21). Here we test the UI against a mock that serves canned `/api/tiles` and an SSE stream — fast, deterministic, runs in CI without a Kubernetes cluster.

- [ ] **Step 1: Write `ui/playwright.config.ts`**

```ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  retries: 0,
  use: {
    baseURL: 'http://localhost:4173',
    trace: 'retain-on-failure',
  },
  webServer: [
    {
      command: 'node tests/e2e/mock-server.mjs',
      port: 8080,
      reuseExistingServer: false,
    },
    {
      command: 'pnpm run preview --port 4173',
      port: 4173,
      reuseExistingServer: false,
    },
  ],
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
});
```

- [ ] **Step 2: Write `ui/tests/e2e/mock-server.mjs`**

```js
import http from 'node:http';

const view = {
  groups: [
    { id: 'media', name: 'Media', order: 0 },
    { id: 'infra', name: 'Infrastructure', order: 1 },
  ],
  tiles: [
    {
      id: 'media/jellyfin/jellyfin.example.com',
      source: 'httproute',
      name: 'Jellyfin',
      url: 'https://jellyfin.example.com',
      icon: 'jellyfin',
      group: 'media',
      order: 0,
      hidden: false,
      status: { state: 'up', statusCode: 200, latencyMs: 12, checkedAt: '' },
      k8s: { namespace: 'media', httpRouteName: 'jellyfin', gatewayRefs: [{ namespace: 'gw', name: 'ext' }] },
    },
    {
      id: 'infra/grafana/grafana.example.com',
      source: 'httproute',
      name: 'Grafana',
      url: 'https://grafana.example.com',
      icon: 'grafana',
      group: 'infra',
      order: 0,
      hidden: false,
      status: { state: 'down', statusCode: 0, latencyMs: 0, checkedAt: '', error: 'unreachable' },
    },
  ],
};

http.createServer((req, res) => {
  res.setHeader('Access-Control-Allow-Origin', '*');
  if (req.url === '/api/tiles' && req.method === 'GET') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(view));
    return;
  }
  if (req.url === '/api/events' && req.method === 'GET') {
    res.writeHead(200, { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache' });
    res.write(': hello\n\n');
    return; // keep open
  }
  if (req.url?.startsWith('/icons/')) {
    // 1×1 transparent PNG so <img> doesn't 404.
    const png = Buffer.from(
      '89504e470d0a1a0a0000000d49484452000000010000000108060000001f15c4890000000d49444154789c63000100000005000146da77c30000000049454e44ae426082',
      'hex',
    );
    res.writeHead(200, { 'Content-Type': 'image/png' });
    res.end(png);
    return;
  }
  if (req.method === 'PATCH' || req.method === 'POST' || req.method === 'DELETE' || req.method === 'PUT') {
    res.writeHead(200);
    res.end();
    return;
  }
  res.writeHead(404);
  res.end();
}).listen(8080, () => console.log('mock backend on :8080'));
```

- [ ] **Step 3: Add `ui/vite.config.ts` preview config** — already proxies `/api` and `/icons` to `:8080`; that's all we need.

- [ ] **Step 4: Write `ui/tests/e2e/smoke.spec.ts`**

```ts
import { test, expect } from '@playwright/test';

test('renders discovered tiles in groups', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByText('Jellyfin')).toBeVisible();
  await expect(page.getByText('Grafana')).toBeVisible();
  await expect(page.getByText('Media')).toBeVisible();
  await expect(page.getByText('Infrastructure')).toBeVisible();
});

test('search filters tiles', async ({ page }) => {
  await page.goto('/');
  await page.getByLabel('Search tiles').fill('jelly');
  await expect(page.getByText('Jellyfin')).toBeVisible();
  await expect(page.getByText('Grafana')).not.toBeVisible();
});

test('edit mode shows action buttons', async ({ page }) => {
  await page.goto('/');
  await page.getByLabel('Toggle edit mode').click();
  await expect(page.getByLabel('Edit tile').first()).toBeVisible();
});

test('opening tile editor shows k8s info', async ({ page }) => {
  await page.goto('/');
  await page.getByLabel('Toggle edit mode').click();
  await page.getByLabel('Edit tile').first().click();
  await expect(page.getByText('Namespace')).toBeVisible();
  await expect(page.getByText('media')).toBeVisible();
});
```

- [ ] **Step 5: Install browsers and run**

```bash
cd ui
pnpm exec playwright install chromium
pnpm run build
pnpm exec playwright test
```
Expected: 4 tests pass.

- [ ] **Step 6: Add npm script**

```json
{
  "scripts": {
    "test:e2e": "playwright test"
  }
}
```

- [ ] **Step 7: Commit**

```bash
cd .. && git add ui/
git commit -m "test(ui): playwright smoke test against mock backend"
```

---

## Done criteria

- `cd ui && pnpm vitest run` passes (all component tests).
- `cd ui && pnpm test:e2e` passes (Playwright smoke).
- `make build-all` produces `bin/k8s-auto-dash` with embedded UI + icons.
- Running the binary against a real cluster and loading `http://<host>/` shows discovered tiles, grouped by namespace, with status dots.
- Toggling edit mode reveals drag handles, action buttons, and the Hidden Tiles drawer.
- Search box live-filters; theme toggle cycles dark/light/auto and persists.
- Adding a bookmark in edit mode creates one server-side and re-renders.
- `kubectl edit dashboardconfig default` triggers a refetch in connected browsers within ~1s (`config-changed` SSE).

## Coverage check (self-review against the design spec)

- ✅ §"Page structure" header + grouped sections + wide info cards — Tasks 9, 10, 11, 15.
- ✅ §"View vs. edit mode" pencil toggle, drawer, action menu — Tasks 11, 14, 16.
- ✅ §"Drag-and-drop" within/across groups via svelte-dnd-action — Task 17.
- ✅ §"Tile editor" all fields including read-only k8s info and "Reset to auto" — Task 13. (Reset is stub-only in v1; see Known Limitations.)
- ✅ §"Search & filtering" client-side fuzzy filter — Task 16.
- ✅ §"Bookmarks" add/edit via UI — Task 14, 16.
- ✅ §"Mobile-responsive" via CSS grid `auto-fill, minmax` — Task 10.
- ✅ §"Theme" dark/light/auto with localStorage — Tasks 6, 15.
- ✅ §"Live updates" SSE — Tasks 4, 5, 16.
- ✅ §"Icons" bundled selfhst/dashboard-icons + IconPicker — Tasks 12, 18.

## Known limitations carried into v1

- **"Reset to auto" is a no-op stub** because the backend has no
  `DELETE /api/config/tiles/{id}` endpoint. Workarounds: clear fields
  manually in the editor, or hide the tile. Add a backend endpoint and
  wire `onResetTile` in a follow-up.
- **Cross-group drag** updates `group` field correctly but does not
  remove the moved tile from its origin group's order list until the
  next refetch (~ms). Visible flicker possible.
- **`config-changed` toast** is not implemented — refetch happens
  silently. Add a `<Toast>` component in a follow-up.

## Notes for the implementing engineer

- **Task ordering:** 1→4 sequential. 5→7 can be parallel. 8→14
  components can be parallel within phase. 15 must precede 16. 17
  depends on 16. 18–19 can run in either order. 20 is last.
- **`svelte-dnd-action` ergonomics:** the action requires the items
  array to be writable and re-assigned on `consider`/`finalize`. Don't
  derive items from a `$:` reactive expression downstream of dnd, or
  you'll lose drag state mid-gesture.
- **TypeScript strictness:** keep `strict: true`. Don't disable
  checks to silence errors — fix the types.
- **No new runtime dependencies** beyond what Task 1 installed.

