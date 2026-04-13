import { render, screen, fireEvent } from '@testing-library/react';
import { DataTablePagination } from '../../../components/data-table/DataTablePagination';

describe('DataTablePagination', () => {
  it('renders next button enabled when hasNext', () => {
    render(<DataTablePagination hasNext onNext={() => {}} hasPrev={false} onPrev={() => {}} />);
    expect(screen.getByRole('button', { name: /next/i })).not.toBeDisabled();
  });

  it('disables prev button when hasPrev is false', () => {
    render(
      <DataTablePagination hasNext={false} onNext={() => {}} hasPrev={false} onPrev={() => {}} />,
    );
    expect(screen.getByRole('button', { name: /previous/i })).toBeDisabled();
  });

  it('calls onNext when next clicked', () => {
    const onNext = vi.fn();
    render(<DataTablePagination hasNext onNext={onNext} hasPrev={false} onPrev={() => {}} />);
    fireEvent.click(screen.getByRole('button', { name: /next/i }));
    expect(onNext).toHaveBeenCalledOnce();
  });
});
