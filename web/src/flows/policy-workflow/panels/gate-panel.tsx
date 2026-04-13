import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Input } from '@patchiq/ui';
import type { GateConfig } from '../types';

const schema = z.object({
  wait_minutes: z.number().min(1),
  failure_threshold: z.number().min(0),
  health_check: z.boolean().optional(),
});

interface GatePanelProps {
  config: GateConfig;
  onSave: (config: GateConfig) => void;
}

export function GatePanel({ config, onSave }: GatePanelProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<GateConfig>({
    resolver: zodResolver(schema),
    defaultValues: config,
  });

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div>
        <label htmlFor="wait_minutes" className="block text-sm font-medium mb-1">
          Wait Minutes
        </label>
        <Input
          id="wait_minutes"
          type="number"
          {...register('wait_minutes', { valueAsNumber: true })}
        />
        {errors.wait_minutes && (
          <p className="text-sm text-destructive">{errors.wait_minutes.message}</p>
        )}
        <p className="text-xs text-muted-foreground mt-1">
          Duration to pause before proceeding. Use to allow time for monitoring after a deployment
          wave.
        </p>
      </div>

      <div>
        <label htmlFor="failure_threshold" className="block text-sm font-medium mb-1">
          Failure Threshold
        </label>
        <Input
          id="failure_threshold"
          type="number"
          {...register('failure_threshold', { valueAsNumber: true })}
        />
        {errors.failure_threshold && (
          <p className="text-sm text-destructive">{errors.failure_threshold.message}</p>
        )}
        <p className="text-xs text-muted-foreground mt-1">
          Number of endpoint failures allowed before the gate blocks the workflow.
        </p>
      </div>

      <div className="flex items-center gap-2">
        <input type="checkbox" id="health_check" {...register('health_check')} />
        <label htmlFor="health_check" className="text-sm font-medium">
          Health Check
        </label>
      </div>
      <p className="text-xs text-muted-foreground ml-6">
        Run a health check on endpoints before allowing the workflow to continue.
      </p>

      <Button type="submit">Save</Button>
    </form>
  );
}
