import { useEffect, useState } from 'react';
import { CVE_AGE_DATA } from '@/data/mock-data';

export function CVEAgeDistribution() {
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    // Trigger animation after first paint
    const id = requestAnimationFrame(() => setMounted(true));
    return () => cancelAnimationFrame(id);
  }, []);

  const maxCount = Math.max(...CVE_AGE_DATA.map((d) => d.count));
  const total = CVE_AGE_DATA.reduce((sum, d) => sum + d.count, 0);

  return (
    <div
      style={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'space-between',
        fontFamily: 'var(--font-sans)',
      }}
    >
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 10,
          flex: 1,
          justifyContent: 'center',
        }}
      >
        {CVE_AGE_DATA.map((item, i) => {
          const widthPct = maxCount > 0 ? (item.count / maxCount) * 100 : 0;
          return (
            <div
              key={item.bucket}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                opacity: mounted ? 1 : 0,
                transform: mounted ? 'translateX(0)' : 'translateX(-8px)',
                transition: `opacity 0.35s ease, transform 0.35s ease`,
                transitionDelay: `${i * 60}ms`,
              }}
            >
              {/* Bucket label */}
              <span
                style={{
                  width: 60,
                  flexShrink: 0,
                  textAlign: 'right',
                  fontSize: 11,
                  color: 'var(--color-muted)',
                  fontWeight: 500,
                }}
              >
                {item.bucket}
              </span>

              {/* Bar track */}
              <div style={{ flex: 1, position: 'relative', height: 20 }}>
                <div
                  style={{
                    position: 'absolute',
                    inset: 0,
                    borderRadius: 4,
                    background: 'var(--color-separator)',
                    opacity: 0.35,
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
                    transition: `width 0.55s cubic-bezier(0.34,1.56,0.64,1)`,
                    transitionDelay: `${i * 60 + 80}ms`,
                    boxShadow: `0 0 6px ${item.color}55`,
                  }}
                />
              </div>

              {/* Count label */}
              <span
                style={{
                  fontSize: 12,
                  fontWeight: 700,
                  color: item.color,
                  flexShrink: 0,
                  minWidth: 28,
                  textAlign: 'right',
                }}
              >
                {item.count}
              </span>
            </div>
          );
        })}
      </div>

      {/* Footer total */}
      <div
        style={{
          borderTop: '1px solid var(--color-separator)',
          paddingTop: 8,
          marginTop: 8,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>Total Active CVEs</span>
        <span
          style={{
            fontSize: 14,
            fontWeight: 800,
            color: 'var(--color-foreground)',
          }}
        >
          {total}
        </span>
      </div>
    </div>
  );
}
