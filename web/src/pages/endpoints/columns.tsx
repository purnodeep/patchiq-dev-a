import { createColumnHelper } from '@tanstack/react-table';
import { MonoTag, SeverityText } from '@patchiq/ui';
import { ChevronDown, ChevronRight } from 'lucide-react';
import type { Endpoint } from '../../api/hooks/useEndpoints';
import { StatusBadge } from '../../components/StatusBadge';
import { timeAgo } from '../../lib/time';
import { computeRiskScore as computeRisk, riskColor as riskClrFn } from '../../lib/risk';

const columnHelper = createColumnHelper<Endpoint>();

function getOsIcon(osFamily: string): { letter: string } {
  switch (osFamily) {
    case 'linux':
      return { letter: 'L' };
    case 'windows':
      return { letter: 'W' };
    case 'darwin':
      return { letter: 'M' };
    default:
      return { letter: osFamily.charAt(0).toUpperCase() };
  }
}

function computeRiskScore(ep: {
  critical_cve_count?: number;
  high_cve_count?: number;
  medium_cve_count?: number;
  critical_patch_count?: number;
  high_patch_count?: number;
  medium_patch_count?: number;
  cve_count?: number;
  pending_patches_count?: number;
}): number {
  // Prefer CVE severity counts (accurate risk), fall back to patch counts
  const critical = ep.critical_cve_count ?? ep.critical_patch_count;
  const high = ep.high_cve_count ?? ep.high_patch_count;
  const medium = ep.medium_cve_count ?? ep.medium_patch_count;
  if (critical !== undefined || high !== undefined) {
    return computeRisk({
      criticalCves: critical ?? 0,
      highCves: high ?? 0,
      mediumCves: medium ?? 0,
    });
  }
  return computeRisk({ cveCount: ep.cve_count, pendingPatches: ep.pending_patches_count });
}

function riskColor(score: number): string {
  return riskClrFn(score);
}

export const endpointColumns = [
  columnHelper.display({
    id: 'select',
    header: ({ table }) => (
      <input
        type="checkbox"
        checked={table.getIsAllPageRowsSelected()}
        onChange={table.getToggleAllPageRowsSelectedHandler()}
        aria-label="Select all"
        className="h-4 w-4 cursor-pointer"
      />
    ),
    cell: ({ row }) => (
      <input
        type="checkbox"
        checked={row.getIsSelected()}
        onChange={row.getToggleSelectedHandler()}
        aria-label="Select row"
        className="h-4 w-4 cursor-pointer"
      />
    ),
    enableSorting: false,
    enableHiding: false,
  }),
  columnHelper.accessor('hostname', {
    header: 'Hostname',
    cell: (info) => (
      <span
        className="font-semibold"
        style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-emphasis)' }}
      >
        {info.getValue()}
      </span>
    ),
  }),
  columnHelper.accessor((row) => `${row.os_family} ${row.os_version}`, {
    id: 'os',
    header: 'OS',
    cell: ({ row }) => {
      const { letter } = getOsIcon(row.original.os_family);
      return (
        <div className="flex items-center gap-2" data-testid="os-cell">
          <span
            className="inline-flex h-6 w-6 items-center justify-center rounded text-xs font-bold"
            style={{
              backgroundColor: 'var(--bg-card-hover)',
              color: 'var(--text-secondary)',
              border: '1px solid var(--border)',
            }}
          >
            {letter}
          </span>
          <span className="text-sm" style={{ color: 'var(--text-primary)' }}>
            {row.original.os_version}
          </span>
        </div>
      );
    },
  }),
  columnHelper.accessor('status', {
    header: 'Status',
    cell: (info) => <StatusBadge status={info.getValue()} />,
  }),
  columnHelper.accessor('agent_version', {
    header: 'Agent Ver',
    cell: (info) => (
      <span
        style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-muted)', fontSize: '12px' }}
      >
        {info.getValue() ?? '\u2014'}
      </span>
    ),
  }),
  columnHelper.display({
    id: 'risk_score',
    header: 'Risk Score',
    cell: ({ row }) => {
      const score = computeRiskScore(row.original);
      return (
        <span
          className="text-sm font-semibold"
          style={{ fontFamily: 'var(--font-mono)', color: riskColor(score) }}
        >
          {score.toFixed(1)}/10
        </span>
      );
    },
  }),
  columnHelper.accessor('last_seen', {
    header: 'Last Seen',
    cell: (info) => {
      const val = info.getValue();
      return <span style={{ color: 'var(--text-muted)' }}>{val ? timeAgo(val) : '\u2014'}</span>;
    },
  }),
  columnHelper.display({
    id: 'patches_pending',
    header: 'Patches Pending',
    cell: ({ row }) => {
      const { critical_patch_count, high_patch_count, medium_patch_count, pending_patches_count } =
        row.original;
      const hasSeverity =
        (critical_patch_count ?? 0) > 0 ||
        (high_patch_count ?? 0) > 0 ||
        (medium_patch_count ?? 0) > 0;

      if (hasSeverity) {
        return (
          <div className="flex items-center gap-1.5">
            {(critical_patch_count ?? 0) > 0 && (
              <span className="flex items-center gap-0.5">
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--signal-critical)',
                    fontSize: '12px',
                  }}
                >
                  {critical_patch_count}
                </span>
                <SeverityText severity="critical" />
              </span>
            )}
            {(high_patch_count ?? 0) > 0 && (
              <span className="flex items-center gap-0.5">
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--signal-warning)',
                    fontSize: '12px',
                  }}
                >
                  {high_patch_count}
                </span>
                <SeverityText severity="high" />
              </span>
            )}
            {(medium_patch_count ?? 0) > 0 && (
              <span className="flex items-center gap-0.5">
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-secondary)',
                    fontSize: '12px',
                  }}
                >
                  {medium_patch_count}
                </span>
                <SeverityText severity="medium" />
              </span>
            )}
          </div>
        );
      }

      const total = pending_patches_count ?? 0;
      if (!total) return <span style={{ color: 'var(--text-muted)' }}>{'\u2014'}</span>;
      return (
        <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)' }}>
          {total}
        </span>
      );
    },
  }),
  columnHelper.display({
    id: 'tags',
    header: 'Tags',
    cell: (info) => {
      const tags = info.row.original.tags ?? [];
      const meta = info.table.options.meta as { onTagClick?: (tagId: string) => void } | undefined;
      if (tags.length === 0) {
        return <span style={{ color: 'var(--text-muted)' }}>{'\u2014'}</span>;
      }
      return (
        <div className="flex items-center gap-1 flex-wrap" onClick={(e) => e.stopPropagation()}>
          {tags.map((tag) => {
            const label = `${tag.key}:${tag.value}`;
            return (
              <span
                key={tag.id}
                className="cursor-pointer hover:opacity-80 transition-opacity"
                onClick={() => meta?.onTagClick?.(tag.id)}
              >
                <MonoTag>{label}</MonoTag>
              </span>
            );
          })}
        </div>
      );
    },
    enableSorting: false,
  }),
  columnHelper.display({
    id: 'expand',
    header: '',
    cell: (info) => {
      const meta = info.table.options.meta as
        | { expandedRows: Set<string>; toggleRow: (id: string) => void }
        | undefined;
      const isExpanded = meta?.expandedRows.has(info.row.original.id) ?? false;
      return (
        <button
          onClick={(e) => {
            e.stopPropagation();
            meta?.toggleRow(info.row.original.id);
          }}
          className="flex h-6 w-6 items-center justify-center rounded transition-colors"
          style={{ color: 'var(--text-muted)' }}
        >
          {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        </button>
      );
    },
    enableSorting: false,
    enableHiding: false,
  }),
];
