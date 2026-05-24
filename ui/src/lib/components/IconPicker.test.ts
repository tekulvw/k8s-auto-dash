import { render, fireEvent } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import IconPicker from './IconPicker.svelte';

describe('IconPicker', () => {
  it('filters suggestions on input', async () => {
    const { getByPlaceholderText, getAllByRole } = render(IconPicker, {
      props: { value: '' },
    });
    const input = getByPlaceholderText('icon slug or URL') as HTMLInputElement;
    await fireEvent.input(input, { target: { value: 'jel' } });
    const opts = getAllByRole('option').map((o) => o.textContent?.trim());
    expect(opts).toContain('jellyfin');
  });

  it('emits change when suggestion clicked', async () => {
    const onChange = vi.fn();
    const { getByPlaceholderText, getByText } = render(IconPicker, {
      props: { value: '', onchange: onChange },
    });

    await fireEvent.input(getByPlaceholderText('icon slug or URL'), {
      target: { value: 'jelly' },
    });
    await fireEvent.click(getByText('jellyfin'));
    expect(onChange).toHaveBeenCalled();
    expect(onChange.mock.calls.at(-1)?.[0]).toEqual({ value: 'jellyfin' });
  });

  it('accepts URL input verbatim', async () => {
    const onChange = vi.fn();
    const { getByPlaceholderText } = render(IconPicker, {
      props: { value: '', onchange: onChange },
    });
    await fireEvent.input(getByPlaceholderText('icon slug or URL'), {
      target: { value: 'https://example.com/x.png' },
    });
    expect(onChange.mock.calls.at(-1)?.[0]).toEqual({ value: 'https://example.com/x.png' });
  });
});
