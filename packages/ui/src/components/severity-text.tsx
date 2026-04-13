interface SeverityTextProps {
  severity: string;
}

function getSeverityColor(severity: string): string {
  switch (severity.toLowerCase()) {
    case 'critical':
      return 'var(--signal-critical)';
    case 'high':
      return 'var(--signal-warning)';
    case 'medium':
    case 'low':
    default:
      return 'var(--text-secondary)';
  }
}

function capitalize(s: string): string {
  if (s.length === 0) return s;
  return s.charAt(0).toUpperCase() + s.slice(1).toLowerCase();
}

function SeverityText({ severity }: SeverityTextProps) {
  return (
    <span className="text-sm font-medium" style={{ color: getSeverityColor(severity) }}>
      {capitalize(severity)}
    </span>
  );
}

export { SeverityText };
export type { SeverityTextProps };
