interface CVSSScoreProps {
  score: number | string | null | undefined;
  className?: string;
}

function scoreColor(score: number): string {
  if (score >= 9.0) return 'var(--signal-critical)';
  if (score >= 7.0) return 'var(--signal-warning)';
  if (score >= 4.0) return 'var(--text-secondary)';
  return 'var(--text-muted)';
}

export const CVSSScore = ({ score }: CVSSScoreProps) => {
  if (score === null || score === undefined) {
    return (
      <span style={{ fontSize: 13, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>
        —
      </span>
    );
  }
  const numeric = typeof score === 'string' ? parseFloat(score) : score;
  if (isNaN(numeric)) {
    return (
      <span style={{ fontSize: 13, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>
        —
      </span>
    );
  }
  return (
    <span
      style={{
        fontSize: 13,
        fontWeight: 600,
        fontFamily: 'var(--font-mono)',
        color: scoreColor(numeric),
      }}
    >
      {numeric.toFixed(1)}
    </span>
  );
};
