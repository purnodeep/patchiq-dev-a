import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  useWorkflows,
  useWorkflow,
  useWorkflowTemplates,
} from '../../../../flows/policy-workflow/hooks/use-workflows';
import type { ReactNode } from 'react';

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

const wrapper = ({ children }: { children: ReactNode }) => (
  <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
);

beforeEach(() => {
  queryClient.clear();
  vi.restoreAllMocks();
});

describe('useWorkflows', () => {
  it('fetches workflow list', async () => {
    const mockData = { data: [{ id: 'w1', name: 'Test' }], next_cursor: null, total_count: 1 };
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify(mockData), { status: 200 }),
    );

    const { result } = renderHook(() => useWorkflows(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.data).toHaveLength(1);
  });
});

describe('useWorkflow', () => {
  it('fetches single workflow', async () => {
    const mockData = { id: 'w1', name: 'Test', nodes: [], edges: [] };
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify(mockData), { status: 200 }),
    );

    const { result } = renderHook(() => useWorkflow('w1'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.id).toBe('w1');
  });
});

describe('useWorkflowTemplates', () => {
  it('fetches templates', async () => {
    const mockData = [{ id: 'critical-fast-track', name: 'Critical Patch Fast-Track' }];
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify(mockData), { status: 200 }),
    );

    const { result } = renderHook(() => useWorkflowTemplates(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
  });
});
