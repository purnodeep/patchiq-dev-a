import { render, screen } from '@testing-library/react';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { DataTable } from '../../../components/data-table/DataTable';

type TestRow = { id: string; name: string };
const columnHelper = createColumnHelper<TestRow>();
const columns = [columnHelper.accessor('name', { header: 'Name' })];

const Wrapper = ({ data }: { data: TestRow[] }) => {
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });
  return <DataTable table={table} />;
};

describe('DataTable', () => {
  it('renders column headers', () => {
    render(<Wrapper data={[{ id: '1', name: 'Alice' }]} />);
    expect(screen.getByText('Name')).toBeInTheDocument();
  });

  it('renders row data', () => {
    render(<Wrapper data={[{ id: '1', name: 'Alice' }]} />);
    expect(screen.getByText('Alice')).toBeInTheDocument();
  });

  it('renders empty message when no data', () => {
    render(<Wrapper data={[]} />);
    expect(screen.getByText('No results.')).toBeInTheDocument();
  });
});
