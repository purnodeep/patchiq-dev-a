import { ShieldCheck, ShieldAlert, ShieldX } from 'lucide-react';

interface ScoreCardProps {
  frameworkName: string;
  score: number;
  totalCves: number;
  compliantCves: number;
  atRiskCves: number;
  nonCompliantCves: number;
}

function getScoreConfig(score: number) {
  if (score >= 95) {
    return { icon: ShieldCheck, color: 'var(--accent)' };
  }
  if (score >= 80) {
    return { icon: ShieldAlert, color: 'var(--signal-warning)' };
  }
  return { icon: ShieldX, color: 'var(--signal-critical)' };
}

export const ScoreCard = ({
  frameworkName,
  score,
  totalCves,
  compliantCves,
  atRiskCves,
  nonCompliantCves,
}: ScoreCardProps) => {
  const config = getScoreConfig(score);
  const Icon = config.icon;
  const issueCount = atRiskCves + nonCompliantCves;

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderLeft: '1px solid var(--border)',
        borderRadius: 8,
        padding: 20,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <div
            style={{
              width: 38,
              height: 38,
              borderRadius: 8,
              background: 'var(--bg-inset)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Icon style={{ width: 18, height: 18, color: config.color }} />
          </div>
          <div>
            <div
              style={{
                fontFamily: 'var(--font-sans)',
                fontSize: 12,
                color: 'var(--text-muted)',
                marginBottom: 2,
              }}
            >
              {frameworkName}
            </div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 26,
                fontWeight: 700,
                color: config.color,
                lineHeight: 1,
              }}
            >
              {score}%
            </div>
          </div>
        </div>
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            color: issueCount === 0 ? 'var(--accent)' : 'var(--signal-critical)',
          }}
        >
          {issueCount === 0 ? 'All clear' : `${issueCount} issues`}
        </span>
      </div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr 1fr',
          gap: 8,
          marginTop: 16,
          textAlign: 'center',
        }}
      >
        <StatCell value={compliantCves} label="Compliant" color="var(--accent)" />
        <StatCell value={atRiskCves} label="At risk" color="var(--signal-warning)" />
        <StatCell value={nonCompliantCves} label="Non-compliant" color="var(--signal-critical)" />
      </div>

      <div
        style={{
          marginTop: 10,
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          color: 'var(--text-faint)',
        }}
      >
        {totalCves} CVEs evaluated
      </div>
    </div>
  );
};

function StatCell({ value, label, color }: { value: number; label: string; color: string }) {
  return (
    <div>
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 15,
          fontWeight: 700,
          color,
          marginBottom: 2,
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
    </div>
  );
}
