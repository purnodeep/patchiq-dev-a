import { toast } from 'sonner';

import type { components } from '../../api/types';

type AuditEvent = components['schemas']['AuditEvent'];

const HEADERS = [
  'Timestamp',
  'Actor ID',
  'Actor Type',
  'Action',
  'Resource',
  'Resource ID',
  'Type',
];

export function buildAuditCsv(events: AuditEvent[]): string {
  const rows = events.map((e) =>
    [e.timestamp, e.actor_id, e.actor_type, e.action, e.resource, e.resource_id, e.type].join(','),
  );
  return [HEADERS.join(','), ...rows].join('\n');
}

export function downloadCsv(csv: string, filename: string): void {
  let url: string | undefined;
  try {
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
  } catch (err) {
    toast.error(`Failed to export CSV: ${err instanceof Error ? err.message : 'unknown error'}`);
  } finally {
    if (url) {
      URL.revokeObjectURL(url);
    }
  }
}
