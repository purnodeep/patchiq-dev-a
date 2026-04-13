import { useState, useCallback } from 'react';
import { useNavigate } from 'react-router';
import { useForm, FormProvider } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import type { components } from '../../api/types';
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
import { BasicsStep } from './BasicsStep';
import { TargetsStep } from './TargetsStep';
import { PatchesStep } from './PatchesStep';
import { ReviewStep } from './ReviewStep';
import { ImpactPreview } from './ImpactPreview';
import { useCreatePolicy } from '../../api/hooks/usePolicies';
import {
  policyWizardSchema,
  DEFAULT_POLICY_VALUES,
  type PolicyWizardValues,
  type PolicyWizardStepId,
} from './types';

const STEP_ORDER: PolicyWizardStepId[] = ['basics', 'targets', 'patches', 'review'];

const STEP_FIELDS: Record<PolicyWizardStepId, (keyof PolicyWizardValues)[]> = {
  basics: ['name', 'policy_type', 'mode'],
  targets: ['target_selector'],
  patches: ['selection_mode', 'schedule_type'],
  review: [],
};

interface PolicyWizardProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function PolicyWizard({ open, onOpenChange }: PolicyWizardProps) {
  const navigate = useNavigate();
  const createPolicy = useCreatePolicy();
  const [currentStep, setCurrentStep] = useState<PolicyWizardStepId>('basics');
  const [completedSteps, setCompletedSteps] = useState<Set<PolicyWizardStepId>>(new Set());
  const [submitError, setSubmitError] = useState<string | null>(null);

  const form = useForm<PolicyWizardValues>({
    resolver: zodResolver(policyWizardSchema),
    defaultValues: DEFAULT_POLICY_VALUES,
    mode: 'onChange',
  });

  const currentIdx = STEP_ORDER.indexOf(currentStep);
  const isFirstStep = currentIdx === 0;
  const isLastStep = currentIdx === STEP_ORDER.length - 1;

  const goNext = useCallback(async () => {
    if (isLastStep) return;
    const fieldsToValidate = STEP_FIELDS[currentStep];
    if (fieldsToValidate.length > 0) {
      const isValid = await form.trigger(fieldsToValidate);
      if (!isValid) return;
    }
    setCompletedSteps((prev) => new Set([...prev, currentStep]));
    setCurrentStep(STEP_ORDER[currentIdx + 1]);
  }, [currentIdx, currentStep, isLastStep, form]);

  const goBack = useCallback(() => {
    if (isFirstStep) return;
    setCurrentStep(STEP_ORDER[currentIdx - 1]);
  }, [currentIdx, isFirstStep]);

  const goToStep = useCallback((step: PolicyWizardStepId) => {
    setCurrentStep(step);
  }, []);

  const handleSubmit = useCallback(async () => {
    const values = form.getValues();
    setSubmitError(null);
    // Map wizard values → API body (drop UI-only fields)
    const body = {
      name: values.name,
      description: values.description,
      policy_type: values.policy_type,
      mode: values.mode,
      selection_mode: values.selection_mode,
      target_selector: values.target_selector,
      min_severity: values.min_severity,
      cve_ids: values.cve_ids,
      package_regex: values.package_regex,
      exclude_packages: values.exclude_packages,
      schedule_type:
        values.schedule_type === 'maintenance_window' ? 'manual' : values.schedule_type,
      schedule_cron: values.schedule_cron,
      timezone: values.timezone,
      mw_enabled: values.mw_enabled,
      mw_start: values.mw_start,
      mw_end: values.mw_end,
    };
    try {
      await createPolicy.mutateAsync(body as components['schemas']['CreatePolicyRequest']);
      onOpenChange(false);
      setTimeout(() => {
        form.reset(DEFAULT_POLICY_VALUES);
        setCurrentStep('basics');
        setCompletedSteps(new Set());
        setSubmitError(null);
      }, 300);
      navigate('/policies');
    } catch (err: unknown) {
      const apiErr = err as { message?: string; field?: string; code?: string };
      const message =
        apiErr?.message ?? (err instanceof Error ? err.message : null) ?? 'Failed to create policy';
      const field = apiErr?.field;
      setSubmitError(field ? `${message} (field: ${field})` : message);
    }
  }, [form, createPolicy, navigate, onOpenChange]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    setTimeout(() => {
      form.reset(DEFAULT_POLICY_VALUES);
      setCurrentStep('basics');
      setCompletedSteps(new Set());
      setSubmitError(null);
    }, 300);
  }, [form, onOpenChange]);

  return (
    <FormProvider {...form}>
      <Sheet open={open} onOpenChange={handleClose}>
        <SheetContent
          side="right"
          style={{
            width: 680,
            maxWidth: 680,
            padding: 0,
            display: 'flex',
            flexDirection: 'column',
          }}
          showCloseButton
        >
          <SheetHeader style={{ padding: '16px 16px 0' }}>
            <SheetTitle style={{ fontFamily: 'var(--font-display)', fontSize: 15 }}>
              New Policy
            </SheetTitle>
            <SheetDescription style={{ fontSize: 11 }}>
              Define patch scope, target groups, and scheduling rules
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
              {currentStep === 'basics' && <BasicsStep />}
              {currentStep === 'targets' && <TargetsStep />}
              {currentStep === 'patches' && <PatchesStep />}
              {currentStep === 'review' && (
                <ReviewStep
                  onSubmit={handleSubmit}
                  isSubmitting={createPolicy.isPending}
                  error={submitError}
                  onBack={goBack}
                />
              )}
            </div>
            <div style={{ padding: '16px 12px' }}>
              <ImpactPreview currentStep={currentStep} />
            </div>
          </div>

          {/* Footer nav (hidden on review — it has its own submit button) */}
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
