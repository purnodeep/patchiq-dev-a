import { ProgressBar } from './progress-bar';
import type { CategoryBreakdown } from '../../../api/hooks/useCompliance';

function getScoreColor(score: number): string {
  if (score >= 95) return 'var(--accent)';
  if (score >= 80) return 'var(--signal-warning)';
  return 'var(--signal-critical)';
}

interface CategoryCardProps {
  category: CategoryBreakdown;
}

export function CategoryCard({ category }: CategoryCardProps) {
  const score = Math.round(category.score);
  const scoreColor = getScoreColor(score);

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        padding: 16,
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 8,
        }}
      >
        <span
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 13,
            fontWeight: 600,
            color: 'var(--text-primary)',
          }}
        >
          {category.category}
        </span>
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 16,
            fontWeight: 700,
            color: scoreColor,
          }}
        >
          {score}%
        </span>
      </div>
      <ProgressBar value={score} max={100} color={scoreColor} />
      <div
        style={{
          display: 'flex',
          gap: 16,
          marginTop: 8,
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          color: 'var(--text-muted)',
        }}
      >
        <span>
          <span style={{ color: 'var(--accent)', fontWeight: 600 }}>
            {category.passing_controls}
          </span>{' '}
          pass
        </span>
        <span>
          <span
            style={{
              color: category.failing_controls > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
              fontWeight: 600,
            }}
          >
            {category.failing_controls}
          </span>{' '}
          fail
        </span>
        <span>
          <span style={{ fontWeight: 600 }}>{category.na_controls}</span> n/a
        </span>
      </div>
    </div>
  );
}
