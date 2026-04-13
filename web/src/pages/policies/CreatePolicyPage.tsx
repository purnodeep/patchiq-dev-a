import { useState } from 'react';
import { Link, useNavigate } from 'react-router';
import { ArrowLeft } from 'lucide-react';
import { useCan } from '../../app/auth/AuthContext';
import { useCreatePolicy } from '../../api/hooks/usePolicies';
import { PolicyForm, type PolicyFormValues } from './PolicyForm';
import type { components } from '../../api/types';

export const CreatePolicyPage = () => {
  const can = useCan();
  const createPolicy = useCreatePolicy();
  const navigate = useNavigate();
  const [serverError, setServerError] = useState<{ message?: string; field?: string } | null>(null);

  const handleSubmit = async (values: PolicyFormValues) => {
    try {
      setServerError(null);
      await createPolicy.mutateAsync({ ...values } as components['schemas']['CreatePolicyRequest']);
      navigate('/policies');
    } catch (err: unknown) {
      const apiErr = err as { message?: string; field?: string; code?: string };
      const message =
        apiErr?.message ?? (err instanceof Error ? err.message : null) ?? 'Failed to create policy';
      const field = apiErr?.field;
      setServerError({ message, field });
    }
  };

  return (
    <div
      style={{
        padding: 24,
        background: 'var(--bg-page)',
        minHeight: '100%',
        display: 'flex',
        flexDirection: 'column',
        gap: 20,
      }}
    >
      {/* Back link */}
      <Link
        to="/policies"
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 6,
          fontSize: 12,
          color: 'var(--text-muted)',
          textDecoration: 'none',
        }}
        onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--accent)')}
        onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--text-muted)')}
      >
        <ArrowLeft style={{ width: 14, height: 14 }} />
        Back to Policies
      </Link>

      {/* Page heading */}
      <div style={{ paddingBottom: 16, borderBottom: '1px solid var(--border)' }}>
        <h1
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 20,
            fontWeight: 700,
            color: 'var(--text-emphasis)',
            margin: 0,
            letterSpacing: '-0.01em',
          }}
        >
          Create Policy
        </h1>
        <p style={{ fontSize: 13, color: 'var(--text-muted)', marginTop: 4 }}>
          Define how patches are selected and deployed to your endpoints.
        </p>
      </div>

      <PolicyForm
        onSubmit={handleSubmit}
        submitLabel="Create Policy"
        isPending={createPolicy.isPending}
        submitDisabled={!can('policies', 'create')}
        submitDisabledTitle={!can('policies', 'create') ? "You don't have permission" : undefined}
        serverError={serverError}
      />
    </div>
  );
};
