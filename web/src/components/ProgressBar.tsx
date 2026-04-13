interface ProgressBarProps {
  value: number;
  max: number;
  className?: string;
  color?: string;
}

export const ProgressBar = ({ value, max, color }: ProgressBarProps) => {
  const pct = max > 0 ? Math.min(100, Math.round((value / max) * 100)) : 0;
  const barColor = color ?? 'var(--accent)';
  return (
    <div
      style={{
        height: 8,
        width: '100%',
        overflow: 'hidden',
        borderRadius: 9999,
        background: 'var(--bg-inset)',
      }}
    >
      <div
        role="progressbar"
        aria-valuenow={pct}
        aria-valuemin={0}
        aria-valuemax={100}
        style={{
          height: '100%',
          borderRadius: 9999,
          background: barColor,
          width: `${pct}%`,
          transition: 'width 0.5s ease-out',
        }}
      />
    </div>
  );
};
