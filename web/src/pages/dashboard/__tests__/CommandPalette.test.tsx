import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeAll } from 'vitest';
import { MemoryRouter } from 'react-router';
import { CommandPalette } from '../CommandPalette';

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn();
});

describe('CommandPalette', () => {
  it('renders when open', () => {
    render(
      <MemoryRouter>
        <CommandPalette open onOpenChange={() => {}} />
      </MemoryRouter>,
    );
    expect(screen.getByPlaceholderText(/Search everything/)).toBeInTheDocument();
  });

  it('does not render when closed', () => {
    render(
      <MemoryRouter>
        <CommandPalette open={false} onOpenChange={() => {}} />
      </MemoryRouter>,
    );
    expect(screen.queryByPlaceholderText(/Search everything/)).not.toBeInTheDocument();
  });

  it('calls onOpenChange when Escape pressed', () => {
    const onOpenChange = vi.fn();
    render(
      <MemoryRouter>
        <CommandPalette open onOpenChange={onOpenChange} />
      </MemoryRouter>,
    );
    fireEvent.keyDown(screen.getByPlaceholderText(/Search everything/), { key: 'Escape' });
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
