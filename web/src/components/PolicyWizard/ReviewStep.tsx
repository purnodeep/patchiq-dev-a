import { useState } from 'react';
import { useFormContext } from 'react-hook-form';
import { AlertTriangle, Check, Zap, Hand, Eye } from 'lucide-react';
import { useValidateSelector } from '../../api/hooks/useTagSelector';
import type { Selector } from '../../types/targeting';
import type { PolicyWizardValues } from './types';
import { LABEL_STYLE, INPUT, SUMMARY_CARD } from './types';

interface ReviewStepProps {
  onSubmit: () => void;
  isSubmitting: boolean;
  error?: string | null;
  onBack: () => void;
}

function modeIcon(m: string) {
  if (m === 'automatic') return Zap;
  if (m === 'advisory') return Eye;
  return Hand;
}

function patchScopeDetail(values: PolicyWizardValues): string {
  switch (values.selection_mode) {
    case 'all_available':
      return 'All available patches';
    case 'by_severity':
      return `Minimum severity: ${values.min_severity ?? 'unset'}`;
    case 'by_cve_list':
      return `${values.cve_ids?.length ?? 0} CVE ID${(values.cve_ids?.length ?? 0) !== 1 ? 's' : ''} targeted`;
    case 'by_regex':
      return values.package_regex ? `Regex: ${values.package_regex}` : 'By package regex';
  }
}

function scheduleDetail(values: PolicyWizardValues): string {
  if (values.schedule_type === 'manual') return 'Manual — trigger on demand';
  if (values.schedule_type === 'maintenance_window') return 'Next maintenance window';
  return values.schedule_cron ? `Recurring: ${values.schedule_cron}` : 'Recurring (cron not set)';
}

export function ReviewStep({ onSubmit, isSubmitting, error, onBack }: ReviewStepProps) {
  const { watch, setValue } = useFormContext<PolicyWizardValues>();
  const [confirmed, setConfirmed] = useState(false);
  const [expandedCard, setExpandedCard] = useState<string | null>(null);
  const values = watch();

  const selector = (values.target_selector ?? null) as Selector | null;
  const selectorValidation = useValidateSelector(selector);
  const totalEndpoints = selectorValidation.data?.matched_count ?? 0;

  const ModeIcon = modeIcon(values.mode);

  const schedDetail = scheduleDetail(values);
  const tzSuffix = values.timezone && values.timezone !== 'UTC' ? ` (${values.timezone})` : '';

  const summaryItems = [
    {
      icon: (
        <div
          style={{
            width: 16,
            height: 16,
            borderRadius: '50%',
            background: 'color-mix(in srgb, var(--accent) 15%, transparent)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 8,
            fontWeight: 700,
            color: 'var(--accent)',
            flexShrink: 0,
          }}
        >
          {values.policy_type === 'patch' ? 'P' : values.policy_type === 'deploy' ? 'D' : 'C'}
        </div>
      ),
      label:
        values.policy_type === 'patch'
          ? 'Patch Policy'
          : values.policy_type === 'deploy'
            ? 'Deploy Policy'
            : 'Compliance Policy',
      detail:
        values.policy_type === 'patch'
          ? 'Select patches, evaluate, and optionally deploy'
          : values.policy_type === 'deploy'
            ? 'Direct deployment to endpoints'
            : 'Compliance evaluation only, no deployments',
      expandContent: (
        <div>
          {values.policy_type === 'patch' &&
            'Patches are scanned, evaluated for applicability, then optionally deployed based on mode.'}
          {values.policy_type === 'deploy' &&
            'Patches are deployed directly to targeted endpoints without a separate evaluation phase.'}
          {values.policy_type === 'compliance' &&
            'Endpoints are evaluated against compliance rules. No patches are applied.'}
        </div>
      ),
    },
    {
      icon: <ModeIcon style={{ width: 16, height: 16, color: 'var(--accent)', flexShrink: 0 }} />,
      label: `${values.mode.charAt(0).toUpperCase() + values.mode.slice(1)} Mode`,
      detail:
        values.mode === 'automatic'
          ? 'Patches deploy on schedule automatically'
          : values.mode === 'advisory'
            ? 'Report only — no patches applied'
            : 'Patches deploy on demand',
      expandContent: (
        <div>
          {values.mode === 'automatic' &&
            'At eval time: patches are assessed. At deploy time: patches are applied automatically on the configured schedule.'}
          {values.mode === 'advisory' &&
            'At eval time: patches are assessed and reported. No deployment occurs — advisory only.'}
          {values.mode === 'manual' &&
            'At eval time: patches are assessed. At deploy time: an operator must manually trigger deployment.'}
        </div>
      ),
    },
    {
      icon: (
        <div
          style={{
            width: 16,
            height: 16,
            borderRadius: '50%',
            background: 'color-mix(in srgb, var(--accent) 15%, transparent)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 8,
            fontWeight: 700,
            color: 'var(--accent)',
            flexShrink: 0,
          }}
        >
          T
        </div>
      ),
      label: 'Targets',
      detail: `Tag selector · ~${totalEndpoints} endpoint${totalEndpoints !== 1 ? 's' : ''}`,
      expandContent: selector ? (
        <pre style={{ fontFamily: 'var(--font-mono)', fontSize: 10, margin: 0 }}>
          {JSON.stringify(selector, null, 2)}
        </pre>
      ) : (
        <div>No tag predicates defined — zero endpoints will match.</div>
      ),
    },
    {
      icon: (
        <div
          style={{
            width: 16,
            height: 16,
            borderRadius: '50%',
            background: 'color-mix(in srgb, var(--accent) 15%, transparent)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 8,
            fontWeight: 700,
            color: 'var(--accent)',
            flexShrink: 0,
          }}
        >
          P
        </div>
      ),
      label: 'Patches',
      detail: patchScopeDetail(values),
      expandContent: (
        <div>
          {values.selection_mode === 'all_available' && 'All pending patches will be included'}
          {values.selection_mode === 'by_severity' &&
            `Patches with severity ≥ ${values.min_severity}`}
          {values.selection_mode === 'by_cve_list' &&
            values.cve_ids &&
            values.cve_ids.length > 0 && (
              <div>
                {values.cve_ids.map((id) => (
                  <div key={id} style={{ fontFamily: 'var(--font-mono)' }}>
                    {id}
                  </div>
                ))}
              </div>
            )}
          {values.selection_mode === 'by_regex' && (
            <div>
              <div>
                Pattern:{' '}
                <code style={{ fontFamily: 'var(--font-mono)' }}>{values.package_regex}</code>
              </div>
              {values.exclude_packages && values.exclude_packages.length > 0 && (
                <div>Excluded: {values.exclude_packages.join(', ')}</div>
              )}
            </div>
          )}
        </div>
      ),
    },
    {
      icon: (
        <div
          style={{
            width: 16,
            height: 16,
            borderRadius: '50%',
            background: 'color-mix(in srgb, var(--accent) 15%, transparent)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 8,
            fontWeight: 700,
            color: 'var(--accent)',
            flexShrink: 0,
          }}
        >
          S
        </div>
      ),
      label: 'Schedule',
      detail: schedDetail + tzSuffix,
      expandContent: (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
          {values.schedule_cron && (
            <div>
              Cron: <code style={{ fontFamily: 'var(--font-mono)' }}>{values.schedule_cron}</code>
            </div>
          )}
          {values.timezone && <div>Timezone: {values.timezone}</div>}
          {values.mw_enabled && values.mw_start && values.mw_end && (
            <div>
              Maintenance window: {values.mw_start} – {values.mw_end}
            </div>
          )}
        </div>
      ),
    },
  ];

  const handleCreate = () => {
    if (!confirmed) {
      setConfirmed(true);
      return;
    }
    onSubmit();
  };

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Name + description — same layout as DeploymentWizard ReviewStep */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        <div>
          <label style={LABEL_STYLE}>
            Policy Name <span style={{ color: 'var(--signal-critical)' }}>*</span>
          </label>
          <input
            type="text"
            value={values.name}
            onChange={(e) => setValue('name', e.target.value)}
            placeholder="e.g. Linux Critical Patching"
            style={INPUT}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          />
        </div>
        <div>
          <label style={LABEL_STYLE}>Description (optional)</label>
          <textarea
            value={values.description}
            onChange={(e) => setValue('description', e.target.value)}
            placeholder="Notes about this policy..."
            style={{
              ...INPUT,
              minHeight: 60,
              resize: 'vertical',
              lineHeight: 1.5,
              fontFamily: 'var(--font-sans)',
            }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          />
        </div>
      </div>

      {/* Summary — matches DeploymentWizard ReviewStep summary cards exactly */}
      <div>
        <label style={LABEL_STYLE}>Summary</label>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 7 }}>
          {summaryItems.map((item) => (
            <div
              key={item.label}
              style={{ cursor: 'pointer' }}
              onClick={() => setExpandedCard(expandedCard === item.label ? null : item.label)}
            >
              <div style={SUMMARY_CARD}>
                {item.icon}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div
                    style={{
                      fontSize: 12,
                      fontWeight: 600,
                      color: 'var(--text-primary)',
                      marginBottom: 2,
                    }}
                  >
                    {item.label}
                  </div>
                  <div
                    style={{
                      fontSize: 10,
                      color: 'var(--text-muted)',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {item.detail}
                  </div>
                </div>
                <Check
                  style={{ width: 13, height: 13, color: 'var(--signal-healthy)', flexShrink: 0 }}
                />
                <span style={{ fontSize: 10, color: 'var(--text-muted)', marginLeft: 4 }}>
                  {expandedCard === item.label ? '▾' : '▸'}
                </span>
              </div>
              {expandedCard === item.label && item.expandContent && (
                <div
                  style={{
                    padding: '8px 14px 8px 42px',
                    fontSize: 11,
                    color: 'var(--text-secondary)',
                    background: 'color-mix(in srgb, var(--bg-inset) 50%, transparent)',
                    borderRadius: '0 0 7px 7px',
                    marginTop: -4,
                    border: '1px solid var(--border)',
                    borderTop: 'none',
                  }}
                >
                  {item.expandContent}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Error */}
      {error && (
        <div
          style={{
            padding: '10px 14px',
            borderRadius: 8,
            fontSize: 12,
            color: 'var(--signal-critical)',
            background: 'color-mix(in srgb, var(--signal-critical) 8%, transparent)',
            border: '1px solid color-mix(in srgb, var(--signal-critical) 30%, transparent)',
            fontWeight: 500,
          }}
        >
          <div style={{ fontWeight: 600, marginBottom: 2 }}>Failed to create policy</div>
          <div>{error}</div>
        </div>
      )}

      {/* Confirmation warning — exact copy of DeploymentWizard ReviewStep */}
      {confirmed && (
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: 10,
            background: 'color-mix(in srgb, var(--signal-warning) 6%, transparent)',
            border: '1px solid color-mix(in srgb, var(--signal-warning) 30%, transparent)',
            borderRadius: 8,
            padding: '10px 14px',
          }}
        >
          <AlertTriangle
            style={{
              width: 15,
              height: 15,
              color: 'var(--signal-warning)',
              flexShrink: 0,
              marginTop: 1,
            }}
          />
          <div>
            <p
              style={{
                fontSize: 12,
                fontWeight: 600,
                color: 'var(--signal-warning)',
                margin: '0 0 3px',
              }}
            >
              This policy will be saved and activated.
            </p>
            <p
              style={{
                fontSize: 10,
                color: 'color-mix(in srgb, var(--signal-warning) 70%, transparent)',
                margin: 0,
              }}
            >
              Click &quot;Confirm Create&quot; to proceed.
            </p>
          </div>
        </div>
      )}

      {/* Back + Create buttons */}
      <div style={{ display: 'flex', gap: 10 }}>
        <button
          type="button"
          onClick={onBack}
          style={{
            padding: '10px 16px',
            borderRadius: 8,
            fontSize: 13,
            fontWeight: 600,
            cursor: 'pointer',
            border: '1px solid var(--border)',
            background: 'transparent',
            color: 'var(--text-primary)',
            transition: 'opacity 0.15s',
          }}
        >
          ← Back
        </button>
        <button
          type="button"
          onClick={handleCreate}
          disabled={isSubmitting}
          style={{
            flex: 1,
            padding: '10px 16px',
            borderRadius: 8,
            fontSize: 13,
            fontWeight: 600,
            cursor: isSubmitting ? 'not-allowed' : 'pointer',
            border: 'none',
            background: confirmed ? 'var(--signal-healthy)' : 'var(--accent)',
            color: 'var(--btn-accent-text, #000)',
            transition: 'opacity 0.15s, filter 0.15s',
            opacity: isSubmitting ? 0.6 : 1,
            letterSpacing: '0.01em',
          }}
          onMouseEnter={(e) => {
            if (!isSubmitting) e.currentTarget.style.filter = 'brightness(1.1)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.filter = 'none';
          }}
        >
          {isSubmitting ? 'Creating Policy...' : confirmed ? 'Confirm Create' : 'Create Policy'}
        </button>
      </div>
    </div>
  );
}
