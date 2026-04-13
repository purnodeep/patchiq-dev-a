import { Badge } from '@patchiq/ui';
import { useWorkflowExecution } from './hooks/use-workflow-executions';

const statusColors: Record<string, string> = {
  pending: 'bg-gray-100 text-gray-700',
  running: 'bg-yellow-100 text-yellow-700',
  completed: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
  skipped: 'bg-gray-100 text-gray-500',
  paused: 'bg-orange-100 text-orange-700',
  cancelled: 'bg-gray-100 text-gray-500',
};

const dotColorMap: Record<string, string> = {
  completed: 'bg-green-500',
  running: 'bg-yellow-500 animate-pulse',
  failed: 'bg-red-500',
};

interface ExecutionStatusPanelProps {
  workflowId: string;
  executionId: string;
}

export function ExecutionStatusPanel({ workflowId, executionId }: ExecutionStatusPanelProps) {
  const {
    data: execution,
    isLoading,
    isError,
  } = useWorkflowExecution(workflowId, executionId, {
    refetchInterval: 3000,
  });

  if (isLoading)
    return (
      <div className="border-t px-4 py-3 text-sm text-muted-foreground">
        Loading execution status...
      </div>
    );
  if (isError || !execution)
    return (
      <div className="border-t px-4 py-3 text-sm text-destructive">
        Failed to load execution status
      </div>
    );

  return (
    <div className="border-t bg-muted/20 px-4 py-3">
      <div className="flex items-center justify-between mb-2">
        <h4 className="text-sm font-semibold">Execution Status</h4>
        <Badge className={statusColors[execution.status] ?? ''}>{execution.status}</Badge>
      </div>
      {execution.node_executions.length > 0 && (
        <div className="space-y-1">
          {execution.node_executions.map((ne) => (
            <div key={ne.id} className="flex items-center gap-2 text-xs">
              <span
                className={`inline-block h-2 w-2 rounded-full ${dotColorMap[ne.status] ?? 'bg-gray-300'}`}
              />
              <span className="font-mono text-muted-foreground">{ne.node_id.slice(0, 8)}</span>
              <span>{ne.status}</span>
              {ne.error_message && <span className="text-red-500">— {ne.error_message}</span>}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
