/* eslint-disable @typescript-eslint/no-explicit-any */
import { RingChart } from '@patchiq/ui';
import type {
  OverallComplianceScore,
  FrameworkScoreSummary,
} from '../../../api/hooks/useCompliance';
import { timeAgo } from '../../../lib/time';

function getComplianceStatus(score: number) {
  if (score >= 95) return { label: 'Compliant', color: 'var(--accent)' };
  if (score >= 80) return { label: 'Needs Improvement', color: 'var(--signal-warning)' };
  return { label: 'Non-Compliant', color: 'var(--signal-critical)' };
}

interface OverallScoreCardProps {
  score: OverallComplianceScore;
  frameworks: FrameworkScoreSummary[];
  overdueControlsCount?: number;
}

export function OverallScoreCard({
  score,
  frameworks,
  overdueControlsCount,
}: OverallScoreCardProps) {
  // Compute overall as average of framework scores when available, so the
  // ring value, status label, and per-framework counts are always consistent.
  // Exclude frameworks with 0 total controls — they have no meaningful score.
  const frameworkScores = frameworks
    .filter((fw) => (fw.total_controls ?? 0) > 0)
    .map((fw) => (fw.score != null ? parseFloat(fw.score) : null))
    .filter((s): s is number => s !== null);
  const overall =
    frameworkScores.length > 0
      ? Math.round(frameworkScores.reduce((a, b) => a + b, 0) / frameworkScores.length)
      : Math.round(parseFloat(score.overall_score ?? '0'));
  const status = getComplianceStatus(overall);

  const compliantCount = frameworks.filter(
    (fw) => fw.score != null && parseFloat(fw.score) >= 95,
  ).length;
  const needsWorkCount = frameworks.filter(
    (fw) => fw.score != null && parseFloat(fw.score) >= 80 && parseFloat(fw.score) < 95,
  ).length;
  const nonCompliantCount = frameworks.filter(
    (fw) => fw.score == null || parseFloat(fw.score) < 80,
  ).length;
  const overdueCount =
    overdueControlsCount ??
    frameworks.reduce((sum, fw) => sum + (fw.overdue_count ?? 0), 0);
  const totalEndpoints = frameworks.length > 0 ? (frameworks[0].total_endpoints ?? 0) : 0;
  const lastEval = timeAgo(score.last_evaluated_at);

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        boxShadow: 'var(--shadow-sm)',
        padding: '28px 32px',
        display: 'flex',
        alignItems: 'center',
        gap: 40,
      }}
    >
      {/* Ring gauge */}
      <div style={{ flexShrink: 0 }}>
        <RingChart
          value={overall}
          max={100}
          size={128}
          thickness={11}
          color={status.color}
          label="Overall"
        />
      </div>

      {/* Info */}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 18,
            fontWeight: 700,
            color: 'var(--text-emphasis)',
            marginBottom: 4,
            letterSpacing: '-0.01em',
          }}
        >
          Overall Compliance
        </div>
        <div style={{ marginBottom: 10, display: 'flex', alignItems: 'center', gap: 10 }}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              fontWeight: 500,
              color: status.color,
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
            }}
          >
            {status.label}
          </span>
          <span style={{ color: 'var(--text-faint)', fontSize: 11 }}>·</span>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--text-muted)',
            }}
          >
            Last evaluated: <span style={{ color: 'var(--text-secondary)' }}>{lastEval}</span>
          </span>
          <span style={{ color: 'var(--text-faint)', fontSize: 11 }}>·</span>
          <span
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' }}
          >
            Target: <span style={{ color: 'var(--text-secondary)' }}>95%</span>
          </span>
        </div>
        <div
          style={{
            fontSize: 12,
            color: 'var(--text-secondary)',
            marginBottom: 20,
            lineHeight: 1.5,
          }}
        >
          {frameworks.length} active framework{frameworks.length !== 1 ? 's' : ''} across{' '}
          {totalEndpoints} endpoints.
        </div>

        {/* Stats row */}
        <div style={{ display: 'flex', gap: 28 }}>
          <StatItem
            value={compliantCount}
            label="Compliant"
            subtitle="Frameworks scoring 95%+"
            color="var(--accent)"
          />
          <StatItem
            value={needsWorkCount}
            label="Needs Work"
            subtitle="Frameworks scoring 80-94%"
            color="var(--signal-warning)"
          />
          <StatItem
            value={nonCompliantCount}
            label="Non-Compliant"
            subtitle="Frameworks below 80%"
            color="var(--signal-critical)"
          />
          <StatItem
            value={overdueCount}
            label="Overdue Controls"
            subtitle="Controls past SLA deadline"
            color={overdueCount > 0 ? 'var(--signal-critical)' : 'var(--text-muted)'}
          />
        </div>
      </div>
    </div>
  );
}

function StatItem({
  value,
  label,
  subtitle,
  color,
}: {
  value: number;
  label: string;
  subtitle?: string;
  color: string;
}) {
  return (
    <div>
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 22,
          fontWeight: 700,
          color,
          lineHeight: 1,
          marginBottom: 3,
        }}
      >
        {value}
      </div>
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          color: 'var(--text-muted)',
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
        }}
      >
        {label}
      </div>
      {subtitle && (
        <div
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 10,
            color: 'var(--text-muted)',
            marginTop: 2,
            lineHeight: 1.3,
          }}
        >
          {subtitle}
        </div>
      )}
    </div>
  );
}
