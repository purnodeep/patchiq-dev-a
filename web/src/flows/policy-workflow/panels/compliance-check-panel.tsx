import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Input } from '@patchiq/ui';
import type { ComplianceCheckConfig } from '../types';

const schema = z.object({
  framework: z.enum(['CIS', 'PCI-DSS', 'HIPAA', 'NIST', 'ISO27001', 'SOC2']),
  min_score: z.number().min(0).max(100),
  failure_behavior: z.enum(['continue', 'halt']),
});

interface ComplianceCheckPanelProps {
  config: ComplianceCheckConfig;
  onSave: (config: ComplianceCheckConfig) => void;
}

export function ComplianceCheckPanel({ config, onSave }: ComplianceCheckPanelProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ComplianceCheckConfig>({
    resolver: zodResolver(schema),
    defaultValues: {
      framework: config.framework ?? 'CIS',
      min_score: config.min_score ?? 80,
      failure_behavior: config.failure_behavior ?? 'halt',
    },
  });

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div>
        <label htmlFor="framework" className="block text-sm font-medium mb-1">
          Framework
        </label>
        <select
          id="framework"
          {...register('framework')}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="CIS">CIS Benchmarks</option>
          <option value="PCI-DSS">PCI-DSS</option>
          <option value="HIPAA">HIPAA</option>
          <option value="NIST">NIST 800-53</option>
          <option value="ISO27001">ISO 27001</option>
          <option value="SOC2">SOC 2</option>
        </select>
        <p className="text-xs text-muted-foreground mt-1">
          The compliance framework to evaluate against. Results are stored in the compliance
          dashboard.
        </p>
      </div>

      <div>
        <label htmlFor="min_score" className="block text-sm font-medium mb-1">
          Minimum Score (%)
        </label>
        <Input id="min_score" type="number" {...register('min_score', { valueAsNumber: true })} />
        {errors.min_score && <p className="text-sm text-destructive">{errors.min_score.message}</p>}
        <p className="text-xs text-muted-foreground mt-1">
          Endpoints scoring below this percentage will cause the check to fail.
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
          Continue proceeds to the next node if compliance fails. Halt stops the workflow.
        </p>
      </div>

      <Button type="submit">Save</Button>
    </form>
  );
}
