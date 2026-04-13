export function ProgressBar({ value, max, color }: { value: number; max: number; color: string }) {
  const pct = max > 0 ? Math.min(100, (value / max) * 100) : 0;
  return (
    <div
      style={{
        height: 3,
        width: '100%',
        borderRadius: 999,
        background: 'var(--border)',
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          height: '100%',
          width: `${pct}%`,
          borderRadius: 999,
          backgroundColor: color,
          transition: 'width 0.3s ease',
        }}
      />
    </div>
  );
}
