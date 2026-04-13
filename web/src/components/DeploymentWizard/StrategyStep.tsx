import { useFormContext } from 'react-hook-form';
import { Plus, X } from 'lucide-react';
import { Switch } from '@patchiq/ui';
import { useWorkflowTemplates } from '../../api/hooks/useWorkflows';
import type { DeploymentWizardValues, WaveConfig } from '../../types/deployment-wizard';

const LABEL_STYLE: React.CSSProperties = {
  fontSize: 10,
  fontWeight: 600,
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  fontFamily: 'var(--font-mono)',
  marginBottom: 8,
  display: 'block',
};

const INPUT: React.CSSProperties = {
  width: '100%',
  background: 'var(--bg-inset)',
  border: '1px solid var(--border)',
  borderRadius: 5,
  padding: '5px 8px',
  fontSize: 12,
  color: 'var(--text-primary)',
  fontFamily: 'var(--font-mono)',
  outline: 'none',
  transition: 'border-color 0.15s',
  boxSizing: 'border-box',
};

const TOGGLE_CARD: React.CSSProperties = {
  background: 'var(--bg-inset)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  padding: '12px 14px',
};

const scheduleOptions = [
  { value: 'now' as const, label: 'Deploy Now', description: 'Start immediately' },
  { value: 'datetime' as const, label: 'Schedule', description: 'Pick date & time' },
  {
    value: 'maintenance_window' as const,
    label: 'Maint. Window',
    description: 'Next available window',
  },
];

const rebootModes = [
  { value: 'immediate' as const, label: 'Immediate' },
  { value: 'graceful' as const, label: 'Graceful' },
  { value: 'deferred' as const, label: 'Deferred' },
];

export function StrategyStep() {
  const resolvedMode = (document.documentElement.classList.contains('dark') ? 'dark' : 'light') as
    | 'dark'
    | 'light';
  const { watch, setValue } = useFormContext<DeploymentWizardValues>();
  const waves = watch('waves');
  const schedule = watch('schedule');
  const scheduledAt = watch('scheduledAt');
  const rollbackThreshold = watch('rollbackThreshold');
  const autoReboot = watch('autoReboot');
  const rebootMode = watch('rebootMode');
  const rebootGracePeriod = watch('rebootGracePeriod');
  const workflowTemplateId = watch('workflowTemplateId');

  const { data: templates } = useWorkflowTemplates();

  const addWave = () => {
    setValue('waves', [...waves, { maxTargets: 50, successThreshold: 95 }]);
  };

  const removeWave = (index: number) => {
    if (waves.length <= 1) return;
    setValue(
      'waves',
      waves.filter((_, i) => i !== index),
    );
  };

  const updateWave = (index: number, field: keyof WaveConfig, value: number) => {
    setValue(
      'waves',
      waves.map((w, i) => (i === index ? { ...w, [field]: value } : w)),
    );
  };

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 18 }}>
      {/* Waves section */}
      <div>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 10,
          }}
        >
          <label style={{ ...LABEL_STYLE, marginBottom: 0 }}>Deployment Waves</label>
          <button
            type="button"
            onClick={addWave}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
              padding: '4px 10px',
              borderRadius: 5,
              fontSize: 10,
              fontWeight: 600,
              cursor: 'pointer',
              border: '1px solid var(--border)',
              background: 'var(--bg-inset)',
              color: 'var(--text-secondary)',
              transition: 'border-color 0.15s',
            }}
          >
            <Plus style={{ width: 10, height: 10 }} />
            Add Wave
          </button>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {waves.map((wave, idx) => (
            <div
              key={idx}
              style={{
                display: 'flex',
                alignItems: 'flex-end',
                gap: 8,
                background: 'var(--bg-inset)',
                border: '1px solid var(--border)',
                borderRadius: 6,
                padding: '10px 12px',
              }}
            >
              {/* Wave badge */}
              <div
                style={{
                  width: 24,
                  height: 24,
                  borderRadius: '50%',
                  background: 'color-mix(in srgb, var(--accent) 10%, transparent)',
                  border: '1px solid color-mix(in srgb, var(--accent) 25%, transparent)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 9,
                  fontWeight: 700,
                  color: 'var(--accent)',
                  flexShrink: 0,
                  marginBottom: 2,
                }}
              >
                {idx + 1}
              </div>

              <div style={{ flex: 1 }}>
                <div
                  style={{
                    fontSize: 9,
                    color: 'var(--text-muted)',
                    marginBottom: 4,
                    fontFamily: 'var(--font-mono)',
                  }}
                >
                  Max Targets
                </div>
                <input
                  type="number"
                  min={1}
                  value={wave.maxTargets}
                  onChange={(e) => updateWave(idx, 'maxTargets', parseInt(e.target.value, 10) || 1)}
                  style={INPUT}
                  onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
                  onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
                />
              </div>

              <div style={{ flex: 1 }}>
                <div
                  style={{
                    fontSize: 9,
                    color: 'var(--text-muted)',
                    marginBottom: 4,
                    fontFamily: 'var(--font-mono)',
                  }}
                >
                  Success Threshold (%)
                </div>
                <input
                  type="number"
                  min={0}
                  max={100}
                  value={wave.successThreshold}
                  onChange={(e) =>
                    updateWave(idx, 'successThreshold', parseInt(e.target.value, 10) || 0)
                  }
                  style={INPUT}
                  onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
                  onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
                />
              </div>

              {waves.length > 1 && (
                <button
                  type="button"
                  onClick={() => removeWave(idx)}
                  style={{
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    color: 'var(--text-muted)',
                    padding: 4,
                    display: 'flex',
                    marginBottom: 2,
                  }}
                >
                  <X style={{ width: 13, height: 13 }} />
                </button>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Schedule */}
      <div>
        <label style={LABEL_STYLE}>Schedule</label>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(3, 1fr)',
            gap: 8,
            marginBottom: schedule === 'datetime' ? 8 : 0,
          }}
        >
          {scheduleOptions.map((opt) => {
            const isActive = schedule === opt.value;
            return (
              <button
                key={opt.value}
                type="button"
                onClick={() => setValue('schedule', opt.value)}
                style={{
                  padding: '10px 8px',
                  borderRadius: 7,
                  border: `1px solid ${isActive ? 'var(--accent)' : 'var(--border)'}`,
                  background: isActive
                    ? 'color-mix(in srgb, var(--accent) 8%, transparent)'
                    : 'var(--bg-inset)',
                  cursor: 'pointer',
                  transition: 'all 0.15s',
                  textAlign: 'left',
                }}
              >
                <span
                  style={{
                    fontSize: 11,
                    fontWeight: 600,
                    color: isActive ? 'var(--accent)' : 'var(--text-secondary)',
                    display: 'block',
                    marginBottom: 3,
                  }}
                >
                  {opt.label}
                </span>
                <span style={{ fontSize: 9, color: 'var(--text-muted)', display: 'block' }}>
                  {opt.description}
                </span>
              </button>
            );
          })}
        </div>

        {schedule === 'datetime' && (
          <input
            type="datetime-local"
            value={scheduledAt ?? ''}
            onChange={(e) => setValue('scheduledAt', e.target.value)}
            style={{ ...INPUT, colorScheme: resolvedMode }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          />
        )}
      </div>

      {/* Rollback threshold */}
      <div>
        <label style={{ ...LABEL_STYLE, marginBottom: 4 }}>
          Rollback Threshold — {rollbackThreshold}%
        </label>
        <p style={{ fontSize: 10, color: 'var(--text-muted)', marginBottom: 10, marginTop: 0 }}>
          Auto-rollback if failure rate exceeds this percentage
        </p>
        <input
          type="range"
          min={0}
          max={100}
          value={rollbackThreshold}
          onChange={(e) => setValue('rollbackThreshold', parseInt(e.target.value, 10))}
          style={{ width: '100%', accentColor: 'var(--accent)' }}
        />
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            fontSize: 9,
            color: 'var(--text-muted)',
            fontFamily: 'var(--font-mono)',
            marginTop: 4,
          }}
        >
          <span>0% (never)</span>
          <span>100% (aggressive)</span>
        </div>
      </div>

      {/* Auto reboot */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <div style={TOGGLE_CARD}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 12,
              marginBottom: autoReboot ? 12 : 0,
            }}
          >
            <div>
              <div
                style={{
                  fontSize: 12,
                  fontWeight: 500,
                  color: 'var(--text-primary)',
                  marginBottom: 2,
                }}
              >
                Auto Reboot
              </div>
              <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>
                Reboot endpoints after patch installation if required
              </div>
            </div>
            <Switch
              checked={autoReboot}
              onCheckedChange={(checked) => setValue('autoReboot', checked)}
            />
          </div>

          {autoReboot && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              <div>
                <div
                  style={{
                    fontSize: 9,
                    color: 'var(--text-muted)',
                    marginBottom: 6,
                    fontFamily: 'var(--font-mono)',
                  }}
                >
                  Reboot Mode
                </div>
                <div style={{ display: 'flex', gap: 6 }}>
                  {rebootModes.map((mode) => {
                    const isActive = rebootMode === mode.value;
                    return (
                      <button
                        key={mode.value}
                        type="button"
                        onClick={() => setValue('rebootMode', mode.value)}
                        style={{
                          padding: '5px 12px',
                          borderRadius: 5,
                          border: `1px solid ${isActive ? 'var(--accent)' : 'var(--border)'}`,
                          background: isActive
                            ? 'color-mix(in srgb, var(--accent) 8%, transparent)'
                            : 'transparent',
                          cursor: 'pointer',
                          fontSize: 11,
                          fontWeight: 500,
                          color: isActive ? 'var(--accent)' : 'var(--text-muted)',
                          transition: 'all 0.15s',
                        }}
                      >
                        {mode.label}
                      </button>
                    );
                  })}
                </div>
              </div>

              <div>
                <div
                  style={{
                    fontSize: 9,
                    color: 'var(--text-muted)',
                    marginBottom: 4,
                    fontFamily: 'var(--font-mono)',
                  }}
                >
                  Grace Period (seconds)
                </div>
                <input
                  type="number"
                  min={0}
                  value={rebootGracePeriod}
                  onChange={(e) => setValue('rebootGracePeriod', parseInt(e.target.value, 10) || 0)}
                  style={{ ...INPUT, width: 120 }}
                  onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
                  onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
                />
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Workflow template */}
      <div>
        <label style={LABEL_STYLE}>Workflow Template (optional)</label>
        <p style={{ fontSize: 10, color: 'var(--text-muted)', marginBottom: 8, marginTop: 0 }}>
          Attach a workflow to run pre/post deployment steps
        </p>
        <select
          value={workflowTemplateId ?? ''}
          onChange={(e) => setValue('workflowTemplateId', e.target.value || undefined)}
          style={{
            ...INPUT,
            cursor: 'pointer',
          }}
          onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
          onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
        >
          <option value="">None</option>
          {(templates as { data?: { id: string; name: string }[] })?.data?.map(
            (t: { id: string; name: string }) => (
              <option key={t.id} value={t.id}>
                {t.name}
              </option>
            ),
          )}
        </select>
      </div>
    </div>
  );
}
