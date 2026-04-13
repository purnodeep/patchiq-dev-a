import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { MemoryRouter } from 'react-router';
import { WorkflowExecution, type RecentWorkflow } from '../../../pages/dashboard/WorkflowExecution';

const makeWorkflow = (overrides: Partial<RecentWorkflow> = {}): RecentWorkflow => ({
  id: 'wf-1',
  name: 'Deploy Patches',
  status: 'completed',
  started_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
  steps: [
    { name: 'Validate', status: 'completed' },
    { name: 'Deploy', status: 'completed' },
    { name: 'Verify', status: 'completed' },
  ],
  ...overrides,
});

function renderWithRouter(ui: React.ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>);
}

describe('WorkflowExecution', () => {
  it('renders card title "Workflow Executions"', () => {
    renderWithRouter(<WorkflowExecution workflows={[]} />);
    expect(screen.getByText('Workflow Executions')).toBeDefined();
  });

  it('renders workflow names', () => {
    const workflows = [
      makeWorkflow({ id: 'wf-1', name: 'Deploy Patches' }),
      makeWorkflow({ id: 'wf-2', name: 'Security Scan' }),
    ];
    renderWithRouter(<WorkflowExecution workflows={workflows} />);
    expect(screen.getByText('Deploy Patches')).toBeDefined();
    expect(screen.getByText('Security Scan')).toBeDefined();
  });

  it('renders step circles with correct status indicators via aria-label', () => {
    const wf = makeWorkflow({
      steps: [
        { name: 'Validate', status: 'completed' },
        { name: 'Deploy', status: 'running' },
        { name: 'Notify', status: 'pending' },
      ],
    });
    renderWithRouter(<WorkflowExecution workflows={[wf]} />);
    expect(screen.getByLabelText('Validate: completed')).toBeDefined();
    expect(screen.getByLabelText('Deploy: running')).toBeDefined();
    expect(screen.getByLabelText('Notify: pending')).toBeDefined();
  });

  it('shows step circle with failed status indicator', () => {
    const wf = makeWorkflow({
      steps: [
        { name: 'Build', status: 'completed' },
        { name: 'Test', status: 'failed' },
      ],
    });
    renderWithRouter(<WorkflowExecution workflows={[wf]} />);
    expect(screen.getByLabelText('Test: failed')).toBeDefined();
  });

  it('shows "View All" link pointing to /workflows', () => {
    renderWithRouter(<WorkflowExecution workflows={[]} />);
    const link = screen.getByRole('link', { name: /view all/i });
    expect(link).toBeDefined();
    expect(link.getAttribute('href')).toBe('/workflows');
  });

  it('handles empty workflows array', () => {
    renderWithRouter(<WorkflowExecution workflows={[]} />);
    expect(screen.getByText('No recent workflow executions.')).toBeDefined();
  });

  it('shows at most 3 workflows', () => {
    const workflows = Array.from({ length: 5 }, (_, i) =>
      makeWorkflow({ id: `wf-${i}`, name: `Workflow ${i + 1}` }),
    );
    renderWithRouter(<WorkflowExecution workflows={workflows} />);
    expect(screen.getByText('Workflow 1')).toBeDefined();
    expect(screen.getByText('Workflow 3')).toBeDefined();
    expect(screen.queryByText('Workflow 4')).toBeNull();
    expect(screen.queryByText('Workflow 5')).toBeNull();
  });
});
