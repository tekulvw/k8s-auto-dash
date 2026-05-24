import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import Icon from './Icon.svelte';

describe('Icon', () => {
  it('uses /icons/<slug>.png for slugs', () => {
    const { container } = render(Icon, { props: { slug: 'jellyfin', alt: 'Jellyfin' } });
    const img = container.querySelector('img')!;
    expect(img.getAttribute('src')).toBe('/icons/jellyfin.png');
  });

  it('uses raw URL when given http', () => {
    const { container } = render(Icon, { props: { slug: 'https://example.com/x.png' } });
    expect(container.querySelector('img')!.getAttribute('src')).toBe('https://example.com/x.png');
  });

  it('renders fallback when no slug', () => {
    const { container } = render(Icon, { props: { slug: '' } });
    expect(container.querySelector('svg')).toBeTruthy();
  });
});
