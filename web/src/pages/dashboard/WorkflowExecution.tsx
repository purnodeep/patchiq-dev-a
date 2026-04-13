import { Link } from 'react-router';
import { MonoTag } from '@patchiq/ui';

export interface WorkflowStep {
  name: string;
  status: 'completed' | 'running' | 'failed' | 'pending';
}

export interface RecentWorkflow {
  id: string;
  name: string;
  status: 'running' | 'completed' | 'failed';
  started_at: string;
  steps: WorkflowStep[];
}

export interface WorkflowExecutionProps {
  workflows: RecentWorkflow[];
}

function timeAgo(isoString: string): string {
  const diff = Math.floor((Date.now() - new Date(isoString).getTime()) / 1000);
  if (diff < 60) return 'just now';
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

const STATUS_COLORS: Record<WorkflowStep['status'], string> = {
  completed: 'var(--signal-healthy)',
  running: 'var(--accent)',
  failed: 'var(--signal-critical)',
  pending: 'var(--text-muted)',
};

interface StepCircleProps {
  step: WorkflowStep;
}

function StepCircle({ step }: StepCircleProps) {
  const color = STATUS_COLORS[step.status];
  const isPulsing = step.status === 'running';

  return (
    <div
      className="relative flex items-center justify-center w-7 h-7 rounded-full border-2 flex-shrink-0"
      style={{ borderColor: color, backgroundColor: `${color}20` }}
      aria-label={`${step.name}: ${step.status}`}
      title={step.name}
    >
      {isPulsing && (
        <span
          className="absolute inset-0 rounded-full animate-ping opacity-40"
          style={{ backgroundColor: color }}
        />
      )}
      {step.status === 'completed' && (
        <span style={{ color }} className="text-xs font-bold leading-none">
          {'\u2713'}
        </span>
      )}
      {step.status === 'failed' && (
        <span style={{ color }} className="text-xs font-bold leading-none">
          {'\u2717'}
        </span>
      )}
      {step.status === 'pending' && (
        <span style={{ color }} className="text-xs leading-none">
          {'\u25cb'}
        </span>
      )}
      {step.status === 'running' && (
        <span className="w-2 h-2 rounded-full" style={{ backgroundColor: color }} />
      )}
    </div>
  );
}

function lineColor(left: WorkflowStep, right: WorkflowStep): string {
  if (left.status === 'completed' && right.status === 'completed') return 'var(--signal-healthy)';
  if (left.status === 'failed' || right.status === 'failed') return 'var(--signal-critical)';
  return 'var(--text-muted)';
}

interface PipelineProps {
  steps: WorkflowStep[];
}

function Pipeline({ steps }: PipelineProps) {
  if (steps.length === 0) return null;

  return (
    <div className="flex items-center gap-0">
      {steps.map((step, i) => (
        <div key={i} className="flex items-center">
          <StepCircle step={step} />
          {i < steps.length - 1 && (
            <div
              className="h-0.5 w-6 flex-shrink-0"
              style={{ backgroundColor: lineColor(step, steps[i + 1]) }}
              aria-hidden="true"
            />
          )}
        </div>
      ))}
    </div>
  );
}

export function WorkflowExecution({ workflows }: WorkflowExecutionProps) {
  const recent = workflows.slice(0, 3);

  return (
    <div
      className="rounded-lg border"
      style={{
        background: 'var(--bg-card)',
        borderColor: 'var(--border)',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      <div className="flex items-center justify-between p-4 pb-2">
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-emphasis)' }}>
          Workflow Executions
        </h3>
        <Link
          to="/workflows"
          className="text-xs transition-colors"
          style={{ color: 'var(--text-muted)' }}
        >
          View All {'\u2192'}
        </Link>
      </div>
      <div className="p-4 pt-0">
        {recent.length === 0 ? (
          <p className="text-sm" style={{ color: 'var(--text-muted)' }}>
            No recent workflow executions.
          </p>
        ) : (
          <div className="space-y-4">
            {recent.map((wf) => (
              <div key={wf.id} className="flex flex-col gap-2">
                <div className="flex items-center justify-between">
                  <span
                    className="text-sm font-medium truncate"
                    style={{ color: 'var(--text-primary)' }}
                  >
                    {wf.name}
                  </span>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    <MonoTag>{wf.status}</MonoTag>
                    <span className="text-xs" style={{ color: 'var(--text-muted)' }}>
                      {timeAgo(wf.started_at)}
                    </span>
                  </div>
                </div>
                <Pipeline steps={wf.steps} />
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
