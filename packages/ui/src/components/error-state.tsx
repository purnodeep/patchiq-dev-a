import { AlertCircle } from 'lucide-react';
import { Button } from '../components/ui/button';
import { cn } from '../lib/utils';

interface ErrorStateProps {
  title?: string;
  message: string;
  onRetry?: () => void;
  className?: string;
}

function ErrorState({
  title = 'Something went wrong',
  message,
  onRetry,
  className,
}: ErrorStateProps) {
  return (
    <div
      className={cn('flex flex-col items-center justify-center text-center', className)}
      style={{ padding: '40px 20px' }}
    >
      <div
        style={{
          marginBottom: 12,
          borderRadius: 9999,
          background: 'var(--bg-inset)',
          padding: 12,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <AlertCircle size={24} style={{ color: 'var(--signal-critical)' }} />
      </div>
      <h3
        style={{
          fontFamily: 'var(--font-sans)',
          fontSize: 13,
          fontWeight: 500,
          color: 'var(--text-primary)',
          margin: 0,
        }}
      >
        {title}
      </h3>
      <p
        style={{
          marginTop: 4,
          maxWidth: 360,
          fontFamily: 'var(--font-sans)',
          fontSize: 12,
          color: 'var(--text-muted)',
          lineHeight: 1.5,
        }}
      >
        {message}
      </p>
      {onRetry && (
        <div style={{ marginTop: 16 }}>
          <Button type="button" size="sm" onClick={onRetry}>
            Retry
          </Button>
        </div>
      )}
    </div>
  );
}

export { ErrorState };
export type { ErrorStateProps };
