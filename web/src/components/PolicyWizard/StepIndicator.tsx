import { POLICY_WIZARD_STEPS, type PolicyWizardStepId } from './types';

interface StepIndicatorProps {
  currentStep: PolicyWizardStepId;
  completedSteps: Set<PolicyWizardStepId>;
  onStepClick: (step: PolicyWizardStepId) => void;
}

export function StepIndicator({ currentStep, completedSteps, onStepClick }: StepIndicatorProps) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        padding: '10px 16px',
        borderBottom: '1px solid var(--border)',
        background: 'var(--bg-inset)',
        gap: 0,
      }}
    >
      {POLICY_WIZARD_STEPS.map((step, idx) => {
        const isCurrent = currentStep === step.id;
        const isCompleted = completedSteps.has(step.id);
        const isClickable = isCompleted || isCurrent;

        return (
          <div key={step.id} style={{ display: 'flex', alignItems: 'center' }}>
            {idx > 0 && (
              <div
                style={{
                  width: 24,
                  height: 1,
                  background: isCompleted || isCurrent ? 'var(--accent)' : 'var(--border)',
                  flexShrink: 0,
                  transition: 'background 0.2s',
                }}
              />
            )}
            <button
              type="button"
              disabled={!isClickable}
              onClick={() => isClickable && onStepClick(step.id)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 7,
                padding: '5px 10px',
                borderRadius: 20,
                fontSize: 11,
                fontWeight: 500,
                cursor: isClickable ? 'pointer' : 'default',
                border: 'none',
                transition: 'background 0.15s, color 0.15s',
                background: isCurrent
                  ? 'color-mix(in srgb, var(--accent) 12%, transparent)'
                  : isCompleted
                    ? 'color-mix(in srgb, var(--accent) 6%, transparent)'
                    : 'transparent',
                color: isCurrent
                  ? 'var(--accent)'
                  : isCompleted
                    ? 'var(--accent)'
                    : 'var(--text-faint)',
                outline: 'none',
              }}
            >
              <div
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: '50%',
                  flexShrink: 0,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: isCurrent
                    ? 'var(--accent)'
                    : isCompleted
                      ? 'color-mix(in srgb, var(--accent) 15%, transparent)'
                      : 'var(--bg-inset)',
                  border: `1.5px solid ${
                    isCurrent
                      ? 'var(--accent)'
                      : isCompleted
                        ? 'color-mix(in srgb, var(--accent) 40%, transparent)'
                        : 'var(--border)'
                  }`,
                  fontFamily: 'var(--font-mono)',
                  fontSize: 9,
                  fontWeight: 700,
                  color: isCurrent
                    ? 'var(--btn-accent-text, #000)'
                    : isCompleted
                      ? 'var(--accent)'
                      : 'var(--text-faint)',
                  transition: 'all 0.2s',
                }}
              >
                {isCompleted && !isCurrent ? '✓' : step.number}
              </div>
              <span style={{ letterSpacing: '0.01em' }}>{step.label}</span>
            </button>
          </div>
        );
      })}
    </div>
  );
}
