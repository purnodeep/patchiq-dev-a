interface ErrorAlertProps {
  message: string;
  onRetry?: () => void;
}

export const ErrorAlert = ({ message, onRetry }: ErrorAlertProps) => (
  <div
    style={{
      borderRadius: 8,
      border: '1px solid color-mix(in srgb, var(--signal-critical) 1%, transparent)',
      padding: '12px 16px',
      fontSize: 13,
      fontFamily: 'var(--font-sans)',
      background: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
      color: 'var(--signal-critical)',
    }}
  >
    {message}{' '}
    {onRetry && (
      <button
        onClick={onRetry}
        style={{
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          padding: 0,
          color: 'var(--signal-critical)',
          textDecoration: 'underline',
          fontFamily: 'var(--font-sans)',
          fontSize: 13,
        }}
      >
        Retry
      </button>
    )}
  </div>
);
