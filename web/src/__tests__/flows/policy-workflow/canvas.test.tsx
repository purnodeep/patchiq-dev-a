import { render, screen } from '@testing-library/react';
import { ReactFlowProvider } from '@xyflow/react';
import { describe, it, expect, vi } from 'vitest';
import { WorkflowCanvas } from '../../../flows/policy-workflow/canvas';

vi.mock('@xyflow/react', async () => {
  const actual = await vi.importActual('@xyflow/react');
  return {
    ...actual,
    ReactFlow: ({ children }: { children?: React.ReactNode }) => (
      <div data-testid="react-flow">{children}</div>
    ),
    MiniMap: () => <div data-testid="minimap" />,
    Controls: () => <div data-testid="controls" />,
    Background: () => <div data-testid="background" />,
  };
});

describe('WorkflowCanvas', () => {
  it('renders react flow container', () => {
    render(
      <ReactFlowProvider>
        <WorkflowCanvas
          nodes={[]}
          edges={[]}
          onNodesChange={() => {}}
          onEdgesChange={() => {}}
          onNodeClick={() => {}}
        />
      </ReactFlowProvider>,
    );
    expect(screen.getByTestId('react-flow')).toBeInTheDocument();
  });

  it('renders controls and background', () => {
    render(
      <ReactFlowProvider>
        <WorkflowCanvas
          nodes={[]}
          edges={[]}
          onNodesChange={() => {}}
          onEdgesChange={() => {}}
          onNodeClick={() => {}}
        />
      </ReactFlowProvider>,
    );
    expect(screen.getByTestId('controls')).toBeInTheDocument();
    expect(screen.getByTestId('background')).toBeInTheDocument();
  });
});
