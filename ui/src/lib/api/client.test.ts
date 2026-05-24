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
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 204 }));
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
