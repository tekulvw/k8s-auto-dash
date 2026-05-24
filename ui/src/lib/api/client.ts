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
