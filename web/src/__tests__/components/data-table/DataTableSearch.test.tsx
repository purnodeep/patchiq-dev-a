import { render, screen, fireEvent } from '@testing-library/react';
import { DataTableSearch } from '../../../components/data-table/DataTableSearch';

describe('DataTableSearch', () => {
  it('renders input with placeholder', () => {
    render(<DataTableSearch value="" onChange={() => {}} placeholder="Search hosts..." />);
    expect(screen.getByPlaceholderText('Search hosts...')).toBeInTheDocument();
  });

  it('calls onChange when user types', () => {
    const onChange = vi.fn();
    render(<DataTableSearch value="" onChange={onChange} placeholder="Search..." />);
    fireEvent.change(screen.getByPlaceholderText('Search...'), { target: { value: 'web' } });
    expect(onChange).toHaveBeenCalledWith('web');
  });
});
