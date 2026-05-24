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
    const onRestore = vi.fn();
    const { getByText } = render(HiddenDrawer, {
      props: { tiles: [hidden('a', 'A'), hidden('b', 'B')], onrestore: onRestore },
    });

    await fireEvent.click(getByText('A'));
    expect(onRestore).toHaveBeenCalledWith({ id: 'a' });
  });

  it('renders empty state', () => {
    const { getByText } = render(HiddenDrawer, { props: { tiles: [] } });
    expect(getByText(/No hidden tiles/i)).toBeInTheDocument();
  });
});
