import { memo, useCallback, useMemo, useState } from 'react';
import { Link } from 'react-router';
import { useCan } from '../../app/auth/AuthContext';
import {
  getCoreRowModel,
  getExpandedRowModel,
  useReactTable,
  type ExpandedState,
  type ColumnDef,
} from '@tanstack/react-table';
import { Plus, Pencil, Trash2, X, Tag as TagIcon, MoreHorizontal } from 'lucide-react';
import { EmptyState, ErrorState, SkeletonCard } from '@patchiq/ui';
import { toast } from 'sonner';
import { useTags, useCreateTag, useDeleteTag, type Tag } from '../../api/hooks/useTags';
import { EditTagDialog } from '../endpoints/EditTagDialog';
import { DataTable } from '../../components/data-table/DataTable';
import { DataTablePagination } from '../../components/data-table/DataTablePagination';
import { FilterBar, FilterPill, FilterSeparator, FilterSearch } from '../../components/FilterBar';

// ─── Helpers ──────────────────────────────────────────────────────────────────

function relativeTime(dateStr: string | undefined | null): string {
  if (!dateStr) return '—';
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diff = Math.floor((now - then) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 86400 * 30) return `${Math.floor(diff / 86400)}d ago`;
  if (diff < 86400 * 365) return `${Math.floor(diff / (86400 * 30))}mo ago`;
  return `${Math.floor(diff / (86400 * 365))}y ago`;
}

function tagLabel(tag: Tag): string {
  return tag.value ? `${tag.key}:${tag.value}` : tag.key;
}

// ─── Inline Stat Card ─────────────────────────────────────────────────────────

interface StatCardButtonProps {
  label: string;
  value: number | undefined;
  valueColor?: string;
  active?: boolean;
  onClick: () => void;
}

function StatCardButton({ label, value, valueColor, active, onClick }: StatCardButtonProps) {
  const [hovered, setHovered] = useState(false);
  return (
    <button
      type="button"
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        flex: 1,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        padding: '12px 14px',
        background: active ? 'var(--bg-inset)' : 'var(--bg-card)',
        border: `1px solid ${active ? (valueColor ?? 'var(--accent)') : hovered ? 'var(--border-hover)' : 'var(--border)'}`,
        borderRadius: 8,
        cursor: 'pointer',
        transition: 'all 0.15s',
        outline: 'none',
        textAlign: 'left',
      }}
    >
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 22,
          fontWeight: 700,
          lineHeight: 1,
          color: valueColor ?? 'var(--text-emphasis)',
          letterSpacing: '-0.02em',
        }}
      >
        {value ?? '—'}
      </span>
      <span
        style={{
          fontSize: 10,
          fontFamily: 'var(--font-mono)',
          fontWeight: 500,
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
          color: active ? (valueColor ?? 'var(--accent)') : 'var(--text-muted)',
          marginTop: 4,
        }}
      >
        {label}
      </span>
    </button>
  );
}

// ─── Tag Pill ─────────────────────────────────────────────────────────────────

function TagPill({ tag }: { tag: Tag }) {
  const isBoolean = !tag.value;
  return (
    <span
      style={{
        fontSize: 12,
        fontFamily: 'var(--font-mono)',
        color: isBoolean ? 'var(--text-muted)' : 'var(--text-secondary)',
        background: 'var(--bg-card-hover)',
        border: `1px solid ${isBoolean ? 'var(--border)' : 'var(--border-strong)'}`,
        borderRadius: 4,
        padding: '3px 8px',
        whiteSpace: 'nowrap',
      }}
    >
      {isBoolean ? tag.key : `${tag.key}:${tag.value}`}
    </span>
  );
}

// ─── Type Badge ───────────────────────────────────────────────────────────────

function TypeBadge({ tag }: { tag: Tag }) {
  const isBoolean = !tag.value;
  return (
    <span
      style={{
        fontSize: 10,
        fontFamily: 'var(--font-mono)',
        fontWeight: 600,
        textTransform: 'uppercase',
        letterSpacing: '0.05em',
        color: isBoolean ? 'var(--text-muted)' : 'var(--accent)',
        background: isBoolean ? 'transparent' : 'color-mix(in srgb, var(--accent) 8%, transparent)',
        border: `1px solid ${isBoolean ? 'var(--border)' : 'color-mix(in srgb, var(--accent) 20%, transparent)'}`,
        borderRadius: 4,
        padding: '2px 7px',
      }}
    >
      {isBoolean ? 'Boolean' : 'Key-Value'}
    </span>
  );
}

// ─── Expanded Row ─────────────────────────────────────────────────────────────

const TagExpandedRow = memo(function TagExpandedRow({
  tag,
  onEdit,
  onDelete,
}: {
  tag: Tag;
  onEdit: (id: string) => void;
  onDelete: (id: string, label: string) => void;
}) {
  const can = useCan();

  return (
    <div
      style={{
        background: 'var(--bg-inset)',
        padding: '16px 20px 16px 48px',
        display: 'grid',
        gridTemplateColumns: '1fr 1fr',
        gap: 24,
        borderTop: '1px solid var(--border)',
      }}
    >
      {/* Description */}
      <div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            color: 'var(--text-muted)',
            marginBottom: 8,
          }}
        >
          Description
        </div>
        <p
          style={{
            fontSize: 12,
            color: tag.description ? 'var(--text-primary)' : 'var(--text-muted)',
            fontFamily: 'var(--font-sans)',
            margin: 0,
            lineHeight: 1.5,
          }}
        >
          {tag.description || 'No description provided.'}
        </p>
        <div style={{ marginTop: 12, display: 'flex', gap: 8 }}>
          <button
            type="button"
            disabled={!can('endpoints', 'update')}
            title={!can('endpoints', 'update') ? "You don't have permission" : undefined}
            onClick={() => onEdit(tag.id)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 5,
              height: 28,
              padding: '0 12px',
              background: 'transparent',
              border: '1px solid var(--border)',
              borderRadius: 5,
              fontSize: 11,
              fontWeight: 500,
              color: 'var(--text-secondary)',
              cursor: !can('endpoints', 'update') ? 'not-allowed' : 'pointer',
              fontFamily: 'var(--font-sans)',
              opacity: !can('endpoints', 'update') ? 0.5 : 1,
            }}
          >
            <Pencil style={{ width: 11, height: 11 }} />
            Edit
          </button>
          <button
            type="button"
            disabled={!can('endpoints', 'delete')}
            title={!can('endpoints', 'delete') ? "You don't have permission" : undefined}
            onClick={() => onDelete(tag.id, tagLabel(tag))}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 5,
              height: 28,
              padding: '0 12px',
              background: 'transparent',
              border: '1px solid color-mix(in srgb, var(--signal-critical) 1%, transparent)',
              borderRadius: 5,
              fontSize: 11,
              fontWeight: 500,
              color: 'var(--signal-critical)',
              cursor: !can('endpoints', 'delete') ? 'not-allowed' : 'pointer',
              fontFamily: 'var(--font-sans)',
              opacity: !can('endpoints', 'delete') ? 0.5 : 1,
            }}
          >
            <Trash2 style={{ width: 11, height: 11 }} />
            Delete
          </button>
        </div>
      </div>

      {/* Endpoints */}
      <div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            color: 'var(--text-muted)',
            marginBottom: 8,
          }}
        >
          Endpoints ({tag.endpoint_count ?? 0})
        </div>
        {(tag.endpoint_count ?? 0) === 0 ? (
          <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>No endpoints tagged.</span>
        ) : (
          <Link
            to={`/endpoints?tag=${encodeURIComponent(tagLabel(tag))}`}
            style={{ fontSize: 12, color: 'var(--accent)', textDecoration: 'none' }}
          >
            View {tag.endpoint_count} tagged endpoint{(tag.endpoint_count ?? 0) === 1 ? '' : 's'} →
          </Link>
        )}
      </div>
    </div>
  );
});

// ─── Inline Create Form ───────────────────────────────────────────────────────

const inputStyle: React.CSSProperties = {
  height: '36px',
  padding: '0 10px',
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: '6px',
  fontSize: '13px',
  color: 'var(--text-primary)',
  fontFamily: 'var(--font-sans)',
  outline: 'none',
  transition: 'border-color 0.15s, box-shadow 0.15s',
  boxSizing: 'border-box',
};

function FocusInput(props: React.InputHTMLAttributes<HTMLInputElement>) {
  const [focused, setFocused] = useState(false);
  const { style, ...rest } = props;
  return (
    <input
      style={{
        ...inputStyle,
        ...(focused
          ? {
              borderColor: 'var(--accent)',
              boxShadow: '0 0 0 2px color-mix(in srgb, var(--accent) 15%, transparent)',
            }
          : {}),
        ...style,
      }}
      onFocus={() => setFocused(true)}
      onBlur={() => setFocused(false)}
      {...rest}
    />
  );
}

interface CreateFormProps {
  onClose: () => void;
}

function CreateTagForm({ onClose }: CreateFormProps) {
  const [newTagKey, setNewTagKey] = useState('');
  const [newTagValue, setNewTagValue] = useState('');
  const [newTagDescription, setNewTagDescription] = useState('');
  const createTag = useCreateTag();

  const handleCreate = async () => {
    if (!newTagKey.trim()) return;
    try {
      await createTag.mutateAsync({
        key: newTagKey.trim(),
        value: newTagValue.trim(),
        description: newTagDescription.trim() || undefined,
      });
      setNewTagKey('');
      setNewTagValue('');
      setNewTagDescription('');
      onClose();
      const label = newTagValue.trim()
        ? `${newTagKey.trim()}:${newTagValue.trim()}`
        : newTagKey.trim();
      toast.success(`Tag "${label}" created`);
    } catch (err) {
      toast.error(`Failed to create tag: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: '8px',
        boxShadow: 'var(--shadow-sm)',
        padding: '20px',
        display: 'flex',
        flexDirection: 'column',
        gap: '14px',
        marginBottom: 4,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <h3
          style={{
            fontSize: '13px',
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            fontFamily: 'var(--font-sans)',
            margin: 0,
          }}
        >
          New Tag
        </h3>
        <button
          type="button"
          onClick={onClose}
          style={{
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            color: 'var(--text-muted)',
            display: 'flex',
            padding: '2px',
          }}
        >
          <X style={{ width: '14px', height: '14px' }} />
        </button>
      </div>
      <div style={{ display: 'flex', gap: '8px' }}>
        <FocusInput
          value={newTagKey}
          onChange={(e) => setNewTagKey(e.target.value)}
          placeholder="Key (e.g. env, os, role)"
          style={{ flex: 1 }}
        />
        <FocusInput
          value={newTagValue}
          onChange={(e) => setNewTagValue(e.target.value)}
          placeholder="Value (optional, e.g. production)"
          style={{ flex: 1 }}
        />
        <FocusInput
          value={newTagDescription}
          onChange={(e) => setNewTagDescription(e.target.value)}
          placeholder="Description (optional)"
          style={{ flex: 1.5 }}
        />
      </div>
      {newTagKey.trim() && (
        <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
          <span
            style={{
              fontSize: '11px',
              color: 'var(--text-muted)',
              fontFamily: 'var(--font-sans)',
            }}
          >
            Preview:
          </span>
          <span
            style={{
              fontSize: '11px',
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-secondary)',
              background: 'var(--bg-card-hover)',
              border: '1px solid var(--border-strong)',
              borderRadius: '4px',
              padding: '2px 8px',
            }}
          >
            {newTagValue.trim() ? `${newTagKey.trim()}:${newTagValue.trim()}` : newTagKey.trim()}
          </span>
        </div>
      )}
      <div style={{ display: 'flex', gap: '8px' }}>
        <button
          type="button"
          onClick={handleCreate}
          disabled={!newTagKey.trim() || createTag.isPending}
          style={{
            height: '32px',
            padding: '0 14px',
            background:
              !newTagKey.trim() || createTag.isPending
                ? 'color-mix(in srgb, var(--accent) 40%, transparent)'
                : 'var(--accent)',
            border: 'none',
            borderRadius: '6px',
            fontSize: '12px',
            fontWeight: 600,
            color: 'var(--text-on-color, #fff)',
            cursor: !newTagKey.trim() || createTag.isPending ? 'not-allowed' : 'pointer',
            fontFamily: 'var(--font-sans)',
          }}
        >
          {createTag.isPending ? 'Creating...' : 'Create'}
        </button>
        <button
          type="button"
          onClick={onClose}
          style={{
            height: '32px',
            padding: '0 14px',
            background: 'transparent',
            border: '1px solid var(--border)',
            borderRadius: '6px',
            fontSize: '12px',
            fontWeight: 500,
            color: 'var(--text-secondary)',
            cursor: 'pointer',
            fontFamily: 'var(--font-sans)',
          }}
        >
          Cancel
        </button>
      </div>
    </div>
  );
}

// ─── Actions Cell ────────────────────────────────────────────────────────────

const ActionsCell = memo(function ActionsCell({
  tag,
  onEdit,
  onDelete,
  isDeleting,
}: {
  tag: Tag;
  onEdit: (id: string) => void;
  onDelete: (id: string, label: string) => void;
  isDeleting: boolean;
}) {
  const [open, setOpen] = useState(false);
  const can = useCan();
  return (
    <div style={{ position: 'relative' }}>
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          setOpen((v) => !v);
        }}
        style={{
          width: 28,
          height: 28,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'transparent',
          border: '1px solid var(--border)',
          borderRadius: 5,
          cursor: 'pointer',
          color: 'var(--text-muted)',
          padding: 0,
          transition: 'all 0.15s',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.borderColor = 'var(--border-hover)';
          e.currentTarget.style.color = 'var(--text-primary)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.borderColor = 'var(--border)';
          e.currentTarget.style.color = 'var(--text-muted)';
        }}
      >
        <MoreHorizontal style={{ width: 13, height: 13 }} />
      </button>
      {open && (
        <>
          <div style={{ position: 'fixed', inset: 0, zIndex: 10 }} onClick={() => setOpen(false)} />
          <div
            style={{
              position: 'absolute',
              right: 0,
              top: '100%',
              marginTop: 4,
              zIndex: 20,
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              boxShadow: 'var(--shadow-md, 0 4px 16px rgba(0,0,0,0.3))',
              minWidth: 120,
              overflow: 'hidden',
            }}
          >
            <button
              type="button"
              disabled={!can('endpoints', 'update')}
              title={!can('endpoints', 'update') ? "You don't have permission" : undefined}
              onClick={(e) => {
                e.stopPropagation();
                setOpen(false);
                onEdit(tag.id);
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                width: '100%',
                padding: '8px 12px',
                background: 'transparent',
                border: 'none',
                cursor: !can('endpoints', 'update') ? 'not-allowed' : 'pointer',
                fontSize: 12,
                color: 'var(--text-secondary)',
                fontFamily: 'var(--font-sans)',
                textAlign: 'left',
                opacity: !can('endpoints', 'update') ? 0.5 : 1,
              }}
              onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--bg-card-hover)')}
              onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
            >
              <Pencil style={{ width: 12, height: 12 }} />
              Edit
            </button>
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                setOpen(false);
                onDelete(tag.id, tagLabel(tag));
              }}
              disabled={!can('endpoints', 'delete') || isDeleting}
              title={!can('endpoints', 'delete') ? "You don't have permission" : undefined}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                width: '100%',
                padding: '8px 12px',
                background: 'transparent',
                border: 'none',
                cursor: !can('endpoints', 'delete') || isDeleting ? 'not-allowed' : 'pointer',
                fontSize: 12,
                color: 'var(--signal-critical)',
                fontFamily: 'var(--font-sans)',
                textAlign: 'left',
                opacity: !can('endpoints', 'delete') || isDeleting ? 0.5 : 1,
              }}
              onMouseEnter={(e) =>
                (e.currentTarget.style.background =
                  'color-mix(in srgb, var(--signal-critical) 1%, transparent)')
              }
              onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
            >
              <Trash2 style={{ width: 12, height: 12 }} />
              Delete
            </button>
          </div>
        </>
      )}
    </div>
  );
});

// ─── Main Page ────────────────────────────────────────────────────────────────

type TagFilter = 'all' | 'kv' | 'bool' | 'unused';

const PAGE_SIZE = 15;

const getRowId = (row: Tag) => row.id;

export function TagsPage() {
  const can = useCan();
  const [search, setSearch] = useState('');
  const [activeFilter, setActiveFilter] = useState<TagFilter>('all');
  const [showCreate, setShowCreate] = useState(false);
  const [editTagId, setEditTagId] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<ExpandedState>({});
  const [cursors, setCursors] = useState<string[]>([]);

  const currentCursor = cursors[cursors.length - 1];

  const {
    data: allTags,
    isLoading,
    isError,
    refetch,
  } = useTags({
    cursor: currentCursor,
    limit: 100,
    search: search || undefined,
  });

  const deleteTag = useDeleteTag();
  const deleteTagMutate = deleteTag.mutateAsync;
  const deleteTagPending = deleteTag.isPending;

  const tags = useMemo(() => allTags ?? [], [allTags]);

  // Computed counts
  const { kvCount, boolCount, unusedCount, totalCount } = useMemo(() => {
    let kv = 0;
    let bool = 0;
    let unused = 0;
    for (const t of tags) {
      if (t.value) kv++;
      else bool++;
      if ((t.endpoint_count ?? 0) === 0) unused++;
    }
    return { kvCount: kv, boolCount: bool, unusedCount: unused, totalCount: tags.length };
  }, [tags]);

  // Filtering
  const filteredTags = useMemo(
    () =>
      tags.filter((t) => {
        if (activeFilter === 'kv') return !!t.value;
        if (activeFilter === 'bool') return !t.value;
        if (activeFilter === 'unused') return (t.endpoint_count ?? 0) === 0;
        return true;
      }),
    [tags, activeFilter],
  );

  // Header click sorting isn't wired up — filteredTags are rendered in
  // server order. Keeping this as a memoized alias preserves referential
  // stability for downstream memos.
  const sortedData = filteredTags;

  // Pagination (client-side over filtered set)
  const currentPage = cursors.length;
  const pageStart = currentPage * PAGE_SIZE;
  const pagedTags = useMemo(
    () => sortedData.slice(pageStart, pageStart + PAGE_SIZE),
    [sortedData, pageStart],
  );
  const hasNext = pageStart + PAGE_SIZE < sortedData.length;
  const hasPrev = currentPage > 0;

  const handleDelete = useCallback(
    async (id: string, label: string) => {
      if (!window.confirm(`Delete tag "${label}"? This will remove it from all endpoints.`)) return;
      try {
        await deleteTagMutate(id);
        toast.success(`Tag "${label}" deleted`);
      } catch (err) {
        toast.error(
          `Failed to delete tag: ${err instanceof Error ? err.message : 'Unknown error'}`,
        );
      }
    },
    [deleteTagMutate],
  );

  const handleEdit = useCallback((id: string) => setEditTagId(id), []);

  const toggleFilter = (f: TagFilter) => {
    setActiveFilter((prev) => (prev === f ? 'all' : f));
    setCursors([]);
  };

  // ─── Column definitions ─────────────────────────────────────────────────────

  const columns: ColumnDef<Tag>[] = useMemo(
    () => [
      {
        id: 'expand',
        size: 36,
        header: () => null,
        cell: ({ row }) => (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              row.toggleExpanded();
            }}
            style={{
              width: 24,
              height: 24,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'transparent',
              border: 'none',
              borderRadius: 4,
              cursor: 'pointer',
              color: 'var(--text-muted)',
              padding: 0,
              transition: 'color 0.15s, background 0.15s',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.color = 'var(--text-primary)';
              e.currentTarget.style.background = 'var(--bg-card-hover)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.color = 'var(--text-muted)';
              e.currentTarget.style.background = 'transparent';
            }}
          >
            <svg
              width="12"
              height="12"
              viewBox="0 0 12 12"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              style={{
                transform: row.getIsExpanded() ? 'rotate(90deg)' : 'rotate(0deg)',
                transition: 'transform 0.2s',
              }}
            >
              <path d="M4.5 2.5L8.5 6L4.5 9.5" />
            </svg>
          </button>
        ),
      },
      {
        id: 'tag',
        header: 'Tag',
        cell: ({ row }) => <TagPill tag={row.original} />,
      },
      {
        id: 'type',
        header: 'Type',
        size: 100,
        cell: ({ row }) => <TypeBadge tag={row.original} />,
      },
      {
        id: 'endpoints',
        accessorKey: 'endpoint_count',
        header: 'Endpoints',
        size: 100,
        cell: ({ row }) => {
          const count = row.original.endpoint_count ?? 0;
          return (
            <Link
              to={`/endpoints?tag=${encodeURIComponent(tagLabel(row.original))}`}
              onClick={(e) => e.stopPropagation()}
              style={{
                fontSize: 13,
                fontFamily: 'var(--font-mono)',
                color: count > 0 ? 'var(--accent)' : 'var(--text-muted)',
                textDecoration: 'none',
                fontWeight: count > 0 ? 600 : 400,
              }}
              title={`View endpoints with tag ${tagLabel(row.original)}`}
            >
              {count}
            </Link>
          );
        },
      },
      {
        id: 'description',
        accessorKey: 'description',
        header: 'Description',
        cell: ({ getValue }) => (
          <span
            style={{
              fontSize: 12,
              color: 'var(--text-muted)',
              fontFamily: 'var(--font-sans)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              display: 'block',
              maxWidth: 320,
            }}
          >
            {(getValue() as string | null) || '—'}
          </span>
        ),
      },
      {
        id: 'created',
        accessorKey: 'created_at',
        header: 'Created',
        size: 100,
        cell: ({ getValue }) => (
          <span
            style={{
              fontSize: 11,
              color: 'var(--text-muted)',
              fontFamily: 'var(--font-mono)',
            }}
          >
            {relativeTime(getValue() as string)}
          </span>
        ),
      },
      {
        id: 'actions',
        header: '',
        size: 56,
        cell: ({ row }) => (
          <ActionsCell
            tag={row.original}
            onEdit={handleEdit}
            onDelete={handleDelete}
            isDeleting={deleteTagPending}
          />
        ),
      },
    ],
    [handleEdit, handleDelete, deleteTagPending],
  );

  const tableState = useMemo(() => ({ expanded }), [expanded]);
  const table = useReactTable({
    data: pagedTags,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
    onExpandedChange: setExpanded,
    state: tableState,
    getRowId: getRowId,
  });

  const renderExpandedRow = useCallback(
    (tag: Tag) => <TagExpandedRow tag={tag} onEdit={handleEdit} onDelete={handleDelete} />,
    [handleEdit, handleDelete],
  );

  // ─── Render ─────────────────────────────────────────────────────────────────

  return (
    <div
      style={{
        padding: '24px',
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        minHeight: '100%',
        background: 'var(--bg-page)',
      }}
    >
      {/* Page Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          paddingBottom: 16,
          borderBottom: '1px solid var(--border)',
        }}
      >
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: 10 }}>
          <h1
            style={{
              fontSize: 22,
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              margin: 0,
              lineHeight: 1.2,
            }}
          >
            Tags
          </h1>
          {!isLoading && (
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 12,
                fontWeight: 700,
                color: 'var(--accent)',
                background: 'color-mix(in srgb, var(--accent) 10%, transparent)',
                border: '1px solid color-mix(in srgb, var(--accent) 20%, transparent)',
                borderRadius: 20,
                padding: '2px 9px',
                lineHeight: 1.5,
                minWidth: 28,
                display: 'inline-block',
                textAlign: 'center',
              }}
            >
              {totalCount}
            </span>
          )}
        </div>
        <button
          type="button"
          disabled={!can('endpoints', 'create')}
          title={!can('endpoints', 'create') ? "You don't have permission" : undefined}
          onClick={() => setShowCreate((v) => !v)}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            height: '34px',
            padding: '0 14px',
            background: 'var(--accent)',
            border: 'none',
            borderRadius: '6px',
            fontSize: '13px',
            fontWeight: 600,
            color: 'var(--text-on-color, #fff)',
            cursor: !can('endpoints', 'create') ? 'not-allowed' : 'pointer',
            fontFamily: 'var(--font-sans)',
            opacity: !can('endpoints', 'create') ? 0.5 : 1,
          }}
        >
          <Plus style={{ width: '14px', height: '14px' }} />
          Create Tag
        </button>
      </div>

      {/* Inline create form */}
      {showCreate && <CreateTagForm onClose={() => setShowCreate(false)} />}

      {/* Stat Cards */}
      <div style={{ display: 'flex', gap: 8 }}>
        <StatCardButton
          label="Total Tags"
          value={totalCount}
          active={activeFilter === 'all'}
          onClick={() => toggleFilter('all')}
        />
        <StatCardButton
          label="Key-Value"
          value={kvCount}
          valueColor="var(--accent)"
          active={activeFilter === 'kv'}
          onClick={() => toggleFilter('kv')}
        />
        <StatCardButton
          label="Boolean"
          value={boolCount}
          valueColor="var(--text-secondary)"
          active={activeFilter === 'bool'}
          onClick={() => toggleFilter('bool')}
        />
        <StatCardButton
          label="Unused"
          value={unusedCount}
          valueColor="var(--signal-warning)"
          active={activeFilter === 'unused'}
          onClick={() => toggleFilter('unused')}
        />
      </div>

      {/* Filter Bar */}
      <FilterBar>
        <FilterPill
          label="All"
          count={totalCount}
          active={activeFilter === 'all'}
          onClick={() => toggleFilter('all')}
        />
        <FilterPill
          label="Key-Value"
          count={kvCount}
          active={activeFilter === 'kv'}
          onClick={() => toggleFilter('kv')}
        />
        <FilterPill
          label="Boolean"
          count={boolCount}
          active={activeFilter === 'bool'}
          onClick={() => toggleFilter('bool')}
          variant="default"
        />
        <FilterPill
          label="Unused"
          count={unusedCount}
          active={activeFilter === 'unused'}
          onClick={() => toggleFilter('unused')}
          variant="high"
        />
        <FilterSeparator />
        <FilterSearch value={search} onChange={setSearch} placeholder="Search tags..." />
      </FilterBar>

      {/* Table */}
      {isLoading ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {Array.from({ length: 6 }).map((_, i) => (
            <SkeletonCard key={i} lines={2} />
          ))}
        </div>
      ) : isError ? (
        <ErrorState message="Failed to load tags" onRetry={() => void refetch()} />
      ) : filteredTags.length === 0 ? (
        <EmptyState
          icon={<TagIcon className="h-12 w-12" />}
          title={search || activeFilter !== 'all' ? 'No matching tags' : 'No tags yet'}
          description={
            search
              ? `No tags match "${search}".`
              : activeFilter !== 'all'
                ? 'No tags match this filter.'
                : 'Create a tag to organize your endpoints.'
          }
          action={
            !search && activeFilter === 'all'
              ? { label: 'Create Tag', onClick: () => setShowCreate(true) }
              : undefined
          }
        />
      ) : (
        <>
          <DataTable table={table} renderExpandedRow={renderExpandedRow} />
          <DataTablePagination
            hasNext={hasNext}
            hasPrev={hasPrev}
            onNext={() => setCursors((prev) => [...prev, ''])}
            onPrev={() => setCursors((prev) => prev.slice(0, -1))}
          />
        </>
      )}

      {/* Edit dialog */}
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
