import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Input } from '@patchiq/ui';
import type { ScanConfig } from '../types';

const schema = z.object({
  scan_type: z.enum(['inventory', 'vulnerability', 'compliance']),
  timeout_minutes: z.number().min(1),
  failure_behavior: z.enum(['continue', 'halt']),
});

interface ScanPanelProps {
  config: ScanConfig;
  onSave: (config: ScanConfig) => void;
}

export function ScanPanel({ config, onSave }: ScanPanelProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ScanConfig>({
    resolver: zodResolver(schema),
    defaultValues: {
      scan_type: config.scan_type ?? 'inventory',
      timeout_minutes: config.timeout_minutes ?? 30,
      failure_behavior: config.failure_behavior ?? 'halt',
    },
  });

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div>
        <label htmlFor="scan_type" className="block text-sm font-medium mb-1">
          Scan Type
        </label>
        <select
          id="scan_type"
          {...register('scan_type')}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="inventory">Inventory</option>
          <option value="vulnerability">Vulnerability</option>
          <option value="compliance">Compliance</option>
        </select>
        <p className="text-xs text-muted-foreground mt-1">
          Inventory collects installed software. Vulnerability checks for known CVEs. Compliance
          evaluates against frameworks.
        </p>
      </div>

      <div>
        <label htmlFor="timeout_minutes" className="block text-sm font-medium mb-1">
          Timeout (minutes)
        </label>
        <Input
          id="timeout_minutes"
          type="number"
          {...register('timeout_minutes', { valueAsNumber: true })}
        />
        {errors.timeout_minutes && (
          <p className="text-sm text-destructive">{errors.timeout_minutes.message}</p>
        )}
        <p className="text-xs text-muted-foreground mt-1">
          Max time to wait for the scan to complete on all endpoints.
        </p>
      </div>

      <div>
        <label htmlFor="failure_behavior" className="block text-sm font-medium mb-1">
          On Failure
        </label>
        <select
          id="failure_behavior"
          {...register('failure_behavior')}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="halt">Halt workflow</option>
          <option value="continue">Continue</option>
        </select>
        <p className="text-xs text-muted-foreground mt-1">
          Continue proceeds to the next node if the scan fails. Halt stops the workflow.
        </p>
      </div>

      <Button type="submit">Save</Button>
    </form>
  );
}
