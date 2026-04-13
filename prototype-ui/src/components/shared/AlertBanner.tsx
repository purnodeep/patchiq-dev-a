import { useState } from 'react';
import { AlertTriangle, X } from 'lucide-react';

interface AlertBannerProps {
  messages: string[];
  onDismiss?: () => void;
}

export function AlertBanner({ messages, onDismiss }: AlertBannerProps) {
  const [dismissed, setDismissed] = useState(false);
  if (dismissed || messages.length === 0) return null;

  return (
    <div
      className="glass-card flex items-center gap-3 border-warning/20 px-4 py-3"
      style={{ borderColor: 'color-mix(in srgb, var(--color-warning) 20%, transparent)' }}
    >
      <div
        className="h-[7px] w-[7px] shrink-0 rounded-full bg-warning"
        style={{ boxShadow: '0 0 10px color-mix(in srgb, var(--color-warning) 50%, transparent)' }}
      />
      <AlertTriangle size={16} className="shrink-0 text-warning" />
      <div className="flex flex-wrap items-center gap-x-1 text-xs">
        {messages.map((msg, i) => (
          <span key={i}>
            {i === 0 ? (
              <span className="font-semibold">{msg}</span>
            ) : (
              <>
                <span className="text-muted">·</span> <span className="text-muted">{msg}</span>
              </>
            )}
          </span>
        ))}
      </div>
      <button
        onClick={() => {
          setDismissed(true);
          onDismiss?.();
        }}
        className="ml-auto text-muted hover:text-foreground"
      >
        <X size={16} />
      </button>
    </div>
  );
}
