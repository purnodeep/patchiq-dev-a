import { useState } from 'react';
import { useFormContext } from 'react-hook-form';
import { AlertTriangle, Check, Package, ScrollText, Terminal } from 'lucide-react';
import type { DeploymentWizardValues } from '../../types/deployment-wizard';

interface ReviewStepProps {
  onDeploy: () => void;
  isDeploying: boolean;
}

const INPUT: React.CSSProperties = {
  width: '100%',
  background: 'var(--bg-inset)',
  border: '1px solid var(--border)',
  borderRadius: 6,
  padding: '7px 10px',
  fontSize: 12,
  color: 'var(--text-primary)',
  fontFamily: 'var(--font-sans)',
  outline: 'none',
  transition: 'border-color 0.15s',
  boxSizing: 'border-box',
};

const LABEL_STYLE: React.CSSProperties = {
  fontSize: 10,
  fontWeight: 600,
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  fontFamily: 'var(--font-mono)',
  marginBottom: 6,
  display: 'block',
};

const SUMMARY_CARD: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 12,
  background: 'var(--bg-inset)',
  border: '1px solid var(--border)',
  borderRadius: 7,
  padding: '10px 14px',
};

export function ReviewStep({ onDeploy, isDeploying }: ReviewStepProps) {
  const { watch, setValue } = useFormContext<DeploymentWizardValues>();
  const [confirmed, setConfirmed] = useState(false);
  const values = watch();

  const sourceIcon = {
    catalog: Package,
    policy: ScrollText,
    adhoc: Terminal,
  }[values.sourceType];
  const SourceIcon = sourceIcon;

  const sourceLabel = {
    catalog: 'Patch Catalog',
    policy: 'Policy',
    adhoc: 'Ad-hoc Packages',
  }[values.sourceType];

  const sourceDetail = (): string => {
    switch (values.sourceType) {
      case 'catalog': {
        const count = values.patchIds?.length ?? 0;
        return `${count} ${count === 1 ? 'patch' : 'patches'} selected`;
      }
      case 'policy':
        return values.policyId ? `Policy ${values.policyId.slice(0, 8)}...` : 'No policy selected';
      case 'adhoc': {
        const count = values.adhocPackages?.length ?? 0;
        return `${count} ${count === 1 ? 'package' : 'packages'}`;
      }
    }
  };

  const targetConditions =
    values.targetExpression?.conditions?.length ?? (values.targetExpression?.tag ? 1 : 0);
  const targetMode = values.targetMode ?? 'all';

  const handleDeploy = () => {
    if (!confirmed) {
      setConfirmed(true);
      return;
    }
    onDeploy();
  };

  const summaryItems = [
    {
      icon: <SourceIcon style={{ width: 16, height: 16, color: 'var(--accent)', flexShrink: 0 }} />,
      label: sourceLabel,
      detail: sourceDetail(),
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
      detail:
        targetMode === 'select'
          ? `${values.endpointIds?.length ?? 0} endpoint${(values.endpointIds?.length ?? 0) !== 1 ? 's' : ''} selected`
          : targetMode === 'tags' && targetConditions > 0
            ? `${targetConditions} tag ${targetConditions === 1 ? 'condition' : 'conditions'}`
            : 'All endpoints',
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
      label: 'Strategy',
      detail: `${values.waves.length} wave${values.waves.length !== 1 ? 's' : ''} · ${
        values.schedule === 'now'
          ? 'Immediate'
          : values.schedule === 'datetime'
            ? `Scheduled: ${values.scheduledAt ?? 'TBD'}`
            : 'Maintenance window'
      } · Rollback at ${values.rollbackThreshold}%`,
    },
  ];

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Name + description */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        <div>
          <label style={LABEL_STYLE}>Deployment Name (optional)</label>
          <input
            type="text"
            value={values.name ?? ''}
            onChange={(e) => setValue('name', e.target.value)}
            placeholder="Auto-generated if empty"
            style={INPUT}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          />
        </div>
        <div>
          <label style={LABEL_STYLE}>Description (optional)</label>
          <textarea
            value={values.description ?? ''}
            onChange={(e) => setValue('description', e.target.value)}
            placeholder="Notes about this deployment..."
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

      {/* Summary */}
      <div>
        <label style={LABEL_STYLE}>Summary</label>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 7 }}>
          {summaryItems.map((item) => (
            <div key={item.label} style={SUMMARY_CARD}>
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
            </div>
          ))}
        </div>
      </div>

      {/* Confirmation warning */}
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
              {values.schedule === 'now'
                ? 'This will begin patch installation immediately.'
                : 'This deployment will be scheduled.'}
            </p>
            <p
              style={{
                fontSize: 10,
                color: 'color-mix(in srgb, var(--signal-warning) 70%, transparent)',
                margin: 0,
              }}
            >
              Click &quot;Confirm Deploy&quot; to proceed.
            </p>
          </div>
        </div>
      )}

      {/* Deploy button */}
      <button
        type="button"
        onClick={handleDeploy}
        disabled={isDeploying}
        style={{
          width: '100%',
          padding: '10px 16px',
          borderRadius: 8,
          fontSize: 13,
          fontWeight: 600,
          cursor: isDeploying ? 'not-allowed' : 'pointer',
          border: 'none',
          background: confirmed ? 'var(--signal-critical)' : 'var(--accent)',
          color: 'var(--btn-accent-text, #000)',
          transition: 'opacity 0.15s, filter 0.15s',
          opacity: isDeploying ? 0.6 : 1,
          letterSpacing: '0.01em',
        }}
        onMouseEnter={(e) => {
          if (!isDeploying) e.currentTarget.style.filter = 'brightness(1.1)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.filter = 'none';
        }}
      >
        {isDeploying
          ? 'Creating deployment...'
          : confirmed
            ? 'Confirm Deploy'
            : values.schedule === 'now'
              ? 'Deploy Now'
              : 'Schedule Deployment'}
      </button>
    </div>
  );
}
