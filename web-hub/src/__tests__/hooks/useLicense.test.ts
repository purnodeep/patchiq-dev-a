import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import React from 'react';
import { useLicense } from '../../api/hooks/useLicenses';

vi.mock('../../api/fetch', () => ({
  apiFetch: vi.fn(),
}));

import { apiFetch } from '../../api/fetch';

const mockApiFetch = vi.mocked(apiFetch);

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return React.createElement(QueryClientProvider, { client: qc }, children);
}

const mockLicense = {
  id: 'lic-1',
  client_id: null,
  client_hostname: null,
  tier: 'enterprise' as const,
  max_endpoints: 500,
  issued_at: '2026-01-01T00:00:00Z',
  expires_at: '2027-01-01T00:00:00Z',
  revoked_at: null,
  customer_name: 'Acme Corp',
  customer_email: 'admin@acme.com',
  notes: null,
  license_key: 'XXXX-XXXX-ABCD',
  client_endpoint_count: null,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

describe('useLicense', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('unwraps {license: ...} envelope and returns license data', async () => {
    mockApiFetch.mockResolvedValue({
      json: () => Promise.resolve({ license: mockLicense }),
    } as Response);

    const { result } = renderHook(() => useLicense('lic-1'), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.customer_name).toBe('Acme Corp');
    expect(result.current.data?.tier).toBe('enterprise');
    expect(result.current.data?.max_endpoints).toBe(500);
  });

  it('throws when envelope is missing', async () => {
    mockApiFetch.mockResolvedValue({
      json: () => Promise.resolve({ wrong_key: {} }),
    } as Response);

    const { result } = renderHook(() => useLicense('lic-1'), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });

  it('does not fetch when id is empty string', () => {
    const { result } = renderHook(() => useLicense(''), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});
