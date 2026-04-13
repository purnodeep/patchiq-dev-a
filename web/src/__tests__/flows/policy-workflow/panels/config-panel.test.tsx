import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { ConfigPanel } from '../../../../flows/policy-workflow/panels/config-panel';

describe('ConfigPanel', () => {
  it('renders trigger panel fields', () => {
    render(
      <ConfigPanel
        nodeType="trigger"
        nodeLabel="My Trigger"
        config={{ trigger_type: 'manual' }}
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );
    expect(screen.getByText('My Trigger')).toBeInTheDocument();
    expect(screen.getByLabelText(/trigger type/i)).toBeInTheDocument();
  });

  it('renders approval panel fields', () => {
    render(
      <ConfigPanel
        nodeType="approval"
        nodeLabel="Approval"
        config={{ approver_roles: [], timeout_hours: 24 }}
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );
    expect(screen.getByLabelText(/timeout hours/i)).toBeInTheDocument();
  });

  it('renders complete panel as read-only summary', () => {
    render(
      <ConfigPanel
        nodeType="complete"
        nodeLabel="Done"
        config={{ generate_report: true, notify_on_complete: false }}
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );
    expect(screen.getByText('Done')).toBeInTheDocument();
    expect(screen.getByText(/generate report/i)).toBeInTheDocument();
    expect(screen.getByText('Yes')).toBeInTheDocument();
  });

  it('renders filter panel fields', () => {
    render(
      <ConfigPanel
        nodeType="filter"
        nodeLabel="OS Filter"
        config={{ os_types: ['linux'] }}
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );
    expect(screen.getByText('OS Filter')).toBeInTheDocument();
    expect(screen.getByLabelText(/min severity/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/os types/i)).toBeInTheDocument();
  });

  it('renders wave panel fields', () => {
    render(
      <ConfigPanel
        nodeType="deployment_wave"
        nodeLabel="Wave 1"
        config={{ percentage: 10 }}
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );
    expect(screen.getByLabelText(/percentage/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/max parallel/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/timeout/i)).toBeInTheDocument();
  });

  it('renders gate panel fields', () => {
    render(
      <ConfigPanel
        nodeType="gate"
        nodeLabel="Health Gate"
        config={{ wait_minutes: 60, failure_threshold: 5 }}
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );
    expect(screen.getByLabelText(/wait minutes/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/failure threshold/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/health check/i)).toBeInTheDocument();
  });

  it('renders script panel fields', () => {
    render(
      <ConfigPanel
        nodeType="script"
        nodeLabel="Pre-check"
        config={{
          script_body: '',
          script_type: 'shell',
          timeout_minutes: 5,
          failure_behavior: 'halt',
        }}
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );
    expect(screen.getByLabelText(/script type/i)).toBeInTheDocument();
    expect(screen.getByTestId('script-editor')).toBeInTheDocument();
  });

  it('renders notification panel fields', () => {
    render(
      <ConfigPanel
        nodeType="notification"
        nodeLabel="Alert"
        config={{ channel: 'slack', target: '#ops' }}
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );
    expect(screen.getByLabelText(/channel/i)).toBeInTheDocument();
  });

  it('renders rollback panel fields', () => {
    render(
      <ConfigPanel
        nodeType="rollback"
        nodeLabel="Rollback"
        config={{ strategy: 'snapshot_restore', failure_threshold: 10 }}
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );
    expect(screen.getByLabelText(/strategy/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/target deployment/i)).toBeInTheDocument();
  });

  it('renders decision panel fields', () => {
    render(
      <ConfigPanel
        nodeType="decision"
        nodeLabel="Branch"
        config={{ field: 'os', operator: 'equals', value: 'linux' }}
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );
    expect(screen.getByLabelText(/field/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/true label/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/false label/i)).toBeInTheDocument();
  });

  it('script panel shows validation error for empty body', async () => {
    render(
      <ConfigPanel
        nodeType="script"
        nodeLabel="Pre-check"
        config={{
          script_body: '',
          script_type: 'shell',
          timeout_minutes: 5,
          failure_behavior: 'halt',
        }}
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );
    const saveButton = screen.getByRole('button', { name: /save/i });
    fireEvent.click(saveButton);
    await waitFor(() => {
      expect(screen.getByText(/script body is required/i)).toBeInTheDocument();
    });
  });

  it('trigger panel shows cron field when cron type selected', () => {
    render(
      <ConfigPanel
        nodeType="trigger"
        nodeLabel="Cron Trigger"
        config={{ trigger_type: 'cron', cron_expression: '0 * * * *' }}
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );
    expect(screen.getByLabelText(/cron expression/i)).toBeInTheDocument();
  });

  it('rollback panel shows script field for script strategy', () => {
    render(
      <ConfigPanel
        nodeType="rollback"
        nodeLabel="Rollback"
        config={{ strategy: 'script', failure_threshold: 10 }}
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
      />,
    );
    expect(screen.getByLabelText(/rollback script/i)).toBeInTheDocument();
  });
});
