import { z } from 'zod';

export const policyWizardSchema = z.object({
  // Basics
  name: z.string().min(1, 'Name is required').max(255),
  description: z.string().max(1000),
  policy_type: z.enum(['patch', 'deploy', 'compliance']),
  mode: z.enum(['automatic', 'manual', 'advisory']),
  // Targets — tag-based selector (nullable; server treats null as "no endpoints")
  target_selector: z.any().nullable().optional(),
  respect_maintenance_window: z.boolean(),
  online_only: z.boolean(),
  // Patches
  selection_mode: z.enum(['all_available', 'by_severity', 'by_cve_list', 'by_regex']),
  min_severity: z.enum(['critical', 'high', 'medium', 'low']).optional(),
  cve_ids: z.array(z.string()),
  package_regex: z.string(),
  exclude_packages: z.array(z.string()),
  // Schedule
  schedule_type: z.enum(['manual', 'recurring', 'maintenance_window']),
  schedule_cron: z.string(),
  timezone: z.string(),
  mw_enabled: z.boolean(),
  mw_start: z.string(),
  mw_end: z.string(),
});

export type PolicyWizardValues = z.infer<typeof policyWizardSchema>;

export const DEFAULT_POLICY_VALUES: PolicyWizardValues = {
  name: '',
  description: '',
  policy_type: 'patch',
  mode: 'manual',
  target_selector: null,
  respect_maintenance_window: true,
  online_only: false,
  selection_mode: 'all_available',
  min_severity: undefined,
  cve_ids: [],
  package_regex: '',
  exclude_packages: [],
  schedule_type: 'manual',
  schedule_cron: '',
  timezone: 'UTC',
  mw_enabled: false,
  mw_start: '',
  mw_end: '',
};

export type PolicyWizardStepId = 'basics' | 'targets' | 'patches' | 'review';

export const POLICY_WIZARD_STEPS: { id: PolicyWizardStepId; label: string; number: string }[] = [
  { id: 'basics', label: 'Basics', number: '1' },
  { id: 'targets', label: 'Targets', number: '2' },
  { id: 'patches', label: 'Patches', number: '3' },
  { id: 'review', label: 'Review', number: '4' },
];

// Shared style tokens — copied verbatim from DeploymentWizard
export const LABEL_STYLE: React.CSSProperties = {
  fontSize: 10,
  fontWeight: 600,
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  fontFamily: 'var(--font-mono)',
  marginBottom: 8,
  display: 'block',
};

export const INPUT: React.CSSProperties = {
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

export const TOGGLE_CARD: React.CSSProperties = {
  background: 'var(--bg-inset)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  padding: '12px 14px',
};

export const SUMMARY_CARD: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 12,
  background: 'var(--bg-inset)',
  border: '1px solid var(--border)',
  borderRadius: 7,
  padding: '10px 14px',
};

export const COMMON_TIMEZONES = [
  'UTC',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'Europe/London',
  'Europe/Paris',
  'Europe/Berlin',
  'Asia/Kolkata',
  'Asia/Tokyo',
  'Asia/Shanghai',
  'Australia/Sydney',
  'Pacific/Auckland',
];
