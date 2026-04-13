import { cn } from '../lib/utils';

type RiskLevel = 'critical' | 'high' | 'medium' | 'healthy';

interface DotMosaicItem {
  id: string;
  risk: RiskLevel;
}

interface DotMosaicProps {
  data: DotMosaicItem[];
  className?: string;
}

const RISK_COLORS: Record<RiskLevel, string> = {
  critical: 'var(--signal-critical)',
  high: 'var(--signal-warning)',
  medium: 'var(--text-muted)',
  healthy: 'var(--signal-healthy)',
};

function buildAriaLabel(data: DotMosaicItem[]): string {
  const counts: Partial<Record<RiskLevel, number>> = {};
  for (const item of data) {
    counts[item.risk] = (counts[item.risk] ?? 0) + 1;
  }
  const parts = (Object.entries(counts) as [RiskLevel, number][]).map(
    ([risk, count]) => `${count} ${risk}`,
  );
  return `Endpoint distribution: ${parts.join(', ')}`;
}

function DotMosaic({ data, className }: DotMosaicProps) {
  return (
    <div
      role="img"
      aria-label={buildAriaLabel(data)}
      className={cn('flex flex-wrap', className)}
      style={{ gap: 2 }}
    >
      {data.map((item) => (
        <div
          key={item.id}
          data-slot="dot"
          title={item.id}
          className="rounded-full"
          style={{
            width: 6,
            height: 6,
            backgroundColor: RISK_COLORS[item.risk],
          }}
        />
      ))}
    </div>
  );
}

export { DotMosaic };
export type { DotMosaicProps, DotMosaicItem, RiskLevel };
