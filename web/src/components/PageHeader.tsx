import type { ReactNode } from 'react';

interface Breadcrumb {
  label: string;
  href?: string;
}

interface PageHeaderProps {
  breadcrumbs?: Breadcrumb[];
  title: string;
  count?: number;
  actions?: ReactNode;
  className?: string;
}

export const PageHeader = ({ title, count, actions }: PageHeaderProps) => (
  <div
    style={{
      display: 'flex',
      alignItems: 'center',
      gap: 12,
      paddingBottom: 16,
      marginBottom: 20,
      borderBottom: '1px solid var(--border)',
    }}
  >
    <div style={{ flex: 1 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <h1
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 22,
            fontWeight: 600,
            letterSpacing: '-0.02em',
            color: 'var(--text-emphasis)',
            margin: 0,
          }}
        >
          {title}
        </h1>
        {count != null && (
          <span
            style={{
              borderRadius: 9999,
              border: '1px solid var(--border)',
              backgroundColor: 'var(--bg-card-hover)',
              color: 'var(--text-secondary)',
              padding: '2px 10px',
              fontSize: 13,
              fontWeight: 600,
              fontFamily: 'var(--font-mono)',
            }}
          >
            {count}
          </span>
        )}
      </div>
    </div>
    {actions && <div style={{ display: 'flex', gap: 8 }}>{actions}</div>}
  </div>
);
