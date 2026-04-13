import { ShieldOff } from 'lucide-react';
import { useNavigate } from 'react-router';
import { useAuth } from './AuthContext';

interface RequirePermissionProps {
  resource: string;
  action: string;
  children: React.ReactNode;
}

export const RequirePermission = ({ resource, action, children }: RequirePermissionProps) => {
  const { can } = useAuth();
  const navigate = useNavigate();

  if (can(resource, action)) {
    return <>{children}</>;
  }

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '60vh',
      }}
    >
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: '1rem',
          padding: '2.5rem',
          borderRadius: '0.75rem',
          border: '1px solid var(--border)',
          backgroundColor: 'var(--bg-elevated)',
          maxWidth: '24rem',
          textAlign: 'center',
        }}
      >
        <ShieldOff size={48} style={{ color: 'var(--text-muted)' }} />
        <h2
          style={{
            fontSize: '1.25rem',
            fontWeight: 600,
            color: 'var(--text-primary)',
            margin: 0,
          }}
        >
          Access Restricted
        </h2>
        <p
          style={{
            fontSize: '0.875rem',
            color: 'var(--text-muted)',
            margin: 0,
            lineHeight: 1.5,
          }}
        >
          You don't have permission to access this page. Contact your administrator to request
          access.
        </p>
        <button
          onClick={() => navigate('/')}
          style={{
            marginTop: '0.5rem',
            padding: '0.5rem 1.25rem',
            borderRadius: '0.375rem',
            border: '1px solid var(--border)',
            backgroundColor: 'var(--bg-elevated)',
            color: 'var(--text-primary)',
            fontSize: '0.875rem',
            fontWeight: 500,
            cursor: 'pointer',
          }}
        >
          Go to Dashboard
        </button>
      </div>
    </div>
  );
};
