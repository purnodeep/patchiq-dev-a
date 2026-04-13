import type { CompleteConfig } from '../types';

interface CompletePanelProps {
  config: CompleteConfig;
  onSave: (config: CompleteConfig) => void;
}

export function CompletePanel({ config }: CompletePanelProps) {
  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">Read-only workflow summary</p>
      <div className="rounded-md border p-3 space-y-2 text-sm">
        <div className="flex justify-between">
          <span className="text-muted-foreground">Generate Report</span>
          <span className="font-medium">{config.generate_report ? 'Yes' : 'No'}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-muted-foreground">Notify on Complete</span>
          <span className="font-medium">{config.notify_on_complete ? 'Yes' : 'No'}</span>
        </div>
      </div>
    </div>
  );
}
