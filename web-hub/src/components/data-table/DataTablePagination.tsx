import { Button } from '@patchiq/ui';
import { ChevronLeft, ChevronRight } from 'lucide-react';

interface DataTablePaginationProps {
  hasNext: boolean;
  hasPrev: boolean;
  onNext: () => void;
  onPrev: () => void;
}

export const DataTablePagination = ({
  hasNext,
  hasPrev,
  onNext,
  onPrev,
}: DataTablePaginationProps) => (
  <div
    style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'flex-end',
      gap: 8,
      padding: '12px 0',
    }}
  >
    <Button
      variant="outline"
      size="sm"
      onClick={onPrev}
      disabled={!hasPrev}
      aria-label="Previous page"
    >
      <ChevronLeft style={{ width: 14, height: 14 }} />
      Previous
    </Button>
    <Button variant="outline" size="sm" onClick={onNext} disabled={!hasNext} aria-label="Next page">
      Next
      <ChevronRight style={{ width: 14, height: 14 }} />
    </Button>
  </div>
);
