interface SyncDotsProps {
  synced: number;
  total: number;
}

const MAX_DOTS = 20;

export function SyncDots({ synced, total }: SyncDotsProps) {
  if (total === 0)
    return (
      <span className="text-xs" style={{ color: 'var(--text-faint)' }}>
        --
      </span>
    );

  // For large counts, use a progress bar instead of individual dots.
  if (total > MAX_DOTS) {
    const pct = Math.round((synced / total) * 100);
    return (
      <div className="flex items-center gap-2" title={`${synced} of ${total} synced`}>
        <div
          className="h-2 w-16 rounded-full overflow-hidden"
          style={{ background: 'var(--border)' }}
        >
          <div
            className="h-full rounded-full"
            style={{ width: `${pct}%`, background: 'var(--accent)' }}
          />
        </div>
      </div>
    );
  }

  const dots = Array.from({ length: total }, (_, i) => i < synced);

  return (
    <div className="flex flex-wrap gap-1" title={`${synced} of ${total} synced`}>
      {dots.map((isSynced, i) => (
        <span
          key={i}
          className="inline-block h-2 w-2 rounded-full"
          style={{ background: isSynced ? 'var(--accent)' : 'var(--border)' }}
        />
      ))}
    </div>
  );
}
