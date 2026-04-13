export function SLAEmptyState() {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 8,
        padding: '24px 0',
      }}
    >
      <svg
        viewBox="0 0 24 24"
        style={{ width: 32, height: 32, color: 'var(--text-faint)', opacity: 0.5 }}
        fill="none"
        stroke="currentColor"
        strokeWidth={1.5}
      >
        <circle cx="12" cy="12" r="10" />
        <polyline points="12 6 12 12 16 14" />
      </svg>
      <span style={{ fontSize: 12, color: 'var(--text-faint)' }}>
        SLA monitoring not configured
      </span>
    </div>
  );
}
