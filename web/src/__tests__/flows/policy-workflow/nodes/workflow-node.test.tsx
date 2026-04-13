import { render, screen } from '@testing-library/react';
import { ReactFlowProvider } from '@xyflow/react';
import { describe, it, expect } from 'vitest';
import { WorkflowNode } from '../../../../flows/policy-workflow/nodes/workflow-node';
import type { WorkflowNodeType } from '../../../../flows/policy-workflow/types';

const renderNode = (nodeType: WorkflowNodeType, label: string) => {
  const props = {
    id: 'n1',
    data: { nodeType, label, config: {} },
    type: 'workflowNode',
    selected: false,
    isConnectable: true,
    positionAbsoluteX: 0,
    positionAbsoluteY: 0,
    zIndex: 0,
    draggable: false,
    dragging: false,
    selectable: true,
    deletable: true,
  };
  return render(
    <ReactFlowProvider>
      <WorkflowNode {...props} />
    </ReactFlowProvider>,
  );
};

describe('WorkflowNode', () => {
  it('renders trigger node with label', () => {
    renderNode('trigger', 'My Trigger');
    expect(screen.getByText('My Trigger')).toBeInTheDocument();
  });

  it('renders filter node with label', () => {
    renderNode('filter', 'OS Filter');
    expect(screen.getByText('OS Filter')).toBeInTheDocument();
  });

  it('renders type badge', () => {
    renderNode('approval', 'Need Approval');
    expect(screen.getByText('Approval')).toBeInTheDocument();
  });
});
