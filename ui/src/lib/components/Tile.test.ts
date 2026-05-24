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

  it('edit mode: emits edit event on menu click', async () => {
    const onEdit = vi.fn();
    const { getByLabelText } = render(Tile, {
      props: { tile: sample, editing: true, onedit: onEdit },
    });
    await fireEvent.click(getByLabelText('Edit tile'));
    expect(onEdit).toHaveBeenCalledWith({ id: 'media/jellyfin/jellyfin.example.com' });
  });
});
