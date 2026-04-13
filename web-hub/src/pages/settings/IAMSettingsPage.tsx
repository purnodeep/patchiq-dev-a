import { useCallback, useState } from 'react';
import { IAMSettings } from './IAMSettings';
import { useSettings, useUpsertSetting } from '../../api/hooks/useSettings';
import { SkeletonCard } from '@patchiq/ui';

export const IAMSettingsPage = () => {
  const { data: settings, isLoading, isError, error } = useSettings();
  const upsertSetting = useUpsertSetting();
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const handleSave = useCallback(
    async (entries: Record<string, unknown>) => {
      setSaving(true);
      setSaveError(null);
      try {
        await Promise.all(
          Object.entries(entries).map(([key, value]) => upsertSetting.mutateAsync({ key, value })),
        );
      } catch (err) {
        setSaveError(err instanceof Error ? err.message : 'Failed to save settings');
      } finally {
        setSaving(false);
      }
    },
    [upsertSetting],
  );

  if (isLoading) {
    return (
      <div style={{ padding: '28px 40px 80px', maxWidth: 800 }}>
        <SkeletonCard />
      </div>
    );
  }

  if (isError) {
    return (
      <div style={{ padding: '28px 40px 80px', maxWidth: 800 }}>
        <p
          style={{ fontSize: 13, color: 'var(--signal-critical)', fontFamily: 'var(--font-sans)' }}
        >
          Failed to load settings: {error?.message ?? 'Unknown error'}
        </p>
      </div>
    );
  }

  const s = (settings ?? {}) as Record<string, unknown>;

  return (
    <div style={{ padding: '28px 40px 80px', maxWidth: 800 }}>
      {/* Section header */}
      <div style={{ paddingBottom: 16, borderBottom: '1px solid var(--border)', marginBottom: 24 }}>
        <h2
          style={{
            fontSize: 18,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            letterSpacing: '-0.02em',
            margin: 0,
          }}
        >
          Identity & Access
        </h2>
        <p
          style={{
            fontSize: 12,
            color: 'var(--text-muted)',
            margin: '4px 0 0',
          }}
        >
          Single sign-on configuration, OIDC provider settings, and role mappings.
        </p>
      </div>

      {saveError && (
        <div
          style={{
            marginBottom: 16,
            padding: '10px 14px',
            borderRadius: 6,
            background: 'var(--signal-critical-subtle)',
            border: '1px solid var(--signal-critical)',
            color: 'var(--signal-critical)',
            fontSize: 13,
            fontFamily: 'var(--font-sans)',
          }}
        >
          {saveError}
        </div>
      )}

      <IAMSettings settings={s} onSave={handleSave} saving={saving} />
    </div>
  );
};
