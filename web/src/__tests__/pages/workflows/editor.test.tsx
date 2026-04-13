import { render, screen } from '@testing-library/react';
import { createMemoryRouter, RouterProvider } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi } from 'vitest';

vi.mock('react-router', async () => {
  const actual = await vi.importActual('react-router');
  return {
    ...actual,
    useBlocker: () => ({ state: 'unblocked', proceed: vi.fn(), reset: vi.fn() }),
  };
});

vi.mock('@xyflow/react', async () => {
  const actual = await vi.importActual('@xyflow/react');
  return {
    ...actual,
    ReactFlow: ({ children }: { children?: React.ReactNode }) => (
      <div data-testid="react-flow">{children}</div>
    ),
    ReactFlowProvider: ({ children }: { children?: React.ReactNode }) => <div>{children}</div>,
    MiniMap: () => <div data-testid="minimap" />,
    Controls: () => <div data-testid="controls" />,
    Background: () => <div data-testid="background" />,
    useNodesState: () => [[], vi.fn(), vi.fn()],
    useEdgesState: () => [[], vi.fn(), vi.fn()],
    useReactFlow: () => ({
      screenToFlowPosition: vi.fn(),
      setNodes: vi.fn(),
      setEdges: vi.fn(),
      getNodes: () => [],
      getEdges: () => [],
    }),
  };
});

vi.mock('../../../flows/policy-workflow/hooks/use-workflows', () => ({
  useWorkflow: () => ({ data: null, isLoading: false, isError: false, refetch: vi.fn() }),
  useCreateWorkflow: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkflow: () => ({ mutateAsync: vi.fn(), isPending: false }),
  usePublishWorkflow: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useWorkflowTemplates: () => ({ data: [], isLoading: false }),
}));

vi.mock('../../../flows/policy-workflow/hooks/use-elk-layout', () => ({
  computeLayout: vi.fn().mockResolvedValue({ nodes: [], edges: [] }),
}));

// Must import AFTER mocks
const { WorkflowEditorPage } = await import('../../../pages/workflows/editor');

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

function renderEditor() {
  const router = createMemoryRouter([{ path: '*', element: <WorkflowEditorPage /> }], {
    initialEntries: ['/workflows/new'],
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

describe('WorkflowEditorPage', () => {
  it('renders palette and canvas', () => {
    renderEditor();
    expect(screen.getByText('Nodes')).toBeInTheDocument();
    expect(screen.getByTestId('react-flow')).toBeInTheDocument();
  });

  it('renders save button', () => {
    renderEditor();
    // In create mode (no :id param) the button says "Publish"; in edit mode it says "Save"
    expect(screen.getByRole('button', { name: /save|publish/i })).toBeInTheDocument();
  });
});
