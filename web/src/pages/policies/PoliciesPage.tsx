import { useState, useMemo, Fragment } from 'react';
import { Link, useNavigate } from 'react-router';
import { Plus, RefreshCw, Upload, ChevronDown, ChevronRight } from 'lucide-react';
import { Skeleton } from '@patchiq/ui';
import {
  usePolicies,
  useTogglePolicy,
  useBulkPolicyAction,
  useEvaluatePolicy,
} from '../../api/hooks/usePolicies';
import { DataTablePagination } from '../../components/data-table';
import { timeAgo } from '../../lib/time';
import type { components } from '../../api/types';
import { useCan } from '../../app/auth/AuthContext';
import { CreatePolicyDialog } from './CreatePolicyDialog';
import './policies.css';

type Policy = components['schemas']['Policy'];
// eslint-disable-next-line @typescript-eslint/no-explicit-any -- PolicyDetail fields not yet in generated types
type AnyPolicy = any;

// ─── Constants ────────────────────────────────────────────────────────────────

const modeColor: Record<string, string> = {
  automatic: 'var(--signal-healthy)',
  manual: 'var(--accent)',
  advisory: 'var(--text-muted)',
};

const modeLabel: Record<string, string> = {
  automatic: 'Automatic',
  manual: 'Manual',
  advisory: 'Advisory',
};

// ─── Shared cell style constants ──────────────────────────────────────────────

const TH: React.CSSProperties = {
  padding: '9px 12px',
  textAlign: 'left',
  fontFamily: 'var(--font-mono)',
  fontSize: 10,
  fontWeight: 600,
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  whiteSpace: 'nowrap',
};

const TD: React.CSSProperties = {
  padding: '10px 12px',
  borderBottom: '1px solid var(--border)',
  verticalAlign: 'middle',
  fontSize: 13,
};

// ─── Stat Card ────────────────────────────────────────────────────────────────

interface StatCardProps {
  label: string;
  value: number | undefined;
  valueColor?: string;
  active?: boolean;
  onClick: () => void;
}

function StatCard({ label, value, valueColor, active, onClick }: StatCardProps) {
  const [hovered, setHovered] = useState(false);
  return (
    <button
      type="button"
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        flex: 1,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        padding: '12px 14px',
        background: active ? 'color-mix(in srgb, white 3%, transparent)' : 'var(--bg-card)',
        border: `1px solid ${active ? (valueColor ?? 'var(--accent)') : hovered ? 'var(--border-hover)' : 'var(--border)'}`,
        borderRadius: 8,
        cursor: 'pointer',
        transition: 'all 0.15s',
        outline: 'none',
        textAlign: 'left',
      }}
    >
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 22,
          fontWeight: 700,
          lineHeight: 1,
          color: valueColor ?? 'var(--text-emphasis)',
          letterSpacing: '-0.02em',
        }}
      >
        {value ?? '—'}
      </span>
      <span
        style={{
          fontSize: 10,
          fontFamily: 'var(--font-mono)',
          fontWeight: 500,
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
          color: active ? (valueColor ?? 'var(--accent)') : 'var(--text-muted)',
          marginTop: 4,
        }}
      >
        {label}
      </span>
    </button>
  );
}

// ─── Custom Checkbox ──────────────────────────────────────────────────────────

function CB({ on, onClick }: { on: boolean; onClick: (e: React.MouseEvent) => void }) {
  return (
    <div
      onClick={onClick}
      style={{
        width: 14,
        height: 14,
        borderRadius: 3,
        border: `1.5px solid ${on ? 'var(--accent)' : 'var(--border-hover)'}`,
        background: on ? 'var(--accent)' : 'transparent',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        cursor: 'pointer',
        flexShrink: 0,
      }}
    >
      {on && (
        <svg width="8" height="6" viewBox="0 0 8 6" fill="none">
          <path
            d="M1 3L3 5L7 1"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      )}
    </div>
  );
}

// ─── Toggle Switch ────────────────────────────────────────────────────────────

function ToggleSwitch({ policy }: { policy: Policy }) {
  const toggle = useTogglePolicy(policy.id);
  const can = useCan();
  const on = policy.enabled;
  return (
    <button
      type="button"
      title={
        !can('policies', 'update')
          ? "You don't have permission"
          : on
            ? 'Disable policy'
            : 'Enable policy'
      }
      onClick={(e) => {
        e.stopPropagation();
        toggle.mutate(!on);
      }}
      disabled={toggle.isPending || !can('policies', 'update')}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        width: 32,
        height: 18,
        borderRadius: 100,
        background: on ? 'var(--signal-healthy)' : 'var(--bg-inset)',
        border: `1px solid ${on ? 'var(--signal-healthy)' : 'var(--border-hover)'}`,
        cursor: toggle.isPending ? 'wait' : !can('policies', 'update') ? 'not-allowed' : 'pointer',
        opacity: !can('policies', 'update') ? 0.5 : 1,
        padding: 2,
        transition: 'background 0.2s, border-color 0.2s',
        flexShrink: 0,
        outline: 'none',
      }}
    >
      <span
        style={{
          display: 'block',
          width: 12,
          height: 12,
          borderRadius: '50%',
          background: on ? '#fff' : 'var(--text-muted)',
          transform: on ? 'translateX(14px)' : 'translateX(0)',
          transition: 'transform 0.2s, background 0.2s',
          flexShrink: 0,
        }}
      />
    </button>
  );
}

// ─── Expand Indicator ─────────────────────────────────────────────────────────

function ExpandButton({
  expanded,
  onClick,
}: {
  expanded: boolean;
  onClick: (e: React.MouseEvent) => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        background: 'transparent',
        border: 'none',
        color: 'var(--text-muted)',
        cursor: 'pointer',
        padding: '2px',
        borderRadius: 3,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        transition: 'color 0.15s',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.color = 'var(--text-primary)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.color = 'var(--text-muted)';
      }}
    >
      {expanded ? (
        <ChevronDown className="h-3.5 w-3.5" />
      ) : (
        <ChevronRight className="h-3.5 w-3.5" />
      )}
    </button>
  );
}

// ─── Expanded Row ─────────────────────────────────────────────────────────────

const secLbl: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 10,
  fontWeight: 600,
  letterSpacing: '0.06em',
  textTransform: 'uppercase',
  color: 'var(--text-muted)',
  marginBottom: 10,
};

function ExpandedRowContent({ policy }: { policy: Policy }) {
  const can = useCan();
  const evaluate = useEvaluatePolicy(policy.id);
  const policyAny = policy as AnyPolicy;
  const groups = policyAny.group_names ?? [];
  const endpointCount = policyAny.target_endpoints_count ?? 0;
  const compliantCount = policyAny.last_eval_compliant_count ?? 0;
  const totalCount = policyAny.last_eval_endpoint_count ?? 0;
  const nonCompliant = totalCount - compliantCount;

  const CARD: React.CSSProperties = {
    background: 'var(--bg-inset)',
    border: '1px solid var(--border)',
    borderRadius: 6,
    padding: '12px 14px',
  };
  const TH: React.CSSProperties = {
    padding: '6px 10px',
    textAlign: 'left',
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 600,
    letterSpacing: '0.05em',
    color: 'var(--text-muted)',
    textTransform: 'uppercase',
    borderBottom: '1px solid var(--border)',
  };
  const TD: React.CSSProperties = {
    padding: '6px 10px',
    fontSize: 12,
    color: 'var(--text-primary)',
    borderBottom: '1px solid var(--border)',
  };

  return (
    <tr>
      <td colSpan={11} style={{ padding: 0, borderBottom: '1px solid var(--border)' }}>
        <div
          style={{
            padding: '8px 10px',
            background: 'var(--bg-page)',
            borderTop: '1px solid var(--border)',
            display: 'flex',
            gap: 8,
            alignItems: 'stretch',
          }}
        >
          {/* Groups table */}
          <div style={{ ...CARD, flex: '0 0 500px' }}>
            <div style={secLbl}>Groups</div>
            {groups.length > 0 ? (
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr>
                    <th style={TH}>Group</th>
                    <th style={{ ...TH, textAlign: 'right' }}>Endpoints</th>
                  </tr>
                </thead>
                <tbody>
                  {/* eslint-disable-next-line @typescript-eslint/no-explicit-any -- untyped group_names */}
                  {groups.map((name: any) => (
                    <tr key={name}>
                      <td style={TD}>{name}</td>
                      <td
                        style={{
                          ...TD,
                          textAlign: 'right',
                          fontFamily: 'var(--font-mono)',
                          color: 'var(--text-muted)',
                        }}
                      >
                        {endpointCount}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>No groups assigned</span>
            )}
          </div>

          {/* Evaluation Summary table */}
          <div style={{ ...CARD, flex: '0 0 480px', marginLeft: 24 }}>
            <div style={secLbl}>Evaluation Summary</div>
            {totalCount > 0 ? (
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr>
                    <th style={TH}>Metric</th>
                    <th style={{ ...TH, textAlign: 'right' }}>Count</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td style={TD}>Total Endpoints</td>
                    <td
                      style={{
                        ...TD,
                        textAlign: 'right',
                        fontFamily: 'var(--font-mono)',
                        color: 'var(--text-primary)',
                      }}
                    >
                      {totalCount}
                    </td>
                  </tr>
                  <tr>
                    <td style={TD}>Compliant</td>
                    <td
                      style={{
                        ...TD,
                        textAlign: 'right',
                        fontFamily: 'var(--font-mono)',
                        color: 'var(--signal-healthy)',
                      }}
                    >
                      {compliantCount}
                    </td>
                  </tr>
                  <tr>
                    <td style={TD}>Need Patches</td>
                    <td
                      style={{
                        ...TD,
                        textAlign: 'right',
                        fontFamily: 'var(--font-mono)',
                        color: nonCompliant > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
                      }}
                    >
                      {nonCompliant}
                    </td>
                  </tr>
                </tbody>
              </table>
            ) : (
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>No evaluations yet</span>
            )}
            <div style={{ display: 'flex', gap: 6 }}>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  evaluate.mutate();
                }}
                disabled={evaluate.isPending || !can('policies', 'read')}
                title={!can('policies', 'read') ? "You don't have permission" : undefined}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '5px 10px',
                  background: 'transparent',
                  color: 'var(--text-secondary)',
                  border: '1px solid var(--border)',
                  borderRadius: 5,
                  fontSize: 11,
                  cursor:
                    evaluate.isPending || !can('policies', 'read') ? 'not-allowed' : 'pointer',
                  fontFamily: 'var(--font-sans)',
                  opacity: !can('policies', 'read') ? 0.5 : 1,
                }}
              >
                <RefreshCw style={{ width: 11, height: 11 }} />
                {evaluate.isPending ? 'Evaluating...' : 'Evaluate Now'}
              </button>
              <button
                type="button"
                onClick={(e) => e.stopPropagation()}
                disabled={!can('deployments', 'create')}
                title={!can('deployments', 'create') ? "You don't have permission" : undefined}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '5px 10px',
                  background: 'var(--accent)',
                  color: 'var(--btn-accent-text, #000)',
                  border: 'none',
                  borderRadius: 5,
                  fontSize: 11,
                  fontWeight: 600,
                  cursor: !can('deployments', 'create') ? 'not-allowed' : 'pointer',
                  fontFamily: 'var(--font-sans)',
                  opacity: !can('deployments', 'create') ? 0.5 : 1,
                }}
              >
                <Upload style={{ width: 11, height: 11 }} />
                Deploy Now
              </button>
            </div>
          </div>
        </div>
      </td>
    </tr>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export const PoliciesPage = () => {
  const [modeFilter, setModeFilter] = useState<'automatic' | 'manual' | 'advisory' | undefined>();
  const [enabledFilter, setEnabledFilter] = useState<string | undefined>();
  const [search, setSearch] = useState('');
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [cursors, setCursors] = useState<string[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const currentCursor = cursors[cursors.length - 1];

  const can = useCan();
  const bulkAction = useBulkPolicyAction();
  const navigate = useNavigate();

  const { data, isLoading, isError, refetch } = usePolicies({
    cursor: currentCursor,
    limit: 25,
    mode: modeFilter,
    enabled: enabledFilter,
    search: search || undefined,
  });

  const policies = useMemo(() => data?.data ?? [], [data?.data]);

  // ─── Stat Counts ────────────────────────────────────────────────────────────
  const stats = useMemo(() => {
    let automatic = 0;
    let enabled = 0;
    let failedEval = 0;
    for (const p of policies) {
      const px = p as AnyPolicy;
      if (p.mode === 'automatic') automatic++;
      if (p.enabled) enabled++;
      if (px.last_eval_pass === false) failedEval++;
    }
    return { automatic, enabled, failedEval };
  }, [policies]);

  const selectedIds = useMemo(() => [...selected], [selected]);

  const togSel = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const togExp = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const allSel = policies.length > 0 && selected.size === policies.length;
  const togAll = () => {
    setSelected(allSel ? new Set() : new Set(policies.map((p) => p.id)));
  };

  if (isError) {
    return (
      <div style={{ padding: 24 }}>
        <div
          style={{
            borderRadius: 8,
            border: '1px solid color-mix(in srgb, var(--signal-critical) 30%, transparent)',
            background: 'color-mix(in srgb, var(--signal-critical) 8%, transparent)',
            padding: '12px 16px',
            fontSize: 13,
            color: 'var(--signal-critical)',
          }}
        >
          Failed to load policies.{' '}
          <button
            onClick={() => refetch()}
            style={{
              textDecoration: 'underline',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              color: 'inherit',
              fontSize: 'inherit',
            }}
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div
      style={{
        background: 'var(--bg-page)',
        padding: 24,
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        minHeight: '100%',
      }}
    >
      {/* ─── Stat Cards ─────────────────────────────────────────────────────── */}
      <div style={{ display: 'flex', gap: 8 }}>
        <StatCard
          label="Total"
          value={data?.total_count}
          active={!modeFilter && !enabledFilter}
          onClick={() => {
            setModeFilter(undefined);
            setEnabledFilter(undefined);
            setCursors([]);
          }}
        />
        <StatCard
          label="Automatic"
          value={stats.automatic}
          valueColor="var(--signal-healthy)"
          active={modeFilter === 'automatic'}
          onClick={() => {
            setModeFilter(modeFilter === 'automatic' ? undefined : 'automatic');
            setCursors([]);
          }}
        />
        <StatCard
          label="Enabled"
          value={stats.enabled}
          valueColor="var(--accent)"
          active={enabledFilter === 'true'}
          onClick={() => {
            setEnabledFilter(enabledFilter === 'true' ? undefined : 'true');
            setCursors([]);
          }}
        />
        <StatCard
          label="Failed Eval"
          value={stats.failedEval}
          valueColor="var(--signal-critical)"
          onClick={() => {}}
        />
      </div>

      {/* ─── Filter Bar + Actions ───────────────────────────────────────────── */}
      <div style={{ display: 'flex', alignItems: 'stretch', gap: 8 }}>
        {/* Filter Bar */}
        <div
          style={{
            flex: 1,
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: '10px 14px',
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            flexWrap: 'wrap',
            boxShadow: 'var(--shadow-sm)',
          }}
        >
          {/* Search */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '5px 10px',
              background: 'var(--bg-inset)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              flex: 1,
              maxWidth: 360,
              transition: 'border-color 0.15s',
            }}
            onFocusCapture={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlurCapture={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          >
            <svg
              width="12"
              height="12"
              viewBox="0 0 24 24"
              fill="none"
              stroke="var(--text-muted)"
              strokeWidth="2.5"
              aria-hidden="true"
            >
              <circle cx="11" cy="11" r="8" />
              <path d="M21 21l-4.35-4.35" />
            </svg>
            <input
              type="text"
              aria-label="Search policies"
              placeholder="Search policies..."
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setCursors([]);
              }}
              style={{
                background: 'transparent',
                border: 'none',
                outline: 'none',
                fontSize: 12,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-sans)',
                width: '100%',
              }}
            />
            {search && (
              <button
                type="button"
                aria-label="Clear search"
                onClick={() => {
                  setSearch('');
                  setCursors([]);
                }}
                style={{
                  width: 16,
                  height: 16,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'transparent',
                  border: 'none',
                  cursor: 'pointer',
                  color: 'var(--text-muted)',
                  padding: 0,
                }}
              >
                <svg
                  width="10"
                  height="10"
                  viewBox="0 0 10 10"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                >
                  <path d="M2 2l6 6M8 2l-6 6" />
                </svg>
              </button>
            )}
          </div>

          {/* Mode filter pills */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            {(
              [
                ['All', undefined, 'var(--accent)'],
                ['Automatic', 'automatic', 'var(--signal-healthy)'],
                ['Manual', 'manual', 'var(--accent)'],
                ['Advisory', 'advisory', 'var(--text-muted)'],
              ] as const
            ).map(([label, value, color]) => {
              const active = modeFilter === value;
              return (
                <button
                  key={label}
                  type="button"
                  onClick={() => {
                    setModeFilter(value);
                    setCursors([]);
                  }}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    padding: '3px 9px',
                    borderRadius: 100,
                    fontSize: 11,
                    fontWeight: 500,
                    cursor: 'pointer',
                    fontFamily: 'var(--font-sans)',
                    border: `1px solid ${active ? `color-mix(in srgb, ${color} 30%, transparent)` : 'transparent'}`,
                    background: active
                      ? `color-mix(in srgb, ${color} 10%, transparent)`
                      : 'transparent',
                    color: active ? color : 'var(--text-muted)',
                    transition: 'all 0.15s',
                  }}
                >
                  {label}
                </button>
              );
            })}
          </div>

          {/* Enabled pills */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            {(
              [
                ['All', undefined],
                ['Enabled', 'true'],
                ['Disabled', 'false'],
              ] as const
            ).map(([label, value]) => {
              const active = enabledFilter === value;
              return (
                <button
                  key={label}
                  type="button"
                  onClick={() => {
                    setEnabledFilter(value);
                    setCursors([]);
                  }}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    padding: '3px 9px',
                    borderRadius: 100,
                    fontSize: 11,
                    fontWeight: 500,
                    cursor: 'pointer',
                    fontFamily: 'var(--font-sans)',
                    border: `1px solid ${active ? 'color-mix(in srgb, var(--accent) 30%, transparent)' : 'transparent'}`,
                    background: active
                      ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
                      : 'transparent',
                    color: active ? 'var(--accent)' : 'var(--text-muted)',
                    transition: 'all 0.15s',
                  }}
                >
                  {label}
                </button>
              );
            })}
          </div>
        </div>

        {/* Create Policy button */}
        <button
          type="button"
          onClick={() => setCreateOpen(true)}
          disabled={!can('policies', 'create')}
          title={!can('policies', 'create') ? "You don't have permission" : undefined}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            padding: '5px 12px',
            borderRadius: 6,
            fontSize: 12,
            fontWeight: 600,
            cursor: !can('policies', 'create') ? 'not-allowed' : 'pointer',
            border: '1px solid var(--accent)',
            background: 'var(--accent)',
            color: 'var(--btn-accent-text, #000)',
            fontFamily: 'var(--font-sans)',
            whiteSpace: 'nowrap',
            flexShrink: 0,
            opacity: !can('policies', 'create') ? 0.5 : 1,
          }}
        >
          <Plus style={{ width: 13, height: 13 }} />
          Create Policy
        </button>
      </div>

      {/* ─── Bulk Action Bar ─────────────────────────────────────────────────── */}
      {selectedIds.length > 0 && (
        <div
          style={{
            position: 'sticky',
            top: 0,
            zIndex: 10,
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: '10px 16px',
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            boxShadow: 'var(--shadow-sm)',
          }}
        >
          <span style={{ fontSize: 12, color: 'var(--text-secondary)', marginRight: 4 }}>
            <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>
              {selectedIds.length}
            </span>{' '}
            selected
          </span>
          <button
            type="button"
            disabled={!can('deployments', 'create')}
            title={!can('deployments', 'create') ? "You don't have permission" : undefined}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 6,
              padding: '5px 12px',
              borderRadius: 6,
              fontSize: 12,
              fontWeight: 500,
              cursor: !can('deployments', 'create') ? 'not-allowed' : 'pointer',
              border: '1px solid var(--accent-border)',
              background: 'var(--accent-subtle)',
              color: 'var(--accent)',
              fontFamily: 'var(--font-sans)',
              opacity: !can('deployments', 'create') ? 0.5 : 1,
            }}
          >
            Deploy Selected
          </button>
          <button
            type="button"
            onClick={() => bulkAction.mutate({ ids: selectedIds, action: 'enable' })}
            disabled={!can('policies', 'update')}
            title={!can('policies', 'update') ? "You don't have permission" : undefined}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              padding: '5px 10px',
              borderRadius: 6,
              fontSize: 12,
              color: 'var(--text-secondary)',
              border: '1px solid var(--border)',
              background: 'transparent',
              cursor: !can('policies', 'update') ? 'not-allowed' : 'pointer',
              fontFamily: 'var(--font-sans)',
              opacity: !can('policies', 'update') ? 0.5 : 1,
            }}
          >
            Enable
          </button>
          <button
            type="button"
            onClick={() => bulkAction.mutate({ ids: selectedIds, action: 'disable' })}
            disabled={!can('policies', 'update')}
            title={!can('policies', 'update') ? "You don't have permission" : undefined}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              padding: '5px 10px',
              borderRadius: 6,
              fontSize: 12,
              color: 'var(--text-secondary)',
              border: '1px solid var(--border)',
              background: 'transparent',
              cursor: !can('policies', 'update') ? 'not-allowed' : 'pointer',
              fontFamily: 'var(--font-sans)',
              opacity: !can('policies', 'update') ? 0.5 : 1,
            }}
          >
            Disable
          </button>
          <button
            type="button"
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              padding: '5px 10px',
              borderRadius: 6,
              fontSize: 12,
              color: 'var(--text-secondary)',
              border: '1px solid var(--border)',
              background: 'transparent',
              cursor: 'pointer',
              fontFamily: 'var(--font-sans)',
            }}
          >
            Clone
          </button>
          <button
            type="button"
            onClick={() => bulkAction.mutate({ ids: selectedIds, action: 'delete' })}
            disabled={!can('policies', 'delete')}
            title={!can('policies', 'delete') ? "You don't have permission" : undefined}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              padding: '5px 10px',
              borderRadius: 6,
              fontSize: 12,
              color: 'var(--signal-critical)',
              border: '1px solid color-mix(in srgb, var(--signal-critical) 25%, transparent)',
              background: 'color-mix(in srgb, var(--signal-critical) 8%, transparent)',
              cursor: !can('policies', 'delete') ? 'not-allowed' : 'pointer',
              fontFamily: 'var(--font-sans)',
              opacity: !can('policies', 'delete') ? 0.5 : 1,
            }}
          >
            Delete
          </button>
          <button
            type="button"
            onClick={() => setSelected(new Set())}
            style={{
              marginLeft: 'auto',
              fontSize: 12,
              color: 'var(--accent)',
              cursor: 'pointer',
              background: 'none',
              border: 'none',
              fontFamily: 'var(--font-sans)',
            }}
          >
            Clear
          </button>
        </div>
      )}

      {/* ─── Table / Loading / Empty ─────────────────────────────────────────── */}
      {isLoading ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-12 rounded-md" />
          ))}
        </div>
      ) : policies.length === 0 && !search && !modeFilter && !enabledFilter ? (
        <div
          style={{
            height: 300,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 13,
            color: 'var(--text-muted)',
          }}
        >
          No policies defined. Create your first policy to start managing patches.
        </div>
      ) : (
        <>
          <div
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              overflow: 'hidden',
              boxShadow: 'var(--shadow-sm)',
            }}
          >
            <div style={{ overflowX: 'auto' }}>
              <table
                style={{
                  width: '100%',
                  borderCollapse: 'separate',
                  borderSpacing: 0,
                  minWidth: 900,
                }}
              >
                <thead>
                  <tr
                    style={{
                      background: 'var(--bg-inset)',
                      borderBottom: '1px solid var(--border)',
                    }}
                  >
                    {/* Checkbox */}
                    <th
                      style={{ ...TH, width: 36, padding: '9px 12px 9px 16px', textAlign: 'left' }}
                    >
                      <CB
                        on={allSel}
                        onClick={(e) => {
                          e.stopPropagation();
                          togAll();
                        }}
                      />
                    </th>
                    {/* Expand */}
                    <th style={{ ...TH, width: 28 }} />
                    {/* Name */}
                    <th style={{ ...TH, width: 220 }}>Name</th>
                    {/* Mode */}
                    <th style={{ ...TH, width: 90 }}>Mode</th>
                    {/* Schedule */}
                    <th style={{ ...TH, width: 130 }}>Schedule</th>
                    {/* Groups */}
                    <th style={{ ...TH, width: 160 }}>Groups</th>
                    {/* Scope */}
                    <th style={{ ...TH, width: 100 }}>Scope</th>
                    {/* Enabled */}
                    <th style={{ ...TH, width: 60 }}>On</th>
                    {/* Endpoints */}
                    <th style={{ ...TH, width: 70, textAlign: 'center' }}>Endpoints</th>
                    {/* Last Eval */}
                    <th style={{ ...TH, width: 110 }}>Last Eval</th>
                    {/* Compliance */}
                    <th style={{ ...TH, width: 120 }}>Compliance</th>
                  </tr>
                </thead>
                <tbody>
                  {policies.length === 0 ? (
                    <tr>
                      <td
                        colSpan={11}
                        style={{
                          padding: '48px 24px',
                          textAlign: 'center',
                          fontSize: 13,
                          color: 'var(--text-muted)',
                        }}
                      >
                        No policies match the current filters.
                      </td>
                    </tr>
                  ) : (
                    policies.map((policy) => {
                      const px = policy as AnyPolicy;
                      const isExpanded = expanded.has(policy.id);
                      const isSel = selected.has(policy.id);
                      const groups: string[] = px.group_names ?? [];
                      const compliant = px.last_eval_compliant_count ?? 0;
                      const total = px.last_eval_endpoint_count ?? 0;
                      const mc = modeColor[policy.mode] ?? 'var(--text-muted)';
                      const ml = modeLabel[policy.mode] ?? policy.mode;
                      const isRecurring = policy.schedule_type === 'recurring';
                      const cron = px.schedule_cron as string | undefined;

                      return (
                        <Fragment key={policy.id}>
                          <PolicyRow
                            policy={policy}
                            px={px}
                            isSel={isSel}
                            isExpanded={isExpanded}
                            groups={groups}
                            compliant={compliant}
                            total={total}
                            mc={mc}
                            ml={ml}
                            isRecurring={isRecurring}
                            cron={cron}
                            onNavigate={() => navigate(`/policies/${policy.id}`)}
                            onToggleSel={(e) => {
                              e.stopPropagation();
                              togSel(policy.id);
                            }}
                            onToggleExp={(e) => {
                              e.stopPropagation();
                              togExp(policy.id);
                            }}
                          />
                          {isExpanded && <ExpandedRowContent policy={policy} />}
                        </Fragment>
                      );
                    })
                  )}
                </tbody>
              </table>
            </div>
          </div>

          <DataTablePagination
            hasNext={!!data?.next_cursor}
            hasPrev={cursors.length > 0}
            onNext={() => {
              if (data?.next_cursor) setCursors((prev) => [...prev, data.next_cursor!]);
            }}
            onPrev={() => setCursors((prev) => prev.slice(0, -1))}
          />
        </>
      )}

      <CreatePolicyDialog open={createOpen} onOpenChange={setCreateOpen} />
    </div>
  );
};

// ─── Policy Table Row (extracted to keep jsx concise) ─────────────────────────

function PolicyRow({
  policy,
  px,
  isSel,
  isExpanded,
  groups,
  compliant,
  total,
  mc,
  ml,
  isRecurring,
  cron,
  onNavigate,
  onToggleSel,
  onToggleExp,
}: {
  policy: Policy;
  px: AnyPolicy;
  isSel: boolean;
  isExpanded: boolean;
  groups: string[];
  compliant: number;
  total: number;
  mc: string;
  ml: string;
  isRecurring: boolean;
  cron: string | undefined;
  onNavigate: () => void;
  onToggleSel: (e: React.MouseEvent) => void;
  onToggleExp: (e: React.MouseEvent) => void;
}) {
  const [hovered, setHovered] = useState(false);

  const rowBg = isSel
    ? 'color-mix(in srgb, var(--accent) 5%, var(--bg-card))'
    : hovered
      ? 'var(--table-row-hover, rgba(255,255,255,0.02))'
      : 'transparent';

  return (
    <tr
      onClick={onNavigate}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        cursor: 'pointer',
        background: rowBg,
        transition: 'background 0.1s',
      }}
    >
      {/* Checkbox */}
      <td style={{ ...TD, width: 36, padding: '10px 12px 10px 16px' }}>
        <CB on={isSel} onClick={onToggleSel} />
      </td>

      {/* Expand */}
      <td style={{ ...TD, width: 28, padding: '10px 4px' }}>
        <ExpandButton expanded={isExpanded} onClick={onToggleExp} />
      </td>

      {/* Name */}
      <td style={{ ...TD, width: 220 }}>
        <Link
          to={`/policies/${policy.id}`}
          onClick={(e) => e.stopPropagation()}
          style={{
            color: 'var(--accent)',
            fontWeight: 500,
            textDecoration: 'none',
            display: 'block',
            maxWidth: 200,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            fontSize: 13,
          }}
          title={policy.name}
        >
          {policy.name}
        </Link>
      </td>

      {/* Mode */}
      <td style={{ ...TD, width: 90 }}>
        <span
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            padding: '2px 8px',
            borderRadius: 100,
            fontSize: 10,
            fontFamily: 'var(--font-mono)',
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.04em',
            border: `1px solid color-mix(in srgb, ${mc} 30%, transparent)`,
            background: `color-mix(in srgb, ${mc} 10%, transparent)`,
            color: mc,
            whiteSpace: 'nowrap',
          }}
        >
          {ml}
        </span>
      </td>

      {/* Schedule */}
      <td style={{ ...TD, width: 130 }}>
        {isRecurring ? (
          <div>
            <div
              style={{
                fontSize: 11,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-primary)',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                maxWidth: 115,
              }}
              title={cron ?? 'Recurring'}
            >
              {cron ?? 'Recurring'}
            </div>
            <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 1 }}>
              Next: scheduled
            </div>
          </div>
        ) : (
          <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>On demand</span>
        )}
      </td>

      {/* Groups */}
      <td style={{ ...TD, width: 160 }}>
        {groups.length === 0 ? (
          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>—</span>
        ) : (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 3, maxWidth: 145 }}>
            {groups.slice(0, 2).map((name) => (
              <span
                key={name}
                title={name}
                style={{
                  display: 'inline-block',
                  maxWidth: 68,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  padding: '2px 6px',
                  background: 'var(--bg-inset)',
                  border: '1px solid var(--border)',
                  borderRadius: 4,
                  fontSize: 10,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-secondary)',
                }}
              >
                {name}
              </span>
            ))}
            {groups.length > 2 && (
              <span
                style={{
                  padding: '2px 6px',
                  background: 'var(--bg-inset)',
                  border: '1px solid var(--border)',
                  borderRadius: 4,
                  fontSize: 10,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-muted)',
                }}
              >
                +{groups.length - 2}
              </span>
            )}
          </div>
        )}
      </td>

      {/* Scope */}
      <td style={{ ...TD, width: 100 }}>
        {(() => {
          const parts: string[] = [];
          if (policy.min_severity) parts.push(policy.min_severity);
          if (policy.selection_mode) parts.push(policy.selection_mode.replace('by_', ''));
          const text = parts.length > 0 ? parts.join(', ') : '—';
          return (
            <span
              title={text}
              style={{
                fontSize: 11,
                color: 'var(--text-muted)',
                display: 'block',
                maxWidth: 88,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {text}
            </span>
          );
        })()}
      </td>

      {/* Enabled toggle */}
      <td style={{ ...TD, width: 60 }} onClick={(e) => e.stopPropagation()}>
        <ToggleSwitch policy={policy} />
      </td>

      {/* Endpoints */}
      <td style={{ ...TD, width: 70, textAlign: 'center' }}>
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 13,
            fontWeight: 600,
            color: 'var(--text-primary)',
          }}
        >
          {px.target_endpoints_count ?? 0}
        </span>
      </td>

      {/* Last Eval */}
      <td style={{ ...TD, width: 110 }}>
        {!px.last_evaluated_at ? (
          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>—</span>
        ) : (
          <div>
            <span
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 3,
                fontSize: 11,
                fontWeight: 600,
                color: px.last_eval_pass ? 'var(--signal-healthy)' : 'var(--signal-critical)',
              }}
            >
              {px.last_eval_pass ? '✓ Pass' : '✗ Fail'}
            </span>
            <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 1 }}>
              {timeAgo(px.last_evaluated_at)}
            </div>
          </div>
        )}
      </td>

      {/* Compliance */}
      <td style={{ ...TD, width: 120 }}>
        {!px.last_evaluated_at ? (
          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>—</span>
        ) : (
          <div>
            <div style={{ fontSize: 11, color: 'var(--text-primary)' }}>
              {compliant}/{total} compliant
            </div>
            <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 1 }}>
              {timeAgo(px.last_evaluated_at)}
            </div>
          </div>
        )}
      </td>
    </tr>
  );
}
