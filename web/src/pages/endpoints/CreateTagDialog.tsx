import { useState, useEffect } from 'react';
import {
  Button,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  Input,
} from '@patchiq/ui';
import { useCan } from '../../app/auth/AuthContext';
import { useCreateTag, useAssignTag } from '../../api/hooks/useTags';
import { useEndpoints } from '../../api/hooks/useEndpoints';
import { StatusBadge } from '../../components/StatusBadge';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  preSelectedEndpointId?: string;
}

export function CreateTagDialog({ open, onOpenChange, preSelectedEndpointId }: Props) {
  const can = useCan();
  const [tagKey, setTagKey] = useState('');
  const [tagValue, setTagValue] = useState('');
  const [description, setDescription] = useState('');
  const [selectedIds, setSelectedIds] = useState<Set<string>>(
    preSelectedEndpointId ? new Set([preSelectedEndpointId]) : new Set(),
  );
  const [epSearch, setEpSearch] = useState('');
  const [assignError, setAssignError] = useState<string | null>(null);
  const createTag = useCreateTag();
  const assignTag = useAssignTag();
  const { data: epData } = useEndpoints({ limit: 200, search: epSearch || undefined });
  const endpoints = epData?.data ?? [];

  const resetForm = () => {
    setTagKey('');
    setTagValue('');
    setDescription('');
    setSelectedIds(preSelectedEndpointId ? new Set([preSelectedEndpointId]) : new Set());
    setEpSearch('');
    setAssignError(null);
    createTag.reset();
  };

  const handleSubmit = async () => {
    setAssignError(null);
    let newTag: { id: string } | undefined;
    try {
      newTag = await createTag.mutateAsync({
        key: tagKey,
        value: tagValue,
        description: description || undefined,
      });
    } catch (err) {
      // createTag.isError will be set by TanStack Query; log for debugging
      console.error('Failed to create tag:', err);
      return;
    }

    if (selectedIds.size > 0 && newTag?.id) {
      try {
        await assignTag.mutateAsync({
          tagId: newTag.id,
          endpointIds: Array.from(selectedIds),
        });
      } catch (err) {
        // Tag created but endpoint assignment failed
        const { toast } = await import('sonner');
        toast.warning(
          `Tag created but failed to assign endpoints: ${err instanceof Error ? err.message : 'Unknown error'}`,
        );
      }
    }

    onOpenChange(false);
    resetForm();
  };

  useEffect(() => {
    if (open && preSelectedEndpointId) {
      setSelectedIds(new Set([preSelectedEndpointId]));
    }
  }, [open, preSelectedEndpointId]);

  const toggleEndpoint = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        onOpenChange(o);
        if (!o) resetForm();
      }}
    >
      <DialogContent className="max-w-lg max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle style={{ fontFamily: 'var(--font-display)' }}>Create Tag</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 overflow-y-auto flex-1">
          <div className="space-y-1">
            <label className="text-sm font-medium">Key</label>
            <Input
              value={tagKey}
              onChange={(e) => setTagKey(e.target.value)}
              placeholder="e.g., env"
              maxLength={100}
            />
          </div>

          <div className="space-y-1">
            <label className="text-sm font-medium">Value</label>
            <Input
              value={tagValue}
              onChange={(e) => setTagValue(e.target.value)}
              placeholder="e.g., production"
              maxLength={100}
            />
          </div>

          <div className="space-y-1">
            <label className="text-sm font-medium">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional description..."
              className="flex min-h-[60px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              rows={2}
            />
          </div>

          <div className="space-y-1">
            <label className="text-sm font-medium">Endpoints ({selectedIds.size} selected)</label>
            <Input
              value={epSearch}
              onChange={(e) => setEpSearch(e.target.value)}
              placeholder="Search endpoints..."
              className="mb-2"
            />
            <div className="max-h-48 overflow-y-auto rounded-md border border-border">
              {endpoints.map((ep) => (
                <label
                  key={ep.id}
                  className="flex items-center gap-2 px-3 py-2 hover:bg-muted/50 cursor-pointer border-b border-border last:border-b-0"
                >
                  <input
                    type="checkbox"
                    checked={selectedIds.has(ep.id)}
                    onChange={() => toggleEndpoint(ep.id)}
                    className="h-4 w-4"
                  />
                  <span className="font-mono text-sm">{ep.hostname}</span>
                  <StatusBadge status={ep.status} />
                </label>
              ))}
              {endpoints.length === 0 && (
                <div className="px-3 py-4 text-center text-sm text-muted-foreground">
                  No endpoints found.
                </div>
              )}
            </div>
          </div>

          {createTag.isError && (
            <p className="text-sm text-destructive">
              {createTag.error instanceof Error ? createTag.error.message : 'Failed to create tag'}
            </p>
          )}
          {assignError && (
            <p className="text-sm text-amber-600">
              Warning: Tag created but endpoint assignment failed — {assignError}
            </p>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={() => void handleSubmit()}
            disabled={
              !can('endpoints', 'create') ||
              !tagKey.trim() ||
              createTag.isPending ||
              assignTag.isPending
            }
            title={!can('endpoints', 'create') ? "You don't have permission" : undefined}
          >
            {createTag.isPending
              ? 'Creating...'
              : assignTag.isPending
                ? 'Assigning...'
                : 'Create Tag'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
