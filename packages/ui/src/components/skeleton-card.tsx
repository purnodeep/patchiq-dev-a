import { cn } from '../lib/utils';

interface SkeletonCardProps {
  className?: string;
  /** Number of shimmer lines to render */
  lines?: number;
}

const LINE_WIDTHS = ['75%', '100%', '60%', '85%', '50%'];

function SkeletonCard({ className, lines = 3 }: SkeletonCardProps) {
  return (
    <div
      className={cn('rounded-lg p-4', className)}
      style={{
        backgroundColor: 'var(--skel-base)',
        borderWidth: '1px',
        borderStyle: 'solid',
        borderColor: 'var(--border-faint)',
      }}
    >
      {Array.from({ length: lines }, (_, i) => (
        <div
          key={i}
          className="rounded"
          style={{
            height: 12,
            marginTop: i > 0 ? 10 : 0,
            width: LINE_WIDTHS[i % LINE_WIDTHS.length],
            background: `linear-gradient(90deg, var(--skel-base) 25%, var(--skel-highlight) 50%, var(--skel-base) 75%)`,
            backgroundSize: '200% 100%',
            animation: `shimmer var(--shimmer-duration) infinite linear`,
          }}
          data-slot="shimmer-line"
        />
      ))}
    </div>
  );
}

export { SkeletonCard };
export type { SkeletonCardProps };
