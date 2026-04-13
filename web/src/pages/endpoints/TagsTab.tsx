import { useState } from 'react';
import { Badge, Skeleton } from '@patchiq/ui';
import { ChevronDown, ChevronRight, Pencil, Trash2 } from 'lucide-react';
import { useCan } from '../../app/auth/AuthContext';
import { useTags, useTag, useDeleteTag } from '../../api/hooks/useTags';
import { StatusBadge } from '../../components/StatusBadge';
import { FilterBar, FilterSearch } from '../../components/FilterBar';
import { EditTagDialog } from './EditTagDialog';
import { timeAgo } from '../../lib/time';

export function TagsTab() {
  const [search, setSearch] = useState('');
  const { data, isLoading } = useTags({ limit: 100, search: search || undefined });
  const tags = data ?? [];
  const [expandedTag, setExpandedTag] = useState<string | null>(null);
  const [editTagId, setEditTagId] = useState<string | null>(null);
  const deleteTag = useDeleteTag();

  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-12 rounded-md" />
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <FilterBar>
        <FilterSearch value={search} onChange={setSearch} placeholder="Search tags..." />
      </FilterBar>

      <div className="rounded-lg border border-border">
        <table className="w-full text-sm">
          <thead className="border-b bg-muted/50">
            <tr>
              <th className="h-10 px-4 text-left font-medium text-muted-foreground w-8" />
              <th className="h-10 px-4 text-left font-medium text-muted-foreground">Tag Name</th>
              <th className="h-10 px-4 text-left font-medium text-muted-foreground">Description</th>
              <th className="h-10 px-4 text-left font-medium text-muted-foreground">Endpoints</th>
              <th className="h-10 px-4 text-left font-medium text-muted-foreground">Created</th>
              <th className="h-10 px-4 text-left font-medium text-muted-foreground">Actions</th>
            </tr>
          </thead>
          <tbody>
            {tags.map((tag) => {
              const isExpanded = expandedTag === tag.id;
              return (
                <TagRow
                  key={tag.id}
                  tag={tag}
                  isExpanded={isExpanded}
                  onToggle={() => setExpandedTag(isExpanded ? null : tag.id)}
                  onEdit={() => setEditTagId(tag.id)}
                  onDelete={() => deleteTag.mutate(tag.id)}
                />
              );
            })}
            {tags.length === 0 && (
              <tr>
                <td colSpan={6} className="h-24 text-center text-muted-foreground">
                  No tags found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {editTagId && (
        <EditTagDialog
          open={!!editTagId}
          onOpenChange={(o) => {
            if (!o) setEditTagId(null);
          }}
          tagId={editTagId}
        />
      )}
    </div>
  );
}

function TagRow({
  tag,
  isExpanded,
  onToggle,
  onEdit,
  onDelete,
}: {
  tag: {
    id: string;
    key: string;
    value: string;
    description: string | null;
    endpoint_count?: number;
    created_at: string;
  };
  isExpanded: boolean;
  onToggle: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const can = useCan();
  return (
    <>
      <tr className="border-b hover:bg-muted/50 transition-colors">
        <td className="px-4 py-3">
          <button onClick={onToggle} className="text-muted-foreground hover:text-foreground">
            {isExpanded ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
          </button>
        </td>
        <td className="px-4 py-3 font-medium">
          {tag.key}:{tag.value}
        </td>
        <td className="px-4 py-3 text-muted-foreground max-w-[200px] truncate">
          {tag.description || '—'}
        </td>
        <td className="px-4 py-3">
          <Badge variant="outline">{tag.endpoint_count ?? 0}</Badge>
        </td>
        <td className="px-4 py-3 text-muted-foreground">{timeAgo(tag.created_at)}</td>
        <td className="px-4 py-3">
          <div className="flex items-center gap-1">
            <button
              onClick={onEdit}
              disabled={!can('endpoints', 'update')}
              title={!can('endpoints', 'update') ? "You don't have permission" : undefined}
              className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
              style={{
                opacity: !can('endpoints', 'update') ? 0.5 : 1,
                cursor: !can('endpoints', 'update') ? 'not-allowed' : 'pointer',
              }}
            >
              <Pencil className="h-3.5 w-3.5" />
            </button>
            <button
              onClick={onDelete}
              disabled={!can('endpoints', 'delete')}
              title={!can('endpoints', 'delete') ? "You don't have permission" : undefined}
              className="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive"
              style={{
                opacity: !can('endpoints', 'delete') ? 0.5 : 1,
                cursor: !can('endpoints', 'delete') ? 'not-allowed' : 'pointer',
              }}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        </td>
      </tr>
      {isExpanded && <TagMembersRow tagId={tag.id} />}
    </>
  );
}

function TagMembersRow({ tagId }: { tagId: string }) {
  const { data: tag, isLoading } = useTag(tagId);

  if (isLoading) {
    return (
      <tr>
        <td colSpan={6} className="px-8 py-3">
          <Skeleton className="h-8 w-full" />
        </td>
      </tr>
    );
  }

  const members = tag?.members ?? [];

  return (
    <tr className="bg-muted/20">
      <td colSpan={6} className="px-8 py-3">
        {members.length === 0 ? (
          <span className="text-sm text-muted-foreground">No endpoints assigned.</span>
        ) : (
          <div className="flex flex-wrap gap-2">
            {members.map((m) => (
              <span
                key={m.id}
                className="inline-flex items-center gap-1.5 rounded-full bg-background border border-border px-2.5 py-1 text-xs"
              >
                <span className="font-mono">{m.hostname}</span>
                <StatusBadge status={m.status as Parameters<typeof StatusBadge>[0]['status']} />
              </span>
            ))}
          </div>
        )}
      </td>
    </tr>
  );
}
