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
