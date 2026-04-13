import { render, screen } from '@testing-library/react';
import { KEVBadge } from '../../components/KEVBadge';

describe('KEVBadge', () => {
  it('renders KEV badge with "Yes" when due date is provided', () => {
    render(<KEVBadge dueDate="2026-04-01" />);
    expect(screen.getByText('Yes')).toBeInTheDocument();
  });

  it('renders em dash when due date is null', () => {
    render(<KEVBadge dueDate={null} />);
    expect(screen.getByText('—')).toBeInTheDocument();
  });

  it('shows due date in title', () => {
    render(<KEVBadge dueDate="2026-04-01" />);
    expect(screen.getByTitle(/2026-04-01/)).toBeInTheDocument();
  });
});
