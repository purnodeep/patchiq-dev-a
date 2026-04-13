import { Button } from '@patchiq/ui';
import { Settings, BookOpen } from 'lucide-react';
import type { CustomFrameworkResponse } from '../../../api/hooks/useCompliance';

interface CustomFrameworkCardProps {
  framework: CustomFrameworkResponse;
  onEdit: (framework: CustomFrameworkResponse) => void;
}

export function CustomFrameworkCard({ framework, onEdit }: CustomFrameworkCardProps) {
  const controlCount = framework.control_count ?? framework.controls?.length ?? 0;
  const scoringLabels: Record<string, string> = {
    average: 'Average',
    strictest: 'Strictest',
    worst_case: 'Worst case',
    weighted: 'Weighted',
  };
  const scoringLabel =
    scoringLabels[framework.scoring_method ?? 'average'] ?? framework.scoring_method;

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 12,
        padding: 16,
        display: 'flex',
        flexDirection: 'column',
        gap: 12,
        position: 'relative',
      }}
    >
      {/* Custom badge */}
      <div
        style={{
          position: 'absolute',
          top: 12,
          right: 12,
          fontFamily: 'var(--font-mono)',
          fontSize: 9,
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          background: 'var(--accent-subtle)',
          color: 'var(--accent)',
          border: '1px solid var(--accent)',
          borderRadius: 4,
          padding: '2px 6px',
          opacity: 0.9,
        }}
      >
        Custom
      </div>

      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10, paddingRight: 56 }}>
        <div
          style={{
            width: 36,
            height: 36,
            borderRadius: 8,
            background: 'var(--bg-inset)',
            border: '1px solid var(--border)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexShrink: 0,
          }}
        >
          <BookOpen size={16} style={{ color: 'var(--text-secondary)' }} />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 13,
              fontWeight: 600,
              color: 'var(--text-primary)',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {framework.name}
          </div>
          <div
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 10,
              color: 'var(--text-muted)',
              marginTop: 2,
            }}
          >
            v{framework.version}
          </div>
        </div>
      </div>

      {/* Description */}
      {framework.description && (
        <div
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 12,
            color: 'var(--text-secondary)',
            lineHeight: 1.5,
            display: '-webkit-box',
            WebkitLineClamp: 2,
            WebkitBoxOrient: 'vertical',
            overflow: 'hidden',
          }}
        >
          {framework.description}
        </div>
      )}

      {/* Stats row */}
      <div
        style={{
          display: 'flex',
          gap: 12,
          borderTop: '1px solid var(--border)',
          paddingTop: 10,
        }}
      >
        <div>
          <div
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 16,
              fontWeight: 700,
              color: 'var(--text-emphasis)',
              lineHeight: 1,
            }}
          >
            {controlCount}
          </div>
          <div
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 10,
              color: 'var(--text-muted)',
              marginTop: 2,
              textTransform: 'uppercase',
              letterSpacing: '0.04em',
            }}
          >
            Controls
          </div>
        </div>
        <div
          style={{
            width: 1,
            background: 'var(--border)',
            alignSelf: 'stretch',
          }}
        />
        <div>
          <div
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              fontWeight: 600,
              color: 'var(--text-primary)',
              lineHeight: 1,
              textTransform: 'capitalize',
            }}
          >
            {scoringLabel}
          </div>
          <div
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 10,
              color: 'var(--text-muted)',
              marginTop: 2,
              textTransform: 'uppercase',
              letterSpacing: '0.04em',
            }}
          >
            Scoring
          </div>
        </div>
      </div>

      {/* Actions */}
      <Button
        variant="outline"
        size="sm"
        onClick={() => onEdit(framework)}
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 11,
          width: '100%',
          justifyContent: 'center',
        }}
      >
        <Settings style={{ width: 12, height: 12, marginRight: 6 }} />
        Edit Framework
      </Button>
    </div>
  );
}
