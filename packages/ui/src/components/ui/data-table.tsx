import * as React from 'react';
import { cn } from '@/lib/utils';

type DataTableProps = React.HTMLAttributes<HTMLTableElement>;

const DataTable = React.forwardRef<HTMLTableElement, DataTableProps>(
  ({ className, ...props }, ref) => (
    <div className="relative w-full overflow-auto">
      <table ref={ref} className={cn('w-full caption-bottom text-sm', className)} {...props} />
    </div>
  ),
);
DataTable.displayName = 'DataTable';

export { DataTable };
