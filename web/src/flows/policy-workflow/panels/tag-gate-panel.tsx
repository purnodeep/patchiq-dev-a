import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Input } from '@patchiq/ui';
import type { TagGateConfig } from '../types';

const schema = z.object({
  required_tags: z.string().min(1, 'At least one tag is required'),
  match_mode: z.enum(['all', 'any']),
});

type FormValues = { required_tags: string; match_mode: 'all' | 'any' };

interface TagGatePanelProps {
  config: TagGateConfig;
  onSave: (config: TagGateConfig) => void;
}

export function TagGatePanel({ config, onSave }: TagGatePanelProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      required_tags: (config.required_tags ?? []).join(', '),
      match_mode: config.match_mode ?? 'all',
    },
  });

  const onSubmit = (data: FormValues) => {
    onSave({
      required_tags: data.required_tags
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean),
      match_mode: data.match_mode,
    });
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div>
        <label htmlFor="required_tags" className="block text-sm font-medium mb-1">
          Required Tags
        </label>
        <Input id="required_tags" placeholder="tag1, tag2, tag3" {...register('required_tags')} />
        <p className="text-xs text-muted-foreground mt-1">Comma-separated list of tags</p>
        {errors.required_tags && (
          <p className="text-sm text-destructive">{errors.required_tags.message}</p>
        )}
      </div>

      <div>
        <label htmlFor="match_mode" className="block text-sm font-medium mb-1">
          Match Mode
        </label>
        <select
          id="match_mode"
          {...register('match_mode')}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="all">All tags must match</option>
          <option value="any">Any tag matches</option>
        </select>
        <p className="text-xs text-muted-foreground mt-1">
          &quot;All&quot; requires every tag to be present. &quot;Any&quot; passes if at least one
          tag matches.
        </p>
      </div>

      <Button type="submit">Save</Button>
    </form>
  );
}
