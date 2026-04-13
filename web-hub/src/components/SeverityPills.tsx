interface SeverityPillsProps {
  counts: { all: number; critical: number; high: number; medium: number; low: number };
  selected: string;
  onSelect: (severity: string) => void;
}

const SEVERITY_CONFIG: { key: string; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'critical', label: 'Critical' },
  { key: 'high', label: 'High' },
  { key: 'medium', label: 'Medium' },
  { key: 'low', label: 'Low' },
];

export function SeverityPills({ counts, selected, onSelect }: SeverityPillsProps) {
  return (
    <div className="flex flex-wrap gap-2">
      {SEVERITY_CONFIG.map(({ key, label }) => {
        const isActive = selected === key;
        return (
          <button
            key={key}
            onClick={() => onSelect(key)}
            className="inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-sm font-medium transition-colors"
            style={{
              background: isActive ? 'var(--accent)' : 'var(--bg-card)',
              color: isActive ? 'var(--text-emphasis)' : 'var(--text-muted)',
              border: `1px solid ${isActive ? 'var(--accent-border)' : 'var(--border)'}`,
            }}
          >
            {label}
            <span
              className="rounded-full px-1.5 py-0.5 text-xs"
              style={{ background: 'var(--bg-inset)' }}
            >
              {counts[key as keyof typeof counts]}
            </span>
          </button>
        );
      })}
    </div>
  );
}
