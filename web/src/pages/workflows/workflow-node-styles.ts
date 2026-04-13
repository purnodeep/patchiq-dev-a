import type { CSSProperties } from 'react';
import {
  Zap,
  Filter,
  ShieldCheck,
  Rocket,
  Timer,
  Bell,
  CircleCheckBig,
  GitBranch,
  RotateCcw,
} from 'lucide-react';

// Precision Clarity design system: 4 node categories, restrained color
// Trigger = accent (emerald), Action = neutral border only, Gate/Decision = amber, Error/Rollback = red

export const nodeTypeStyle: Record<
  string,
  { bg: string; border: string; text: string; leftBar: string; cls: string }
> = {
  trigger: {
    bg: 'var(--bg-elevated)',
    border: 'var(--accent)',
    text: 'var(--accent)',
    leftBar: 'var(--accent)',
    cls: '',
  },
  filter: {
    bg: 'var(--bg-elevated)',
    border: 'var(--border)',
    text: 'var(--text-secondary)',
    leftBar: 'var(--border-strong)',
    cls: '',
  },
  approval: {
    bg: 'var(--bg-elevated)',
    border: 'color-mix(in srgb, var(--signal-warning) 27%, transparent)',
    text: 'var(--signal-warning)',
    leftBar: 'var(--signal-warning)',
    cls: '',
  },
  deployment_wave: {
    bg: 'var(--bg-elevated)',
    border: 'var(--border)',
    text: 'var(--text-secondary)',
    leftBar: 'var(--border-strong)',
    cls: '',
  },
  gate: {
    bg: 'var(--bg-elevated)',
    border: 'color-mix(in srgb, var(--signal-warning) 27%, transparent)',
    text: 'var(--signal-warning)',
    leftBar: 'var(--signal-warning)',
    cls: '',
  },
  script: {
    bg: 'var(--bg-elevated)',
    border: 'var(--border)',
    text: 'var(--text-secondary)',
    leftBar: 'var(--border-strong)',
    cls: '',
  },
  notification: {
    bg: 'var(--bg-elevated)',
    border: 'var(--border)',
    text: 'var(--text-secondary)',
    leftBar: 'var(--border-strong)',
    cls: '',
  },
  rollback: {
    bg: 'var(--bg-elevated)',
    border: 'color-mix(in srgb, var(--signal-critical) 27%, transparent)',
    text: 'var(--signal-critical)',
    leftBar: 'var(--signal-critical)',
    cls: '',
  },
  decision: {
    bg: 'var(--bg-elevated)',
    border: 'color-mix(in srgb, var(--signal-warning) 27%, transparent)',
    text: 'var(--signal-warning)',
    leftBar: 'var(--signal-warning)',
    cls: '',
  },
  complete: {
    bg: 'var(--bg-elevated)',
    border: 'var(--border)',
    text: 'var(--text-muted)',
    leftBar: 'var(--border)',
    cls: '',
  },
  reboot: {
    bg: 'var(--bg-elevated)',
    border: 'color-mix(in srgb, var(--signal-critical) 27%, transparent)',
    text: 'var(--signal-critical)',
    leftBar: 'var(--signal-critical)',
    cls: '',
  },
  scan: {
    bg: 'var(--bg-elevated)',
    border: 'var(--border)',
    text: 'var(--text-secondary)',
    leftBar: 'var(--border-strong)',
    cls: '',
  },
  tag_gate: {
    bg: 'var(--bg-elevated)',
    border: 'color-mix(in srgb, var(--signal-warning) 27%, transparent)',
    text: 'var(--signal-warning)',
    leftBar: 'var(--signal-warning)',
    cls: '',
  },
  compliance_check: {
    bg: 'var(--bg-elevated)',
    border: 'var(--border)',
    text: 'var(--text-secondary)',
    leftBar: 'var(--border-strong)',
    cls: '',
  },
};

export const nodeTypeIcon: Record<
  string,
  React.ComponentType<{ className?: string; style?: CSSProperties }>
> = {
  trigger: Zap,
  filter: Filter,
  approval: ShieldCheck,
  deployment_wave: Rocket,
  gate: Timer,
  script: GitBranch,
  notification: Bell,
  rollback: RotateCcw,
  decision: GitBranch,
  complete: CircleCheckBig,
  reboot: RotateCcw,
  scan: Filter,
  tag_gate: Timer,
  compliance_check: ShieldCheck,
};

// Ordered type list for card mini-pipeline approximation (when we don't have per-node data)
export const defaultPipelineTypes = [
  'trigger',
  'filter',
  'approval',
  'deployment_wave',
  'gate',
  'notification',
  'complete',
];
