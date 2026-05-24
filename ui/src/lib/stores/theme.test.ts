import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { get } from 'svelte/store';
import { createThemeStore } from './theme';

describe('theme store', () => {
  beforeEach(() => { localStorage.clear(); document.documentElement.removeAttribute('data-theme'); });
  afterEach(() => { localStorage.clear(); });

  it('defaults to dark when nothing persisted', () => {
    const s = createThemeStore();
    expect(get(s)).toBe('dark');
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
  });

  it('persists changes to localStorage', () => {
    const s = createThemeStore();
    s.set('light');
    expect(localStorage.getItem('theme')).toBe('light');
    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
  });

  it('reads existing localStorage value on init', () => {
    localStorage.setItem('theme', 'light');
    const s = createThemeStore();
    expect(get(s)).toBe('light');
  });
});
