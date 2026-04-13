import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { SLACountdown } from '../../../pages/dashboard/SLACountdown';

const mockUseSLADeadlines = vi.fn();

vi.mock('../../../api/hooks/useDashboard', () => ({
  useSLADeadlines: () => mockUseSLADeadlines(),
}));

describe('SLACountdown', () => {
  it('renders card title "SLA Countdown"', () => {
    mockUseSLADeadlines.mockReturnValue({ data: [], isLoading: false, isError: false });
    render(<SLACountdown />);
    expect(screen.getByText('SLA Countdown')).toBeInTheDocument();
  });

  it('shows "SLA monitoring not configured" when no data', () => {
    mockUseSLADeadlines.mockReturnValue({ data: [], isLoading: false, isError: false });
    render(<SLACountdown />);
    expect(screen.getByText('SLA monitoring not configured')).toBeInTheDocument();
  });

  it('shows loading state', () => {
    mockUseSLADeadlines.mockReturnValue({ data: undefined, isLoading: true, isError: false });
    render(<SLACountdown />);
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('shows error state when API fails', () => {
    mockUseSLADeadlines.mockReturnValue({ data: undefined, isLoading: false, isError: true });
    render(<SLACountdown />);
    expect(screen.getByText('Failed to load SLA deadlines')).toBeInTheDocument();
  });

  it('renders SLA timers from real data', () => {
    mockUseSLADeadlines.mockReturnValue({
      data: [
        {
          endpoint_id: '11111111-1111-1111-1111-111111111111',
          hostname: 'web-01',
          severity: 'high',
          patch_name: 'CVE-2025-1234',
          remaining_seconds: 3 * 86400,
        },
      ],
      isLoading: false,
      isError: false,
    });
    render(<SLACountdown />);
    expect(screen.getByText('CVE-2025-1234')).toBeInTheDocument();
  });

  it('shows OVERDUE for past deadlines', () => {
    mockUseSLADeadlines.mockReturnValue({
      data: [
        {
          endpoint_id: '22222222-2222-2222-2222-222222222222',
          hostname: 'db-02',
          severity: 'critical',
          patch_name: 'CVE-2025-9999',
          remaining_seconds: -3600,
        },
      ],
      isLoading: false,
      isError: false,
    });
    render(<SLACountdown />);
    expect(screen.getByText('OVERDUE')).toBeInTheDocument();
  });
});
