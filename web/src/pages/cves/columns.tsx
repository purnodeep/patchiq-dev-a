import { createColumnHelper } from '@tanstack/react-table';
import { Link } from 'react-router';
import { ChevronRight } from 'lucide-react';
import type { CVEListItem } from '../../types/cves';
import { SEVERITY_ORDER } from '../../types/shared';
import { SeverityBadge } from '../../components/SeverityBadge';
import { CVSSBar } from '../../components/CVSSBar';
import { KEVBadge } from '../../components/KEVBadge';
import { ExploitBadge } from '../../components/ExploitBadge';
import { AttackVectorBadge } from '../../components/AttackVectorBadge';

const columnHelper = createColumnHelper<CVEListItem>();

export const cveColumns = [
  columnHelper.accessor('cve_id', {
    header: 'CVE ID',
    cell: (info) => (
      <Link
        to={`/cves/${info.row.original.id}`}
        onClick={(e) => e.stopPropagation()}
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 12,
          fontWeight: 500,
          color: 'var(--accent)',
          whiteSpace: 'nowrap',
          textDecoration: 'none',
        }}
        onMouseEnter={(e) =>
          ((e.currentTarget as HTMLAnchorElement).style.textDecoration = 'underline')
        }
        onMouseLeave={(e) => ((e.currentTarget as HTMLAnchorElement).style.textDecoration = 'none')}
      >
        {info.getValue()}
      </Link>
    ),
  }),
  columnHelper.accessor('cvss_v3_score', {
    header: 'CVSS Score',
    cell: (info) => <CVSSBar score={info.getValue()} />,
    sortingFn: (rowA, rowB) => {
      const a = rowA.original.cvss_v3_score ?? -1;
      const b = rowB.original.cvss_v3_score ?? -1;
      return a - b;
    },
  }),
  columnHelper.accessor('severity', {
    header: 'Severity',
    cell: (info) => <SeverityBadge severity={info.getValue()} />,
    sortingFn: (rowA, rowB) =>
      SEVERITY_ORDER[rowA.original.severity] - SEVERITY_ORDER[rowB.original.severity],
  }),
  columnHelper.accessor('attack_vector', {
    header: 'Attack Vector',
    cell: (info) => <AttackVectorBadge vector={info.getValue()} />,
  }),
  columnHelper.accessor('exploit_available', {
    header: 'Exploit',
    cell: (info) => <ExploitBadge exploitAvailable={info.getValue()} />,
  }),
  columnHelper.accessor('cisa_kev_due_date', {
    header: 'KEV',
    cell: (info) => <KEVBadge dueDate={info.getValue()} />,
  }),
  columnHelper.accessor('patch_count', {
    header: 'Affected Pkgs',
    cell: (info) => (
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 12,
          color: 'var(--text-primary)',
          fontVariantNumeric: 'tabular-nums',
        }}
      >
        {info.getValue()}
      </span>
    ),
  }),
  columnHelper.accessor('affected_endpoint_count', {
    header: 'Affected Endpoints',
    cell: (info) => (
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 12,
          color: 'var(--text-primary)',
          fontVariantNumeric: 'tabular-nums',
        }}
      >
        {info.getValue()}
      </span>
    ),
    sortingFn: 'basic',
  }),
  columnHelper.accessor('patch_available', {
    header: 'Patches',
    cell: (info) =>
      info.getValue() ? (
        <span
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            borderRadius: 9999,
            background: 'color-mix(in srgb, var(--signal-healthy) 10%, transparent)',
            border: '1px solid color-mix(in srgb, var(--signal-healthy) 25%, transparent)',
            padding: '2px 8px',
            fontSize: 10,
            fontWeight: 600,
            fontFamily: 'var(--font-mono)',
            color: 'var(--signal-healthy)',
          }}
        >
          Available
        </span>
      ) : (
        <span style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 11 }}>
          —
        </span>
      ),
  }),
  columnHelper.accessor('published_at', {
    header: 'Published',
    cell: (info) => {
      const val = info.getValue();
      if (!val)
        return (
          <span
            style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 11 }}
          >
            —
          </span>
        );
      return (
        <span
          style={{
            fontSize: 11,
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-secondary)',
          }}
        >
          {new Date(val).toLocaleDateString()}
        </span>
      );
    },
  }),
  columnHelper.display({
    id: 'expand',
    cell: ({ row }) => (
      <ChevronRight
        style={{
          width: 14,
          height: 14,
          color: 'var(--text-muted)',
          transition: 'transform 0.15s ease',
          transform: row.getIsExpanded() ? 'rotate(90deg)' : 'rotate(0deg)',
        }}
      />
    ),
  }),
];
