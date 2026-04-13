import { useEffect } from 'react';
import { useFormContext } from 'react-hook-form';
import { Zap, Hand, Eye } from 'lucide-react';
import type { PolicyWizardValues } from './types';
import { LABEL_STYLE, INPUT } from './types';

const typeOptions = [
  {
    value: 'patch' as const,
    label: 'Patch Policy',
    description: 'Select patches by severity, CVE, or regex. Evaluate and optionally auto-deploy.',
    color: 'var(--accent)',
  },
  {
    value: 'deploy' as const,
    label: 'Deploy Policy',
    description: 'Target specific updates for direct deployment to endpoints.',
    color: 'var(--signal-healthy)',
  },
  {
    value: 'compliance' as const,
    label: 'Compliance Policy',
    description: 'Evaluate patch compliance on a schedule. Report only, no deployments.',
    color: 'var(--text-muted)',
  },
];

// Large source-style cards with icon — matches DeploymentWizard SourceStep card design
const modeOptions = [
  {
    value: 'automatic' as const,
    label: 'Automatic',
    description:
      'Evaluates on schedule. Matching patches deploy automatically within the maintenance window.',
    Icon: Zap,
    color: 'var(--signal-healthy)',
  },
  {
    value: 'manual' as const,
    label: 'Manual',
    description:
      'Evaluates on schedule. Patches are queued but NOT deployed until you click Deploy.',
    Icon: Hand,
    color: 'var(--accent)',
  },
  {
    value: 'advisory' as const,
    label: 'Advisory',
    description: 'Evaluates on schedule. Reports compliance only. No patches are ever deployed.',
    Icon: Eye,
    color: 'var(--text-muted)',
  },
];

export function BasicsStep() {
  const {
    register,
    watch,
    setValue,
    formState: { errors },
  } = useFormContext<PolicyWizardValues>();

  const mode = watch('mode');
  const policyType = watch('policy_type');

  useEffect(() => {
    if (policyType === 'compliance') setValue('mode', 'advisory');
  }, [policyType, setValue]);

  function getFilteredModes(type: string) {
    if (type === 'deploy') return modeOptions.filter((o) => o.value !== 'advisory');
    if (type === 'compliance') return modeOptions.filter((o) => o.value === 'advisory');
    return modeOptions;
  }

  const filteredModes = getFilteredModes(policyType);

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 18 }}>
      {/* Policy Type */}
      <div>
        <label style={LABEL_STYLE}>Policy Type</label>
        <div style={{ display: 'flex', gap: 8 }}>
          {typeOptions.map((opt) => {
            const isActive = policyType === opt.value;
            return (
              <button
                key={opt.value}
                type="button"
                onClick={() => setValue('policy_type', opt.value)}
                style={{
                  flex: 1,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  gap: 8,
                  padding: '14px 10px',
                  borderRadius: 8,
                  border: `1.5px solid ${isActive ? opt.color : 'var(--border)'}`,
                  background: isActive
                    ? `color-mix(in srgb, ${opt.color} 8%, transparent)`
                    : 'var(--bg-inset)',
                  cursor: 'pointer',
                  transition: 'border-color 0.15s, background 0.15s',
                  outline: 'none',
                  textAlign: 'center',
                }}
              >
                <div>
                  <div
                    style={{
                      fontSize: 12,
                      fontWeight: 600,
                      color: isActive ? opt.color : 'var(--text-primary)',
                      marginBottom: 3,
                    }}
                  >
                    {opt.label}
                  </div>
                  <div style={{ fontSize: 10, color: 'var(--text-muted)', lineHeight: 1.3 }}>
                    {opt.description}
                  </div>
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* Name */}
      <div>
        <label style={LABEL_STYLE}>
          Policy Name <span style={{ color: 'var(--signal-critical)' }}>*</span>
        </label>
        <input
          {...register('name')}
          placeholder="e.g. Linux Critical Patching"
          style={INPUT}
          onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
          onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
        />
        {errors.name && (
          <p style={{ marginTop: 4, fontSize: 11, color: 'var(--signal-critical)' }}>
            {errors.name.message}
          </p>
        )}
      </div>

      {/* Description */}
      <div>
        <label style={LABEL_STYLE}>Description (optional)</label>
        <textarea
          {...register('description')}
          rows={2}
          placeholder="What does this policy do?"
          style={{
            ...INPUT,
            height: 'auto',
            minHeight: 60,
            padding: '8px 10px',
            resize: 'vertical',
            lineHeight: 1.5,
          }}
          onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
          onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
        />
      </div>

      {/* Mode — large icon cards matching DeploymentWizard SourceStep */}
      <div>
        <label style={LABEL_STYLE}>Deployment Mode</label>
        <div style={{ display: 'flex', gap: 8 }}>
          {filteredModes.map((opt) => {
            const isActive = mode === opt.value;
            return (
              <button
                key={opt.value}
                type="button"
                onClick={() => setValue('mode', opt.value)}
                style={{
                  flex: 1,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  gap: 8,
                  padding: '14px 10px',
                  borderRadius: 8,
                  border: `1.5px solid ${isActive ? opt.color : 'var(--border)'}`,
                  background: isActive
                    ? `color-mix(in srgb, ${opt.color} 8%, transparent)`
                    : 'var(--bg-inset)',
                  cursor: 'pointer',
                  transition: 'border-color 0.15s, background 0.15s',
                  outline: 'none',
                  textAlign: 'center',
                }}
              >
                <opt.Icon
                  style={{
                    width: 20,
                    height: 20,
                    color: isActive ? opt.color : 'var(--text-muted)',
                    transition: 'color 0.15s',
                  }}
                />
                <div>
                  <div
                    style={{
                      fontSize: 12,
                      fontWeight: 600,
                      color: isActive ? opt.color : 'var(--text-primary)',
                      marginBottom: 3,
                    }}
                  >
                    {opt.label}
                  </div>
                  <div style={{ fontSize: 10, color: 'var(--text-muted)', lineHeight: 1.3 }}>
                    {opt.description}
                  </div>
                </div>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
