import { useState, useEffect } from 'react';
import { Check } from 'lucide-react';
import { useTheme, ACCENT_PRESETS, type ThemeMode } from '../theme';
import { cn } from '../lib/utils';

const DENSITY_KEY = 'patchiq-density';

type Density = 'comfortable' | 'compact';

function getStoredDensity(): Density {
  try {
    const v = localStorage.getItem(DENSITY_KEY);
    if (v === 'comfortable' || v === 'compact') return v;
  } catch {
    /* noop */
  }
  return 'comfortable';
}

const MODE_OPTIONS: { value: ThemeMode; label: string }[] = [
  { value: 'dark', label: 'Dark' },
  { value: 'light', label: 'Light' },
  { value: 'system', label: 'System' },
];

function ThemeConfigurator({ className }: { className?: string }) {
  const { mode, accent, setMode, setAccent } = useTheme();
  const [customHex, setCustomHex] = useState(accent);
  const [density, setDensityState] = useState<Density>(getStoredDensity);

  useEffect(() => {
    setCustomHex(accent);
  }, [accent]);

  function handleCustomHexChange(hex: string) {
    setCustomHex(hex);
    if (/^#[0-9a-f]{6}$/i.test(hex)) {
      setAccent(hex);
    }
  }

  function setDensity(d: Density) {
    setDensityState(d);
    try {
      localStorage.setItem(DENSITY_KEY, d);
    } catch {
      /* noop */
    }
  }

  const presetEntries = Object.entries(ACCENT_PRESETS);

  return (
    <div className={cn('space-y-6', className)}>
      {/* Mode picker */}
      <div>
        <label
          className="mb-2 block text-xs uppercase tracking-wider"
          style={{ color: 'var(--text-muted)' }}
        >
          Appearance
        </label>
        <div
          className="inline-flex rounded-md"
          style={{
            borderWidth: '1px',
            borderStyle: 'solid',
            borderColor: 'var(--border)',
          }}
          role="radiogroup"
          aria-label="Theme mode"
        >
          {MODE_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              type="button"
              role="radio"
              aria-checked={mode === opt.value}
              onClick={() => setMode(opt.value)}
              className="px-3 py-1.5 text-sm font-medium first:rounded-l-md last:rounded-r-md"
              style={{
                backgroundColor: mode === opt.value ? 'var(--accent)' : 'transparent',
                color: mode === opt.value ? '#ffffff' : 'var(--text-secondary)',
              }}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Accent color */}
      <div>
        <label
          className="mb-2 block text-xs uppercase tracking-wider"
          style={{ color: 'var(--text-muted)' }}
        >
          Accent Color
        </label>
        <div className="flex flex-wrap gap-2">
          {presetEntries.map(([name, hex]) => (
            <button
              key={name}
              type="button"
              onClick={() => setAccent(hex)}
              className="relative flex h-8 w-8 items-center justify-center rounded-full"
              style={{ backgroundColor: hex }}
              aria-label={`${name} accent color`}
            >
              {accent === hex && <Check size={14} className="text-white" />}
            </button>
          ))}
        </div>
        <div className="mt-3 flex items-center gap-2">
          <label
            className="text-xs"
            style={{ color: 'var(--text-secondary)' }}
            htmlFor="custom-hex-input"
          >
            Custom
          </label>
          <input
            id="custom-hex-input"
            type="text"
            value={customHex}
            onChange={(e) => handleCustomHexChange(e.target.value)}
            className="rounded-md px-2 py-1 text-xs"
            style={{
              backgroundColor: 'var(--bg-card)',
              borderWidth: '1px',
              borderStyle: 'solid',
              borderColor: 'var(--border)',
              color: 'var(--text-primary)',
              fontFamily: 'var(--font-mono)',
              width: 90,
            }}
            placeholder="#10b981"
          />
        </div>
      </div>

      {/* Density toggle */}
      <div>
        <label
          className="mb-2 block text-xs uppercase tracking-wider"
          style={{ color: 'var(--text-muted)' }}
        >
          Density
        </label>
        <div
          className="inline-flex rounded-md"
          style={{
            borderWidth: '1px',
            borderStyle: 'solid',
            borderColor: 'var(--border)',
          }}
          role="radiogroup"
          aria-label="Display density"
        >
          {(['comfortable', 'compact'] as const).map((d) => (
            <button
              key={d}
              type="button"
              role="radio"
              aria-checked={density === d}
              onClick={() => setDensity(d)}
              className="px-3 py-1.5 text-sm font-medium capitalize first:rounded-l-md last:rounded-r-md"
              style={{
                backgroundColor: density === d ? 'var(--accent)' : 'transparent',
                color: density === d ? '#ffffff' : 'var(--text-secondary)',
              }}
            >
              {d}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}

export { ThemeConfigurator };
