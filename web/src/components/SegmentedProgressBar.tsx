interface SegmentedProgressBarProps {
  succeeded: number;
  active: number;
  failed: number;
  pending: number;
  total: number;
  className?: string;
  showLabel?: boolean;
}

export const SegmentedProgressBar = ({
  succeeded,
  active,
  failed,
  pending,
  total,
  showLabel,
}: SegmentedProgressBarProps) => {
  const pct = (v: number) => (total > 0 ? (v / total) * 100 : 0);
  const overallPct = total > 0 ? Math.round((succeeded / total) * 100) : 0;

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      <div
        style={{
          display: 'flex',
          height: 8,
          flex: 1,
          overflow: 'hidden',
          borderRadius: 9999,
          background: 'var(--bg-inset)',
        }}
      >
        <div
          data-segment="succeeded"
          style={{
            height: '100%',
            background: 'var(--signal-healthy)',
            width: `${pct(succeeded)}%`,
            transition: 'width 0.5s ease',
          }}
        />
        <div
          data-segment="active"
          style={{
            height: '100%',
            background: 'var(--accent)',
            width: `${pct(active)}%`,
            transition: 'width 0.5s ease',
            animation: active > 0 ? 'pulse 2s cubic-bezier(0.4,0,0.6,1) infinite' : undefined,
          }}
        />
        <div
          data-segment="failed"
          style={{
            height: '100%',
            background: 'var(--signal-critical)',
            width: `${pct(failed)}%`,
            transition: 'width 0.5s ease',
          }}
        />
        <div
          data-segment="pending"
          style={{
            height: '100%',
            background: 'var(--border-strong)',
            width: `${pct(pending)}%`,
            transition: 'width 0.5s ease',
          }}
        />
      </div>
      {showLabel && (
        <span
          style={{
            flexShrink: 0,
            fontSize: 10,
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-muted)',
          }}
        >
          {overallPct}%
        </span>
      )}
    </div>
  );
};
