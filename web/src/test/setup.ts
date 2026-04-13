import '@testing-library/jest-dom/vitest';

// jsdom does not implement window.matchMedia; provide a minimal stub
// so components using the use-mobile hook (from @patchiq/ui) can render.
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string): MediaQueryList => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
});
