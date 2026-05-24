import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import Group from './Group.svelte';
import type { Tile as TTile } from '$lib/api/types';

const tile = (id: string, name: string): TTile => ({
  id, source: 'httproute', name, url: `https://${id}`, icon: '',
  group: 'g', order: 0, hidden: false,
  status: { state: 'up', statusCode: 200, latencyMs: 0, checkedAt: '' },
});

describe('Group', () => {
  it('renders the group name and tiles', () => {
    const { getByText } = render(Group, {
      props: { group: { id: 'g', name: 'My Group', order: 0 }, tiles: [tile('a','A'), tile('b','B')], editing: false },
    });
    expect(getByText('My Group')).toBeInTheDocument();
    expect(getByText('A')).toBeInTheDocument();
    expect(getByText('B')).toBeInTheDocument();
  });

  it('hides hidden tiles when not editing', () => {
    const hidden = { ...tile('h', 'Hidden'), hidden: true };
    const { queryByText } = render(Group, {
      props: { group: { id: 'g', name: 'G', order: 0 }, tiles: [hidden], editing: false },
    });
    expect(queryByText('Hidden')).toBeNull();
  });

  it('shows hidden tiles when editing', () => {
    const hidden = { ...tile('h', 'Hidden'), hidden: true };
    const { getByText } = render(Group, {
      props: { group: { id: 'g', name: 'G', order: 0 }, tiles: [hidden], editing: true },
    });
    expect(getByText('Hidden')).toBeInTheDocument();
  });
});
