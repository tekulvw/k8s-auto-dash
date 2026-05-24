import '@testing-library/jest-dom/vitest';

// Polyfill localStorage for jsdom when running without proper URL
if (typeof localStorage !== 'undefined' && typeof localStorage.setItem !== 'function') {
  const store = new Map();
  const storage: Storage = {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => { store.set(key, value); },
    removeItem: (key: string) => { store.delete(key); },
    clear: () => { store.clear(); },
    key: (index: number) => [...store.keys()][index] ?? null,
    get length() { return store.size; },
    [Symbol.toStringTag]: 'Storage',
  };
  Object.defineProperty(globalThis, 'localStorage', { value: storage, configurable: true, writable: true });
}
