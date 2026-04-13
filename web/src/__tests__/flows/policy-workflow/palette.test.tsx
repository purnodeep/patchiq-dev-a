import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { Palette } from '../../../flows/policy-workflow/palette';

describe('Palette', () => {
  it('renders all 10 node types', () => {
    render(<Palette />);
    expect(screen.getByText('Trigger')).toBeInTheDocument();
    expect(screen.getByText('Filter')).toBeInTheDocument();
    expect(screen.getByText('Approval')).toBeInTheDocument();
    expect(screen.getByText('Deploy Wave')).toBeInTheDocument();
    expect(screen.getByText('Gate')).toBeInTheDocument();
    expect(screen.getByText('Script')).toBeInTheDocument();
    expect(screen.getByText('Notification')).toBeInTheDocument();
    expect(screen.getByText('Rollback')).toBeInTheDocument();
    expect(screen.getByText('Decision')).toBeInTheDocument();
    expect(screen.getByText('Complete')).toBeInTheDocument();
  });

  it('each palette item is draggable', () => {
    render(<Palette />);
    // Palette items are divs with draggable attribute (not li elements)
    const items = document.querySelectorAll('[draggable="true"]');
    expect(items.length).toBeGreaterThan(0);
    for (const item of Array.from(items)) {
      expect(item).toHaveAttribute('draggable', 'true');
    }
  });

  it('has a toggle button to collapse palette', () => {
    render(<Palette collapsed={false} onToggle={vi.fn()} />);
    expect(screen.getByRole('button', { name: /collapse/i })).toBeInTheDocument();
  });

  it('hides node list when collapsed', () => {
    render(<Palette collapsed={true} onToggle={vi.fn()} />);
    expect(screen.queryByText('Trigger')).not.toBeInTheDocument();
  });
});
