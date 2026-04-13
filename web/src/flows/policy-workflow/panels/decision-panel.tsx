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
import type { DecisionConfig } from '../types';

const schema = z.object({
  field: z.string().min(1, 'Field is required'),
  operator: z.enum(['equals', 'not_equals', 'in', 'gt', 'lt']),
  value: z.string(),
  true_label: z.string().optional(),
  false_label: z.string().optional(),
});

interface DecisionPanelProps {
  config: DecisionConfig;
  onSave: (config: DecisionConfig) => void;
}

export function DecisionPanel({ config, onSave }: DecisionPanelProps) {
  const { register, handleSubmit, watch, setValue } = useForm<DecisionConfig>({
    resolver: zodResolver(schema),
    defaultValues: config,
  });

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div>
        <label htmlFor="field" className="block text-sm font-medium mb-1">
          Field
        </label>
        <Input id="field" {...register('field')} />
        <p className="text-xs text-muted-foreground mt-1">
          The data field to evaluate (e.g. &quot;os_type&quot;, &quot;patch_count&quot;,
          &quot;severity&quot;).
        </p>
      </div>

      <div>
        <label htmlFor="operator" className="block text-sm font-medium mb-1">
          Operator
        </label>
        <Select
          value={watch('operator')}
          onValueChange={(v) => setValue('operator', v as DecisionConfig['operator'])}
        >
          <SelectTrigger id="operator">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="equals">Equals</SelectItem>
            <SelectItem value="not_equals">Not Equals</SelectItem>
            <SelectItem value="in">In</SelectItem>
            <SelectItem value="gt">Greater Than</SelectItem>
            <SelectItem value="lt">Less Than</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground mt-1">
          How to compare the field against the value.
        </p>
      </div>

      <div>
        <label htmlFor="value" className="block text-sm font-medium mb-1">
          Value
        </label>
        <Input id="value" {...register('value')} />
        <p className="text-xs text-muted-foreground mt-1">
          The value to compare against. For &quot;in&quot; operator, use comma-separated values.
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor="true_label" className="block text-sm font-medium mb-1">
            True Label
          </label>
          <Input id="true_label" placeholder="Yes" {...register('true_label')} />
        </div>
        <div>
          <label htmlFor="false_label" className="block text-sm font-medium mb-1">
            False Label
          </label>
          <Input id="false_label" placeholder="No" {...register('false_label')} />
        </div>
      </div>
      <p className="text-xs text-muted-foreground mt-1">
        Custom labels shown on the outgoing edges for each branch.
      </p>

      <Button type="submit">Save</Button>
    </form>
  );
}
