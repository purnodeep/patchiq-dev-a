import { render, screen } from '@testing-library/react';
import { TimelineView } from '../../../pages/audit/TimelineView';
import type { components } from '../../../api/types';

type AuditEvent = components['schemas']['AuditEvent'];

const EVENTS: AuditEvent[] = [
  {
    id: 'evt1',
    tenant_id: 't1',
    type: 'endpoint.enrolled',
    actor_id: 'admin@acme.com',
    actor_type: 'user',
    resource: 'endpoint',
    resource_id: 'e1',
    action: 'enrolled',
    payload: { hostname: 'prod-web-01' } as unknown as Record<string, never>,
    metadata: {} as unknown as Record<string, never>,
    timestamp: '2026-03-10T08:14:00Z',
  },
  {
    id: 'evt2',
    tenant_id: 't1',
    type: 'deployment.created',
    actor_id: 'system',
    actor_type: 'system',
    resource: 'deployment',
    resource_id: 'dep-abc',
    action: 'created',
    payload: { deployment_id: 'dep-abc' } as unknown as Record<string, never>,
    metadata: {} as unknown as Record<string, never>,
    timestamp: '2026-03-10T07:22:00Z',
  },
  {
    id: 'evt3',
    tenant_id: 't1',
    type: 'policy.updated',
    actor_id: 'admin@acme.com',
    actor_type: 'user',
    resource: 'policy',
    resource_id: 'pol-1',
    action: 'updated',
    payload: {} as Record<string, never>,
    metadata: {} as unknown as Record<string, never>,
    timestamp: '2026-03-09T12:00:00Z',
  },
];

describe('TimelineView', () => {
  it('groups events by date with date separators', () => {
    render(<TimelineView events={EVENTS} />);
    expect(screen.getByText(/March 10/)).toBeInTheDocument();
    expect(screen.getByText(/March 9/)).toBeInTheDocument();
  });

  it('renders event summary text', () => {
    render(<TimelineView events={EVENTS} />);
    // Should show actor names in event text
    const adminTexts = screen.getAllByText(/admin@acme\.com/);
    expect(adminTexts.length).toBeGreaterThan(0);
  });

  it('renders category badges for each event', () => {
    render(<TimelineView events={EVENTS} />);
    expect(screen.getByText('Deployment')).toBeInTheDocument();
    expect(screen.getByText('Policy')).toBeInTheDocument();
  });

  it('shows UTC time for each event', () => {
    render(<TimelineView events={EVENTS} />);
    // TimelineView renders time with seconds via toLocaleTimeString (e.g. "08:14:00 UTC")
    expect(screen.getByText(/08:14.*UTC/)).toBeInTheDocument();
    expect(screen.getByText(/07:22.*UTC/)).toBeInTheDocument();
  });

  it('shows empty state when no events', () => {
    render(<TimelineView events={[]} />);
    expect(screen.getByText(/no audit events found/i)).toBeInTheDocument();
  });
});
