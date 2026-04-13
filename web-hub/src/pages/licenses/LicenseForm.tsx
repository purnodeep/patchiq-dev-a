import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Button,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@patchiq/ui';
import { toast } from 'sonner';
import { useCreateLicense } from '../../api/hooks/useLicenses';

const licenseSchema = z.object({
  customer_name: z.string().min(1, 'Customer name is required'),
  customer_email: z.string().email('Invalid email').optional().or(z.literal('')),
  tier: z.string().min(1, 'Tier is required'),
  max_endpoints: z.number().min(1, 'Must be at least 1'),
  expires_at: z.string().min(1, 'Expiration date is required'),
  notes: z.string().optional(),
});

type LicenseFormData = z.infer<typeof licenseSchema>;

interface LicenseFormProps {
  open: boolean;
  onSuccess: () => void;
  onCancel: () => void;
}

export const LicenseForm = ({ open, onSuccess, onCancel }: LicenseFormProps) => {
  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors },
  } = useForm<LicenseFormData>({
    resolver: zodResolver(licenseSchema),
    defaultValues: {
      customer_name: '',
      customer_email: '',
      tier: '',
      max_endpoints: 10,
      expires_at: '',
      notes: '',
    },
  });

  useEffect(() => {
    if (open) {
      reset({
        customer_name: '',
        customer_email: '',
        tier: '',
        max_endpoints: 10,
        expires_at: '',
        notes: '',
      });
    }
  }, [open, reset]);

  const createMutation = useCreateLicense();

  const onSubmit = (data: LicenseFormData) => {
    createMutation.mutate(
      {
        customer_name: data.customer_name,
        customer_email: data.customer_email || undefined,
        tier: data.tier,
        max_endpoints: data.max_endpoints,
        expires_at: new Date(data.expires_at).toISOString(),
        notes: data.notes || undefined,
      },
      {
        onSuccess: () => {
          toast.success('License issued successfully');
          onSuccess();
        },
        onError: (err) => {
          toast.error(err instanceof Error ? err.message : 'Failed to issue license');
        },
      },
    );
  };

  const tierValue = watch('tier');

  return (
    <Dialog
      open={open}
      onOpenChange={(v: boolean) => {
        if (!v) onCancel();
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Issue License</DialogTitle>
        </DialogHeader>
        <form onSubmit={(e) => void handleSubmit(onSubmit)(e)} className="space-y-4">
          <div>
            <label className="text-sm font-medium">Customer Name</label>
            <Input {...register('customer_name')} placeholder="Acme Corp" />
            {errors.customer_name && (
              <p className="text-sm text-destructive mt-1">{errors.customer_name.message}</p>
            )}
          </div>

          <div>
            <label className="text-sm font-medium">Customer Email</label>
            <Input {...register('customer_email')} placeholder="admin@acme.com" type="email" />
            {errors.customer_email && (
              <p className="text-sm text-destructive mt-1">{errors.customer_email.message}</p>
            )}
          </div>

          <div>
            <label className="text-sm font-medium">Tier</label>
            <Select value={tierValue} onValueChange={(v: string) => setValue('tier', v)}>
              <SelectTrigger>
                <SelectValue placeholder="Select tier" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="community">Community</SelectItem>
                <SelectItem value="professional">Professional</SelectItem>
                <SelectItem value="enterprise">Enterprise</SelectItem>
                <SelectItem value="msp">MSP</SelectItem>
              </SelectContent>
            </Select>
            {errors.tier && <p className="text-sm text-destructive mt-1">{errors.tier.message}</p>}
          </div>

          <div>
            <label className="text-sm font-medium">Max Endpoints</label>
            <Input {...register('max_endpoints', { valueAsNumber: true })} type="number" min={1} />
            {errors.max_endpoints && (
              <p className="text-sm text-destructive mt-1">{errors.max_endpoints.message}</p>
            )}
          </div>

          <div>
            <label className="text-sm font-medium">Expires At</label>
            <Input {...register('expires_at')} type="date" />
            {errors.expires_at && (
              <p className="text-sm text-destructive mt-1">{errors.expires_at.message}</p>
            )}
          </div>

          <div>
            <label className="text-sm font-medium">Notes</label>
            <textarea
              {...register('notes')}
              className="flex min-h-[60px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              placeholder="Optional notes"
            />
          </div>

          {createMutation.error && (
            <p className="text-sm text-destructive">
              {createMutation.error instanceof Error
                ? createMutation.error.message
                : 'Failed to create license'}
            </p>
          )}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? 'Creating...' : 'Issue License'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
