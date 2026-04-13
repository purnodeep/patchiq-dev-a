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
import type { NotificationConfig } from '../types';

const schema = z.object({
  channel: z.enum(['email', 'slack', 'webhook', 'pagerduty']),
  target: z.string().min(1, 'Target is required'),
  message_template: z.string().optional(),
});

interface NotificationPanelProps {
  config: NotificationConfig;
  onSave: (config: NotificationConfig) => void;
}

export function NotificationPanel({ config, onSave }: NotificationPanelProps) {
  const { register, handleSubmit, watch, setValue } = useForm<NotificationConfig>({
    resolver: zodResolver(schema),
    defaultValues: config,
  });

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div>
        <label htmlFor="channel" className="block text-sm font-medium mb-1">
          Channel
        </label>
        <Select
          value={watch('channel')}
          onValueChange={(v) => setValue('channel', v as NotificationConfig['channel'])}
        >
          <SelectTrigger id="channel">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="email">Email</SelectItem>
            <SelectItem value="slack">Slack</SelectItem>
            <SelectItem value="webhook">Webhook</SelectItem>
            <SelectItem value="pagerduty">PagerDuty</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground mt-1">
          The delivery channel. Configure channels in Settings &gt; Notifications.
        </p>
      </div>

      <div>
        <label htmlFor="target" className="block text-sm font-medium mb-1">
          Target
        </label>
        <Input id="target" {...register('target')} />
        <p className="text-xs text-muted-foreground mt-1">
          Email address, Slack channel, webhook URL, or PagerDuty service key.
        </p>
      </div>

      <div>
        <label htmlFor="message_template" className="block text-sm font-medium mb-1">
          Message Template
        </label>
        <textarea
          id="message_template"
          className="w-full min-h-[80px] rounded-md border border-input bg-background px-3 py-2 text-sm"
          {...register('message_template')}
        />
        <p className="text-xs text-muted-foreground mt-1">
          Optional template with variables like {'{{workflow_name}}'}, {'{{status}}'},{' '}
          {'{{node_count}}'}.
        </p>
      </div>

      <Button type="submit">Save</Button>
    </form>
  );
}
