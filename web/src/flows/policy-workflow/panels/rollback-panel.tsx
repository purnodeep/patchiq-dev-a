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
import type { RollbackConfig } from '../types';

const schema = z.object({
  strategy: z.enum(['snapshot_restore', 'package_downgrade', 'script']),
  failure_threshold: z.number().min(0),
  rollback_script: z.string().optional(),
  target_deployment: z.string().optional(),
});

interface RollbackPanelProps {
  config: RollbackConfig;
  onSave: (config: RollbackConfig) => void;
}

export function RollbackPanel({ config, onSave }: RollbackPanelProps) {
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<RollbackConfig>({
    resolver: zodResolver(schema),
    defaultValues: config,
  });

  const strategy = watch('strategy');

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div>
        <label htmlFor="strategy" className="block text-sm font-medium mb-1">
          Strategy
        </label>
        <Select
          value={strategy}
          onValueChange={(v) => setValue('strategy', v as RollbackConfig['strategy'])}
        >
          <SelectTrigger id="strategy">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="snapshot_restore">Snapshot Restore</SelectItem>
            <SelectItem value="package_downgrade">Package Downgrade</SelectItem>
            <SelectItem value="script">Script</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground mt-1">
          Snapshot restore reverts to a pre-deployment state. Package downgrade reinstalls the
          previous version.
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
          Number of endpoint failures that triggers this rollback.
        </p>
      </div>

      <div>
        <label htmlFor="target_deployment" className="block text-sm font-medium mb-1">
          Target Deployment
        </label>
        <Input
          id="target_deployment"
          placeholder="Deployment ID or name"
          {...register('target_deployment')}
        />
        <p className="text-xs text-muted-foreground mt-1">
          The deployment to roll back. Leave empty to target the most recent deployment.
        </p>
      </div>

      {strategy === 'script' && (
        <div>
          <label htmlFor="rollback_script" className="block text-sm font-medium mb-1">
            Rollback Script
          </label>
          <textarea
            id="rollback_script"
            className="w-full min-h-[120px] rounded-md border border-input bg-background px-3 py-2 text-sm font-mono"
            {...register('rollback_script')}
          />
          <p className="text-xs text-muted-foreground mt-1">
            Custom script executed on each endpoint to perform the rollback.
          </p>
        </div>
      )}

      <Button type="submit">Save</Button>
    </form>
  );
}
