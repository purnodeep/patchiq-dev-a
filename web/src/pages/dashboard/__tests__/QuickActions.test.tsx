import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { MemoryRouter } from 'react-router';
import { QuickActions } from '../QuickActions';

describe('QuickActions', () => {
  it('renders action buttons', () => {
    render(
      <MemoryRouter>
        <QuickActions />
      </MemoryRouter>,
    );
    expect(screen.getByText('New Deployment')).toBeInTheDocument();
    expect(screen.getByText('Scan All')).toBeInTheDocument();
    expect(screen.getByText('Review Critical')).toBeInTheDocument();
  });
});
