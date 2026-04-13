import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Input } from '@patchiq/ui';
import type { RebootConfig } from '../types';

const schema = z.object({
  timeout_minutes: z.number().min(1),
  force_reboot: z.boolean().optional(),
  failure_behavior: z.enum(['continue', 'halt']),
});

interface RebootPanelProps {
  config: RebootConfig;
  onSave: (config: RebootConfig) => void;
}

export function RebootPanel({ config, onSave }: RebootPanelProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RebootConfig>({
    resolver: zodResolver(schema),
    defaultValues: {
      timeout_minutes: config.timeout_minutes ?? 10,
      force_reboot: config.force_reboot ?? false,
      failure_behavior: config.failure_behavior ?? 'halt',
    },
  });

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
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
          Max time to wait for endpoints to come back online after reboot.
        </p>
      </div>

      <div className="flex items-center gap-2">
        <input type="checkbox" id="force_reboot" {...register('force_reboot')} />
        <label htmlFor="force_reboot" className="text-sm font-medium">
          Force Reboot
        </label>
      </div>
      <p className="text-xs text-muted-foreground ml-6">
        Force reboot even if users are logged in or applications are running.
      </p>

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
          Continue proceeds to the next node if reboot fails. Halt stops the workflow.
        </p>
      </div>

      <Button type="submit">Save</Button>
    </form>
  );
}
