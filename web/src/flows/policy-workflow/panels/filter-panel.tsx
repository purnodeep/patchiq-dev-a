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
import type { FilterConfig } from '../types';

const schema = z.object({
  os_types: z.array(z.string()).optional(),
  tags: z.array(z.string()).optional(),
  min_severity: z.string().optional(),
  package_regex: z.string().optional(),
});

interface FilterPanelProps {
  config: FilterConfig;
  onSave: (config: FilterConfig) => void;
}

export function FilterPanel({ config, onSave }: FilterPanelProps) {
  const { register, handleSubmit, watch, setValue } = useForm<FilterConfig>({
    resolver: zodResolver(schema),
    defaultValues: config,
  });

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div>
        <label htmlFor="os_types" className="block text-sm font-medium mb-1">
          OS Types
        </label>
        <Input
          id="os_types"
          defaultValue={config.os_types?.join(', ') ?? ''}
          onChange={(e) =>
            setValue(
              'os_types',
              e.target.value
                .split(',')
                .map((s) => s.trim())
                .filter(Boolean),
            )
          }
        />
        <p className="text-xs text-muted-foreground mt-1">Comma-separated (e.g. linux, windows)</p>
      </div>

      <div>
        <label htmlFor="tags" className="block text-sm font-medium mb-1">
          Tags
        </label>
        <Input
          id="tags"
          defaultValue={config.tags?.join(', ') ?? ''}
          onChange={(e) =>
            setValue(
              'tags',
              e.target.value
                .split(',')
                .map((s) => s.trim())
                .filter(Boolean),
            )
          }
        />
        <p className="text-xs text-muted-foreground mt-1">
          Comma-separated &quot;key=value&quot; tag predicates (e.g. env=prod, role=db).
          Endpoints must carry every listed tag to pass the filter.
        </p>
      </div>

      <div>
        <label htmlFor="min_severity" className="block text-sm font-medium mb-1">
          Min Severity
        </label>
        <Select
          value={watch('min_severity') ?? ''}
          onValueChange={(v) => setValue('min_severity', v)}
        >
          <SelectTrigger id="min_severity">
            <SelectValue placeholder="Select severity" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="low">Low</SelectItem>
            <SelectItem value="medium">Medium</SelectItem>
            <SelectItem value="high">High</SelectItem>
            <SelectItem value="critical">Critical</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground mt-1">
          Only include endpoints with patches at or above this severity.
        </p>
      </div>

      <div>
        <label htmlFor="package_regex" className="block text-sm font-medium mb-1">
          Package Regex
        </label>
        <Input id="package_regex" {...register('package_regex')} />
        <p className="text-xs text-muted-foreground mt-1">
          Regex pattern to match package names (e.g. &quot;openssl.*&quot;).
        </p>
      </div>

      <Button type="submit">Save</Button>
    </form>
  );
}
