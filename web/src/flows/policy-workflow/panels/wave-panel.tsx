import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Input } from '@patchiq/ui';
import type { DeploymentWaveConfig } from '../types';

const schema = z.object({
  percentage: z.number().min(1).max(100),
  max_parallel: z.number().min(1).optional(),
  timeout_minutes: z.number().min(1).optional(),
  success_threshold: z.number().min(0).max(100).optional(),
});

interface WavePanelProps {
  config: DeploymentWaveConfig;
  onSave: (config: DeploymentWaveConfig) => void;
}

export function WavePanel({ config, onSave }: WavePanelProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<DeploymentWaveConfig>({
    resolver: zodResolver(schema),
    defaultValues: config,
  });

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div>
        <label htmlFor="percentage" className="block text-sm font-medium mb-1">
          Percentage
        </label>
        <Input id="percentage" type="number" {...register('percentage', { valueAsNumber: true })} />
        <p className="text-xs text-muted-foreground mt-1">
          Percentage of endpoints to include in this wave (1-100%).
        </p>
        {errors.percentage && (
          <p className="text-sm text-destructive">{errors.percentage.message}</p>
        )}
      </div>

      <div>
        <label htmlFor="max_parallel" className="block text-sm font-medium mb-1">
          Max Parallel
        </label>
        <Input
          id="max_parallel"
          type="number"
          {...register('max_parallel', { valueAsNumber: true })}
        />
        <p className="text-xs text-muted-foreground mt-1">
          Maximum number of endpoints to deploy to concurrently.
        </p>
        {errors.max_parallel && (
          <p className="text-sm text-destructive">{errors.max_parallel.message}</p>
        )}
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
        <p className="text-xs text-muted-foreground mt-1">
          Max time to wait for this wave to complete before timing out.
        </p>
        {errors.timeout_minutes && (
          <p className="text-sm text-destructive">{errors.timeout_minutes.message}</p>
        )}
      </div>

      <div>
        <label htmlFor="success_threshold" className="block text-sm font-medium mb-1">
          Success Threshold (%)
        </label>
        <Input
          id="success_threshold"
          type="number"
          {...register('success_threshold', { valueAsNumber: true })}
        />
        <p className="text-xs text-muted-foreground mt-1">
          Minimum success rate (0-100%) required before proceeding to the next wave.
        </p>
        {errors.success_threshold && (
          <p className="text-sm text-destructive">{errors.success_threshold.message}</p>
        )}
      </div>

      <Button type="submit">Save</Button>
    </form>
  );
}
