import { buildAuditCsv } from '../../../pages/audit/export-csv';

const events = [
  {
    id: '01ABC',
    tenant_id: 't1',
    type: 'endpoint.created',
    actor_id: 'system',
    actor_type: 'system' as const,
    resource: 'endpoint',
    resource_id: 'e1',
    action: 'create',
    payload: {},
    metadata: {},
    timestamp: '2026-03-06T10:00:00Z',
  },
];

describe('buildAuditCsv', () => {
  it('produces CSV with header and one row', () => {
    const csv = buildAuditCsv(events);
    const lines = csv.split('\n');
    expect(lines[0]).toBe('Timestamp,Actor ID,Actor Type,Action,Resource,Resource ID,Type');
    expect(lines[1]).toBe('2026-03-06T10:00:00Z,system,system,create,endpoint,e1,endpoint.created');
  });

  it('returns only header for empty array', () => {
    const csv = buildAuditCsv([]);
    const lines = csv.split('\n').filter(Boolean);
    expect(lines).toHaveLength(1);
  });
});
