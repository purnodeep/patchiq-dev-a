import type { EndpointDetail, EndpointPatch, Endpoint } from '../../api/hooks/useEndpoints';

export function buildEndpointReportCsv(endpoint: EndpointDetail, patches: EndpointPatch[]): string {
  const lines: string[] = [];

  lines.push('Endpoint Overview');
  lines.push('Field,Value');
  lines.push(`Hostname,${endpoint.hostname}`);
  lines.push(`OS,${endpoint.os_family} ${endpoint.os_version}`);
  lines.push(`Status,${endpoint.status}`);
  lines.push(`IP Address,${endpoint.ip_address ?? ''}`);
  lines.push(`Agent Version,${endpoint.agent_version ?? ''}`);
  lines.push(`Architecture,${endpoint.arch ?? ''}`);
  lines.push(`Kernel Version,${endpoint.kernel_version ?? ''}`);
  lines.push(`CPU Model,${endpoint.cpu_model ?? ''}`);
  lines.push(`CPU Cores,${endpoint.cpu_cores ?? ''}`);
  lines.push(`CPU Usage (%),${endpoint.cpu_usage_percent ?? ''}`);
  lines.push(`Memory Total (MB),${endpoint.memory_total_mb ?? ''}`);
  lines.push(`Disk Total (GB),${endpoint.disk_total_gb ?? ''}`);
  lines.push(`Last Seen,${endpoint.last_seen ?? ''}`);
  lines.push(`Last Scan,${endpoint.last_scan ?? ''}`);
  lines.push(`Enrolled At,${endpoint.enrolled_at ?? ''}`);
  lines.push('');

  lines.push('Patches');
  lines.push('Name,Version,Severity,Status,CVE Count');
  for (const p of patches) {
    lines.push(`"${p.name}",${p.version},${p.severity},${p.status},${p.cve_count}`);
  }

  return lines.join('\n');
}

/** Fetches a server-side CSV export with the given filters and triggers a download. */
export async function downloadEndpointExport(params: {
  status?: string;
  os_family?: string;
  tag_id?: string;
  search?: string;
}): Promise<void> {
  const qs = new URLSearchParams();
  if (params.status) qs.set('status', params.status);
  if (params.os_family) qs.set('os_family', params.os_family);
  if (params.tag_id) qs.set('tag_id', params.tag_id);
  if (params.search) qs.set('search', params.search);

  const res = await fetch(`/api/v1/endpoints/export?${qs.toString()}`, {
    credentials: 'include',
  });
  if (!res.ok) throw new Error(`Export failed: ${res.status}`);

  const csv = await res.text();
  const date = new Date().toISOString().slice(0, 10);
  downloadCsvString(csv, `endpoints-export-${date}.csv`);
}

const CSV_HEADERS = [
  'Hostname',
  'OS Family',
  'OS Version',
  'Status',
  'Agent Version',
  'IP Address',
  'Architecture',
  'Kernel Version',
  'Last Seen',
  'Pending Patches',
  'Critical Patches',
  'CVEs',
  'Tags',
];

function csvField(v: string | number | null | undefined): string {
  const s = v == null ? '' : String(v);
  if (s.includes(',') || s.includes('"') || s.includes('\n')) {
    return `"${s.replace(/"/g, '""')}"`;
  }
  return s;
}

/** Builds a CSV string from a list of already-loaded Endpoint objects (for bulk selection export). */
export function buildEndpointCsv(endpoints: Endpoint[]): string {
  const rows = endpoints.map((e) =>
    [
      csvField(e.hostname),
      csvField(e.os_family),
      csvField(e.os_version),
      csvField(e.status),
      csvField(e.agent_version),
      csvField(e.ip_address),
      csvField(e.arch),
      csvField(e.kernel_version),
      csvField(e.last_seen),
      csvField(e.pending_patches_count),
      csvField(e.critical_patch_count),
      csvField(e.cve_count),
      csvField(e.tags?.map((t) => `${t.key}:${t.value}`).join('; ') ?? ''),
    ].join(','),
  );
  return [CSV_HEADERS.join(','), ...rows].join('\n');
}

export function downloadCsvString(csv: string, filename: string): void {
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
