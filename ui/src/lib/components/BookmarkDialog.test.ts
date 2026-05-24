import { render, fireEvent } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import BookmarkDialog from './BookmarkDialog.svelte';

describe('BookmarkDialog', () => {
  it('emits create with payload', async () => {
    const onCreate = vi.fn();
    const { getByLabelText, getByText } = render(BookmarkDialog, {
      props: { group: 'infra', oncreate: onCreate },
    });

    await fireEvent.input(getByLabelText('Name'), { target: { value: 'Router' } });
    await fireEvent.input(getByLabelText('URL'), { target: { value: 'https://r' } });
    await fireEvent.click(getByText('Add'));

    expect(onCreate.mock.calls[0][0]).toMatchObject({
      name: 'Router', url: 'https://r', group: 'infra',
    });
    expect(onCreate.mock.calls[0][0].id).toBeTruthy();
  });
});
