import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import { IconRail } from '../app/layout/IconRail';

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

const navLabels = [
  'Overview',
  'Patches',
  'Hardware',
  'Software',
  'Services',
  'History',
  'Logs',
  'Settings',
];

describe('IconRail', () => {
  const renderIconRail = () =>
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <IconRail />
        </MemoryRouter>
      </QueryClientProvider>,
    );

  it('renders without crashing', () => {
    const { container } = renderIconRail();
    expect(container).toBeTruthy();
  });

  it.each(navLabels)('renders navigation item: %s', (label) => {
    renderIconRail();
    expect(screen.getByLabelText(label)).toBeInTheDocument();
  });
});
