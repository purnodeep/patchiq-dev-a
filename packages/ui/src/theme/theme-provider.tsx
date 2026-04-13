import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';

type ThemeMode = 'dark' | 'light' | 'system';

interface ThemeContextValue {
  mode: ThemeMode;
  resolvedMode: 'dark' | 'light';
  accent: string;
  setMode: (mode: ThemeMode) => void;
  setAccent: (hex: string) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

const ACCENT_PRESETS: Record<string, string> = {
  forest: '#10b981',
  amethyst: '#7c3aed',
  ocean: '#3b82f6',
  arctic: '#06b6d4',
  ruby: '#f43f5e',
  ember: '#f97316',
  twilight: '#6366f1',
  mint: '#14b8a6',
};

const MODE_KEY = 'patchiq-theme-mode';
const ACCENT_KEY = 'patchiq-theme-accent';

function getStoredMode(): ThemeMode {
  try {
    const v = localStorage.getItem(MODE_KEY);
    if (v === 'dark' || v === 'light' || v === 'system') return v;
  } catch {
    /* noop */
  }
  return 'dark';
}

function getStoredAccent(): string | null {
  try {
    const v = localStorage.getItem(ACCENT_KEY);
    if (v && /^#[0-9a-f]{6}$/i.test(v)) return v;
  } catch {
    /* noop */
  }
  return null; // null = use CSS-defined default (emerald dark, amethyst light)
}

function resolveMode(mode: ThemeMode): 'dark' | 'light' {
  if (mode !== 'system') return mode;
  try {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  } catch {
    return 'dark';
  }
}

function computeAccentSubtle(hex: string): string {
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  return `rgba(${r},${g},${b},0.08)`;
}

function computeAccentBorder(hex: string): string {
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  return `rgba(${r},${g},${b},0.3)`;
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<ThemeMode>(getStoredMode);
  const [accent, setAccentState] = useState<string | null>(getStoredAccent);

  const resolved = resolveMode(mode);

  const setMode = useCallback((m: ThemeMode) => {
    setModeState(m);
    try {
      localStorage.setItem(MODE_KEY, m);
    } catch {
      /* noop */
    }
  }, []);

  const setAccent = useCallback((hex: string) => {
    setAccentState(hex);
    try {
      localStorage.setItem(ACCENT_KEY, hex);
    } catch {
      /* noop */
    }
  }, []);

  useEffect(() => {
    const el = document.documentElement;
    if (resolved === 'light') {
      el.classList.add('light');
      el.classList.remove('dark');
    } else {
      el.classList.add('dark');
      el.classList.remove('light');
    }
  }, [resolved]);

  useEffect(() => {
    if (mode !== 'system') return;
    try {
      const mq = window.matchMedia('(prefers-color-scheme: dark)');
      const handler = () => {
        const el = document.documentElement;
        const next = mq.matches ? 'dark' : 'light';
        if (next === 'light') {
          el.classList.add('light');
          el.classList.remove('dark');
        } else {
          el.classList.add('dark');
          el.classList.remove('light');
        }
      };
      mq.addEventListener('change', handler);
      return () => mq.removeEventListener('change', handler);
    } catch {
      /* noop */
    }
  }, [mode]);

  useEffect(() => {
    const el = document.documentElement;
    if (accent) {
      // User has explicitly chosen a custom accent — override CSS defaults
      el.style.setProperty('--accent', accent);
      el.style.setProperty('--accent-subtle', computeAccentSubtle(accent));
      el.style.setProperty('--accent-border', computeAccentBorder(accent));
    } else {
      // No custom accent — let CSS tokens.css handle it (emerald dark, amethyst light)
      el.style.removeProperty('--accent');
      el.style.removeProperty('--accent-subtle');
      el.style.removeProperty('--accent-border');
    }
  }, [accent]);

  // Resolve accent for consumers: custom if set, otherwise theme-dependent default
  const resolvedAccent = accent ?? (resolved === 'light' ? '#7c3aed' : '#10b981');

  return (
    <ThemeContext.Provider
      value={{ mode, resolvedMode: resolved, accent: resolvedAccent, setMode, setAccent }}
    >
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider');
  return ctx;
}

export { ACCENT_PRESETS };
export type { ThemeMode };
