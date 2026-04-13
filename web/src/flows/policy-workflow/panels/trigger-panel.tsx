import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  Button,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@patchiq/ui';
import type { TriggerConfig } from '../types';

const schema = z.object({
  trigger_type: z.enum(['manual', 'cron', 'cve_severity', 'policy_evaluation']),
  cron_expression: z.string().optional(),
  severity_threshold: z.string().optional(),
});

interface TriggerPanelProps {
  config: TriggerConfig;
  onSave: (config: TriggerConfig) => void;
}

export function TriggerPanel({ config, onSave }: TriggerPanelProps) {
  const { register, handleSubmit, watch, setValue } = useForm<TriggerConfig>({
    resolver: zodResolver(schema),
    defaultValues: config,
  });

  const triggerType = watch('trigger_type');

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div>
        <label htmlFor="trigger_type" className="block text-sm font-medium mb-1">
          Trigger Type
        </label>
        <Select
          value={triggerType}
          onValueChange={(v) => setValue('trigger_type', v as TriggerConfig['trigger_type'])}
        >
          <SelectTrigger id="trigger_type">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="manual">Manual</SelectItem>
            <SelectItem value="cron">Cron</SelectItem>
            <SelectItem value="cve_severity">CVE Severity</SelectItem>
            <SelectItem value="policy_evaluation">Policy Evaluation</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground mt-1">
          How this workflow gets started. Manual triggers require explicit execution.
        </p>
      </div>

      {triggerType === 'cron' && (
        <div>
          <label htmlFor="cron_expression" className="block text-sm font-medium mb-1">
            Cron Expression
          </label>
          <Input
            id="cron_expression"
            placeholder="e.g. 0 2 * * *"
            {...register('cron_expression')}
          />
          <p className="text-xs text-muted-foreground mt-1">
            Standard cron format (minute hour day month weekday). Example: &quot;0 2 * * *&quot;
            runs daily at 2 AM.
          </p>
        </div>
      )}

      {triggerType === 'cve_severity' && (
        <div>
          <label htmlFor="severity_threshold" className="block text-sm font-medium mb-1">
            Severity Threshold
          </label>
          <Select
            value={watch('severity_threshold') ?? ''}
            onValueChange={(v) => setValue('severity_threshold', v)}
          >
            <SelectTrigger id="severity_threshold">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="low">Low</SelectItem>
              <SelectItem value="medium">Medium</SelectItem>
              <SelectItem value="high">High</SelectItem>
              <SelectItem value="critical">Critical</SelectItem>
            </SelectContent>
          </Select>
          <p className="text-xs text-muted-foreground mt-1">
            Minimum CVSS severity level that triggers this workflow.
          </p>
        </div>
      )}

      <Button type="submit">Save</Button>
    </form>
  );
}
