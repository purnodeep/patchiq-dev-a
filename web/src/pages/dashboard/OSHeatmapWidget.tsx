import { useNavigate } from 'react-router';
import { SkeletonCard, ErrorState } from '@patchiq/ui';
import { useTopEndpointsByRisk } from '@/api/hooks/useDashboard';

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  transition: 'border-color 150ms ease',
  display: 'flex',
  flexDirection: 'column',
};

function riskColor(score: number): string {
  const s = Math.max(0, Math.min(100, score)) / 100;
  if (s >= 0.7) return 'var(--signal-critical)';
  if (s >= 0.4) return 'var(--signal-warning)';
  return 'var(--text-muted)';
}

export function OSHeatmapWidget() {
  const navigate = useNavigate();
  const { data: endpoints, isLoading, error, refetch } = useTopEndpointsByRisk();

  if (isLoading)
    return (
      <div style={{ ...cardStyle, padding: 16 }}>
        <SkeletonCard lines={4} />
      </div>
    );
  if (error)
    return (
      <div style={{ ...cardStyle, padding: 16 }}>
        <ErrorState message="Failed to load heatmap" onRetry={() => refetch()} />
      </div>
    );

  return (
    <div
      style={cardStyle}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--text-faint)';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)';
      }}
    >
      <div
        style={{
          padding: '16px 20px 0',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
          Risk Heatmap
        </span>
        <span style={{ fontSize: 11, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}>
          {(endpoints ?? []).length} endpoints
        </span>
      </div>
      <div
        style={{
          padding: '12px 20px 18px',
          flex: 1,
          display: 'grid',
          gridTemplateColumns: 'repeat(5, 1fr)',
          gap: 6,
          alignContent: 'start',
        }}
      >
        {(endpoints ?? []).slice(0, 30).map((ep) => {
          const name =
            ep.hostname.length > 10 ? ep.hostname.substring(0, 10) + '\u2026' : ep.hostname;
          return (
            <div
              key={ep.hostname}
              role="button"
              tabIndex={0}
              onClick={() => void navigate(`/endpoints?q=${encodeURIComponent(ep.hostname)}`)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  void navigate(`/endpoints?q=${encodeURIComponent(ep.hostname)}`);
                }
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                textAlign: 'center',
                backgroundColor: riskColor(ep.risk_score),
                color: 'var(--text-on-color, #fff)',
                height: 30,
                fontSize: 9.5,
                padding: 4,
                fontFamily: 'var(--font-mono)',
                fontWeight: 600,
                borderRadius: 5,
                cursor: 'pointer',
                transition: 'transform 150ms',
              }}
              title={`${ep.hostname}\nRisk: ${ep.risk_score ?? 0} \u2022 CVEs: ${ep.cve_count}\n\nClick to open endpoint`}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'scale(1.05)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'scale(1)';
              }}
            >
              {name}
            </div>
          );
        })}
        {(endpoints ?? []).length === 0 && (
          <div
            style={{
              gridColumn: '1 / -1',
              textAlign: 'center',
              padding: 16,
              color: 'var(--text-muted)',
              fontSize: 12,
            }}
          >
            No endpoint data
          </div>
        )}
      </div>
    </div>
  );
}
