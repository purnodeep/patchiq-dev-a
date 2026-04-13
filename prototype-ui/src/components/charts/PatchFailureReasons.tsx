import { useEffect, useState } from 'react';
import { PATCH_FAILURE_REASONS_DATA } from '@/data/mock-data';

export function PatchFailureReasons() {
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    const id = requestAnimationFrame(() => setMounted(true));
    return () => cancelAnimationFrame(id);
  }, []);

  const sorted = [...PATCH_FAILURE_REASONS_DATA].sort((a, b) => b.count - a.count);
  const maxCount = sorted.length > 0 ? sorted[0].count : 1;
  const total = sorted.reduce((sum, d) => sum + d.count, 0);

  return (
    <div
      style={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        fontFamily: 'var(--font-sans)',
      }}
    >
      {/* Total header */}
      <div style={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center' }}>
        <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>
          Total failures:{' '}
          <span style={{ fontWeight: 700, color: 'var(--color-danger)' }}>{total}</span>
        </span>
      </div>

      {/* Bar rows */}
      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          gap: 8,
        }}
      >
        {sorted.map((item, i) => {
          const widthPct = maxCount > 0 ? (item.count / maxCount) * 100 : 0;
          return (
            <div
              key={item.reason}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                opacity: mounted ? 1 : 0,
                transform: mounted ? 'translateX(0)' : 'translateX(-6px)',
                transition: `opacity 0.3s ease, transform 0.3s ease`,
                transitionDelay: `${i * 55}ms`,
              }}
            >
              {/* Reason label */}
              <span
                style={{
                  width: 140,
                  flexShrink: 0,
                  fontSize: 11,
                  color: 'var(--color-muted)',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  textAlign: 'right',
                }}
                title={item.reason}
              >
                {item.reason}
              </span>

              {/* Bar track */}
              <div style={{ flex: 1, position: 'relative', height: 18 }}>
                <div
                  style={{
                    position: 'absolute',
                    inset: 0,
                    borderRadius: 4,
                    background: 'var(--color-separator)',
                    opacity: 0.3,
                  }}
                />
                <div
                  style={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    height: '100%',
                    width: mounted ? `${widthPct}%` : '0%',
                    borderRadius: 4,
                    background: item.color,
                    transition: `width 0.5s cubic-bezier(0.34,1.56,0.64,1)`,
                    transitionDelay: `${i * 55 + 80}ms`,
                    boxShadow: `0 0 5px ${item.color}44`,
                  }}
                />
              </div>

              {/* Count */}
              <span
                style={{
                  fontSize: 12,
                  fontWeight: 700,
                  color: item.color,
                  flexShrink: 0,
                  minWidth: 24,
                  textAlign: 'right',
                }}
              >
                {item.count}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
