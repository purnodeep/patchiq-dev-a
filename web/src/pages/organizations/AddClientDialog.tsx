import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { extractApiError } from '../../api/errors';
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
} from '@patchiq/ui';
import { toast } from 'sonner';
import { useProvisionTenant } from '../../api/hooks/useOrganizations';

const schema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters').max(255),
  slug: z
    .string()
    .min(2, 'Slug must be at least 2 characters')
    .max(63)
    .regex(/^[a-z0-9]+(-[a-z0-9]+)*$/, 'Slug must be lowercase kebab-case (e.g. acme-corp)'),
});

type FormValues = z.infer<typeof schema>;

interface Props {
  orgId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// AddClientDialog provisions a new child tenant under the active organization.
// It is opened from ClientsPage.
export function AddClientDialog({ orgId, open, onOpenChange }: Props) {
  const provision = useProvisionTenant();
  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: '', slug: '' },
  });

  useEffect(() => {
    if (!open) {
      reset({ name: '', slug: '' });
      provision.reset();
    }
    // Intentionally omitting `provision` from deps: TanStack mutation objects
    // are reference-stable per render but ESLint cannot prove that, and
    // including it would loop on every reset.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, reset]);

  const onSubmit = handleSubmit(async (values) => {
    try {
      await provision.mutateAsync({
        orgId,
        body: { name: values.name, slug: values.slug },
      });
      toast.success(`Client "${values.name}" provisioned`);
      onOpenChange(false);
    } catch (err) {
      const apiError = extractApiError(err);
      // Slug conflicts are field-level — surface inline so the user can edit
      // without re-reading a toast. Everything else goes to a toast banner.
      if (apiError.code === 'SLUG_TAKEN') {
        setError('slug', { type: 'server', message: apiError.message });
        return;
      }
      toast.error(apiError.message);
    }
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Add client</DialogTitle>
          <DialogDescription>
            Provision a new client tenant inside this organization.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={onSubmit} className="space-y-4">
          <div className="space-y-1">
            <label htmlFor="client-name" className="text-sm font-medium">
              Name
            </label>
            <Input id="client-name" placeholder="Acme Corporation" {...register('name')} />
            {errors.name && (
              <p role="alert" className="text-xs text-destructive">
                {errors.name.message}
              </p>
            )}
          </div>

          <div className="space-y-1">
            <label htmlFor="client-slug" className="text-sm font-medium">
              Slug
            </label>
            <Input id="client-slug" placeholder="acme-corp" {...register('slug')} />
            {errors.slug && (
              <p role="alert" className="text-xs text-destructive">
                {errors.slug.message}
              </p>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting || provision.isPending}>
              {provision.isPending ? 'Creating…' : 'Create client'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
