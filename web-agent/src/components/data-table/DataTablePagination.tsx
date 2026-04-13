import { Button } from '@patchiq/ui';
import { ChevronLeft, ChevronRight } from 'lucide-react';

interface DataTablePaginationProps {
  pageIndex: number;
  pageCount: number;
  hasNext: boolean;
  hasPrev: boolean;
  onNext: () => void;
  onPrev: () => void;
}

export const DataTablePagination = ({
  pageIndex,
  pageCount,
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
    <span
      style={{
        marginRight: 'auto',
        fontSize: 12,
        color: 'var(--text-muted)',
        fontFamily: 'var(--font-sans)',
      }}
    >
      Page {pageIndex + 1} of {pageCount}
    </span>
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
