import { writable } from 'svelte/store';
import { browser } from '$app/environment';

export type Theme = 'dark' | 'light' | 'auto';
const VALID: Theme[] = ['dark', 'light', 'auto'];

function resolveAuto(): 'dark' | 'light' {
  if (!browser) return 'dark';
  return window.matchMedia?.('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

function apply(t: Theme) {
  if (!browser) return;
  const effective = t === 'auto' ? resolveAuto() : t;
  document.documentElement.setAttribute('data-theme', effective);
}

export function createThemeStore() {
  const initial: Theme = (() => {
    if (!browser) return 'dark';
    const v = localStorage.getItem('theme') as Theme | null;
    return v && VALID.includes(v) ? v : 'dark';
  })();
  apply(initial);

  const { subscribe, set } = writable<Theme>(initial);
  return {
    subscribe,
    set(t: Theme) {
      if (browser) localStorage.setItem('theme', t);
      apply(t);
      set(t);
    },
  };
}

export const theme = createThemeStore();
