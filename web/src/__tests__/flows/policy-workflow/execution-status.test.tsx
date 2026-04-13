import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi } from 'vitest';
import { ExecutionStatusPanel } from '../../../flows/policy-workflow/execution-status';

vi.mock('../../../flows/policy-workflow/hooks/use-workflow-executions', () => ({
  useWorkflowExecution: () => ({
    data: {
      id: 'e1',
      status: 'running',
      node_executions: [
        { id: 'ne1', node_id: 'n1', status: 'completed', error_message: '' },
        { id: 'ne2', node_id: 'n2', status: 'running', error_message: '' },
        { id: 'ne3', node_id: 'n3', status: 'pending', error_message: '' },
      ],
    },
    isLoading: false,
  }),
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

describe('ExecutionStatusPanel', () => {
  it('renders execution status', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <ExecutionStatusPanel workflowId="w1" executionId="e1" />
      </QueryClientProvider>,
    );
    expect(screen.getAllByText('running').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('Execution Status')).toBeInTheDocument();
  });

  it('shows node execution statuses', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <ExecutionStatusPanel workflowId="w1" executionId="e1" />
      </QueryClientProvider>,
    );
    expect(screen.getByText('completed')).toBeInTheDocument();
    expect(screen.getByText('pending')).toBeInTheDocument();
  });
});
