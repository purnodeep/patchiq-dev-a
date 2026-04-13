import { Clock, AlertTriangle } from 'lucide-react';
import type { ComplianceEvaluation } from '../../../api/hooks/useCompliance';

interface SlaTrackerProps {
  evaluations: ComplianceEvaluation[];
}

function getUrgencyStyle(daysRemaining: number | null): { text: string; color: string } {
  if (daysRemaining === null || daysRemaining < 0) {
    return { text: 'Overdue', color: 'var(--signal-critical)' };
  }
  if (daysRemaining === 0) {
    return { text: 'Due today', color: 'var(--signal-critical)' };
  }
  if (daysRemaining <= 3) {
    return { text: `${daysRemaining}d left`, color: 'var(--signal-warning)' };
  }
  if (daysRemaining <= 7) {
    return { text: `${daysRemaining}d left`, color: 'var(--signal-warning)' };
  }
  return { text: `${daysRemaining}d left`, color: 'var(--text-secondary)' };
}

export const SlaTracker = ({ evaluations }: SlaTrackerProps) => {
  const urgent = evaluations
    .filter((e) => e.state === 'AT_RISK' || e.state === 'NON_COMPLIANT')
    .sort((a, b) => {
      const aDays = a.days_remaining ?? -Infinity;
      const bDays = b.days_remaining ?? -Infinity;
      return aDays - bDays;
    })
    .slice(0, 10);

  const headerRow = (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        marginBottom: 12,
      }}
    >
      <Clock style={{ width: 15, height: 15, color: 'var(--text-muted)' }} />
      <span
        style={{
          fontFamily: 'var(--font-sans)',
          fontSize: 14,
          fontWeight: 600,
          color: 'var(--text-primary)',
        }}
      >
        SLA Deadlines
      </span>
    </div>
  );

  if (urgent.length === 0) {
    return (
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          padding: '16px 20px',
        }}
      >
        {headerRow}
        <p
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 12,
            color: 'var(--text-muted)',
            margin: 0,
          }}
        >
          No urgent SLA deadlines.
        </p>
      </div>
    );
  }

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        padding: '16px 20px',
      }}
    >
      {headerRow}
      <ul
        style={{
          listStyle: 'none',
          margin: 0,
          padding: 0,
          display: 'flex',
          flexDirection: 'column',
          gap: 8,
        }}
      >
        {urgent.map((e) => {
          const urgency = getUrgencyStyle(e.days_remaining ?? null);
          return (
            <li
              key={e.id}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <AlertTriangle
                  style={{ width: 13, height: 13, color: urgency.color, flexShrink: 0 }}
                />
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 11,
                    color: 'var(--text-secondary)',
                  }}
                >
                  {e.cve_id}
                </span>
              </div>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 11,
                  fontWeight: 500,
                  color: urgency.color,
                }}
              >
                {urgency.text}
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
};
