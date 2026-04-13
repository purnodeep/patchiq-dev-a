import { render, screen, fireEvent } from '@testing-library/react';
import { ActivityStream } from '../../../pages/audit/ActivityStream';
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
    timestamp: '2026-03-10T08:00:00Z',
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
    payload: { deployment_id: 'dep-abc', target_count: 45 } as unknown as Record<string, never>,
    metadata: {} as unknown as Record<string, never>,
    timestamp: '2026-03-09T07:00:00Z',
  },
];

describe('ActivityStream', () => {
  it('renders actor initials in avatars', () => {
    render(<ActivityStream events={EVENTS} expandedId={null} onToggleExpand={() => {}} />);
    expect(screen.getByText('AD')).toBeInTheDocument(); // admin@acme.com
    expect(screen.getByText('DP')).toBeInTheDocument(); // system + deployment.created → DP
  });

  it('renders category badges', () => {
    render(<ActivityStream events={EVENTS} expandedId={null} onToggleExpand={() => {}} />);
    expect(screen.getByText('Endpoint')).toBeInTheDocument();
    expect(screen.getByText('Deployment')).toBeInTheDocument();
  });

  it('calls onToggleExpand when event card is clicked', () => {
    const onToggle = vi.fn();
    render(<ActivityStream events={EVENTS} expandedId={null} onToggleExpand={onToggle} />);
    // Click the first event row — rows are <tr> elements with inline cursor:pointer style
    const rows = document.querySelectorAll('tr[style*="cursor: pointer"]');
    expect(rows.length).toBeGreaterThan(0);
    fireEvent.click(rows[0]);
    expect(onToggle).toHaveBeenCalledWith('evt1');
  });

  it('shows JSON payload when event is expanded', () => {
    render(<ActivityStream events={EVENTS} expandedId="evt1" onToggleExpand={() => {}} />);
    expect(screen.getByText('Event Payload')).toBeInTheDocument();
    // JSON content should be visible in pre block
    const preEl = document.querySelector('pre');
    expect(preEl?.textContent).toContain('prod-web-01');
  });

  it('does not show payload when event is not expanded', () => {
    render(<ActivityStream events={EVENTS} expandedId={null} onToggleExpand={() => {}} />);
    expect(screen.queryByText('Event Payload')).not.toBeInTheDocument();
  });

  it('shows empty state when no events', () => {
    render(<ActivityStream events={[]} expandedId={null} onToggleExpand={() => {}} />);
    expect(screen.getByText(/no audit events found/i)).toBeInTheDocument();
  });
});
