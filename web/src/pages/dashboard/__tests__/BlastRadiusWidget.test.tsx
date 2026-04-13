import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import { BlastRadiusWidget } from '../BlastRadiusWidget';

vi.mock('@/api/hooks/useDashboard', () => ({
  useBlastRadius: () => ({
    data: {
      cve: { id: '1', cve_id: 'CVE-2024-1234', cvss: 9.8, affected_count: 38 },
      groups: [
        { name: 'Windows Servers', os: 'windows', host_count: 18 },
        { name: 'Ubuntu 22.04', os: 'ubuntu', host_count: 12 },
      ],
    },
    isLoading: false,
    error: null,
  }),
}));

const qc = new QueryClient();

describe('BlastRadiusWidget', () => {
  it('renders CVE ID', () => {
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <BlastRadiusWidget />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(screen.getByText('CVE-2024-1234')).toBeInTheDocument();
  });

  it('renders group nodes', () => {
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <BlastRadiusWidget />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(screen.getByText('Windows Servers')).toBeInTheDocument();
    expect(screen.getByText('Ubuntu 22.04')).toBeInTheDocument();
  });
});
