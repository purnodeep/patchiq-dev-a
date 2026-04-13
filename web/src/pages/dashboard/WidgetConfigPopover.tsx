import { useEffect, useRef, useCallback } from 'react';
import type { WidgetConfigSchema, WidgetConfig, WidgetConfigCategory } from './types';

interface WidgetConfigPopoverProps {
  configSchema: WidgetConfigSchema;
  config: WidgetConfig;
  onChange: (config: WidgetConfig) => void;
  onClose: () => void;
}

const CATEGORY_ORDER: { key: WidgetConfigCategory; label: string }[] = [
  { key: 'data', label: 'Data' },
  { key: 'display', label: 'Display' },
  { key: 'behavior', label: 'Behavior' },
];

export function WidgetConfigPopover({ configSchema, config, onChange, onClose }: WidgetConfigPopoverProps) {
  const ref = useRef<HTMLDivElement>(null);

  const handleClickOutside = useCallback(
    (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClose();
      }
    },
    [onClose],
  );

  useEffect(() => {
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [handleClickOutside]);

  const fields = Object.entries(configSchema);
  const grouped = CATEGORY_ORDER.map((cat) => ({
    ...cat,
    fields: fields.filter(([, f]) => f.category === cat.key),
  })).filter((g) => g.fields.length > 0);

  const handleReset = () => {
    const defaults: WidgetConfig = {};
    for (const [key, field] of fields) {
      defaults[key] = field.default;
    }
    onChange(defaults);
  };

  const updateField = (key: string, value: unknown) => {
    onChange({ ...config, [key]: value });
  };

  return (
    <div
      ref={ref}
      style={{
        position: 'absolute',
        top: 28,
        right: 0,
        zIndex: 100,
        width: 240,
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        boxShadow: '0 8px 24px rgba(0,0,0,0.3)',
        padding: 12,
        fontSize: 12,
        color: 'var(--text-secondary)',
      }}
      onClick={(e) => e.stopPropagation()}
    >
      {grouped.map((group) => (
        <div key={group.key} style={{ marginBottom: 10 }}>
          <div
            style={{
              fontSize: 10,
              fontWeight: 600,
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
              color: 'var(--text-muted)',
              marginBottom: 6,
            }}
          >
            {group.label}
          </div>
          {group.fields.map(([key, field]) => (
            <div key={key} style={{ marginBottom: 8 }}>
              <label
                style={{
                  display: 'block',
                  fontSize: 11,
                  color: 'var(--text-secondary)',
                  marginBottom: 3,
                }}
              >
                {field.label}
              </label>
              {field.type === 'select' && (
                <select
                  value={String(config[key] ?? field.default)}
                  onChange={(e) => updateField(key, e.target.value)}
                  style={{
                    width: '100%',
                    padding: '4px 6px',
                    borderRadius: 4,
                    border: '1px solid var(--border)',
                    background: 'var(--bg-page)',
                    color: 'var(--text-primary)',
                    fontSize: 11,
                    outline: 'none',
                  }}
                >
                  {field.options?.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </select>
              )}
              {field.type === 'multi-select' && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
                  {field.options?.map((opt) => {
                    const currentVal = (config[key] ?? field.default) as string[];
                    const checked = currentVal.includes(opt.value);
                    return (
                      <label
                        key={opt.value}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 5,
                          fontSize: 11,
                          cursor: 'pointer',
                        }}
                      >
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={() => {
                            const next = checked
                              ? currentVal.filter((v) => v !== opt.value)
                              : [...currentVal, opt.value];
                            updateField(key, next);
                          }}
                          style={{ accentColor: 'var(--accent)' }}
                        />
                        {opt.label}
                      </label>
                    );
                  })}
                </div>
              )}
              {field.type === 'toggle' && (
                <label
                  style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}
                >
                  <input
                    type="checkbox"
                    checked={Boolean(config[key] ?? field.default)}
                    onChange={(e) => updateField(key, e.target.checked)}
                    style={{ accentColor: 'var(--accent)' }}
                  />
                  <span style={{ fontSize: 11 }}>
                    {Boolean(config[key] ?? field.default) ? 'On' : 'Off'}
                  </span>
                </label>
              )}
              {field.type === 'number' && (
                <input
                  type="number"
                  value={Number(config[key] ?? field.default)}
                  min={field.min}
                  max={field.max}
                  onChange={(e) => updateField(key, Number(e.target.value))}
                  style={{
                    width: '100%',
                    padding: '4px 6px',
                    borderRadius: 4,
                    border: '1px solid var(--border)',
                    background: 'var(--bg-page)',
                    color: 'var(--text-primary)',
                    fontSize: 11,
                    outline: 'none',
                  }}
                />
              )}
            </div>
          ))}
        </div>
      ))}
      <button
        onClick={handleReset}
        style={{
          width: '100%',
          padding: '5px 0',
          borderRadius: 4,
          border: '1px solid var(--border)',
          background: 'transparent',
          color: 'var(--text-muted)',
          fontSize: 10,
          cursor: 'pointer',
          transition: 'color 150ms',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.color = 'var(--text-primary)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.color = 'var(--text-muted)';
        }}
      >
        Reset to defaults
      </button>
    </div>
  );
}
