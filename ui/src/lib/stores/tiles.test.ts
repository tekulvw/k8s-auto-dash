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
