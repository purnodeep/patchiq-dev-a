import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { toast } from 'sonner';
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
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@patchiq/ui';
import { useCreateCatalogEntry } from '../../api/hooks/useCatalog';

// Matches the actual backend catalogRequest struct (types.ts is stale)
interface CreateCatalogEntryRequest {
  name: string;
  vendor: string;
  os_family: string;
  version: string;
  severity: string;
  description?: string;
  installer_type?: string;
  cve_ids?: string[];
}

const catalogSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  vendor: z.string().min(1, 'Vendor is required'),
  os_family: z.string().min(1, 'OS family is required'),
  version: z.string().min(1, 'Version is required'),
  severity: z.enum(['critical', 'high', 'medium', 'low', 'none']),
  description: z.string().optional(),
  installer_type: z.string().optional(),
  cve_ids: z.string().optional(),
});

type CatalogFormData = z.infer<typeof catalogSchema>;

interface CatalogFormProps {
  open: boolean;
  onSuccess: () => void;
  onCancel: () => void;
}

export const CatalogForm = ({ open, onSuccess, onCancel }: CatalogFormProps) => {
  const createMutation = useCreateCatalogEntry();

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors },
  } = useForm<CatalogFormData>({
    resolver: zodResolver(catalogSchema),
    defaultValues: {
      name: '',
      vendor: '',
      os_family: '',
      version: '',
      severity: 'medium',
      description: '',
      installer_type: '',
      cve_ids: '',
    },
  });

  const osFamily = watch('os_family');
  const severityValue = watch('severity');

  const onSubmit = (data: CatalogFormData) => {
    const cveIdList = data.cve_ids
      ? data.cve_ids
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean)
      : [];

    const payload: CreateCatalogEntryRequest = {
      name: data.name,
      vendor: data.vendor,
      os_family: data.os_family,
      version: data.version,
      severity: data.severity,
      description: data.description || undefined,
      installer_type: data.installer_type || undefined,
      cve_ids: cveIdList.length > 0 ? cveIdList : undefined,
    };

    createMutation.mutate(payload as Parameters<typeof createMutation.mutate>[0], {
      onSuccess: () => {
        toast.success('Catalog entry created');
        reset();
        onSuccess();
      },
      onError: (err: Error) => {
        toast.error(`Failed to create entry: ${err.message}`);
      },
    });
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(v: boolean) => {
        if (!v) {
          reset();
          onCancel();
        }
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add Catalog Entry</DialogTitle>
          <DialogDescription>
            Manually add a patch or vulnerability entry to the catalog.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={(e) => void handleSubmit(onSubmit)(e)} className="space-y-4">
          <div>
            <label className="text-sm font-medium">Name</label>
            <Input {...register('name')} placeholder="e.g. KB5001234" />
            {errors.name && <p className="text-sm text-destructive mt-1">{errors.name.message}</p>}
          </div>

          <div>
            <label className="text-sm font-medium">Vendor</label>
            <Input {...register('vendor')} placeholder="e.g. Microsoft, Canonical" />
            {errors.vendor && (
              <p className="text-sm text-destructive mt-1">{errors.vendor.message}</p>
            )}
          </div>

          <div>
            <label className="text-sm font-medium">OS Family</label>
            <Select value={osFamily ?? ''} onValueChange={(v: string) => setValue('os_family', v)}>
              <SelectTrigger>
                <SelectValue placeholder="Select OS family" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="windows">Windows</SelectItem>
                <SelectItem value="linux">Linux</SelectItem>
                <SelectItem value="macos">macOS</SelectItem>
                <SelectItem value="ubuntu">Ubuntu</SelectItem>
                <SelectItem value="debian">Debian</SelectItem>
                <SelectItem value="rhel">RHEL</SelectItem>
              </SelectContent>
            </Select>
            {errors.os_family && (
              <p className="text-sm text-destructive mt-1">{errors.os_family.message}</p>
            )}
          </div>

          <div>
            <label className="text-sm font-medium">Version</label>
            <Input {...register('version')} placeholder="e.g. 1.0.0" />
            {errors.version && (
              <p className="text-sm text-destructive mt-1">{errors.version.message}</p>
            )}
          </div>

          <div>
            <label className="text-sm font-medium">Severity</label>
            <Select
              value={severityValue ?? ''}
              onValueChange={(v: string) => setValue('severity', v as CatalogFormData['severity'])}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select severity" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="critical">Critical</SelectItem>
                <SelectItem value="high">High</SelectItem>
                <SelectItem value="medium">Medium</SelectItem>
                <SelectItem value="low">Low</SelectItem>
                <SelectItem value="none">None</SelectItem>
              </SelectContent>
            </Select>
            {errors.severity && (
              <p className="text-sm text-destructive mt-1">{errors.severity.message}</p>
            )}
          </div>

          <div>
            <label className="text-sm font-medium">Installer Type</label>
            <Input {...register('installer_type')} placeholder="e.g. msi, deb, rpm" />
          </div>

          <div>
            <label className="text-sm font-medium">CVE IDs</label>
            <Input {...register('cve_ids')} placeholder="e.g. CVE-2024-1234, CVE-2024-5678" />
            <p className="text-xs text-muted-foreground mt-1">Comma-separated list of CVE IDs</p>
          </div>

          <div>
            <label className="text-sm font-medium">Description</label>
            <textarea
              {...register('description')}
              className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              placeholder="Optional description"
            />
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                reset();
                onCancel();
              }}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? 'Creating...' : 'Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
