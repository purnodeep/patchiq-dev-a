import {
  Play,
  Filter,
  ShieldCheck,
  Waves,
  GanttChart,
  Terminal,
  Bell,
  Undo2,
  Diamond,
  CheckCircle2,
  RotateCcw,
  Scan,
  Tag,
  ClipboardCheck,
  type LucideIcon,
} from 'lucide-react';
import type { WorkflowNodeType } from './types';

export interface NodeTypeInfo {
  label: string;
  description: string;
  icon: LucideIcon;
  color: string;
  bgColor: string;
  borderColor: string;
}

export const NODE_TYPE_REGISTRY: Record<WorkflowNodeType, NodeTypeInfo> = {
  trigger: {
    label: 'Trigger',
    description:
      'Defines how this workflow starts. Choose manual for on-demand execution, cron for scheduled runs, or CVE severity to auto-trigger when vulnerabilities are detected.',
    icon: Play,
    color: 'text-green-700',
    bgColor: 'bg-green-50',
    borderColor: 'border-green-300',
  },
  filter: {
    label: 'Filter',
    description:
      'Narrows which endpoints this workflow targets. Filter by OS type, endpoint group, tags, severity level, or package name patterns.',
    icon: Filter,
    color: 'text-blue-700',
    bgColor: 'bg-blue-50',
    borderColor: 'border-blue-300',
  },
  approval: {
    label: 'Approval',
    description:
      'Pauses the workflow until a designated role approves. Configure timeout behavior to auto-reject or escalate if no response.',
    icon: ShieldCheck,
    color: 'text-orange-700',
    bgColor: 'bg-orange-50',
    borderColor: 'border-orange-300',
  },
  deployment_wave: {
    label: 'Deploy Wave',
    description:
      'Deploys patches to a percentage of endpoints in a controlled rollout. Use multiple waves for canary or progressive deployments.',
    icon: Waves,
    color: 'text-purple-700',
    bgColor: 'bg-purple-50',
    borderColor: 'border-purple-300',
  },
  gate: {
    label: 'Gate',
    description:
      'Pauses the workflow for a set duration to monitor health. Optionally runs a health check before allowing the next step.',
    icon: GanttChart,
    color: 'text-yellow-700',
    bgColor: 'bg-yellow-50',
    borderColor: 'border-yellow-300',
  },
  script: {
    label: 'Script',
    description:
      'Runs a custom shell or PowerShell script on target endpoints. Use for pre/post-deployment checks, cleanup, or custom logic.',
    icon: Terminal,
    color: 'text-gray-700',
    bgColor: 'bg-gray-50',
    borderColor: 'border-gray-300',
  },
  notification: {
    label: 'Notification',
    description:
      'Sends an alert via email, Slack, webhook, or PagerDuty. Use to notify teams of deployment progress or failures.',
    icon: Bell,
    color: 'text-teal-700',
    bgColor: 'bg-teal-50',
    borderColor: 'border-teal-300',
  },
  rollback: {
    label: 'Rollback',
    description:
      'Reverts endpoints to their previous state. Choose between snapshot restore, package downgrade, or a custom rollback script.',
    icon: Undo2,
    color: 'text-red-700',
    bgColor: 'bg-red-50',
    borderColor: 'border-red-300',
  },
  decision: {
    label: 'Decision',
    description:
      'Branches the workflow based on a condition. Evaluates a field against a value and routes to different paths for true/false outcomes.',
    icon: Diamond,
    color: 'text-amber-700',
    bgColor: 'bg-amber-50',
    borderColor: 'border-amber-300',
  },
  complete: {
    label: 'Complete',
    description:
      'Marks the workflow as finished. Optionally generates a summary report and sends a completion notification.',
    icon: CheckCircle2,
    color: 'text-green-700',
    bgColor: 'bg-green-50',
    borderColor: 'border-green-300',
  },
  reboot: {
    label: 'Reboot',
    description:
      'Restarts target endpoints after patching. Configure timeout and whether to force reboot unresponsive machines.',
    icon: RotateCcw,
    color: 'text-rose-700',
    bgColor: 'bg-rose-50',
    borderColor: 'border-rose-300',
  },
  scan: {
    label: 'Scan',
    description:
      'Runs an inventory, vulnerability, or compliance scan on endpoints. Use after deployment to verify patch application.',
    icon: Scan,
    color: 'text-cyan-700',
    bgColor: 'bg-cyan-50',
    borderColor: 'border-cyan-300',
  },
  tag_gate: {
    label: 'Tag Gate',
    description:
      'Gates the workflow based on endpoint tags. Only allows endpoints with specific tags to proceed.',
    icon: Tag,
    color: 'text-indigo-700',
    bgColor: 'bg-indigo-50',
    borderColor: 'border-indigo-300',
  },
  compliance_check: {
    label: 'Compliance Check',
    description:
      'Evaluates endpoints against a compliance framework (CIS, PCI-DSS, HIPAA, etc.) and blocks if the score is below threshold.',
    icon: ClipboardCheck,
    color: 'text-emerald-700',
    bgColor: 'bg-emerald-50',
    borderColor: 'border-emerald-300',
  },
};

export const ALL_NODE_TYPES: WorkflowNodeType[] = Object.keys(
  NODE_TYPE_REGISTRY,
) as WorkflowNodeType[];
