import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import React from 'react';
import { useClient } from '../../api/hooks/useClients';

vi.mock('../../api/fetch', () => ({
  apiFetch: vi.fn(),
}));

import { apiFetch } from '../../api/fetch';

const mockApiFetch = vi.mocked(apiFetch);

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return React.createElement(QueryClientProvider, { client: qc }, children);
}

describe('useClient', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('unwraps {client: ...} envelope and returns client data', async () => {
    const mockClient = {
      id: '123',
      hostname: 'test-pm.internal',
      status: 'approved',
      endpoint_count: 42,
    };
    mockApiFetch.mockResolvedValue({
      json: () => Promise.resolve({ client: mockClient }),
    } as Response);

    const { result } = renderHook(() => useClient('123'), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.hostname).toBe('test-pm.internal');
    expect(result.current.data?.endpoint_count).toBe(42);
  });

  it('throws when envelope is missing', async () => {
    mockApiFetch.mockResolvedValue({
      json: () => Promise.resolve({ wrong_key: {} }),
    } as Response);

    const { result } = renderHook(() => useClient('123'), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });

  it('does not fetch when id is undefined', () => {
    const { result } = renderHook(() => useClient(undefined), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});
