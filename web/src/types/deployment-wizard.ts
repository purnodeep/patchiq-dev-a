import { z } from 'zod';

// --- Tag Expression Types ---

export interface TagExpression {
  op?: 'AND' | 'OR' | 'NOT';
  tag?: string;
  value?: string;
  conditions?: TagExpression[];
}

// --- Wave Config ---

export interface WaveConfig {
  tagExpression?: TagExpression;
  maxTargets: number;
  successThreshold: number;
}

// --- Wizard Form Values ---

export interface DeploymentWizardValues {
  // Step 1: Source
  sourceType: 'catalog' | 'policy' | 'adhoc';
  patchIds?: string[];
  policyId?: string;
  adhocPackages?: { name: string; version: string }[];

  // Step 2: Targets
  targetMode: 'all' | 'tags' | 'select';
  targetExpression?: TagExpression;
  endpointIds: string[];
  respectMaintenanceWindow: boolean;
  excludePendingDeployments: boolean;

  // Step 3: Strategy
  waves: WaveConfig[];
  schedule: 'now' | 'datetime' | 'maintenance_window';
  scheduledAt?: string;
  rollbackThreshold: number;
  autoReboot: boolean;
  rebootMode: 'immediate' | 'graceful' | 'deferred';
  rebootGracePeriod: number;
  workflowTemplateId?: string;

  // Step 4: Review
  name?: string;
  description?: string;
}

// --- Wizard Steps ---

export const WIZARD_STEPS = [
  { id: 'source', label: 'Source', number: 1 },
  { id: 'targets', label: 'Targets', number: 2 },
  { id: 'strategy', label: 'Strategy', number: 3 },
  { id: 'review', label: 'Review', number: 4 },
] as const;

export type WizardStepId = (typeof WIZARD_STEPS)[number]['id'];

// --- Zod Schemas for Validation ---

export const tagExpressionSchema: z.ZodType<TagExpression> = z.lazy(() =>
  z.object({
    op: z.enum(['AND', 'OR', 'NOT']).optional(),
    tag: z.string().optional(),
    value: z.string().optional(),
    conditions: z.array(tagExpressionSchema).optional(),
  }),
);

export const waveConfigSchema = z.object({
  tagExpression: tagExpressionSchema.optional(),
  maxTargets: z.number().min(1, 'Max targets must be at least 1'),
  successThreshold: z.number().min(0).max(100, 'Threshold must be 0-100'),
});

export const adhocPackageSchema = z.object({
  name: z.string().min(1, 'Package name is required'),
  version: z.string().min(1, 'Version is required'),
});

export const deploymentWizardSchema = z.object({
  // Step 1
  sourceType: z.enum(['catalog', 'policy', 'adhoc']),
  patchIds: z.array(z.string()).optional(),
  policyId: z.string().optional(),
  adhocPackages: z.array(adhocPackageSchema).optional(),

  // Step 2
  targetMode: z.enum(['all', 'tags', 'select']),
  targetExpression: tagExpressionSchema.optional(),
  endpointIds: z.array(z.string()).default([]),
  respectMaintenanceWindow: z.boolean(),
  excludePendingDeployments: z.boolean(),

  // Step 3
  waves: z.array(waveConfigSchema),
  schedule: z.enum(['now', 'datetime', 'maintenance_window']),
  scheduledAt: z.string().optional(),
  rollbackThreshold: z.number().min(0).max(100),
  autoReboot: z.boolean(),
  rebootMode: z.enum(['immediate', 'graceful', 'deferred']),
  rebootGracePeriod: z.number().min(0),
  workflowTemplateId: z.string().optional(),

  // Step 4
  name: z.string().optional(),
  description: z.string().optional(),
});

// --- Default Values ---

export const DEFAULT_WIZARD_VALUES: DeploymentWizardValues = {
  sourceType: 'catalog',
  patchIds: [],
  targetMode: 'all',
  endpointIds: [],
  respectMaintenanceWindow: true,
  excludePendingDeployments: true,
  waves: [{ maxTargets: 100, successThreshold: 95 }],
  schedule: 'now',
  rollbackThreshold: 10,
  autoReboot: false,
  rebootMode: 'graceful',
  rebootGracePeriod: 300,
};

// --- Wizard Initial State (for opening from different entry points) ---

export interface DeploymentWizardInitialState {
  sourceType?: 'catalog' | 'policy' | 'adhoc';
  patchIds?: string[];
  policyId?: string;
  startStep?: WizardStepId;
}
