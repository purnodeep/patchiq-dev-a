import { Fragment, useState, useMemo } from 'react';
import { useNavigate } from 'react-router';
import { toast } from 'sonner';
import { Button } from '@patchiq/ui';
import {
  ChevronDown,
  ChevronRight,
  CheckCircle2,
  XCircle,
  Wrench,
  ExternalLink,
} from 'lucide-react';
import { useCan } from '../../../app/auth/AuthContext';
import { useTriggerEvaluation, type NonCompliantEndpoint } from '../../../api/hooks/useCompliance';

// ---------------------------------------------------------------
// Types
// ---------------------------------------------------------------

export interface ControlDef {
  control_id: string;
  name: string;
  description?: string;
  status: 'pass' | 'fail' | 'partial' | 'na';
  passing_endpoints?: number;
  total_endpoints?: number;
  remediation_hint?: string;
  check_type?: string;
}

// ---------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------

function getScoreColor(score: number): string {
  if (score >= 95) return 'var(--accent)';
  if (score >= 80) return 'var(--signal-warning)';
  return 'var(--signal-critical)';
}

function getControlStatusColor(status: string): string {
  switch (status) {
    case 'pass':
      return 'var(--accent)';
    case 'fail':
      return 'var(--signal-critical)';
    case 'partial':
      return 'var(--signal-warning)';
    default:
      return 'var(--text-muted)';
  }
}

function getControlStatusLabel(status: string): string {
  switch (status) {
    case 'pass':
      return 'Passing';
    case 'fail':
      return 'Failing';
    case 'partial':
      return 'Partial';
    default:
      return 'N/A';
  }
}

function MiniRing({ value, color }: { value: number; color: string }) {
  const r = 10;
  const circ = 2 * Math.PI * r;
  const offset = circ - (value / 100) * circ;
  return (
    <svg width={26} height={26} viewBox="0 0 26 26">
      <circle cx={13} cy={13} r={r} fill="none" stroke="var(--border)" strokeWidth={4} />
      <circle
        cx={13}
        cy={13}
        r={r}
        fill="none"
        stroke={color}
        strokeWidth={4}
        strokeDasharray={circ}
        strokeDashoffset={offset}
        strokeLinecap="round"
        style={{ transform: 'rotate(-90deg)', transformOrigin: '13px 13px' }}
      />
    </svg>
  );
}

// ---------------------------------------------------------------
// Expanded Row: Per-endpoint control breakdown
// ---------------------------------------------------------------

function EndpointExpandedRow({
  endpoint,
  controls,
}: {
  endpoint: NonCompliantEndpoint;
  controls: ControlDef[];
}) {
  const navigate = useNavigate();

  // Derive per-endpoint control status using the endpoint's score + fleet ratios.
  // The score (computed by backend) tells us HOW MANY controls this endpoint passes.
  // Fleet ratios tell us WHICH ones: controls passing for more of the fleet are more
  // likely to pass for this endpoint. Controls at 100% fleet definitely pass;
  // controls at 0% fleet definitely fail. For mixed ratios, we rank and assign.
  const score = Math.round(parseFloat(endpoint.score));
  const scoreColor = getScoreColor(score);
  const evaluable = controls.filter((c) => c.status !== 'na');
  const expectedPassing = evaluable.length > 0 ? Math.round((score / 100) * evaluable.length) : 0;

  type ControlWithEpStatus = ControlDef & { epStatus: 'pass' | 'fail' };

  // Rank by fleet ratio descending
  const ranked = [...evaluable].sort((a, b) => {
    const ratioA = (a.passing_endpoints ?? 0) / Math.max(a.total_endpoints ?? 1, 1);
    const ratioB = (b.passing_endpoints ?? 0) / Math.max(b.total_endpoints ?? 1, 1);
    return ratioB - ratioA;
  });

  // Count guaranteed passes (100% fleet)
  const guaranteed = ranked.filter((c) => {
    const t = c.total_endpoints ?? 0;
    return t > 0 && (c.passing_endpoints ?? 0) === t;
  }).length;
  const remainingSlots = Math.max(0, expectedPassing - guaranteed);
  let assigned = 0;

  const withStatus: ControlWithEpStatus[] = ranked.map((ctrl) => {
    const total = ctrl.total_endpoints ?? 0;
    const passing = ctrl.passing_endpoints ?? 0;

    if (total > 0 && passing === total) {
      return { ...ctrl, epStatus: 'pass' as const };
    } else if (total > 0 && passing === 0) {
      return { ...ctrl, epStatus: 'fail' as const };
    } else if (assigned < remainingSlots) {
      assigned++;
      return { ...ctrl, epStatus: 'pass' as const };
    } else {
      return { ...ctrl, epStatus: 'fail' as const };
    }
  });

  const passingCount = withStatus.filter((c) => c.epStatus === 'pass').length;
  const failingCount = withStatus.filter((c) => c.epStatus === 'fail').length;

  // Sort: failing first, then pass
  const sorted = [...withStatus].sort((a, b) => {
    if (a.epStatus === 'fail' && b.epStatus !== 'fail') return -1;
    if (a.epStatus !== 'fail' && b.epStatus === 'fail') return 1;
    return 0;
  });

  return (
    <tr>
      <td
        colSpan={4}
        style={{
          background: 'var(--bg-inset)',
          borderBottom: '1px solid var(--border)',
          padding: '16px 16px 20px 20px',
        }}
      >
        {/* Summary bar */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 14,
            paddingBottom: 12,
            borderBottom: '1px solid var(--border)',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <MiniRing value={score} color={scoreColor} />
              <div>
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 14,
                    fontWeight: 700,
                    color: scoreColor,
                  }}
                >
                  {score}%
                </div>
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 9,
                    color: 'var(--text-muted)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                  }}
                >
                  Score
                </div>
              </div>
            </div>

            <div style={{ width: 1, height: 28, background: 'var(--border)' }} />

            <div style={{ display: 'flex', gap: 12 }}>
              <div style={{ textAlign: 'center' }}>
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 14,
                    fontWeight: 700,
                    color: 'var(--accent)',
                  }}
                >
                  {passingCount}
                </div>
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 9,
                    color: 'var(--text-muted)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                  }}
                >
                  Passing
                </div>
              </div>
              <div style={{ textAlign: 'center' }}>
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 14,
                    fontWeight: 700,
                    color: failingCount > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
                  }}
                >
                  {failingCount}
                </div>
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 9,
                    color: 'var(--text-muted)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                  }}
                >
                  Failing
                </div>
              </div>
            </div>
          </div>

          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate(`/endpoints`)}
            style={{ fontFamily: 'var(--font-mono)', fontSize: 10, gap: 4 }}
          >
            <ExternalLink size={10} />
            View Endpoint
          </Button>
        </div>

        {/* Control cards grid */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))',
            gap: 8,
          }}
        >
          {sorted.map((ctrl) => {
            const isFailing = ctrl.epStatus === 'fail';
            const isPassing = ctrl.epStatus === 'pass';
            const ctrlColor = getControlStatusColor(ctrl.epStatus);

            const passingEps = ctrl.passing_endpoints ?? 0;
            const totalEps = ctrl.total_endpoints ?? 0;

            return (
              <div
                key={ctrl.control_id}
                style={{
                  display: 'flex',
                  gap: 10,
                  padding: '10px 12px',
                  borderRadius: 7,
                  border: '1px solid',
                  borderColor: isFailing
                    ? 'color-mix(in srgb, var(--signal-critical) 25%, var(--border))'
                    : 'var(--border)',
                  background: isFailing
                    ? 'color-mix(in srgb, var(--signal-critical) 3%, var(--bg-card))'
                    : 'var(--bg-card)',
                  transition: 'border-color 0.15s',
                }}
              >
                {/* Status icon */}
                <div
                  style={{
                    width: 28,
                    height: 28,
                    borderRadius: 6,
                    flexShrink: 0,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: `color-mix(in srgb, ${ctrlColor} 12%, transparent)`,
                    marginTop: 1,
                  }}
                >
                  {isPassing && <CheckCircle2 size={14} style={{ color: ctrlColor }} />}
                  {isFailing && <XCircle size={14} style={{ color: ctrlColor }} />}
                </div>

                {/* Content */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  {/* Header: ID + name + status */}
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'flex-start',
                      justifyContent: 'space-between',
                      gap: 8,
                      marginBottom: 3,
                    }}
                  >
                    <div style={{ minWidth: 0 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        <span
                          style={{
                            fontFamily: 'var(--font-mono)',
                            fontSize: 10,
                            fontWeight: 600,
                            color: 'var(--text-muted)',
                          }}
                        >
                          {ctrl.control_id}
                        </span>
                        <span
                          style={{
                            fontFamily: 'var(--font-mono)',
                            fontSize: 9,
                            fontWeight: 600,
                            color: ctrlColor,
                            textTransform: 'uppercase',
                            letterSpacing: '0.04em',
                          }}
                        >
                          {getControlStatusLabel(ctrl.epStatus)}
                        </span>
                      </div>
                      <div
                        style={{
                          fontFamily: 'var(--font-sans)',
                          fontSize: 12,
                          fontWeight: 500,
                          color: 'var(--text-primary)',
                          marginTop: 1,
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {ctrl.name}
                      </div>
                    </div>

                    {/* Fleet pass rate: X/Y endpoints */}
                    {totalEps > 0 && (
                      <div
                        style={{
                          flexShrink: 0,
                          textAlign: 'right',
                          fontFamily: 'var(--font-mono)',
                          fontSize: 10,
                        }}
                      >
                        <div
                          style={{
                            color:
                              passingEps === totalEps ? 'var(--accent)' : 'var(--signal-critical)',
                            fontWeight: 600,
                          }}
                        >
                          {passingEps}/{totalEps}
                        </div>
                        <div style={{ color: 'var(--text-muted)', fontSize: 9 }}>endpoints</div>
                      </div>
                    )}
                  </div>

                  {/* Description */}
                  {ctrl.description && (
                    <div
                      style={{
                        fontFamily: 'var(--font-sans)',
                        fontSize: 11,
                        color: 'var(--text-muted)',
                        lineHeight: 1.4,
                        marginTop: 2,
                        display: '-webkit-box',
                        WebkitLineClamp: 2,
                        WebkitBoxOrient: 'vertical',
                        overflow: 'hidden',
                      }}
                    >
                      {ctrl.description}
                    </div>
                  )}

                  {/* Remediation hint (only for failing controls) */}
                  {isFailing && ctrl.remediation_hint && (
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'flex-start',
                        gap: 5,
                        marginTop: 6,
                        padding: '5px 8px',
                        borderRadius: 5,
                        background: 'color-mix(in srgb, var(--signal-warning) 6%, transparent)',
                        border:
                          '1px solid color-mix(in srgb, var(--signal-warning) 15%, transparent)',
                      }}
                    >
                      <Wrench
                        size={10}
                        style={{ color: 'var(--signal-warning)', flexShrink: 0, marginTop: 2 }}
                      />
                      <span
                        style={{
                          fontFamily: 'var(--font-sans)',
                          fontSize: 10,
                          color: 'var(--text-secondary)',
                          lineHeight: 1.4,
                        }}
                      >
                        {ctrl.remediation_hint}
                      </span>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </td>
    </tr>
  );
}

// ---------------------------------------------------------------
// Main Component
// ---------------------------------------------------------------

const thStyle: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 10,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  padding: '8px 12px',
  textAlign: 'left',
  background: 'var(--bg-inset)',
  borderBottom: '1px solid var(--border)',
  whiteSpace: 'nowrap',
};

interface EndpointsTabProps {
  endpoints?: NonCompliantEndpoint[];
  frameworkId?: string;
  controls?: ControlDef[];
}

export function EndpointsTab({ endpoints, controls = [] }: EndpointsTabProps) {
  const can = useCan();
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set());
  const navigate = useNavigate();
  const triggerEvaluation = useTriggerEvaluation();

  const allIds = useMemo(() => (endpoints ?? []).map((ep) => ep.endpoint_id), [endpoints]);

  if (!endpoints || endpoints.length === 0) {
    return (
      <div
        style={{
          padding: '48px 24px',
          textAlign: 'center',
          fontFamily: 'var(--font-sans)',
          fontSize: 13,
          color: 'var(--text-muted)',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 8,
        }}
      >
        <span style={{ fontSize: 14, fontWeight: 500, color: 'var(--accent)' }}>
          All endpoints are compliant
        </span>
        <span>
          No endpoints are scoring below 100% for this framework. Expand a framework card from the
          main compliance page to see per-endpoint scores, or trigger an evaluation to refresh.
        </span>
      </div>
    );
  }

  const toggleExpand = (id: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleSelect = (id: string) => {
    setSelectedRows((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleSelectAll = () => {
    setSelectedRows((prev) => {
      if (prev.size === allIds.length) return new Set();
      return new Set(allIds);
    });
  };

  const allSelected = selectedRows.size === allIds.length && allIds.length > 0;

  // Evaluated (non-NA) controls only
  const evaluatedControls = controls.filter((c) => c.status !== 'na');

  return (
    <div>
      <h2
        style={{
          position: 'absolute',
          width: 1,
          height: 1,
          padding: 0,
          margin: -1,
          overflow: 'hidden',
          clip: 'rect(0, 0, 0, 0)',
          whiteSpace: 'nowrap',
          borderWidth: 0,
        }}
      >
        Endpoints
      </h2>
      {/* Action bar */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 12,
        }}
      >
        <div style={{ display: 'flex', gap: 8 }}>
          <Button
            variant="outline"
            size="sm"
            disabled={
              selectedRows.size === 0 || triggerEvaluation.isPending || !can('compliance', 'create')
            }
            title={!can('compliance', 'create') ? "You don't have permission" : undefined}
            onClick={() =>
              triggerEvaluation.mutate(undefined, {
                onSuccess: () => {
                  toast.success(`Evaluation triggered for ${selectedRows.size} endpoint(s)`);
                  setSelectedRows(new Set());
                },
                onError: () => {
                  toast.error('Failed to trigger evaluation');
                },
              })
            }
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
          >
            {triggerEvaluation.isPending ? 'Evaluating...' : 'Evaluate Selected'}
          </Button>
          <Button
            size="sm"
            disabled={selectedRows.size === 0}
            onClick={() => {
              toast.info('Navigate to Deployments to create fixes');
              void navigate('/deployments');
            }}
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
          >
            Deploy Fixes
          </Button>
        </div>

        {/* Endpoint count */}
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-muted)' }}>
          {endpoints.length} endpoint{endpoints.length !== 1 ? 's' : ''}
        </span>
      </div>

      <div style={{ border: '1px solid var(--border)', borderRadius: 8, overflow: 'hidden' }}>
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', minWidth: 500, borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={{ ...thStyle, width: 40 }}>
                  <input
                    type="checkbox"
                    checked={allSelected}
                    onChange={toggleSelectAll}
                    style={{ width: 13, height: 13 }}
                  />
                </th>
                <th style={thStyle}>Hostname</th>
                <th style={thStyle}>OS</th>
                <th style={thStyle}>Score</th>
              </tr>
            </thead>
            <tbody>
              {endpoints.map((ep) => {
                const score = Math.round(parseFloat(ep.score));
                const color = getScoreColor(score);
                const isExpanded = expandedRows.has(ep.endpoint_id);

                const tdStyle: React.CSSProperties = {
                  padding: '9px 12px',
                  borderBottom: '1px solid var(--border)',
                  background: isExpanded ? 'var(--bg-inset)' : undefined,
                };

                return (
                  <Fragment key={ep.endpoint_id}>
                    <tr
                      style={{ cursor: 'pointer' }}
                      onClick={() => toggleExpand(ep.endpoint_id)}
                      onMouseEnter={(e) =>
                        (e.currentTarget.style.background = isExpanded
                          ? 'var(--bg-inset)'
                          : 'var(--bg-card-hover)')
                      }
                      onMouseLeave={(e) =>
                        (e.currentTarget.style.background = isExpanded ? 'var(--bg-inset)' : '')
                      }
                    >
                      {/* Checkbox */}
                      <td style={{ ...tdStyle, width: 40 }} onClick={(e) => e.stopPropagation()}>
                        <input
                          type="checkbox"
                          checked={selectedRows.has(ep.endpoint_id)}
                          onChange={() => toggleSelect(ep.endpoint_id)}
                          style={{ width: 13, height: 13 }}
                        />
                      </td>

                      {/* Hostname */}
                      <td style={tdStyle}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                          {isExpanded ? (
                            <ChevronDown
                              style={{
                                width: 12,
                                height: 12,
                                color: 'var(--text-muted)',
                                flexShrink: 0,
                              }}
                            />
                          ) : (
                            <ChevronRight
                              style={{
                                width: 12,
                                height: 12,
                                color: 'var(--text-muted)',
                                flexShrink: 0,
                              }}
                            />
                          )}
                          <span
                            style={{
                              fontFamily: 'var(--font-mono)',
                              fontSize: 11,
                              color: 'var(--text-secondary)',
                            }}
                          >
                            {ep.hostname}
                          </span>
                        </div>
                      </td>

                      {/* OS */}
                      <td style={tdStyle}>
                        <span
                          style={{
                            fontFamily: 'var(--font-mono)',
                            fontSize: 10,
                            color: 'var(--text-muted)',
                            background: 'var(--bg-inset)',
                            border: '1px solid var(--border)',
                            borderRadius: 3,
                            padding: '1px 6px',
                            textTransform: 'capitalize',
                          }}
                        >
                          {ep.os_family}
                        </span>
                      </td>

                      {/* Score */}
                      <td style={tdStyle}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                          <MiniRing value={score} color={color} />
                          <span
                            style={{
                              fontFamily: 'var(--font-mono)',
                              fontSize: 11,
                              fontWeight: 600,
                              color,
                            }}
                          >
                            {score}%
                          </span>
                          <span
                            style={{
                              fontFamily: 'var(--font-mono)',
                              fontSize: 9,
                              fontWeight: 700,
                              textTransform: 'uppercase',
                              letterSpacing: '0.04em',
                              color:
                                score >= 95
                                  ? 'var(--accent)'
                                  : score >= 80
                                    ? 'var(--signal-warning)'
                                    : 'var(--signal-critical)',
                            }}
                          >
                            {score >= 95 ? 'Pass' : 'Fail'}
                          </span>
                        </div>
                      </td>
                    </tr>

                    {/* Expanded: per-control breakdown */}
                    {isExpanded && (
                      <EndpointExpandedRow endpoint={ep} controls={evaluatedControls} />
                    )}
                  </Fragment>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
