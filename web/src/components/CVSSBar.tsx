interface CVSSBarProps {
  score: number | null;
  className?: string;
}

export function CVSSBar({ score }: CVSSBarProps) {
  if (score === null || score === undefined) {
    return (
      <span style={{ fontSize: 11, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>
        —
      </span>
    );
  }

  const barColor =
    score >= 9
      ? 'var(--signal-critical)'
      : score >= 7
        ? 'var(--signal-warning)'
        : score >= 4
          ? 'var(--signal-medium, #eab308)'
          : 'var(--signal-healthy)';
  const textColor = barColor;

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
      <div
        style={{
          width: 64,
          height: 6,
          borderRadius: 9999,
          background: 'var(--bg-inset)',
          overflow: 'hidden',
          flexShrink: 0,
        }}
      >
        <div
          style={{
            height: '100%',
            borderRadius: 9999,
            background: barColor,
            width: `${score * 10}%`,
          }}
        />
      </div>
      <span
        style={{
          fontSize: 11,
          fontWeight: 700,
          fontFamily: 'var(--font-mono)',
          color: textColor,
          fontVariantNumeric: 'tabular-nums',
        }}
      >
        {score.toFixed(1)}
      </span>
    </div>
  );
}
