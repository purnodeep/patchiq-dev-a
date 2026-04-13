import { useEffect, useState } from 'react';
import { ENDPOINT_COVERAGE_DATA } from '@/data/mock-data';

function colorForPct(pct: number): string {
  if (pct >= 95) return 'var(--color-success)';
  if (pct >= 80) return 'var(--color-warning)';
  return 'var(--color-danger)';
}

export function EndpointCoverageMap() {
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    const id = requestAnimationFrame(() => setMounted(true));
    return () => cancelAnimationFrame(id);
  }, []);

  const totalEndpoints = ENDPOINT_COVERAGE_DATA.reduce((sum, d) => sum + d.total, 0);
  const totalCovered = ENDPOINT_COVERAGE_DATA.reduce((sum, d) => sum + d.covered, 0);

  return (
    <div
      style={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        fontFamily: 'var(--font-sans)',
      }}
    >
      {/* Row list */}
      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          minHeight: 0,
          overflowY: 'auto',
        }}
      >
        {ENDPOINT_COVERAGE_DATA.map((dept, i) => {
          const color = colorForPct(dept.pct);
          return (
            <div
              key={dept.dept}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '5px 0',
                borderBottom:
                  i < ENDPOINT_COVERAGE_DATA.length - 1
                    ? '1px solid var(--color-separator)'
                    : undefined,
              }}
            >
              {/* Dept name */}
              <span
                style={{
                  width: 100,
                  flexShrink: 0,
                  fontSize: 10,
                  fontWeight: 700,
                  color: 'var(--color-foreground)',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
                title={dept.dept}
              >
                {dept.dept}
              </span>

              {/* Progress bar */}
              <div
                style={{
                  flex: 1,
                  height: 8,
                  borderRadius: 4,
                  background: 'var(--color-separator)',
                  overflow: 'hidden',
                  position: 'relative',
                }}
              >
                <div
                  style={{
                    height: '100%',
                    borderRadius: 4,
                    background: color,
                    width: mounted ? `${dept.pct}%` : '0%',
                    transition: `width 0.6s cubic-bezier(0.34,1.56,0.64,1)`,
                    transitionDelay: `${i * 50}ms`,
                    boxShadow: `0 0 5px ${color === 'var(--color-success)' ? '#34c75955' : color === 'var(--color-warning)' ? '#ff950055' : '#ff3b3055'}`,
                  }}
                />
              </div>

              {/* X/Y covered count */}
              <span
                style={{
                  width: 48,
                  flexShrink: 0,
                  fontSize: 9,
                  color: 'var(--color-muted)',
                  textAlign: 'right',
                  whiteSpace: 'nowrap',
                }}
              >
                {dept.covered}/{dept.total}
              </span>

              {/* Pct badge */}
              <span
                style={{
                  flexShrink: 0,
                  fontSize: 9,
                  fontWeight: 700,
                  color: color,
                  background: `color-mix(in srgb, ${color} 12%, transparent)`,
                  border: `1px solid color-mix(in srgb, ${color} 30%, transparent)`,
                  borderRadius: 10,
                  padding: '1px 6px',
                  minWidth: 34,
                  textAlign: 'center',
                }}
              >
                {dept.pct}%
              </span>
            </div>
          );
        })}
      </div>

      {/* Summary footer */}
      <div
        style={{
          borderTop: '1px solid var(--color-separator)',
          paddingTop: 8,
          marginTop: 4,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>
          {totalCovered}/{totalEndpoints} endpoints have agent coverage
        </span>
        <span
          style={{
            fontSize: 12,
            fontWeight: 700,
            color: colorForPct(Math.round((totalCovered / totalEndpoints) * 100)),
          }}
        >
          {Math.round((totalCovered / totalEndpoints) * 100)}%
        </span>
      </div>
    </div>
  );
}
