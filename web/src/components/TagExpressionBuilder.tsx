import { useState } from 'react';
import { X, Plus } from 'lucide-react';
import { Button } from '@patchiq/ui';
import { useTags } from '../api/hooks/useTags';
import type { TagExpression } from '../types/deployment-wizard';

interface TagExpressionBuilderProps {
  value: TagExpression | undefined;
  onChange: (expr: TagExpression | undefined) => void;
  endpointCount?: number;
  endpointCountLoading?: boolean;
}

/** Tag chips use a consistent monochrome style to stay within the design system. */
const TAG_COLORS: Array<{ bg: string; color: string; border: string }> = [
  {
    bg: 'color-mix(in srgb, var(--text-muted) 10%, transparent)',
    color: 'var(--text-secondary)',
    border: 'color-mix(in srgb, var(--text-muted) 25%, transparent)',
  },
  {
    bg: 'color-mix(in srgb, var(--text-muted) 10%, transparent)',
    color: 'var(--text-secondary)',
    border: 'color-mix(in srgb, var(--text-muted) 25%, transparent)',
  },
  {
    bg: 'color-mix(in srgb, var(--text-muted) 10%, transparent)',
    color: 'var(--text-secondary)',
    border: 'color-mix(in srgb, var(--text-muted) 25%, transparent)',
  },
  {
    bg: 'color-mix(in srgb, var(--text-muted) 10%, transparent)',
    color: 'var(--text-secondary)',
    border: 'color-mix(in srgb, var(--text-muted) 25%, transparent)',
  },
  {
    bg: 'color-mix(in srgb, var(--text-muted) 10%, transparent)',
    color: 'var(--text-secondary)',
    border: 'color-mix(in srgb, var(--text-muted) 25%, transparent)',
  },
  {
    bg: 'color-mix(in srgb, var(--text-muted) 10%, transparent)',
    color: 'var(--text-secondary)',
    border: 'color-mix(in srgb, var(--text-muted) 25%, transparent)',
  },
];

function getConditions(expr: TagExpression | undefined): TagExpression[] {
  if (!expr) return [];
  if (expr.conditions) return expr.conditions;
  if (expr.tag) return [expr];
  return [];
}

function buildExpression(conditions: TagExpression[], op: 'AND' | 'OR'): TagExpression | undefined {
  if (conditions.length === 0) return undefined;
  if (conditions.length === 1) return conditions[0];
  return { op, conditions };
}

const inputStyle: React.CSSProperties = {
  display: 'flex',
  height: 32,
  width: '100%',
  borderRadius: 6,
  border: '1px solid var(--border)',
  background: 'var(--bg-card)',
  padding: '0 8px',
  fontSize: 12,
  fontFamily: 'var(--font-sans)',
  color: 'var(--text-primary)',
  outline: 'none',
  boxSizing: 'border-box',
};

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: 10,
  fontFamily: 'var(--font-mono)',
  color: 'var(--text-muted)',
  marginBottom: 4,
  textTransform: 'uppercase',
  letterSpacing: '0.04em',
};

export function TagExpressionBuilder({
  value,
  onChange,
  endpointCount,
  endpointCountLoading,
}: TagExpressionBuilderProps) {
  const [selectedTag, setSelectedTag] = useState('');
  const [tagValue, setTagValue] = useState('');
  const [combineOp, setCombineOp] = useState<'AND' | 'OR'>(
    value?.op === 'AND' || value?.op === 'OR' ? value.op : 'AND',
  );
  const { data: tagsData } = useTags({ limit: 100 });

  const conditions = getConditions(value);
  const tags = tagsData ?? [];

  const addCondition = () => {
    if (!selectedTag) return;
    const newCondition: TagExpression = { tag: selectedTag, value: tagValue || undefined };
    const updated = [...conditions, newCondition];
    onChange(buildExpression(updated, combineOp));
    setSelectedTag('');
    setTagValue('');
  };

  const removeCondition = (index: number) => {
    const updated = conditions.filter((_, i) => i !== index);
    onChange(buildExpression(updated, combineOp));
  };

  const toggleOp = () => {
    const newOp = combineOp === 'AND' ? 'OR' : 'AND';
    setCombineOp(newOp);
    if (conditions.length > 1) {
      onChange(buildExpression(conditions, newOp));
    }
  };

  const dotColor = endpointCountLoading
    ? 'var(--signal-warning)'
    : conditions.length > 0
      ? 'var(--signal-healthy)'
      : 'var(--text-muted)';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* Current conditions */}
      {conditions.length > 0 && (
        <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 6 }}>
          {conditions.map((cond, idx) => {
            const tagColor = TAG_COLORS[idx % TAG_COLORS.length];
            return (
              <div key={idx} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                {idx > 0 && (
                  <button
                    type="button"
                    onClick={toggleOp}
                    style={{
                      borderRadius: 4,
                      padding: '2px 6px',
                      fontSize: 10,
                      fontWeight: 700,
                      fontFamily: 'var(--font-mono)',
                      cursor: 'pointer',
                      border: 'none',
                      transition: 'all 0.1s ease',
                      background:
                        combineOp === 'AND'
                          ? 'color-mix(in srgb, var(--accent) 15%, transparent)'
                          : 'color-mix(in srgb, var(--signal-warning) 15%, transparent)',
                      color: combineOp === 'AND' ? 'var(--accent)' : 'var(--signal-warning)',
                    }}
                  >
                    {combineOp}
                  </button>
                )}
                <span
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: 4,
                    borderRadius: 4,
                    border: `1px solid ${tagColor.border}`,
                    padding: '2px 8px 2px 8px',
                    fontSize: 11,
                    fontFamily: 'var(--font-mono)',
                    background: tagColor.bg,
                    color: tagColor.color,
                  }}
                >
                  <span style={{ fontWeight: 600 }}>{cond.tag}</span>
                  {cond.value && (
                    <>
                      <span style={{ opacity: 0.5 }}>:</span>
                      <span>{cond.value}</span>
                    </>
                  )}
                  <button
                    type="button"
                    onClick={() => removeCondition(idx)}
                    style={{
                      background: 'none',
                      border: 'none',
                      cursor: 'pointer',
                      padding: '1px',
                      borderRadius: 9999,
                      display: 'flex',
                      alignItems: 'center',
                      color: tagColor.color,
                      opacity: 0.7,
                      marginLeft: 2,
                    }}
                  >
                    <X style={{ width: 9, height: 9 }} />
                  </button>
                </span>
              </div>
            );
          })}
        </div>
      )}

      {/* Add condition row */}
      <div style={{ display: 'flex', alignItems: 'flex-end', gap: 8 }}>
        <div style={{ flex: 1 }}>
          <label style={labelStyle}>Tag</label>
          <select
            value={selectedTag}
            onChange={(e) => setSelectedTag(e.target.value)}
            style={inputStyle}
          >
            <option value="">Select tag...</option>
            {tags.map((t) => (
              <option key={t.id} value={`${t.key}:${t.value}`}>
                {t.key}:{t.value} ({t.endpoint_count ?? 0})
              </option>
            ))}
          </select>
        </div>
        <div style={{ flex: 1 }}>
          <label style={labelStyle}>Value (optional)</label>
          <input
            type="text"
            value={tagValue}
            onChange={(e) => setTagValue(e.target.value)}
            placeholder="e.g. production"
            style={inputStyle}
          />
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={addCondition}
          disabled={!selectedTag}
          style={{ height: 32, padding: '0 8px' }}
        >
          <Plus style={{ width: 14, height: 14 }} />
        </Button>
      </div>

      {/* Endpoint count */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          borderRadius: 6,
          background: 'var(--bg-inset)',
          border: '1px solid var(--border)',
          padding: '8px 12px',
        }}
      >
        <div
          style={{
            width: 8,
            height: 8,
            borderRadius: '50%',
            background: dotColor,
            flexShrink: 0,
            animation: endpointCountLoading
              ? 'pulse 2s cubic-bezier(0.4,0,0.6,1) infinite'
              : undefined,
          }}
        />
        <span
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 11,
            color: 'var(--text-secondary)',
          }}
        >
          {endpointCountLoading
            ? 'Resolving targets...'
            : conditions.length > 0
              ? `${endpointCount ?? 0} endpoint${endpointCount !== 1 ? 's' : ''} match this criteria`
              : endpointCount != null
                ? `All endpoints (${endpointCount} total)`
                : 'Add tag conditions to target specific endpoints'}
        </span>
      </div>
    </div>
  );
}
