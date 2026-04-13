import { createColumnHelper } from '@tanstack/react-table';
import type { PatchListItem } from '../../types/patches';
import { SEVERITY_ORDER } from '../../types/shared';
import { SeverityBadge } from '../../components/SeverityBadge';

const columnHelper = createColumnHelper<PatchListItem>();

const OS_LETTER_MAP: Record<string, string> = {
  windows: 'W',
  ubuntu: 'U',
  rhel: 'R',
  debian: 'D',
  linux: 'L',
  darwin: 'M',
  macos: 'M',
  centos: 'C',
  fedora: 'F',
  opensuse: 'S',
};

function getOsStyle(osFamily: string) {
  const key = osFamily.toLowerCase();
  return {
    bg: 'var(--bg-card-hover)',
    text: 'var(--text-secondary)',
    border: 'var(--border-strong)',
    letter: OS_LETTER_MAP[key] ?? osFamily[0]?.toUpperCase() ?? '?',
  };
}

function cvssColor(score: number): string {
  if (score >= 9) return 'var(--signal-critical)';
  if (score >= 7) return '#f97316';
  if (score >= 4) return 'var(--signal-warning)';
  return 'var(--accent)';
}

function remColor(pct: number): string {
  if (pct >= 100) return 'var(--signal-healthy)';
  if (pct >= 70) return 'var(--accent)';
  if (pct >= 50) return 'var(--signal-warning)';
  return 'var(--signal-critical)';
}

function MiniBar({
  value,
  max,
  color,
  width,
}: {
  value: number;
  max: number;
  color: string;
  width: number;
}) {
  const pct = Math.min(100, (value / max) * 100);
  return (
    <div
      style={{
        width,
        height: 5,
        background: 'color-mix(in srgb, white 10%, transparent)',
        borderRadius: 3,
        overflow: 'hidden',
        display: 'inline-block',
      }}
    >
      <div
        style={{
          width: `${pct}%`,
          height: '100%',
          background: color,
          borderRadius: 3,
        }}
      />
    </div>
  );
}

export const patchColumns = [
  columnHelper.accessor('name', {
    header: 'Patch Name',
    cell: (info) => (
      <span className="font-mono font-semibold text-primary text-xs">{info.getValue()}</span>
    ),
  }),
  columnHelper.accessor('version', {
    header: 'Version / KB',
    cell: (info) => <span className="text-[11px] text-muted-foreground">{info.getValue()}</span>,
  }),
  columnHelper.accessor('severity', {
    header: 'Severity',
    cell: (info) => <SeverityBadge severity={info.getValue()} />,
    sortingFn: (rowA, rowB) =>
      SEVERITY_ORDER[rowA.original.severity] - SEVERITY_ORDER[rowB.original.severity],
  }),
  columnHelper.accessor('os_family', {
    header: 'OS',
    cell: (info) => {
      const os = info.getValue();
      const style = getOsStyle(os);
      return (
        <div className="flex items-center gap-1.5">
          <div
            style={{
              width: 18,
              height: 18,
              borderRadius: 4,
              background: style.bg,
              color: style.text,
              border: `1px solid ${style.border}`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 9,
              fontWeight: 700,
              flexShrink: 0,
            }}
          >
            {style.letter}
          </div>
          <span className="text-[11px]">{os}</span>
        </div>
      );
    },
  }),
  columnHelper.accessor('cve_count', {
    header: 'CVE Count',
    cell: (info) => <span className="text-xs">{info.getValue()}</span>,
    sortingFn: 'basic',
  }),
  columnHelper.accessor('highest_cvss_score', {
    header: 'CVSS Highest',
    cell: (info) => {
      const score = info.getValue() ?? 0;
      const color = cvssColor(score);
      return (
        <div className="flex items-center gap-1.5">
          <MiniBar value={score} max={10} color={color} width={60} />
          <span style={{ fontSize: 11, fontWeight: 600, color }}>{score.toFixed(1)}</span>
        </div>
      );
    },
    sortingFn: 'basic',
  }),
  columnHelper.accessor('affected_endpoint_count', {
    header: 'Affected Endpoints',
    cell: (info) => {
      const total = info.getValue();
      const deployed = info.row.original.endpoints_deployed_count ?? 0;
      return (
        <span className="text-xs">
          {total} <span className="text-muted-foreground text-[10px]">({deployed} deployed)</span>
        </span>
      );
    },
    sortingFn: 'basic',
  }),
  columnHelper.accessor('remediation_pct', {
    header: 'Remediation %',
    cell: (info) => {
      const pct = info.getValue() ?? 0;
      const color = remColor(pct);
      return (
        <div className="flex items-center gap-1.5">
          <MiniBar value={pct} max={100} color={color} width={70} />
          <span style={{ fontSize: 11, color }}>{pct}%</span>
        </div>
      );
    },
    sortingFn: 'basic',
  }),
  columnHelper.accessor('released_at', {
    header: 'Released',
    cell: (info) => {
      const val = info.getValue() ?? info.row.original.created_at;
      if (!val) return <span className="text-muted-foreground text-[11px]">—</span>;
      const d = new Date(val);
      const iso = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
      return <span className="text-[11px] text-muted-foreground">{iso}</span>;
    },
  }),
  columnHelper.accessor('status', {
    header: 'Status',
    cell: (info) => {
      const status = info.getValue();
      const pct = info.row.original.remediation_pct ?? 0;

      let displayStatus: string;
      let cls: string;

      if (status === 'superseded') {
        displayStatus = 'Superseded';
        cls = 'bg-white/10 text-muted-foreground border-border';
      } else if (status === 'recalled') {
        displayStatus = 'Not Applicable';
        cls = 'bg-white/10 text-muted-foreground border-border';
      } else if (pct >= 100) {
        displayStatus = 'Deployed';
        cls = 'bg-green-500/10 text-green-500 border-green-500/20';
      } else {
        displayStatus = 'Pending';
        cls = 'bg-amber-500/10 text-amber-500 border-amber-500/20';
      }

      return (
        <span
          className={`inline-flex items-center rounded border px-1.5 py-0.5 text-[10px] font-semibold whitespace-nowrap ${cls}`}
        >
          {displayStatus}
        </span>
      );
    },
  }),
];
