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
