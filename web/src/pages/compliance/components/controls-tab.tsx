import { useState } from 'react';
import { useNavigate } from 'react-router';
import { toast } from 'sonner';
import {
  Button,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@patchiq/ui';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { useFrameworkControls, type ControlResult } from '../../../api/hooks/useCompliance';
import { ProgressBar } from './progress-bar';

const thStyle: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 10,
  fontWeight: 600,
  textTransform: 'uppercase' as const,
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  padding: '8px 12px',
  textAlign: 'left' as const,
  background: 'var(--bg-inset)',
  borderBottom: '1px solid var(--border)',
  whiteSpace: 'nowrap' as const,
};

function getStatusStyle(status: string): { label: string; color: string; tooltip?: string } {
  switch (status) {
    case 'pass':
      return { label: 'Pass', color: 'var(--accent)' };
    case 'fail':
      return { label: 'Fail', color: 'var(--signal-critical)' };
    case 'partial':
      return { label: 'Partial', color: 'var(--signal-warning)' };
    default:
      return {
        label: 'N/A',
        color: 'var(--text-muted)',
        tooltip: 'Not yet monitored — evaluator not configured for this control',
      };
  }
}

function formatSla(control: ControlResult): { text: string; color: string; ariaLabel?: string } {
  if (control.status === 'na') return { text: '—', color: 'var(--text-muted)' };
  if ((control.days_overdue ?? 0) > 0) {
    return {
      text: `⚠ ${control.days_overdue}d overdue`,
      color: 'var(--signal-critical)',
      ariaLabel: `${control.days_overdue} days overdue`,
    };
  }
  if (control.sla_deadline_at) {
    const deadline = new Date(control.sla_deadline_at);
    const daysLeft = Math.ceil((deadline.getTime() - Date.now()) / 86_400_000);
    if (daysLeft <= 7) {
      return {
        text: `${daysLeft}d remaining`,
        color: 'var(--signal-critical)',
        ariaLabel: `${daysLeft} days remaining`,
      };
    }
    if (daysLeft <= 30) {
      return {
        text: `${daysLeft}d remaining`,
        color: 'var(--signal-warning)',
        ariaLabel: `${daysLeft} days remaining`,
      };
    }
    return {
      text: `${daysLeft}d remaining`,
      color: 'var(--text-secondary)',
      ariaLabel: `${daysLeft} days remaining`,
    };
  }
  return { text: '✓ On track', color: 'var(--accent)' };
}

function getExpandedContent(control: ControlResult) {
  if (control.status === 'pass') {
    return {
      evidenceTitle: 'Evidence',
      evidence: `All ${control.total_endpoints} endpoints passing this control. Last evaluation completed successfully.`,
      patchTitle: 'Satisfying Patches',
      patches: 'No patches required — configuration control',
      patchColor: 'var(--text-muted)',
      remediation: 'No action required',
      remediationColor: 'var(--accent)',
    };
  }
  if (control.status === 'fail') {
    const failing = control.total_endpoints - control.passing_endpoints;
    return {
      evidenceTitle: 'Failing Evidence',
      evidence: `${failing} endpoints failing this control. ${control.remediation_hint || 'Review endpoint configurations.'}`,
      patchTitle: 'Missing Patches',
      patches: `${failing} endpoints require remediation`,
      patchColor: 'var(--signal-critical)',
      remediation: control.remediation_hint || `Deploy fix to ${failing} affected endpoints.`,
      remediationColor: 'var(--text-primary)',
    };
  }
  if (control.status === 'partial') {
    const failing = control.total_endpoints - control.passing_endpoints;
    return {
      evidenceTitle: 'Evidence',
      evidence: `${control.passing_endpoints}/${control.total_endpoints} endpoints compliant. ${failing} endpoints need attention.`,
      patchTitle: 'Pending Patches',
      patches: `${failing} endpoints partially compliant`,
      patchColor: 'var(--signal-warning)',
      remediation: control.remediation_hint || 'Complete remediation on remaining endpoints.',
      remediationColor: 'var(--text-primary)',
    };
  }
  return {
    evidenceTitle: 'Evidence',
    evidence: 'Not applicable to current environment.',
    patchTitle: 'Patches',
    patches: 'N/A',
    patchColor: 'var(--text-muted)',
    remediation: 'No action required',
    remediationColor: 'var(--text-muted)',
  };
}

interface ControlsTabProps {
  frameworkId: string;
}

export function ControlsTab({ frameworkId }: ControlsTabProps) {
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [search, setSearch] = useState('');
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  const { data: controls, isLoading } = useFrameworkControls(frameworkId, {
    status: statusFilter === 'all' ? undefined : statusFilter,
    search: search || undefined,
  });

  const toggleRow = (controlId: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(controlId)) next.delete(controlId);
      else next.add(controlId);
      return next;
    });
  };

  if (isLoading) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full rounded" />
        ))}
      </div>
    );
  }

  // Deduplicate controls by control_id (keeps last occurrence per id).
  // Guards against the API returning duplicate rows across evaluation runs.
  const seen = new Map<string, ControlResult>();
  for (const ctrl of controls ?? []) {
    seen.set(ctrl.control_id, ctrl);
  }

  // Group deduplicated controls by category
  const grouped = new Map<string, ControlResult[]>();
  for (const ctrl of seen.values()) {
    const existing = grouped.get(ctrl.category) ?? [];
    existing.push(ctrl);
    grouped.set(ctrl.category, existing);
  }

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
        Controls
      </h2>
      {/* Filter bar */}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10, marginBottom: 16 }}>
        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[140px]">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Status</SelectItem>
            <SelectItem value="pass">Pass</SelectItem>
            <SelectItem value="fail">Fail</SelectItem>
            <SelectItem value="partial">Partial</SelectItem>
            <SelectItem value="na">N/A</SelectItem>
          </SelectContent>
        </Select>
        <Input
          placeholder="Search controls..."
          style={{ width: 220, fontFamily: 'var(--font-sans)', fontSize: 12 }}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {/* Table */}
      <div
        style={{
          border: '1px solid var(--border)',
          borderRadius: 8,
          overflow: 'hidden',
        }}
      >
        <div style={{ overflowX: 'auto' }}>
          <table
            style={{
              width: '100%',
              minWidth: 860,
              borderCollapse: 'collapse',
            }}
          >
            <thead>
              <tr>
                <th style={{ ...thStyle, width: 30 }} />
                <th style={thStyle}>Control ID</th>
                <th style={thStyle}>Name</th>
                <th style={thStyle}>Status</th>
                <th style={thStyle}>Passing / Total</th>
                <th style={thStyle}>SLA</th>
                <th style={thStyle}>Action</th>
              </tr>
            </thead>
            <tbody>
              {Array.from(grouped.entries()).map(([category, ctrls]) => (
                <CategoryGroup
                  key={category}
                  category={category}
                  controls={ctrls}
                  expandedRows={expandedRows}
                  onToggle={toggleRow}
                />
              ))}
              {grouped.size === 0 && (
                <tr>
                  <td
                    colSpan={7}
                    style={{
                      padding: '32px',
                      textAlign: 'center',
                      fontFamily: 'var(--font-sans)',
                      fontSize: 13,
                      color: 'var(--text-muted)',
                    }}
                  >
                    No controls found matching your filters.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function CategoryGroup({
  category,
  controls,
  expandedRows,
  onToggle,
}: {
  category: string;
  controls: ControlResult[];
  expandedRows: Set<string>;
  onToggle: (id: string) => void;
}) {
  return (
    <>
      <tr>
        <td
          colSpan={7}
          style={{
            padding: '6px 12px',
            background: 'var(--bg-inset)',
            borderBottom: '1px solid var(--border)',
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 700,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            color: 'var(--text-muted)',
          }}
        >
          {category} — {controls.length} controls
        </td>
      </tr>
      {controls.map((ctrl) => (
        <ControlRow
          key={`${category}-${ctrl.control_id}`}
          control={ctrl}
          isExpanded={expandedRows.has(ctrl.control_id)}
          onToggle={() => onToggle(ctrl.control_id)}
        />
      ))}
    </>
  );
}

function ControlRow({
  control,
  isExpanded,
  onToggle,
}: {
  control: ControlResult;
  isExpanded: boolean;
  onToggle: () => void;
}) {
  const navigate = useNavigate();
  const statusStyle = getStatusStyle(control.status);
  const sla = formatSla(control);
  const isOverdue = (control.days_overdue ?? 0) > 0;

  const tdStyle: React.CSSProperties = {
    padding: '9px 12px',
    borderBottom: '1px solid var(--border)',
    background: isExpanded ? 'var(--bg-inset)' : undefined,
  };

  return (
    <>
      <tr
        style={{
          cursor: 'pointer',
          borderLeft: isOverdue ? '2px solid var(--signal-critical)' : undefined,
          transition: 'background 0.1s ease',
        }}
        onClick={onToggle}
        onMouseEnter={(e) =>
          ((e.currentTarget as HTMLTableRowElement).style.background = 'var(--bg-card-hover)')
        }
        onMouseLeave={(e) =>
          ((e.currentTarget as HTMLTableRowElement).style.background = isExpanded
            ? 'var(--bg-inset)'
            : '')
        }
      >
        {/* Expand icon */}
        <td style={{ ...tdStyle, width: 30, textAlign: 'center' }}>
          {isExpanded ? (
            <ChevronDown style={{ width: 12, height: 12, color: 'var(--text-muted)' }} />
          ) : (
            <ChevronRight style={{ width: 12, height: 12, color: 'var(--text-muted)' }} />
          )}
        </td>

        {/* Control ID */}
        <td style={tdStyle}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--text-secondary)',
            }}
          >
            {control.control_id}
          </span>
        </td>

        {/* Name */}
        <td style={tdStyle}>
          <span
            style={{ fontFamily: 'var(--font-sans)', fontSize: 12, color: 'var(--text-primary)' }}
          >
            {control.name}
          </span>
        </td>

        {/* Status */}
        <td style={tdStyle}>
          {statusStyle.tooltip ? (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      fontWeight: 600,
                      color: statusStyle.color,
                      textTransform: 'uppercase',
                      letterSpacing: '0.04em',
                      cursor: 'help',
                      borderBottom: '1px dashed var(--text-muted)',
                    }}
                  >
                    {statusStyle.label}
                  </span>
                </TooltipTrigger>
                <TooltipContent>{statusStyle.tooltip}</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          ) : (
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                fontWeight: 600,
                color: statusStyle.color,
                textTransform: 'uppercase',
                letterSpacing: '0.04em',
              }}
            >
              {statusStyle.label}
            </span>
          )}
        </td>

        {/* Passing/Total */}
        <td style={tdStyle}>
          {control.status === 'na' ? (
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                color: 'var(--text-muted)',
              }}
            >
              N/A
            </span>
          ) : (
            <div>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 11,
                  color: 'var(--text-secondary)',
                }}
              >
                <span style={{ color: 'var(--accent)' }}>{control.passing_endpoints}</span>
                <span style={{ color: 'var(--text-faint)' }}>/{control.total_endpoints}</span>
              </span>
              <div style={{ marginTop: 4 }}>
                <ProgressBar
                  value={control.passing_endpoints}
                  max={control.total_endpoints || 1}
                  color={statusStyle.color}
                />
              </div>
            </div>
          )}
        </td>

        {/* SLA */}
        <td style={tdStyle}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              fontWeight: 500,
              color: sla.color,
            }}
            aria-label={sla.ariaLabel}
          >
            {sla.text}
          </span>
        </td>

        {/* Action */}
        <td style={tdStyle} onClick={(e) => e.stopPropagation()}>
          <Button
            variant={
              control.status === 'fail' || control.status === 'partial' ? 'default' : 'ghost'
            }
            size="sm"
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
            onClick={() => {
              if (control.status === 'fail' || control.status === 'partial') {
                toast.info('Navigate to Deployments to create a fix for this control');
                void navigate('/deployments');
              } else {
                onToggle();
              }
            }}
          >
            {control.status === 'fail' || control.status === 'partial' ? 'Deploy Fix' : 'Details'}
          </Button>
        </td>
      </tr>

      {isExpanded && <ExpandedRow control={control} />}
    </>
  );
}

function ExpandedRow({ control }: { control: ControlResult }) {
  const navigate = useNavigate();
  const content = getExpandedContent(control);

  return (
    <tr>
      <td
        colSpan={7}
        style={{
          background: 'var(--bg-inset)',
          borderBottom: '1px solid var(--border)',
          padding: '14px 16px 14px 44px',
        }}
      >
        {/* Control description */}
        {control.description && (
          <div
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 12,
              color: 'var(--text-secondary)',
              lineHeight: 1.6,
              marginBottom: 14,
              padding: '8px 10px',
              background: 'color-mix(in srgb, var(--bg-page) 60%, var(--bg-inset))',
              borderRadius: 6,
              border: '1px solid var(--border)',
            }}
          >
            {control.description}
          </div>
        )}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 20 }}>
          {/* Evidence */}
          <div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                fontWeight: 600,
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                color: 'var(--text-muted)',
                marginBottom: 6,
              }}
            >
              {content.evidenceTitle}
            </div>
            <div
              style={{
                fontFamily: 'var(--font-sans)',
                fontSize: 11,
                color: 'var(--text-secondary)',
                lineHeight: 1.5,
              }}
            >
              {content.evidence}
            </div>
          </div>

          {/* Patches */}
          <div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                fontWeight: 600,
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                color: 'var(--text-muted)',
                marginBottom: 6,
              }}
            >
              {content.patchTitle}
            </div>
            <div
              style={{
                fontFamily: 'var(--font-sans)',
                fontSize: 11,
                color: content.patchColor,
                lineHeight: 1.5,
              }}
            >
              {content.patches}
            </div>
          </div>

          {/* Remediation */}
          <div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                fontWeight: 600,
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                color: 'var(--text-muted)',
                marginBottom: 6,
              }}
            >
              Remediation
            </div>
            <div
              style={{
                fontFamily: 'var(--font-sans)',
                fontSize: 11,
                color: content.remediationColor,
                lineHeight: 1.5,
              }}
            >
              {content.remediation}
            </div>
            {(control.status === 'fail' || control.status === 'partial') && (
              <Button
                size="sm"
                style={{ marginTop: 10, fontFamily: 'var(--font-mono)', fontSize: 11 }}
                onClick={() => {
                  toast.info('Navigate to Deployments to create a fix for this control');
                  void navigate('/deployments');
                }}
              >
                Deploy Fix
              </Button>
            )}
          </div>
        </div>
      </td>
    </tr>
  );
}
