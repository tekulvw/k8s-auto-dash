import { render, fireEvent } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import { get } from 'svelte/store';
import Header from './Header.svelte';
import { editMode } from '$lib/stores/edit';
import { searchQuery } from '$lib/stores/search';

describe('Header', () => {
  it('toggles edit mode when pencil clicked', async () => {
    editMode.set(false);
    const { getByLabelText } = render(Header, { props: { title: 'Home' } });
    await fireEvent.click(getByLabelText('Toggle edit mode'));
    expect(get(editMode)).toBe(true);
  });

  it('updates searchQuery on input', async () => {
    searchQuery.set('');
    const { getByPlaceholderText } = render(Header, { props: { title: 'Home' } });
    await fireEvent.input(getByPlaceholderText('Search'), { target: { value: 'jelly' } });
    expect(get(searchQuery)).toBe('jelly');
  });
});
