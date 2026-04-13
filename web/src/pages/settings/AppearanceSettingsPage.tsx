import { useState, useCallback } from 'react';
import { useTheme } from '@patchiq/ui';

const ACCENT_COLORS = [
  'var(--accent)',
  '#7c3aed',
  '#3b82f6',
  'var(--signal-warning)',
  'var(--signal-critical)',
  '#ec4899',
  '#06b6d4',
  '#8b5cf6',
];

const COMPACT_KEY = 'patchiq-compact-mode';
const MONO_KEY = 'patchiq-monospace-data';

function getStoredBool(key: string, fallback: boolean): boolean {
  try {
    const v = localStorage.getItem(key);
    if (v === 'true') return true;
    if (v === 'false') return false;
  } catch {
    /* noop */
  }
  return fallback;
}

export function AppearanceSettingsPage() {
  const { mode, accent, setMode, setAccent } = useTheme();

  const [compactMode, setCompactMode] = useState(() => getStoredBool(COMPACT_KEY, false));
  const [monoData, setMonoData] = useState(() => getStoredBool(MONO_KEY, false));

  const toggleCompact = useCallback(() => {
    setCompactMode((prev) => {
      const next = !prev;
      try {
        localStorage.setItem(COMPACT_KEY, String(next));
      } catch {
        /* noop */
      }
      return next;
    });
  }, []);

  const toggleMono = useCallback(() => {
    setMonoData((prev) => {
      const next = !prev;
      try {
        localStorage.setItem(MONO_KEY, String(next));
      } catch {
        /* noop */
      }
      return next;
    });
  }, []);

  return (
    <div style={{ padding: '28px 40px 80px', maxWidth: 680 }}>
      {/* Section header */}
      <div style={{ paddingBottom: 16, borderBottom: '1px solid var(--border)', marginBottom: 24 }}>
        <h2
          style={{
            fontSize: 18,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            letterSpacing: '-0.02em',
            margin: 0,
          }}
        >
          Appearance
        </h2>
        <p
          style={{
            fontSize: 12,
            color: 'var(--text-muted)',
            margin: '4px 0 0',
          }}
        >
          Theme, accent color, and display preferences.
        </p>
      </div>

      {/* Theme sub-label */}
      <label
        style={{
          display: 'block',
          fontSize: 10,
          fontWeight: 600,
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          color: 'var(--text-muted)',
          fontFamily: 'var(--font-mono)',
          marginBottom: 10,
        }}
      >
        Theme
      </label>

      {/* Theme cards */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, maxWidth: 400 }}>
        {/* Dark theme card */}
        <button
          type="button"
          onClick={() => setMode('dark')}
          style={{
            background: 'var(--bg-card)',
            border: `2px solid ${mode === 'dark' ? accent : 'var(--border)'}`,
            borderRadius: 10,
            padding: 0,
            cursor: 'pointer',
            overflow: 'hidden',
            textAlign: 'left',
          }}
        >
          {/* Preview */}
          <div
            style={{
              height: 80,
              display: 'flex',
              background: '#111118',
              borderBottom: '1px solid var(--border)',
            }}
          >
            {/* Mini sidebar */}
            <div
              style={{
                width: 40,
                background: '#0c0c10',
                borderRight: '1px solid #1e1e2a',
                display: 'flex',
                flexDirection: 'column',
                padding: '10px 6px',
                gap: 6,
              }}
            >
              <div style={{ width: '100%', height: 4, borderRadius: 2, background: '#2a2a3a' }} />
              <div style={{ width: '100%', height: 4, borderRadius: 2, background: '#2a2a3a' }} />
              <div
                style={{
                  width: '70%',
                  height: 4,
                  borderRadius: 2,
                  background: accent,
                  opacity: 0.7,
                }}
              />
              <div style={{ width: '100%', height: 4, borderRadius: 2, background: '#2a2a3a' }} />
            </div>
            {/* Content area */}
            <div
              style={{
                flex: 1,
                padding: '10px 8px',
                display: 'flex',
                flexDirection: 'column',
                gap: 6,
              }}
            >
              <div style={{ width: '60%', height: 4, borderRadius: 2, background: '#2a2a3a' }} />
              <div style={{ width: '80%', height: 4, borderRadius: 2, background: '#1e1e2a' }} />
              <div style={{ width: '40%', height: 4, borderRadius: 2, background: '#1e1e2a' }} />
              <div
                style={{
                  marginTop: 'auto',
                  width: '30%',
                  height: 8,
                  borderRadius: 3,
                  background: accent,
                  opacity: 0.5,
                }}
              />
            </div>
          </div>
          {/* Label row */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '10px 12px',
            }}
          >
            <span
              style={{
                fontSize: 12,
                fontWeight: 500,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              Dark
            </span>
            <div
              style={{
                width: 16,
                height: 16,
                borderRadius: '50%',
                border: `2px solid ${mode === 'dark' ? accent : 'var(--border-strong, var(--border))'}`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              {mode === 'dark' && (
                <div style={{ width: 8, height: 8, borderRadius: '50%', background: accent }} />
              )}
            </div>
          </div>
        </button>

        {/* Light theme card */}
        <button
          type="button"
          onClick={() => setMode('light')}
          style={{
            background: 'var(--bg-card)',
            border: `2px solid ${mode === 'light' ? accent : 'var(--border)'}`,
            borderRadius: 10,
            padding: 0,
            cursor: 'pointer',
            overflow: 'hidden',
            textAlign: 'left',
          }}
        >
          {/* Preview */}
          <div
            style={{
              height: 80,
              display: 'flex',
              background: '#f8f9fa',
              borderBottom: '1px solid var(--border)',
            }}
          >
            {/* Mini sidebar */}
            <div
              style={{
                width: 40,
                background: '#ffffff',
                borderRight: '1px solid #e5e7eb',
                display: 'flex',
                flexDirection: 'column',
                padding: '10px 6px',
                gap: 6,
              }}
            >
              <div style={{ width: '100%', height: 4, borderRadius: 2, background: '#e5e7eb' }} />
              <div style={{ width: '100%', height: 4, borderRadius: 2, background: '#e5e7eb' }} />
              <div
                style={{
                  width: '70%',
                  height: 4,
                  borderRadius: 2,
                  background: accent,
                  opacity: 0.6,
                }}
              />
              <div style={{ width: '100%', height: 4, borderRadius: 2, background: '#e5e7eb' }} />
            </div>
            {/* Content area */}
            <div
              style={{
                flex: 1,
                padding: '10px 8px',
                display: 'flex',
                flexDirection: 'column',
                gap: 6,
              }}
            >
              <div style={{ width: '60%', height: 4, borderRadius: 2, background: '#d1d5db' }} />
              <div style={{ width: '80%', height: 4, borderRadius: 2, background: '#e5e7eb' }} />
              <div style={{ width: '40%', height: 4, borderRadius: 2, background: '#e5e7eb' }} />
              <div
                style={{
                  marginTop: 'auto',
                  width: '30%',
                  height: 8,
                  borderRadius: 3,
                  background: accent,
                  opacity: 0.4,
                }}
              />
            </div>
          </div>
          {/* Label row */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '10px 12px',
            }}
          >
            <span
              style={{
                fontSize: 12,
                fontWeight: 500,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              Light
            </span>
            <div
              style={{
                width: 16,
                height: 16,
                borderRadius: '50%',
                border: `2px solid ${mode === 'light' ? accent : 'var(--border-strong, var(--border))'}`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              {mode === 'light' && (
                <div style={{ width: 8, height: 8, borderRadius: '50%', background: accent }} />
              )}
            </div>
          </div>
        </button>
      </div>

      {/* Divider */}
      <div style={{ height: 1, background: 'var(--border)', margin: '24px 0' }} />

      {/* Accent Color */}
      <label
        style={{
          display: 'block',
          fontSize: 10,
          fontWeight: 600,
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          color: 'var(--text-muted)',
          fontFamily: 'var(--font-mono)',
          marginBottom: 4,
        }}
      >
        Accent Color
      </label>
      <p
        style={{
          fontSize: 11,
          color: 'var(--text-faint)',
          fontFamily: 'var(--font-sans)',
          margin: '0 0 12px',
        }}
      >
        Applied to active states, primary buttons, and focus rings.
      </p>

      {/* Accent swatches */}
      <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
        {ACCENT_COLORS.map((color) => {
          const isSelected = accent === color;
          return (
            <button
              key={color}
              type="button"
              onClick={() => setAccent(color)}
              aria-label={`Accent color ${color}`}
              style={{
                width: 32,
                height: 32,
                borderRadius: 8,
                background: color,
                border: isSelected ? '2px solid var(--text-emphasis)' : '2px solid transparent',
                cursor: 'pointer',
                padding: 0,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                transition: 'border-color 0.15s, transform 0.1s',
                boxShadow: isSelected ? `0 0 0 2px var(--bg-card), 0 0 0 4px ${color}` : 'none',
              }}
            />
          );
        })}
      </div>

      {/* Divider */}
      <div style={{ height: 1, background: 'var(--border)', margin: '24px 0' }} />

      {/* Toggle rows */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
        {/* Compact mode */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '14px 0',
            borderBottom: '1px solid var(--border)',
          }}
        >
          <div>
            <div
              style={{
                fontSize: 13,
                fontWeight: 500,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              Compact mode
            </div>
            <div
              style={{
                fontSize: 11,
                color: 'var(--text-faint)',
                fontFamily: 'var(--font-sans)',
                marginTop: 2,
              }}
            >
              Reduce spacing and padding throughout the interface.
            </div>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={compactMode}
            onClick={toggleCompact}
            style={{
              width: 36,
              height: 20,
              borderRadius: 10,
              background: compactMode ? accent : 'var(--bg-inset)',
              border: `1px solid ${compactMode ? accent : 'var(--border)'}`,
              padding: 2,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              transition: 'background 0.15s, border-color 0.15s',
              flexShrink: 0,
            }}
          >
            <div
              style={{
                width: 14,
                height: 14,
                borderRadius: '50%',
                background: compactMode ? 'var(--btn-accent-text, #fff)' : 'var(--text-faint)',
                transform: compactMode ? 'translateX(16px)' : 'translateX(0)',
                transition: 'transform 0.15s, background 0.15s',
              }}
            />
          </button>
        </div>

        {/* Show monospace data */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '14px 0',
          }}
        >
          <div>
            <div
              style={{
                fontSize: 13,
                fontWeight: 500,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              Show monospace data
            </div>
            <div
              style={{
                fontSize: 11,
                color: 'var(--text-faint)',
                fontFamily: 'var(--font-sans)',
                marginTop: 2,
              }}
            >
              Use monospace font for IDs, hashes, and technical values.
            </div>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={monoData}
            onClick={toggleMono}
            style={{
              width: 36,
              height: 20,
              borderRadius: 10,
              background: monoData ? accent : 'var(--bg-inset)',
              border: `1px solid ${monoData ? accent : 'var(--border)'}`,
              padding: 2,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              transition: 'background 0.15s, border-color 0.15s',
              flexShrink: 0,
            }}
          >
            <div
              style={{
                width: 14,
                height: 14,
                borderRadius: '50%',
                background: monoData ? 'var(--btn-accent-text, #fff)' : 'var(--text-faint)',
                transform: monoData ? 'translateX(16px)' : 'translateX(0)',
                transition: 'transform 0.15s, background 0.15s',
              }}
            />
          </button>
        </div>
      </div>
    </div>
  );
}
