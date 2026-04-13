import { useState } from 'react';
import { useNavigate } from 'react-router';
import { Skeleton } from '@patchiq/ui';
import { useCan } from '../../../app/auth/AuthContext';
import { useEndpointPatches, useDeployCritical } from '../../../api/hooks/useEndpoints';

interface PatchesTabProps {
  endpointId: string;
}

type SeverityFilter = 'all' | 'critical' | 'high' | 'medium' | 'low';

// ── design tokens ──────────────────────────────────────────────
const S = {
  card: {
    background: 'var(--bg-card)',
    border: '1px solid var(--border)',
    borderRadius: 8,
    boxShadow: 'var(--shadow-sm)',
    overflow: 'hidden' as const,
  },
  cardTitle: {
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 500,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    color: 'var(--text-muted)',
    padding: '12px 16px',
    borderBottom: '1px solid var(--border)',
    background: 'var(--bg-inset)',
  },
  th: {
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 500,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    color: 'var(--text-muted)',
    padding: '9px 12px',
    background: 'var(--bg-inset)',
    borderBottom: '1px solid var(--border)',
    textAlign: 'left' as const,
    whiteSpace: 'nowrap' as const,
  },
  td: {
    padding: '10px 12px',
    borderBottom: '1px solid var(--border)',
    color: 'var(--text-primary)',
    fontSize: 13,
  },
};

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'var(--signal-critical)',
  high: 'var(--signal-warning)',
  medium: 'var(--text-faint)',
  low: 'var(--text-faint)',
  none: 'var(--text-faint)',
};

const STATUS_COLOR: Record<string, string> = {
  pending: 'var(--signal-warning)',
  sent: 'var(--signal-warning)',
  executing: 'var(--signal-warning)',
  running: 'var(--signal-warning)',
  succeeded: 'var(--signal-healthy)',
  failed: 'var(--signal-critical)',
  cancelled: 'var(--text-faint)',
};

function cvssColor(score: number): string {
  if (score >= 9) return 'var(--signal-critical)';
  if (score >= 7) return 'var(--signal-warning)';
  if (score >= 4) return 'var(--text-secondary)';
  return 'var(--text-faint)';
}

function severityToCvss(severity: string): number {
  switch (severity) {
    case 'critical':
      return 9.5;
    case 'high':
      return 7.5;
    case 'medium':
      return 5.0;
    case 'low':
      return 2.5;
    default:
      return 0;
  }
}

export function PatchesTab({ endpointId }: PatchesTabProps) {
  const navigate = useNavigate();
  const can = useCan();
  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>('all');
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [confirmingCritical, setConfirmingCritical] = useState(false);
  const [deployError, setDeployError] = useState<string | null>(null);

  const { data, isLoading, error } = useEndpointPatches(endpointId);
  const deployCritical = useDeployCritical();
  const deploySelected = useDeployCritical();

  const patches = data?.data ?? [];

  const criticalCount = patches.filter((p) => p.severity === 'critical').length;
  const highCount = patches.filter((p) => p.severity === 'high').length;
  const mediumCount = patches.filter((p) => p.severity === 'medium').length;
  const lowCount = patches.filter((p) => p.severity === 'low').length;
  const installedCount = patches.filter((p) => p.status === 'succeeded').length;
  const failedCount = patches.filter(
    (p) => p.status === 'failed' || p.status === 'cancelled',
  ).length;
  const pendingCount = patches.filter(
    (p) =>
      p.status === 'pending' ||
      p.status === 'sent' ||
      p.status === 'executing' ||
      p.status === 'running',
  ).length;

  const criticalPatches = patches.filter((p) => p.severity === 'critical');

  const filtered = patches.filter((p) => {
    if (severityFilter !== 'all' && p.severity !== severityFilter) return false;
    return true;
  });

  const pendingPatches = filtered.filter(
    (p) =>
      p.status === 'pending' ||
      p.status === 'sent' ||
      p.status === 'executing' ||
      p.status === 'running',
  );
  const installedPatches = filtered.filter((p) => p.status === 'succeeded');
  const availablePatches = filtered.filter((p) => p.status === 'available');

  const toggleSelect = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleAll = (list: typeof pendingPatches) => {
    const allIds = list.map((p) => p.id);
    const allSelected = allIds.every((id) => selected.has(id));
    setSelected((prev) => {
      const next = new Set(prev);
      if (allSelected) allIds.forEach((id) => next.delete(id));
      else allIds.forEach((id) => next.add(id));
      return next;
    });
  };

  const handleDeploySelected = async () => {
    setDeployError(null);
    try {
      await deploySelected.mutateAsync({
        endpointId,
        patchIds: Array.from(selected),
        name: `Selected patches - ${selected.size} patches`,
      });
      setSelected(new Set());
    } catch (err) {
      setDeployError(err instanceof Error ? err.message : 'Deployment failed');
    }
  };

  const handleDeployAllCritical = async () => {
    if (!confirmingCritical) {
      setConfirmingCritical(true);
      return;
    }
    setDeployError(null);
    try {
      await deployCritical.mutateAsync({
        endpointId,
        patchIds: criticalPatches.map((p) => p.id),
        name: `Critical patches - ${criticalPatches.length} patches`,
      });
      setConfirmingCritical(false);
    } catch (err) {
      setDeployError(err instanceof Error ? err.message : 'Deployment failed');
      setConfirmingCritical(false);
    }
  };

  if (isLoading) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ ...S.card, padding: 16 }}>
        <span style={{ fontSize: 13, color: 'var(--signal-critical)' }}>
          Failed to load patches.
        </span>
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Summary stat cards */}
      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' as const, marginBottom: 0 }}>
        {[
          {
            label: 'Pending',
            count: pendingCount,
            color: 'var(--signal-warning)',
            sub: `${criticalCount}C · ${highCount}H · ${mediumCount}M · ${lowCount}L`,
          },
          { label: 'Installed', count: installedCount, color: 'var(--signal-healthy)', sub: null },
          { label: 'Failed', count: failedCount, color: 'var(--signal-critical)', sub: null },
        ].map(({ label, count, color, sub }) => (
          <div
            key={label}
            style={{
              flex: '1 1 140px',
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              padding: '14px 16px',
            }}
          >
            <div
              style={{
                fontSize: 10,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-muted)',
                textTransform: 'uppercase' as const,
                letterSpacing: '0.04em',
              }}
            >
              {label}
            </div>
            <div
              style={{
                fontSize: 28,
                fontWeight: 700,
                fontFamily: 'var(--font-mono)',
                color,
                lineHeight: 1,
                marginTop: 4,
              }}
            >
              {count}
            </div>
            {sub && (
              <div
                style={{
                  fontSize: 10,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-muted)',
                  marginTop: 4,
                }}
              >
                {sub}
              </div>
            )}
          </div>
        ))}

        {/* Actions */}
        <div style={{ marginLeft: 'auto', display: 'flex', gap: 8, alignItems: 'flex-end' }}>
          {selected.size > 0 && (
            <button
              disabled={!can('deployments', 'create') || deploySelected.isPending}
              title={!can('deployments', 'create') ? "You don't have permission" : undefined}
              onClick={() => void handleDeploySelected()}
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 12,
                padding: '6px 14px',
                borderRadius: 6,
                border: '1px solid var(--accent)',
                background: 'var(--accent)',
                color: 'var(--btn-accent-text, #000)',
                cursor: !can('deployments', 'create') ? 'not-allowed' : 'pointer',
                fontWeight: 600,
                opacity: !can('deployments', 'create') ? 0.5 : 1,
              }}
            >
              {deploySelected.isPending ? 'Deploying...' : `Deploy (${selected.size})`}
            </button>
          )}
          {criticalCount > 0 && (
            <button
              disabled={!can('deployments', 'create') || deployCritical.isPending}
              title={!can('deployments', 'create') ? "You don't have permission" : undefined}
              onClick={() => void handleDeployAllCritical()}
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 12,
                padding: '6px 14px',
                borderRadius: 6,
                border: '1px solid var(--signal-critical)',
                background: confirmingCritical ? 'var(--signal-critical)' : 'transparent',
                color: confirmingCritical ? 'var(--text-on-color, #fff)' : 'var(--signal-critical)',
                cursor: !can('deployments', 'create') ? 'not-allowed' : 'pointer',
                fontWeight: 600,
                opacity: !can('deployments', 'create') ? 0.5 : 1,
              }}
            >
              {deployCritical.isPending
                ? 'Deploying...'
                : confirmingCritical
                  ? `Confirm (${criticalPatches.length})`
                  : 'Deploy Critical'}
            </button>
          )}
        </div>
      </div>

      {/* Confirm warning */}
      {confirmingCritical && !deployCritical.isPending && (
        <div
          style={{
            ...S.card,
            padding: '10px 16px',
            borderColor: 'var(--signal-warning)',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <span style={{ fontSize: 12, color: 'var(--signal-warning)' }}>
            This will deploy {criticalPatches.length} critical patch
            {criticalPatches.length !== 1 ? 'es' : ''}. Click confirm to proceed.
          </span>
          <button
            onClick={() => setConfirmingCritical(false)}
            style={{
              fontSize: 11,
              color: 'var(--text-muted)',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              textDecoration: 'underline',
            }}
          >
            Cancel
          </button>
        </div>
      )}

      {deployError && (
        <div style={{ ...S.card, padding: '10px 16px', borderColor: 'var(--signal-critical)' }}>
          <span style={{ fontSize: 12, color: 'var(--signal-critical)' }}>{deployError}</span>
        </div>
      )}

      {/* Severity filter pills */}
      <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' as const }}>
        {(
          [
            { key: 'all', label: 'All', count: patches.length },
            { key: 'critical', label: 'Critical', count: criticalCount },
            { key: 'high', label: 'High', count: highCount },
            { key: 'medium', label: 'Medium', count: mediumCount },
            { key: 'low', label: 'Low', count: lowCount },
          ] as const
        ).map(({ key, label, count }) => {
          const active = severityFilter === key;
          const color = key === 'all' ? 'var(--text-primary)' : SEVERITY_COLOR[key];
          return (
            <button
              key={key}
              onClick={() => setSeverityFilter(key)}
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                padding: '4px 10px',
                borderRadius: 4,
                border: `1px solid ${active ? color : 'var(--border)'}`,
                background: active ? 'var(--bg-inset)' : 'var(--bg-card)',
                color: active ? color : 'var(--text-muted)',
                cursor: 'pointer',
                transition: 'all 0.1s',
              }}
            >
              {label} {count}
            </button>
          );
        })}
      </div>

      {/* Pending patches table */}
      {pendingPatches.length > 0 && (
        <div style={S.card}>
          <div
            style={{
              ...S.cardTitle,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}
          >
            <span>Pending Patches</span>
            <span style={{ color: 'var(--text-faint)' }}>{pendingPatches.length}</span>
          </div>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={{ ...S.th, width: 32 }}>
                    <input
                      type="checkbox"
                      checked={
                        pendingPatches.length > 0 && pendingPatches.every((p) => selected.has(p.id))
                      }
                      onChange={() => toggleAll(pendingPatches)}
                      style={{ cursor: 'pointer', accentColor: 'var(--accent)' }}
                    />
                  </th>
                  <th style={S.th}>Patch Name</th>
                  <th style={S.th}>Severity</th>
                  <th style={S.th}>CVSS</th>
                  <th style={S.th}>CVEs</th>
                  <th style={S.th}>Source</th>
                </tr>
              </thead>
              <tbody>
                {pendingPatches.map((patch) => {
                  const cvss = severityToCvss(patch.severity);
                  const sevColor = SEVERITY_COLOR[patch.severity] ?? 'var(--text-muted)';
                  return (
                    <tr
                      key={patch.id}
                      onMouseEnter={(e) => {
                        (e.currentTarget as HTMLTableRowElement).style.background =
                          'var(--bg-card-hover)';
                      }}
                      onMouseLeave={(e) => {
                        (e.currentTarget as HTMLTableRowElement).style.background = '';
                      }}
                    >
                      <td style={{ ...S.td, textAlign: 'center' }}>
                        <input
                          type="checkbox"
                          checked={selected.has(patch.id)}
                          onChange={() => toggleSelect(patch.id)}
                          style={{ cursor: 'pointer', accentColor: 'var(--accent)' }}
                        />
                      </td>
                      <td style={S.td}>
                        <button
                          onClick={() => void navigate(`/patches/${patch.id}`)}
                          style={{
                            background: 'none',
                            border: 'none',
                            cursor: 'pointer',
                            fontFamily: 'var(--font-sans)',
                            fontSize: 13,
                            color: 'var(--text-primary)',
                            textAlign: 'left',
                            padding: 0,
                          }}
                          onMouseEnter={(e) => {
                            (e.currentTarget as HTMLButtonElement).style.color = 'var(--accent)';
                          }}
                          onMouseLeave={(e) => {
                            (e.currentTarget as HTMLButtonElement).style.color =
                              'var(--text-primary)';
                          }}
                        >
                          {patch.name}
                        </button>
                      </td>
                      <td style={S.td}>
                        <span
                          style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: sevColor }}
                        >
                          {patch.severity || '—'}
                        </span>
                      </td>
                      <td style={S.td}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                          <div
                            style={{
                              width: 48,
                              height: 3,
                              background: 'var(--border)',
                              borderRadius: 2,
                              overflow: 'hidden',
                            }}
                          >
                            <div
                              style={{
                                height: '100%',
                                width: `${(cvss / 10) * 100}%`,
                                background: cvssColor(cvss),
                                borderRadius: 2,
                              }}
                            />
                          </div>
                          <span
                            style={{
                              fontFamily: 'var(--font-mono)',
                              fontSize: 11,
                              color: cvssColor(cvss),
                            }}
                          >
                            {cvss.toFixed(1)}
                          </span>
                        </div>
                      </td>
                      <td
                        style={{
                          ...S.td,
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: 'var(--text-muted)',
                        }}
                      >
                        {patch.cve_count ?? '—'}
                      </td>
                      <td style={{ ...S.td, fontSize: 11, color: 'var(--text-muted)' }}>
                        {patch.os_family ?? '—'}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Installed patches table */}
      {installedPatches.length > 0 && (
        <div style={S.card}>
          <div
            style={{
              ...S.cardTitle,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}
          >
            <span>Recently Installed</span>
            <span style={{ color: 'var(--text-faint)' }}>{installedPatches.length}</span>
          </div>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={S.th}>Patch Name</th>
                  <th style={S.th}>Severity</th>
                  <th style={S.th}>Status</th>
                  <th style={S.th}>Source</th>
                </tr>
              </thead>
              <tbody>
                {installedPatches.map((patch) => {
                  const sevColor = SEVERITY_COLOR[patch.severity] ?? 'var(--text-muted)';
                  const statusColor = STATUS_COLOR[patch.status] ?? 'var(--text-muted)';
                  return (
                    <tr
                      key={patch.id}
                      onMouseEnter={(e) => {
                        (e.currentTarget as HTMLTableRowElement).style.background =
                          'var(--bg-card-hover)';
                      }}
                      onMouseLeave={(e) => {
                        (e.currentTarget as HTMLTableRowElement).style.background = '';
                      }}
                    >
                      <td style={S.td}>
                        <button
                          onClick={() => void navigate(`/patches/${patch.id}`)}
                          style={{
                            background: 'none',
                            border: 'none',
                            cursor: 'pointer',
                            fontSize: 13,
                            color: 'var(--text-primary)',
                            padding: 0,
                          }}
                          onMouseEnter={(e) => {
                            (e.currentTarget as HTMLButtonElement).style.color = 'var(--accent)';
                          }}
                          onMouseLeave={(e) => {
                            (e.currentTarget as HTMLButtonElement).style.color =
                              'var(--text-primary)';
                          }}
                        >
                          {patch.name}
                        </button>
                      </td>
                      <td style={S.td}>
                        <span
                          style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: sevColor }}
                        >
                          {patch.severity || '—'}
                        </span>
                      </td>
                      <td style={S.td}>
                        <span
                          style={{
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: 5,
                            fontSize: 12,
                            color: statusColor,
                          }}
                        >
                          <span
                            style={{
                              width: 6,
                              height: 6,
                              borderRadius: '50%',
                              background: statusColor,
                            }}
                          />
                          {patch.status}
                        </span>
                      </td>
                      <td style={{ ...S.td, fontSize: 11, color: 'var(--text-muted)' }}>
                        {patch.os_family ?? '—'}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Available patches table */}
      {availablePatches.length > 0 && (
        <div style={S.card}>
          <div
            style={{
              ...S.cardTitle,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}
          >
            <span>Available Patches</span>
            <span style={{ color: 'var(--text-faint)' }}>{availablePatches.length}</span>
          </div>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={S.th}>Patch Name</th>
                  <th style={S.th}>Severity</th>
                  <th style={S.th}>CVSS</th>
                  <th style={S.th}>CVEs</th>
                  <th style={S.th}>Status</th>
                </tr>
              </thead>
              <tbody>
                {availablePatches.map((patch) => {
                  const cvss = severityToCvss(patch.severity);
                  const sevColor = SEVERITY_COLOR[patch.severity] ?? 'var(--text-muted)';
                  return (
                    <tr
                      key={patch.id}
                      onMouseEnter={(e) => {
                        (e.currentTarget as HTMLTableRowElement).style.background =
                          'var(--bg-card-hover)';
                      }}
                      onMouseLeave={(e) => {
                        (e.currentTarget as HTMLTableRowElement).style.background = '';
                      }}
                    >
                      <td style={S.td}>
                        <button
                          onClick={() => void navigate(`/patches/${patch.id}`)}
                          style={{
                            background: 'none',
                            border: 'none',
                            cursor: 'pointer',
                            fontFamily: 'var(--font-sans)',
                            fontSize: 13,
                            color: 'var(--text-primary)',
                            textAlign: 'left',
                            padding: 0,
                          }}
                          onMouseEnter={(e) => {
                            (e.currentTarget as HTMLButtonElement).style.color = 'var(--accent)';
                          }}
                          onMouseLeave={(e) => {
                            (e.currentTarget as HTMLButtonElement).style.color =
                              'var(--text-primary)';
                          }}
                        >
                          {patch.name}
                        </button>
                      </td>
                      <td style={S.td}>
                        <span
                          style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: sevColor }}
                        >
                          {patch.severity || '—'}
                        </span>
                      </td>
                      <td style={S.td}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                          <div
                            style={{
                              width: 48,
                              height: 3,
                              background: 'var(--border)',
                              borderRadius: 2,
                              overflow: 'hidden',
                            }}
                          >
                            <div
                              style={{
                                height: '100%',
                                width: `${(cvss / 10) * 100}%`,
                                background: cvssColor(cvss),
                                borderRadius: 2,
                              }}
                            />
                          </div>
                          <span
                            style={{
                              fontFamily: 'var(--font-mono)',
                              fontSize: 11,
                              color: cvssColor(cvss),
                            }}
                          >
                            {cvss.toFixed(1)}
                          </span>
                        </div>
                      </td>
                      <td
                        style={{
                          ...S.td,
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: 'var(--text-muted)',
                        }}
                      >
                        {patch.cve_count ?? '—'}
                      </td>
                      <td style={S.td}>
                        <span
                          style={{
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: 4,
                            fontFamily: 'var(--font-mono)',
                            fontSize: 11,
                            padding: '2px 7px',
                            borderRadius: 4,
                            background: 'var(--bg-inset)',
                            border: '1px solid var(--border)',
                            color: 'var(--text-secondary)',
                          }}
                        >
                          available
                        </span>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Empty state */}
      {pendingPatches.length === 0 &&
        installedPatches.length === 0 &&
        availablePatches.length === 0 && (
          <div
            style={{
              ...S.card,
              padding: 48,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: 8,
            }}
          >
            <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>
              {patches.length === 0
                ? 'No patches found. Run a scan to populate patch data.'
                : 'No patches match the current filter.'}
            </span>
          </div>
        )}
    </div>
  );
}
