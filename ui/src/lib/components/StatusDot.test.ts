import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import StatusDot from './StatusDot.svelte';

describe('StatusDot', () => {
  it.each(['up', 'degraded', 'down', 'unknown'] as const)('renders %s', (state) => {
    const { container } = render(StatusDot, { props: { state } });
    expect(container.querySelector(`.dot-${state}`)).toBeTruthy();
  });
});
