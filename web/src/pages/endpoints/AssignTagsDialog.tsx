import { useState } from 'react';
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@patchiq/ui';
import { useCan } from '../../app/auth/AuthContext';
import { useTags, useAssignTag } from '../../api/hooks/useTags';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  selectedEndpointIds: string[];
}

export function AssignTagsDialog({ open, onOpenChange, selectedEndpointIds }: Props) {
  const can = useCan();
  const { data: tagsData } = useTags({ limit: 100 });
  const tags = tagsData ?? [];
  const [selectedTagIds, setSelectedTagIds] = useState<Set<string>>(new Set());
  const assignTag = useAssignTag();
  const [assigning, setAssigning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const toggleTag = (id: string) => {
    setSelectedTagIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleAssign = async () => {
    if (selectedEndpointIds.length === 0) return;
    setAssigning(true);
    setError(null);
    try {
      for (const tagId of selectedTagIds) {
        await assignTag.mutateAsync({ tagId, endpointIds: selectedEndpointIds });
      }
      setSelectedTagIds(new Set());
      onOpenChange(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to assign tags');
    } finally {
      setAssigning(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle style={{ fontFamily: 'var(--font-display)' }}>
            Assign Tags to {selectedEndpointIds.length} Endpoint
            {selectedEndpointIds.length > 1 ? 's' : ''}
          </DialogTitle>
          <DialogDescription>Select tags to assign to the selected endpoints.</DialogDescription>
        </DialogHeader>

        <div className="max-h-64 overflow-y-auto rounded-md border border-border">
          {tags.map((tag) => (
            <label
              key={tag.id}
              className="flex items-center gap-2 px-3 py-2 hover:bg-muted/50 cursor-pointer border-b border-border last:border-b-0"
            >
              <input
                type="checkbox"
                checked={selectedTagIds.has(tag.id)}
                onChange={() => toggleTag(tag.id)}
                className="h-4 w-4"
              />
              <span className="text-sm font-medium">
                {tag.key}:{tag.value}
              </span>
              {tag.endpoint_count != null && (
                <span className="text-xs text-muted-foreground">
                  ({tag.endpoint_count} endpoints)
                </span>
              )}
            </label>
          ))}
          {tags.length === 0 && (
            <div className="px-3 py-4 text-center text-sm text-muted-foreground">
              No tags exist yet. Create one first.
            </div>
          )}
        </div>

        {error && <p className="text-sm text-destructive">{error}</p>}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleAssign}
            disabled={
              !can('endpoints', 'create') ||
              selectedTagIds.size === 0 ||
              assigning ||
              selectedEndpointIds.length === 0
            }
            title={!can('endpoints', 'create') ? "You don't have permission" : undefined}
          >
            {assigning ? 'Assigning...' : 'Assign'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
