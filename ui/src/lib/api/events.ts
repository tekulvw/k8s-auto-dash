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
