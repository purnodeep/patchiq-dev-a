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
import { useTag, useUpdateTag, useDeleteTag } from '../../api/hooks/useTags';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tagId: string;
}

export function EditTagDialog({ open, onOpenChange, tagId }: Props) {
  const can = useCan();
  const { data: tag } = useTag(tagId);
  const [tagKey, setTagKey] = useState('');
  const [tagValue, setTagValue] = useState('');
  const [description, setDescription] = useState('');
  const [confirmDelete, setConfirmDelete] = useState(false);
  const updateTag = useUpdateTag();
  const deleteTag = useDeleteTag();

  useEffect(() => {
    if (tag) {
      setTagKey(tag.key);
      setTagValue(tag.value);
      setDescription(tag.description ?? '');
    }
  }, [tag]);

  const handleSubmit = () => {
    updateTag.mutate(
      { id: tagId, body: { description: description || undefined } },
      { onSuccess: () => onOpenChange(false) },
    );
  };

  const handleDelete = () => {
    deleteTag.mutate(tagId, { onSuccess: () => onOpenChange(false) });
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        onOpenChange(o);
        if (!o) {
          setConfirmDelete(false);
          updateTag.reset();
          deleteTag.reset();
        }
      }}
    >
      <DialogContent className="max-w-lg max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle style={{ fontFamily: 'var(--font-display)' }}>
            Edit Tag: {tag ? `${tag.key}:${tag.value}` : ''}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 overflow-y-auto flex-1">
          <div className="space-y-1">
            <label className="text-sm font-medium">Key</label>
            <Input
              value={tagKey}
              onChange={(e) => setTagKey(e.target.value)}
              maxLength={100}
              readOnly
            />
          </div>

          <div className="space-y-1">
            <label className="text-sm font-medium">Value</label>
            <Input
              value={tagValue}
              onChange={(e) => setTagValue(e.target.value)}
              maxLength={100}
              readOnly
            />
          </div>

          <div className="space-y-1">
            <label className="text-sm font-medium">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="flex min-h-[60px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              rows={2}
            />
          </div>

          {(updateTag.isError || deleteTag.isError) && (
            <p className="text-sm text-destructive">
              {((updateTag.error ?? deleteTag.error) as Error | null)?.message ??
                'Operation failed'}
            </p>
          )}
        </div>

        <DialogFooter className="flex justify-between">
          <div>
            {!confirmDelete ? (
              <Button
                variant="destructive"
                size="sm"
                disabled={!can('endpoints', 'delete')}
                title={!can('endpoints', 'delete') ? "You don't have permission" : undefined}
                onClick={() => setConfirmDelete(true)}
              >
                Delete Tag
              </Button>
            ) : (
              <div className="flex items-center gap-2">
                <span className="text-sm text-destructive">
                  Remove from all {tag?.endpoint_count ?? 0} endpoints?
                </span>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={handleDelete}
                  disabled={deleteTag.isPending}
                >
                  {deleteTag.isPending ? 'Deleting...' : 'Confirm'}
                </Button>
                <Button variant="outline" size="sm" onClick={() => setConfirmDelete(false)}>
                  Cancel
                </Button>
              </div>
            )}
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={!can('endpoints', 'update') || !tagKey.trim() || updateTag.isPending}
              title={!can('endpoints', 'update') ? "You don't have permission" : undefined}
            >
              {updateTag.isPending ? 'Saving...' : 'Save'}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
