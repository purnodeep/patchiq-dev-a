import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@patchiq/ui';
import { usePatchDeploy, type PatchDeployPayload } from '../../api/hooks/usePatchDeploy';

const schema = z.object({
  name: z.string().min(1, 'Name is required'),
  description: z.string().optional(),
  config_type: z.enum(['install', 'rollback']),
  endpoint_filter: z.enum(['all', 'windows', 'linux', 'critical']),
  scheduled_date: z.string().optional(),
  scheduled_time: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

interface DeploymentModalProps {
  open: boolean;
  patchId: string;
  patchName: string;
  patchVersion?: string;
  patchSeverity?: string;
  onClose: () => void;
  onSuccess?: () => void;
}

export const DeploymentModal = ({
  open,
  patchId,
  patchName,
  patchVersion,
  patchSeverity,
  onClose,
  onSuccess,
}: DeploymentModalProps) => {
  const { mutate, isPending } = usePatchDeploy();

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: patchName ? `${patchName} - Deployment` : '',
      config_type: 'install',
      endpoint_filter: 'all',
    },
  });

  useEffect(() => {
    if (open) {
      reset({
        name: patchName ? `${patchName} - Deployment` : '',
        config_type: 'install',
        endpoint_filter: 'all',
        description: '',
        scheduled_date: '',
        scheduled_time: '',
      });
    }
  }, [open, patchName, reset]);

  const onSubmit = (values: FormValues) => {
    const scheduled_at =
      values.scheduled_date && values.scheduled_time
        ? new Date(`${values.scheduled_date}T${values.scheduled_time}`).toISOString()
        : undefined;

    mutate(
      {
        patchId,
        name: values.name,
        description: values.description,
        config_type: values.config_type,
        scope: values.endpoint_filter,
        target_endpoints: values.endpoint_filter,
        scheduled_at,
      } as PatchDeployPayload,
      {
        onSuccess: () => {
          onSuccess?.();
          onClose();
        },
      },
    );
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) onClose();
      }}
    >
      <DialogContent className="max-w-[680px] max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create Patch Deployment</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4 py-2">
          {/* Deployment Name */}
          <div>
            <label htmlFor="dep-name" className="mb-1.5 block text-xs font-semibold">
              Deployment Name <span className="text-destructive">*</span>
            </label>
            <input
              id="dep-name"
              {...register('name')}
              className="w-full rounded-md border border-border bg-muted/30 px-3 py-2 text-xs text-foreground outline-none focus:border-primary"
              placeholder={`e.g., ${patchName} - Critical Patch`}
              aria-label="Deployment Name"
            />
            {errors.name && (
              <p className="mt-1 text-[11px] text-destructive">{errors.name.message}</p>
            )}
          </div>

          {/* Description */}
          <div>
            <label className="mb-1.5 block text-xs font-semibold">Description</label>
            <textarea
              {...register('description')}
              className="w-full resize-y rounded-md border border-border bg-muted/30 px-3 py-2 text-xs text-foreground outline-none focus:border-primary min-h-[70px]"
              placeholder="Optional: deployment notes, approval info..."
            />
          </div>

          {/* Configuration Type */}
          <div>
            <p className="mb-1.5 text-xs font-semibold">
              Configuration Type <span className="text-destructive">*</span>
            </p>
            <div className="flex gap-4">
              {(['install', 'rollback'] as const).map((t) => (
                <label key={t} className="flex cursor-pointer items-center gap-2">
                  <input type="radio" value={t} {...register('config_type')} />
                  <span className="text-xs capitalize">{t}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Target Endpoints */}
          <div>
            <p className="mb-1.5 text-xs font-semibold">
              Target Endpoints <span className="text-destructive">*</span>
            </p>
            <Select
              value={watch('endpoint_filter')}
              onValueChange={(v) => setValue('endpoint_filter', v as FormValues['endpoint_filter'])}
            >
              <SelectTrigger className="h-9 w-full text-xs">
                <SelectValue placeholder="Select endpoints" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Endpoints</SelectItem>
                <SelectItem value="windows">Windows Only</SelectItem>
                <SelectItem value="linux">Linux Only</SelectItem>
                <SelectItem value="critical">Critical Endpoints</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Patches to Deploy (read-only) */}
          {patchName && (
            <div>
              <p className="mb-1.5 text-xs font-semibold">Patches to Deploy</p>
              <div className="overflow-hidden rounded-md border border-border">
                <table className="w-full text-xs">
                  <thead className="border-b bg-muted/50">
                    <tr>
                      <th className="px-3 py-2 text-left font-medium text-muted-foreground">ID</th>
                      <th className="px-3 py-2 text-left font-medium text-muted-foreground">
                        Version
                      </th>
                      <th className="px-3 py-2 text-left font-medium text-muted-foreground">
                        Severity
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr>
                      <td className="px-3 py-2 font-mono font-semibold text-primary">
                        {patchName}
                      </td>
                      <td className="px-3 py-2 text-muted-foreground">{patchVersion ?? '—'}</td>
                      <td className="px-3 py-2 capitalize text-muted-foreground">
                        {patchSeverity ?? '—'}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Schedule (optional) */}
          <div>
            <p className="mb-1.5 text-xs font-semibold">Schedule Deployment (Optional)</p>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="mb-1 block text-[10px] text-muted-foreground">Start Date</label>
                <input
                  type="date"
                  {...register('scheduled_date')}
                  className="w-full rounded-md border border-border bg-muted/30 px-2 py-1.5 text-xs text-foreground outline-none focus:border-primary"
                />
              </div>
              <div>
                <label className="mb-1 block text-[10px] text-muted-foreground">Start Time</label>
                <input
                  type="time"
                  {...register('scheduled_time')}
                  className="w-full rounded-md border border-border bg-muted/30 px-2 py-1.5 text-xs text-foreground outline-none focus:border-primary"
                />
              </div>
            </div>
          </div>

          <DialogFooter className="pt-2">
            <button
              type="button"
              onClick={onClose}
              className="inline-flex items-center rounded-md border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-muted"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={onClose}
              className="inline-flex items-center rounded-md border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted"
            >
              Save as Draft
            </button>
            <button
              type="submit"
              disabled={isPending}
              className="inline-flex items-center rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-white hover:bg-primary/90 disabled:opacity-50"
            >
              {isPending ? 'Publishing...' : 'Publish'}
            </button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
