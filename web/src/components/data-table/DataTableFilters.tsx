interface DataTableFiltersProps {
  children: React.ReactNode;
  className?: string;
}

export const DataTableFilters = ({ children }: DataTableFiltersProps) => (
  <div
    style={{
      display: 'flex',
      flexWrap: 'wrap',
      alignItems: 'center',
      gap: 8,
    }}
  >
    {children}
  </div>
);
