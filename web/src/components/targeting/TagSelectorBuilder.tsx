import { useEffect, useMemo, useState } from 'react';
import type { PredicateRow, Selector } from '../../types/targeting';
import { rowsToSelector, selectorToRows } from '../../types/targeting';
import { useDistinctTagKeys } from '../../api/hooks/useTagKeys';
import { useValidateSelector } from '../../api/hooks/useTagSelector';

interface Props {
  value: Selector | null | undefined;
  onChange: (next: Selector | null) => void;
  disabled?: boolean;
}

/**
 * Minimal, ships-the-contract-end-to-end selector builder. Supports a
 * flat list of predicates joined by AND: [key eq value], [key in values],
 * [key exists]. Nested boolean expressions (OR, NOT, mixed groups) are a
 * planned follow-up — the underlying AST already supports them, so a
 * richer UI can land without touching the backend contract.
 *
 * Live match count is driven by /api/v1/tags/selectors/validate, debounced
 * at 400ms so typing doesn't spam the server.
 */
export function TagSelectorBuilder({ value, onChange, disabled }: Props) {
  const [rows, setRows] = useState<PredicateRow[]>(() => selectorToRows(value));
  const [debounced, setDebounced] = useState<Selector | null>(() => rowsToSelector(rows));

  const { data: knownKeys } = useDistinctTagKeys();

  // Sync local row state when the parent flips `value` (e.g. form reset).
  useEffect(() => {
    setRows(selectorToRows(value));
  }, [value]);

  // Debounce AST emission so the live-preview call isn't spammed.
  const liveSelector = useMemo(() => rowsToSelector(rows), [rows]);
  useEffect(() => {
    const t = window.setTimeout(() => setDebounced(liveSelector), 400);
    return () => window.clearTimeout(t);
  }, [liveSelector]);

  // Emit changes to parent immediately — the debounce only affects the
  // preview query, not the saved-on-submit AST.
  useEffect(() => {
    onChange(liveSelector);
  }, [liveSelector, onChange]);

  const validation = useValidateSelector(debounced);

  const addRow = () => {
    setRows((prev) => [
      ...prev,
      { id: crypto.randomUUID(), key: '', op: 'eq', value: '', values: [] },
    ]);
  };

  const updateRow = (id: string, patch: Partial<PredicateRow>) => {
    setRows((prev) => prev.map((r) => (r.id === id ? { ...r, ...patch } : r)));
  };

  const removeRow = (id: string) => {
    setRows((prev) => prev.filter((r) => r.id !== id));
  };

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        padding: 12,
        border: '1px solid var(--border)',
        borderRadius: 8,
        background: 'var(--bg-card)',
      }}
    >
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          fontSize: 12,
          color: 'var(--text-secondary)',
        }}
      >
        <span>
          Target endpoints matching <strong>all</strong> of the following tag predicates:
        </span>
        <MatchCountBadge validation={validation} hasRows={rows.length > 0} />
      </div>

      {rows.length === 0 ? (
        <div style={{ fontSize: 12, color: 'var(--text-muted)', fontStyle: 'italic' }}>
          No predicates — policy will match zero endpoints until you add one.
        </div>
      ) : (
        rows.map((row, idx) => (
          <PredicateRowEditor
            key={row.id}
            row={row}
            isFirst={idx === 0}
            disabled={disabled}
            knownKeys={knownKeys ?? []}
            onChange={(patch) => updateRow(row.id, patch)}
            onRemove={() => removeRow(row.id)}
          />
        ))
      )}

      <div>
        <button
          type="button"
          onClick={addRow}
          disabled={disabled}
          style={{
            padding: '6px 12px',
            fontSize: 12,
            borderRadius: 6,
            border: '1px solid var(--border)',
            background: 'transparent',
            color: 'var(--text-primary)',
            cursor: disabled ? 'not-allowed' : 'pointer',
          }}
        >
          + Add predicate
        </button>
      </div>

      {validation.data?.valid === false && validation.data.error && (
        <div style={{ fontSize: 11, color: 'var(--danger, #e53935)' }}>
          {validation.data.error}
        </div>
      )}
    </div>
  );
}

interface RowProps {
  row: PredicateRow;
  isFirst: boolean;
  disabled?: boolean;
  knownKeys: string[];
  onChange: (patch: Partial<PredicateRow>) => void;
  onRemove: () => void;
}

function PredicateRowEditor({ row, isFirst, disabled, knownKeys, onChange, onRemove }: RowProps) {
  const [valuesDraft, setValuesDraft] = useState(row.values.join(', '));

  useEffect(() => {
    setValuesDraft(row.values.join(', '));
  }, [row.values]);

  const listId = `tag-keys-${row.id}`;

  return (
    <div style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' }}>
      <span
        style={{
          fontSize: 11,
          color: 'var(--text-muted)',
          width: 32,
          textAlign: 'center',
        }}
      >
        {isFirst ? 'WHERE' : 'AND'}
      </span>

      <input
        list={listId}
        value={row.key}
        placeholder="key (e.g. env)"
        disabled={disabled}
        onChange={(e) => onChange({ key: e.target.value })}
        style={inputStyle(140)}
      />
      <datalist id={listId}>
        {knownKeys.map((k) => (
          <option key={k} value={k} />
        ))}
      </datalist>

      <select
        value={row.op}
        disabled={disabled}
        onChange={(e) =>
          onChange({ op: e.target.value as PredicateRow['op'] })
        }
        style={inputStyle(90)}
      >
        <option value="eq">equals</option>
        <option value="in">in</option>
        <option value="exists">exists</option>
      </select>

      {row.op === 'eq' && (
        <input
          value={row.value}
          placeholder="value"
          disabled={disabled}
          onChange={(e) => onChange({ value: e.target.value })}
          style={inputStyle(180)}
        />
      )}

      {row.op === 'in' && (
        <input
          value={valuesDraft}
          placeholder="comma-separated values"
          disabled={disabled}
          onChange={(e) => {
            setValuesDraft(e.target.value);
            onChange({
              values: e.target.value
                .split(',')
                .map((v) => v.trim())
                .filter(Boolean),
            });
          }}
          style={inputStyle(240)}
        />
      )}

      <button
        type="button"
        onClick={onRemove}
        disabled={disabled}
        aria-label="remove predicate"
        style={{
          padding: '4px 8px',
          fontSize: 11,
          borderRadius: 4,
          border: '1px solid var(--border)',
          background: 'transparent',
          color: 'var(--text-secondary)',
          cursor: disabled ? 'not-allowed' : 'pointer',
        }}
      >
        ×
      </button>
    </div>
  );
}

interface MatchCountBadgeProps {
  validation: ReturnType<typeof useValidateSelector>;
  hasRows: boolean;
}

function MatchCountBadge({ validation, hasRows }: MatchCountBadgeProps) {
  if (!hasRows) {
    return <span style={badgeStyle('var(--text-muted)')}>0 endpoints</span>;
  }
  if (validation.isFetching) {
    return <span style={badgeStyle('var(--text-muted)')}>counting…</span>;
  }
  if (validation.data?.valid === false) {
    return <span style={badgeStyle('var(--danger, #e53935)')}>invalid</span>;
  }
  const count = validation.data?.matched_count ?? 0;
  return <span style={badgeStyle('var(--accent)')}>{count.toLocaleString()} endpoints</span>;
}

function inputStyle(width: number): React.CSSProperties {
  return {
    padding: '4px 8px',
    fontSize: 12,
    borderRadius: 4,
    border: '1px solid var(--border)',
    background: 'var(--bg-inset)',
    color: 'var(--text-primary)',
    width,
  };
}

function badgeStyle(color: string): React.CSSProperties {
  return {
    padding: '2px 8px',
    fontSize: 11,
    borderRadius: 12,
    border: `1px solid ${color}`,
    color,
    fontWeight: 500,
  };
}
