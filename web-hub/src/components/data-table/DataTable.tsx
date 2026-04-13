import { Fragment } from 'react';
import { flexRender, type Table as TanstackTable } from '@tanstack/react-table';

interface DataTableProps<TData> {
  table: TanstackTable<TData>;
  onRowClick?: (row: TData) => void;
  renderExpandedRow?: (row: TData) => React.ReactNode;
  isRowFailed?: (row: TData) => boolean;
}

export function DataTable<TData>({
  table,
  onRowClick,
  renderExpandedRow,
  isRowFailed,
}: DataTableProps<TData>) {
  return (
    <div
      style={{
        position: 'relative',
        width: '100%',
        overflowX: 'auto',
        borderRadius: 8,
        border: '1px solid var(--border)',
      }}
    >
      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
        <thead>
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <th
                  key={header.id}
                  style={{
                    height: 40,
                    padding: '0 16px',
                    textAlign: 'left',
                    verticalAlign: 'middle',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 11,
                    fontWeight: 600,
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    color: 'var(--text-muted)',
                    background: 'var(--bg-inset)',
                    borderBottom: '1px solid var(--border)',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, header.getContext())}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody>
          {table.getRowModel().rows.length ? (
            table.getRowModel().rows.map((row) => {
              const failed = isRowFailed?.(row.original) ?? false;
              return (
                <Fragment key={row.id}>
                  <tr
                    style={{
                      borderBottom: '1px solid var(--border)',
                      borderLeft: failed ? '2px solid var(--signal-critical)' : undefined,
                      background: row.getIsSelected()
                        ? 'color-mix(in srgb, var(--accent) 4%, transparent)'
                        : undefined,
                      cursor: onRowClick ? 'pointer' : undefined,
                      transition: 'background 0.1s ease',
                    }}
                    onClick={() => onRowClick?.(row.original)}
                    onMouseEnter={(e) => {
                      if (!row.getIsSelected()) {
                        (e.currentTarget as HTMLTableRowElement).style.background =
                          'var(--bg-card-hover)';
                      }
                    }}
                    onMouseLeave={(e) => {
                      if (!row.getIsSelected()) {
                        (e.currentTarget as HTMLTableRowElement).style.background = '';
                      }
                    }}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <td
                        key={cell.id}
                        style={{
                          padding: '10px 16px',
                          verticalAlign: 'middle',
                          fontFamily: 'var(--font-sans)',
                          color: 'var(--text-primary)',
                        }}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    ))}
                  </tr>
                  {renderExpandedRow && row.getIsExpanded() && (
                    <tr
                      style={{
                        borderBottom: '1px solid var(--border)',
                        background: 'var(--bg-inset)',
                      }}
                    >
                      <td colSpan={row.getAllCells().length} style={{ padding: 0 }}>
                        {renderExpandedRow(row.original)}
                      </td>
                    </tr>
                  )}
                </Fragment>
              );
            })
          ) : (
            <tr>
              <td
                colSpan={table.getAllColumns().length}
                style={{
                  height: 96,
                  textAlign: 'center',
                  fontFamily: 'var(--font-sans)',
                  fontSize: 13,
                  color: 'var(--text-muted)',
                }}
              >
                No results.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
