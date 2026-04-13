import { useState, useCallback, useMemo, useEffect } from 'react';
import { useNavigate } from 'react-router';
import { useForm, FormProvider } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
  Button,
} from '@patchiq/ui';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { StepIndicator } from './StepIndicator';
import { SourceStep } from './SourceStep';
import { TargetsStep } from './TargetsStep';
import { StrategyStep } from './StrategyStep';
import { ReviewStep } from './ReviewStep';
import { ImpactPreview } from './ImpactPreview';
import { useCreateDeployment } from '../../api/hooks/useDeployments';
import { api } from '../../api/client';
import type { components } from '../../api/types';
import {
  DEFAULT_WIZARD_VALUES,
  deploymentWizardSchema,
  type DeploymentWizardValues,
  type DeploymentWizardInitialState,
  type WizardStepId,
  type TagExpression,
} from '../../types/deployment-wizard';

function getFirstTag(expr: TagExpression | undefined): string | undefined {
  if (!expr) return undefined;
  if (expr.tag) return expr.tag;
  if (expr.conditions?.length) return getFirstTag(expr.conditions[0]);
  return undefined;
}

interface DeploymentWizardProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialState?: DeploymentWizardInitialState;
}

const STEP_ORDER: WizardStepId[] = ['source', 'targets', 'strategy', 'review'];

export function DeploymentWizard({ open, onOpenChange, initialState }: DeploymentWizardProps) {
  const navigate = useNavigate();
  const createDeployment = useCreateDeployment();

  const defaultValues = useMemo(() => {
    const values = { ...DEFAULT_WIZARD_VALUES };
    if (initialState?.sourceType) values.sourceType = initialState.sourceType;
    if (initialState?.patchIds) values.patchIds = initialState.patchIds;
    if (initialState?.policyId) values.policyId = initialState.policyId;
    return values;
  }, [initialState]);

  const form = useForm<DeploymentWizardValues>({
    resolver: zodResolver(deploymentWizardSchema) as any, // TODO(#332): Zod4 + react-hook-form resolver type mismatch
    defaultValues,
    mode: 'onChange',
  });

  // Reset form when initialState changes (e.g., bulk selection → deploy selected)
  useEffect(() => {
    form.reset(defaultValues);
  }, [defaultValues]);

  const startStep = initialState?.startStep ?? 'source';
  const [currentStep, setCurrentStep] = useState<WizardStepId>(startStep);
  const [completedSteps, setCompletedSteps] = useState<Set<WizardStepId>>(() => {
    const initial = new Set<WizardStepId>();
    // If starting at a later step, mark previous steps as completed
    const startIdx = STEP_ORDER.indexOf(startStep);
    for (let i = 0; i < startIdx; i++) {
      initial.add(STEP_ORDER[i]);
    }
    return initial;
  });
  const [deployError, setDeployError] = useState<string | null>(null);

  const currentIdx = STEP_ORDER.indexOf(currentStep);
  const isFirstStep = currentIdx === 0;
  const isLastStep = currentIdx === STEP_ORDER.length - 1;

  const goNext = useCallback(() => {
    if (isLastStep) return;
    setCompletedSteps((prev) => new Set([...prev, currentStep]));
    setCurrentStep(STEP_ORDER[currentIdx + 1]);
  }, [currentIdx, currentStep, isLastStep]);

  const goBack = useCallback(() => {
    if (isFirstStep) return;
    setCurrentStep(STEP_ORDER[currentIdx - 1]);
  }, [currentIdx, isFirstStep]);

  const goToStep = useCallback((step: WizardStepId) => {
    setCurrentStep(step);
  }, []);

  const handleDeploy = useCallback(async () => {
    const values = form.getValues();
    setDeployError(null);

    // Build the CreateDeploymentRequest from wizard values
    const body: components['schemas']['CreateDeploymentRequest'] = {};

    body.source_type = values.sourceType;

    if (values.sourceType === 'policy' && values.policyId) {
      body.policy_id = values.policyId;
    }
    if (values.patchIds?.length) {
      body.patch_ids = values.patchIds;
    }
    if (values.name?.trim()) {
      body.name = values.name.trim();
    }
    if (values.description?.trim()) {
      body.description = values.description.trim();
    }
    // Resolve endpoint_ids for non-policy source types.
    // Backend requires explicit endpoint_ids for adhoc/catalog deployments.
    if (values.sourceType !== 'policy') {
      const mode = values.targetMode ?? 'all';
      let resolvedIds: string[];
      if (mode === 'select') {
        resolvedIds = values.endpointIds ?? [];
      } else {
        // "all" or "tags" mode: fetch endpoint IDs from API
        try {
          const query: Record<string, unknown> = { limit: 10000 };
          if (mode === 'tags' && values.targetExpression) {
            const firstTag = getFirstTag(values.targetExpression);
            if (firstTag) query.tag_id = firstTag;
          }
          const { data: epData, error: epError } = await api.GET('/api/v1/endpoints', {
            params: { query },
          });
          if (epError) throw epError;
          resolvedIds = (epData?.data ?? []).map((ep) => ep.id);
        } catch {
          setDeployError('Failed to resolve target endpoints');
          return;
        }
      }
      if (resolvedIds.length === 0) {
        setDeployError('No endpoints matched the target criteria');
        return;
      }
      body.endpoint_ids = resolvedIds;
    }
    if (values.targetMode === 'tags' && values.targetExpression) {
      body.target_expression = values.targetExpression;
    }
    if (values.schedule === 'datetime' && values.scheduledAt) {
      body.scheduled_at = new Date(values.scheduledAt).toISOString();
    }
    if (values.waves.length > 0) {
      const waveConfigs = values.waves.map((w) => ({
        percentage: w.maxTargets,
        success_threshold: w.successThreshold / 100,
        error_rate_max: (100 - w.successThreshold) / 100,
        delay_minutes: 0,
      }));
      const totalPct = waveConfigs.reduce((sum, w) => sum + w.percentage, 0);
      if (totalPct !== 100) {
        setDeployError(
          `Wave percentages must sum to 100 (currently ${totalPct}). Adjust wave sizes in the Strategy step.`,
        );
        return;
      }
      body.wave_config = waveConfigs;
    }
    if (values.autoReboot) {
      body.reboot_config = {
        mode: values.rebootMode,
        grace_period: values.rebootGracePeriod,
      };
    }
    if (values.workflowTemplateId) {
      body.workflow_template_id = values.workflowTemplateId;
    }

    try {
      const result = await createDeployment.mutateAsync(body);
      onOpenChange(false);
      form.reset(DEFAULT_WIZARD_VALUES);
      setCurrentStep('source');
      setCompletedSteps(new Set());
      if (result?.id) {
        navigate(`/deployments/${result.id}`);
      }
    } catch (err) {
      setDeployError(err instanceof Error ? err.message : 'Failed to create deployment');
    }
  }, [form, createDeployment, navigate, onOpenChange]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    // Reset after animation
    setTimeout(() => {
      form.reset(DEFAULT_WIZARD_VALUES);
      setCurrentStep('source');
      setCompletedSteps(new Set());
      setDeployError(null);
    }, 300);
  }, [form, onOpenChange]);

  return (
    <FormProvider {...form}>
      <Sheet open={open} onOpenChange={handleClose}>
        <SheetContent
          side="right"
          style={{
            width: 700,
            maxWidth: 700,
            padding: 0,
            display: 'flex',
            flexDirection: 'column',
          }}
          showCloseButton
        >
          <SheetHeader style={{ padding: '16px 16px 0' }}>
            <SheetTitle style={{ fontFamily: 'var(--font-display)', fontSize: 15 }}>
              New Deployment
            </SheetTitle>
            <SheetDescription style={{ fontSize: 11 }}>
              Configure and deploy patches to your endpoints
            </SheetDescription>
          </SheetHeader>

          <StepIndicator
            currentStep={currentStep}
            completedSteps={completedSteps}
            onStepClick={goToStep}
          />

          {/* Step content + impact preview side-by-side */}
          <div style={{ flex: 1, overflowY: 'auto', display: 'flex' }}>
            <div style={{ flex: 1, minWidth: 0 }}>
              {currentStep === 'source' && <SourceStep />}
              {currentStep === 'targets' && <TargetsStep />}
              {currentStep === 'strategy' && <StrategyStep />}
              {currentStep === 'review' && (
                <ReviewStep onDeploy={handleDeploy} isDeploying={createDeployment.isPending} />
              )}
            </div>
            <div style={{ padding: '16px 12px' }}>
              <ImpactPreview currentStep={currentStep} />
            </div>
          </div>

          {/* Error */}
          {deployError && (
            <div style={{ padding: '0 16px' }}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: 10,
                  background: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                  border: '1px solid color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                  borderRadius: 8,
                  padding: '10px 14px',
                }}
              >
                <span style={{ fontSize: 14, lineHeight: 1, flexShrink: 0, marginTop: 1 }}>!</span>
                <div style={{ minWidth: 0 }}>
                  <p
                    style={{
                      fontSize: 12,
                      fontWeight: 600,
                      color: 'var(--signal-critical)',
                      margin: '0 0 2px',
                    }}
                  >
                    Deployment failed
                  </p>
                  <p
                    style={{
                      fontSize: 11,
                      color: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                      margin: 0,
                      wordBreak: 'break-word',
                    }}
                  >
                    {deployError}
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() => setDeployError(null)}
                  style={{
                    background: 'none',
                    border: 'none',
                    color: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                    cursor: 'pointer',
                    padding: 2,
                    fontSize: 14,
                    lineHeight: 1,
                    marginLeft: 'auto',
                    flexShrink: 0,
                  }}
                >
                  x
                </button>
              </div>
            </div>
          )}

          {/* Footer navigation (hidden on review step — it has its own deploy button) */}
          {currentStep !== 'review' && (
            <SheetFooter
              style={{
                display: 'flex',
                flexDirection: 'row',
                justifyContent: 'space-between',
                borderTop: '1px solid var(--border)',
                padding: '12px 16px',
              }}
            >
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={goBack}
                disabled={isFirstStep}
              >
                <ChevronLeft style={{ width: 14, height: 14, marginRight: 4 }} />
                Back
              </Button>
              <Button type="button" size="sm" onClick={goNext}>
                Next
                <ChevronRight style={{ width: 14, height: 14, marginLeft: 4 }} />
              </Button>
            </SheetFooter>
          )}
        </SheetContent>
      </Sheet>
    </FormProvider>
  );
}

// Re-export for convenience
export type { DeploymentWizardInitialState } from '../../types/deployment-wizard';
