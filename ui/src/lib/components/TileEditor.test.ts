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
    const onSave = vi.fn();
    const { getByLabelText, getByText } = render(TileEditor, {
      props: { tile: t, groups, onsave: onSave },
    });

    await fireEvent.input(getByLabelText('Name'), { target: { value: 'Jelly' } });
    await fireEvent.click(getByText('Save'));

    expect(onSave).toHaveBeenCalled();
    expect(onSave.mock.calls[0][0]).toMatchObject({
      id: t.id, name: 'Jelly',
    });
  });

  it('emits reset', async () => {
    const onReset = vi.fn();
    const { getByText } = render(TileEditor, { props: { tile: t, groups, onreset: onReset } });
    await fireEvent.click(getByText('Reset to auto'));
    expect(onReset).toHaveBeenCalledWith({ id: t.id });
  });

  it('shows k8s info read-only', () => {
    const { getByText } = render(TileEditor, { props: { tile: t, groups } });
    expect(getByText('media')).toBeInTheDocument();
    expect(getByText('jelly')).toBeInTheDocument();
  });
});
