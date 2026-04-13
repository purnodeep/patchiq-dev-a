/* eslint-disable @typescript-eslint/no-explicit-any */
import { RingChart, Button } from '@patchiq/ui';
import { Link, useNavigate } from 'react-router';
import { RefreshCw, ArrowRight } from 'lucide-react';
import type { FrameworkScoreSummary } from '../../../api/hooks/useCompliance';
import { timeAgo } from '../../../lib/time';
import { ProgressBar } from './progress-bar';

function getComplianceStatus(score: number) {
  if (score >= 95) return { label: 'Compliant', color: 'var(--accent)' };
  if (score >= 80) return { label: 'Needs Improvement', color: 'var(--signal-warning)' };
  return { label: 'Non-Compliant', color: 'var(--signal-critical)' };
}

// Maps API IDs / raw names to canonical display names
const FRAMEWORK_NAMES: Record<string, string> = {
  cis: 'CIS Controls v8',
  hipaa: 'HIPAA Security Rule',
  nist_800_53: 'NIST 800-53',
  pci_dss: 'PCI DSS v4.0',
  iso_27001: 'ISO 27001',
  soc_2: 'SOC 2 Type II',
};

function normalizeFrameworkName(name: string): string {
  const lower = name.toLowerCase().replace(/[\s-]/g, '_');
  if (FRAMEWORK_NAMES[name.toLowerCase()]) return FRAMEWORK_NAMES[name.toLowerCase()];
  if (FRAMEWORK_NAMES[lower]) return FRAMEWORK_NAMES[lower];
  return name;
}

const FRAMEWORK_SUBTITLES: Record<string, string> = {
  'NIST CSF 2.0': 'National Institute of Standards',
  'NIST 800-53': 'National Institute of Standards',
  'PCI DSS 4.0': 'Payment Card Industry Security',
  'PCI DSS v4.0': 'Payment Card Industry Security',
  'HIPAA Security Rule': 'Health Information Portability',
  'CIS Controls v8': 'Center for Internet Security',
  'ISO 27001:2022': 'Information Security Management',
  'ISO 27001': 'Information Security Management',
  'SOC 2 Type II': 'Service Organization Controls',
};

interface FrameworkCardProps {
  framework: FrameworkScoreSummary;
  onEvaluate: (frameworkId: string) => void;
}

export function FrameworkCard({ framework, onEvaluate }: FrameworkCardProps) {
  const navigate = useNavigate();
  const passing = framework.passing_controls ?? 0;
  const total = framework.total_controls ?? 0;
  const hasControls = total > 0;
  const score =
    hasControls && framework.score != null ? Math.round(parseFloat(framework.score)) : 0;
  const status = hasControls
    ? getComplianceStatus(score)
    : { label: 'No Data', color: 'var(--text-muted)' };
  const endpointsCompliant = framework.endpoints_compliant ?? 0;
  const totalEndpoints = framework.total_endpoints ?? 0;
  const overdue = framework.overdue_count ?? 0;
  const displayName = normalizeFrameworkName(framework.name);
  const subtitle = FRAMEWORK_SUBTITLES[displayName] ?? '';
  const lastEval = timeAgo(framework.evaluated_at);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      navigate(`/compliance/frameworks/${framework.framework_id}`);
    }
  };

  return (
    <div
      onClick={() => navigate(`/compliance/frameworks/${framework.framework_id}`)}
      onKeyDown={handleKeyDown}
      tabIndex={0}
      role="link"
      aria-label={`View ${displayName} compliance details`}
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        boxShadow: 'var(--shadow-sm)',
        padding: 20,
        display: 'flex',
        flexDirection: 'column',
        gap: 0,
        transition: 'border-color 0.15s ease, background 0.15s ease',
        cursor: 'pointer',
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border-hover)';
        (e.currentTarget as HTMLDivElement).style.background = 'var(--bg-card-hover)';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)';
        (e.currentTarget as HTMLDivElement).style.background = 'var(--bg-card)';
      }}
    >
      {/* Header: name + status text */}
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          justifyContent: 'space-between',
          marginBottom: 16,
        }}
      >
        <div>
          <div
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 14,
              fontWeight: 700,
              color: 'var(--text-emphasis)',
              marginBottom: 2,
            }}
          >
            {displayName}
          </div>
          {subtitle && (
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                color: 'var(--text-muted)',
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
              }}
            >
              {subtitle}
            </div>
          )}
        </div>
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 600,
            color: status.color,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
          }}
        >
          {status.label}
        </span>
      </div>

      {/* Gauge row */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 16 }}>
        <RingChart
          value={hasControls ? score : 0}
          max={100}
          size={64}
          thickness={7}
          color={hasControls ? status.color : 'var(--text-muted)'}
          trackColor={hasControls ? undefined : 'var(--border)'}
        />
        <div>
          <div
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 28,
              fontWeight: 700,
              color: hasControls ? status.color : 'var(--text-muted)',
              lineHeight: 1,
              marginBottom: 5,
            }}
          >
            {hasControls ? `${score}%` : '\u2014'}
          </div>
          <div style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' }}>
            {hasControls ? (
              <>
                <span style={{ color: 'var(--accent)', fontWeight: 600 }}>{passing}</span>
                <span style={{ color: 'var(--text-faint)' }}>/{total}</span> controls passing
              </>
            ) : (
              'No controls configured'
            )}
          </div>
        </div>
      </div>

      {/* Endpoint progress */}
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          color: 'var(--text-muted)',
          marginBottom: 5,
        }}
      >
        {totalEndpoints === 0
          ? 'No endpoints enrolled'
          : `${endpointsCompliant}/${totalEndpoints} endpoints meeting SLA`}
      </div>
      <ProgressBar value={endpointsCompliant} max={totalEndpoints || 1} color={status.color} />

      {/* Meta row */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginTop: 12,
          marginBottom: 16,
        }}
      >
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            color: 'var(--text-faint)',
          }}
        >
          Evaluated{' '}
          <span
            style={{ color: 'var(--text-muted)' }}
            title={
              framework.evaluated_at ? new Date(framework.evaluated_at).toLocaleString() : undefined
            }
          >
            {lastEval}
          </span>
        </span>
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 600,
            color: overdue > 0 ? 'var(--signal-critical)' : 'var(--accent)',
          }}
        >
          {overdue > 0 ? `${overdue} overdue` : 'All on track'}
        </span>
      </div>

      {/* Actions */}
      <div
        style={{
          borderTop: '1px solid var(--border)',
          paddingTop: 14,
          display: 'flex',
          gap: 8,
        }}
      >
        <Button variant="ghost" size="sm" asChild style={{ flex: 1, justifyContent: 'center' }}>
          <Link
            to={`/compliance/frameworks/${framework.framework_id}`}
            onClick={(e) => e.stopPropagation()}
            aria-label={`View details for ${displayName}`}
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--accent)',
              display: 'flex',
              alignItems: 'center',
              gap: 4,
            }}
          >
            View Details
            <ArrowRight style={{ width: 12, height: 12 }} />
          </Link>
        </Button>
        <Button
          variant="outline"
          size="sm"
          aria-label={`Evaluate ${displayName}`}
          style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
          onClick={(e) => {
            e.stopPropagation();
            onEvaluate(framework.framework_id);
          }}
        >
          <RefreshCw style={{ width: 11, height: 11, marginRight: 4 }} />
          Evaluate
        </Button>
      </div>
    </div>
  );
}
