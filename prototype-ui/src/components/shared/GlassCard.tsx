import { cn } from '@/lib/utils';

interface GlassCardProps extends React.HTMLAttributes<HTMLDivElement> {
  hover?: boolean;
}

export function GlassCard({ className, hover = true, children, ...props }: GlassCardProps) {
  return (
    <div className={cn('glass-card', hover && 'glass-card-hover', className)} {...props}>
      {children}
    </div>
  );
}
