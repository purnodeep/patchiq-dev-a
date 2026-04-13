import { useMemo } from 'react';
import { RingGauge } from '@patchiq/ui';
import type { PolicyFormValues } from './PolicyForm';
import type { Selector } from '../../types/targeting';
import { useValidateSelector } from '../../api/hooks/useTagSelector';

// ── Design tokens (inline styles matching the PolicyForm's existing style system) ─

const CARD: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  padding: '14px 14px',
};

const LABEL: React.CSSProperties = {
  fontSize: 10,
  fontWeight: 600,
  textTransform: 'uppercase' as const,
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  marginBottom: 8,
  display: 'block',
};

// ── Ring gauge wrapper ─────────────────────────────────────────────────────────

function FleetRingGauge({
  pct,
  matchedCount,
  totalCount,
}: {
  pct: number;
  matchedCount: number;
  totalCount: number;
}) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6 }}>
      <RingGauge value={pct} size={80} strokeWidth={8} colorByValue />
      <div
        style={{
          fontSize: 10,
          color: 'var(--text-muted)',
          textAlign: 'center',
          lineHeight: 1.4,
        }}
      >
        {matchedCount} of {totalCount} endpoints matched
      </div>
    </div>
  );
}

// ── Severity bar ───────────────────────────────────────────────────────────────

function SeverityBar({
  mode,
  minSeverity,
}: {
  mode: PolicyFormValues['selection_mode'];
  minSeverity?: PolicyFormValues['min_severity'];
}) {
  // Approximate visual split based on selection mode
  const segments: { color: string; width: number; label: string }[] = [];

  if (mode === 'all_available') {
    segments.push(
      { color: 'var(--signal-critical)', width: 20, label: 'crit' },
      { color: 'var(--signal-warning)', width: 30, label: 'high' },
      { color: 'var(--border)', width: 50, label: 'other' },
    );
  } else if (mode === 'by_severity') {
    switch (minSeverity) {
      case 'critical':
        segments.push({ color: 'var(--signal-critical)', width: 100, label: 'critical' });
        break;
      case 'high':
        segments.push(
          { color: 'var(--signal-critical)', width: 35, label: 'crit' },
          { color: 'var(--signal-warning)', width: 65, label: 'high' },
        );
        break;
      case 'medium':
        segments.push(
          { color: 'var(--signal-critical)', width: 20, label: 'crit' },
          { color: 'var(--signal-warning)', width: 30, label: 'high' },
          { color: 'var(--text-muted)', width: 50, label: 'med' },
        );
        break;
      default:
        segments.push(
          { color: 'var(--signal-critical)', width: 20, label: 'crit' },
          { color: 'var(--signal-warning)', width: 30, label: 'high' },
          { color: 'var(--border)', width: 50, label: 'other' },
        );
    }
  } else {
    // by_cve_list / by_regex — mixed
    segments.push(
      { color: 'var(--signal-critical)', width: 40, label: 'crit' },
      { color: 'var(--signal-warning)', width: 40, label: 'high' },
      { color: 'var(--border)', width: 20, label: 'other' },
    );
  }

  const severityLabel = useMemo(() => {
    if (mode === 'all_available') return 'All available patches';
    if (mode === 'by_severity') {
      switch (minSeverity) {
        case 'critical':
          return 'Critical only';
        case 'high':
          return 'Critical + High';
        case 'medium':
          return 'Critical + High + Medium';
        default:
          return 'By severity (unset)';
      }
    }
    if (mode === 'by_cve_list') return 'By CVE list';
    if (mode === 'by_regex') return 'By package regex';
    return '';
  }, [mode, minSeverity]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      <div
        style={{ display: 'flex', height: 6, borderRadius: 3, overflow: 'hidden', width: '100%' }}
      >
        {segments.map((s, i) => (
          <div key={i} style={{ width: `${s.width}%`, background: s.color, flexShrink: 0 }} />
        ))}
      </div>
      <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>{severityLabel}</div>
    </div>
  );
}

// ── Schedule preview ───────────────────────────────────────────────────────────

// Parse a simple cron expression: "MIN HOUR * * DOW"
function parseCron(cron: string): { nextRun: string; description: string } | null {
  if (!cron.trim()) return null;

  const parts = cron.trim().split(/\s+/);
  if (parts.length < 5) return null;

  const [min, hour, , , dow] = parts;

  const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

  const parsedMin = parseInt(min, 10);
  const parsedHour = parseInt(hour, 10);
  const parsedDow = dow === '*' ? -1 : parseInt(dow, 10);

  if (isNaN(parsedMin) || isNaN(parsedHour)) return null;

  const now = new Date();
  const next = new Date(now);
  next.setSeconds(0, 0);
  next.setMinutes(parsedMin);
  next.setHours(parsedHour);

  if (parsedDow >= 0 && parsedDow <= 6) {
    // Find next occurrence of that weekday
    const currentDay = next.getDay();
    let daysUntil = (parsedDow - currentDay + 7) % 7;
    if (daysUntil === 0 && next <= now) daysUntil = 7;
    next.setDate(next.getDate() + daysUntil);
  } else {
    // Daily — if time already passed today, move to tomorrow
    if (next <= now) next.setDate(next.getDate() + 1);
  }

  const timeStr = `${String(parsedHour).padStart(2, '0')}:${String(parsedMin).padStart(2, '0')} UTC`;

  const description =
    parsedDow >= 0 && parsedDow <= 6
      ? `Weekly on ${days[parsedDow]} at ${timeStr}`
      : `Daily at ${timeStr}`;

  // Format next run date
  const nextRunDate = next.toLocaleDateString('en-US', {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
  });
  const nextRun = `${nextRunDate}, ${timeStr}`;

  return { nextRun, description };
}

// 7-day calendar strip
function WeekStrip({ highlightDow }: { highlightDow: number }) {
  const DAY_ABBR = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'];
  const today = new Date().getDay();

  return (
    <div style={{ display: 'flex', gap: 3 }}>
      {DAY_ABBR.map((d, i) => {
        const isScheduled = i === highlightDow;
        const isToday = i === today;
        return (
          <div
            key={i}
            style={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: 2,
            }}
          >
            <div
              style={{
                fontSize: 8,
                color: isToday ? 'var(--accent)' : 'var(--text-muted)',
                fontWeight: isToday ? 600 : 400,
              }}
            >
              {d}
            </div>
            <div
              style={{
                width: 18,
                height: 18,
                borderRadius: '50%',
                background: isScheduled ? 'var(--accent)' : 'transparent',
                border: `1px solid ${isScheduled ? 'var(--accent)' : 'var(--border)'}`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              {isScheduled && (
                <div
                  style={{
                    width: 5,
                    height: 5,
                    borderRadius: '50%',
                    background: 'var(--text-emphasis)',
                  }}
                />
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}

function SchedulePreview({
  scheduleType,
  scheduleCron,
}: {
  scheduleType: PolicyFormValues['schedule_type'];
  scheduleCron: string;
}) {
  if (scheduleType === 'manual') {
    return (
      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Manual — trigger on demand</div>
    );
  }

  const parsed = parseCron(scheduleCron);

  if (!parsed) {
    return (
      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Enter a cron expression above</div>
    );
  }

  const parts = scheduleCron.trim().split(/\s+/);
  const dow = parts[4] === '*' ? -1 : parseInt(parts[4], 10);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <div>
        <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>Next run</div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            color: 'var(--text-primary)',
            marginTop: 2,
          }}
        >
          {parsed.nextRun}
        </div>
      </div>
      <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>{parsed.description}</div>
      {dow >= 0 && dow <= 6 && <WeekStrip highlightDow={dow} />}
    </div>
  );
}

// ── Risk reduction estimate ────────────────────────────────────────────────────

function RiskReduction({
  matchedCount,
  selectionMode,
  minSeverity,
}: {
  matchedCount: number;
  selectionMode: PolicyFormValues['selection_mode'];
  minSeverity?: PolicyFormValues['min_severity'];
}) {
  // Heuristic: base risk 4.7, reduction based on severity scope and endpoint count
  const baseRisk = 4.7;
  const reductionFactor = useMemo(() => {
    if (matchedCount === 0) return 0;
    let factor: number;
    if (selectionMode === 'all_available') {
      factor = 0.5;
    } else if (selectionMode === 'by_severity') {
      if (minSeverity === 'critical') factor = 0.35;
      else if (minSeverity === 'high') factor = 0.45;
      else factor = 0.5;
    } else {
      factor = 0.3;
    }
    // Scale slightly with endpoint count (more endpoints = more impact)
    const scale = Math.min(1.3, 1 + matchedCount / 500);
    return Math.min(baseRisk - 0.5, factor * scale);
  }, [matchedCount, selectionMode, minSeverity]);

  const afterRisk = Math.max(0.5, baseRisk - reductionFactor).toFixed(1);
  const reduction = reductionFactor.toFixed(1);

  if (matchedCount === 0) {
    return (
      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>
        Define target predicates to see estimate
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 13,
          fontWeight: 700,
          color: 'var(--signal-healthy)',
        }}
      >
        −{reduction}
      </div>
      <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>
        Avg risk: <span style={{ color: 'var(--signal-critical)' }}>{baseRisk}</span> →{' '}
        <span style={{ color: 'var(--signal-healthy)' }}>{afterRisk}</span>
      </div>
    </div>
  );
}

// ── Main export ────────────────────────────────────────────────────────────────

interface PolicyPreviewProps {
  values: PolicyFormValues;
}

export function PolicyPreview({ values }: PolicyPreviewProps) {
  // Fleet coverage comes from the authoritative backend resolver count —
  // previously this summed endpoint_count across selected groups, which
  // overcounted when groups overlapped. The /tags/selectors/validate
  // endpoint returns the exact tenant-scoped resolver count.
  const selector = (values.target_selector ?? null) as Selector | null;
  const validation = useValidateSelector(selector);
  const matchedCount = validation.data?.matched_count ?? 0;
  // Total count isn't a thing with tag selectors — for the RingGauge we
  // fall back to a soft target so non-zero matches always show as "some
  // coverage" rather than 0/0. Replace with a tenant-wide count call in
  // a follow-up when we add GET /api/v1/endpoints/count.
  const totalCount = matchedCount;
  const pct = matchedCount > 0 ? 100 : 0;

  const patchCountLabel = useMemo(() => {
    switch (values.selection_mode) {
      case 'all_available':
        return 'All available';
      case 'by_severity':
        if (values.min_severity === 'critical') return 'Critical only';
        if (values.min_severity === 'high') return 'Critical + High';
        if (values.min_severity === 'medium') return 'Crit + High + Med';
        return 'By severity';
      case 'by_cve_list':
        return `${values.cve_ids?.length ?? 0} CVE${(values.cve_ids?.length ?? 0) !== 1 ? 's' : ''}`;
      case 'by_regex':
        return 'By regex';
      default:
        return 'Unknown';
    }
  }, [values.selection_mode, values.min_severity, values.cve_ids]);

  return (
    <div
      style={{
        position: 'sticky',
        top: 20,
        width: 280,
        flexShrink: 0,
        display: 'flex',
        flexDirection: 'column',
        gap: 10,
        alignSelf: 'flex-start',
      }}
    >
      {/* Header */}
      <div
        style={{
          fontSize: 11,
          fontWeight: 600,
          color: 'var(--text-muted)',
          paddingBottom: 8,
          borderBottom: '1px solid var(--border)',
        }}
      >
        Target Preview
      </div>

      <div style={{ marginBottom: 12 }}>
        <span style={LABEL}>Policy Type</span>
        <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)' }}>
          {values.policy_type === 'patch'
            ? 'Patch Policy'
            : values.policy_type === 'deploy'
              ? 'Deploy Policy'
              : 'Compliance Policy'}
        </div>
      </div>

      {/* Ring gauge — fleet coverage */}
      <div style={CARD}>
        <span style={LABEL}>Fleet Coverage</span>
        <FleetRingGauge pct={pct} matchedCount={matchedCount} totalCount={totalCount} />
      </div>

      {/* Patch impact */}
      <div style={CARD}>
        <span style={LABEL}>Patch Impact</span>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div style={{ fontSize: 11, color: 'var(--text-primary)', fontWeight: 500 }}>
            {patchCountLabel}
          </div>
          <SeverityBar mode={values.selection_mode} minSeverity={values.min_severity} />
        </div>
      </div>

      {/* Schedule preview */}
      <div style={CARD}>
        <span style={LABEL}>Schedule</span>
        <SchedulePreview scheduleType={values.schedule_type} scheduleCron={values.schedule_cron} />
      </div>

      {/* Risk reduction */}
      <div style={CARD}>
        <span style={LABEL}>Risk Reduction Est.</span>
        <RiskReduction
          matchedCount={matchedCount}
          selectionMode={values.selection_mode}
          minSeverity={values.min_severity}
        />
      </div>
    </div>
  );
}
