import { type ReactNode } from 'react';

interface MonoTagProps {
  children: ReactNode;
}

function MonoTag({ children }: MonoTagProps) {
  return (
    <span
      className="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium"
      style={{
        backgroundColor: 'var(--bg-card-hover)',
        borderWidth: '1px',
        borderStyle: 'solid',
        borderColor: 'var(--border-strong)',
        color: 'var(--text-secondary)',
        fontFamily: 'var(--font-mono)',
      }}
    >
      {children}
    </span>
  );
}

export { MonoTag };
export type { MonoTagProps };
