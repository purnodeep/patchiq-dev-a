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
import type { ApprovalConfig } from '../types';

const schema = z.object({
  approver_roles: z.array(z.string()).min(1, 'At least one approver role required'),
  timeout_hours: z.number().min(1, 'Must be at least 1 hour'),
  escalation_role: z.string().optional(),
  timeout_action: z.enum(['reject', 'escalate']).optional(),
});

interface ApprovalPanelProps {
  config: ApprovalConfig;
  onSave: (config: ApprovalConfig) => void;
}

export function ApprovalPanel({ config, onSave }: ApprovalPanelProps) {
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<ApprovalConfig>({
    resolver: zodResolver(schema),
    defaultValues: config,
  });

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div>
        <label htmlFor="approver_roles" className="block text-sm font-medium mb-1">
          Approver Roles
        </label>
        <Input
          id="approver_roles"
          defaultValue={config.approver_roles?.join(', ') ?? ''}
          onChange={(e) =>
            setValue(
              'approver_roles',
              e.target.value
                .split(',')
                .map((s) => s.trim())
                .filter(Boolean),
            )
          }
        />
        <p className="text-xs text-muted-foreground mt-1">Comma-separated</p>
        {errors.approver_roles && (
          <p className="text-sm text-destructive">{errors.approver_roles.message}</p>
        )}
      </div>

      <div>
        <label htmlFor="timeout_hours" className="block text-sm font-medium mb-1">
          Timeout Hours
        </label>
        <Input
          id="timeout_hours"
          type="number"
          {...register('timeout_hours', { valueAsNumber: true })}
        />
        {errors.timeout_hours && (
          <p className="text-sm text-destructive">{errors.timeout_hours.message}</p>
        )}
        <p className="text-xs text-muted-foreground mt-1">
          Hours to wait for approval before triggering the timeout action.
        </p>
      </div>

      <div>
        <label htmlFor="escalation_role" className="block text-sm font-medium mb-1">
          Escalation Role
        </label>
        <Input id="escalation_role" {...register('escalation_role')} />
        <p className="text-xs text-muted-foreground mt-1">
          Role to escalate to if the timeout action is &quot;escalate&quot;.
        </p>
      </div>

      <div>
        <label htmlFor="timeout_action" className="block text-sm font-medium mb-1">
          Timeout Action
        </label>
        <Select
          value={watch('timeout_action') ?? ''}
          onValueChange={(v) => setValue('timeout_action', v as ApprovalConfig['timeout_action'])}
        >
          <SelectTrigger id="timeout_action">
            <SelectValue placeholder="Select action" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="reject">Reject</SelectItem>
            <SelectItem value="escalate">Escalate</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground mt-1">
          What happens if no one approves within the timeout period.
        </p>
      </div>

      <Button type="submit">Save</Button>
    </form>
  );
}
