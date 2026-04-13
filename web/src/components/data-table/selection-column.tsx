import type { ColumnDef } from '@tanstack/react-table';

/**
 * Creates a checkbox selection column for use with TanStack Table.
 * Add this as the first column in your column definitions.
 */
export function createSelectionColumn<TData>(): ColumnDef<TData> {
  return {
    id: 'select',
    header: ({ table }) => (
      <input
        type="checkbox"
        checked={table.getIsAllPageRowsSelected()}
        onChange={table.getToggleAllPageRowsSelectedHandler()}
        style={{
          width: 14,
          height: 14,
          borderRadius: 3,
          cursor: 'pointer',
          accentColor: 'var(--accent)',
        }}
        aria-label="Select all"
      />
    ),
    cell: ({ row }) => (
      <input
        type="checkbox"
        checked={row.getIsSelected()}
        disabled={!row.getCanSelect()}
        onChange={row.getToggleSelectedHandler()}
        onClick={(e) => e.stopPropagation()}
        style={{
          width: 14,
          height: 14,
          borderRadius: 3,
          cursor: 'pointer',
          accentColor: 'var(--accent)',
        }}
        aria-label="Select row"
      />
    ),
    enableSorting: false,
    enableHiding: false,
    size: 40,
  };
}
