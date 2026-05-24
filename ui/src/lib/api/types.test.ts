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
