import { useState } from 'react';
import { useParams, useNavigate } from 'react-router';
import { Skeleton } from '@patchiq/ui';
import { useCan } from '../../app/auth/AuthContext';
import { usePolicy, useUpdatePolicy } from '../../api/hooks/usePolicies';
import type { components } from '../../api/types';

type UpdatePolicyRequest = components['schemas']['UpdatePolicyRequest'];
import { PolicyForm, type PolicyFormValues } from './PolicyForm';

export const EditPolicyPage = () => {
  const can = useCan();
  const { id } = useParams<{ id: string }>();
  const { data: policy, isLoading } = usePolicy(id!);
  const updatePolicy = useUpdatePolicy(id!);
  const navigate = useNavigate();
  const [serverError, setServerError] = useState<{ message?: string; field?: string } | null>(null);

  if (!id)
    return (
      <div style={{ padding: 24, fontSize: 13, color: 'var(--signal-critical)' }}>
        Policy not found
      </div>
    );

  if (isLoading || !policy) {
    return (
      <div style={{ padding: 24, display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Skeleton className="h-7 w-52" />
        <Skeleton className="h-[400px] rounded-xl" />
      </div>
    );
  }

  const handleSubmit = async (values: PolicyFormValues) => {
    try {
      setServerError(null);
      await updatePolicy.mutateAsync({ ...values } as UpdatePolicyRequest);
      navigate(`/policies/${id}`);
    } catch (err: unknown) {
      const apiErr = err as { message?: string; field?: string; code?: string };
      const message =
        apiErr?.message ?? (err instanceof Error ? err.message : null) ?? 'Failed to update policy';
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
      {/* Page heading */}
      <div style={{ paddingBottom: 16, borderBottom: '1px solid var(--border)' }}>
        <h1
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 22,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            margin: 0,
            letterSpacing: '-0.01em',
          }}
        >
          Edit Policy
        </h1>
        <p style={{ fontSize: 13, color: 'var(--text-muted)', marginTop: 4 }}>
          Editing: <strong style={{ color: 'var(--text-primary)' }}>{policy.name}</strong>
        </p>
      </div>

      <PolicyForm
        defaultValues={{
          name: policy.name,
          description: policy.description ?? '',
          mode: policy.mode,
          selection_mode: policy.selection_mode,
          target_selector: policy.target_selector ?? null,
          policy_type: policy.policy_type ?? 'patch',
          timezone: policy.timezone ?? 'UTC',
          mw_enabled: policy.mw_enabled ?? false,
          min_severity: policy.min_severity ?? undefined,
          cve_ids: policy.cve_ids ?? [],
          package_regex: policy.package_regex ?? '',
          exclude_packages: policy.exclude_packages ?? [],
          schedule_type: policy.schedule_type ?? 'manual',
          schedule_cron: policy.schedule_cron ?? '',
          mw_start: policy.mw_start ?? '',
          mw_end: policy.mw_end ?? '',
        }}
        onSubmit={handleSubmit}
        submitLabel="Save Changes"
        isPending={updatePolicy.isPending}
        submitDisabled={!can('policies', 'update')}
        submitDisabledTitle={!can('policies', 'update') ? "You don't have permission" : undefined}
        serverError={serverError}
      />
    </div>
  );
};
