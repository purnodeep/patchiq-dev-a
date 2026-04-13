import { useNavigate } from 'react-router';
import { useCVEs } from '@/api/hooks/useCVEs';

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  transition: 'border-color 150ms ease',
  display: 'flex',
  flexDirection: 'column',
};

const scoreColors: Record<string, string> = {
  critical: 'var(--signal-critical)',
  high: 'var(--signal-warning)',
  medium: 'var(--text-primary)',
  low: 'var(--text-muted)',
};

const severityColors: Record<string, string> = {
  critical: 'var(--signal-critical)',
  high: 'var(--signal-warning)',
  medium: 'var(--text-secondary)',
  low: 'var(--text-muted)',
};

function capitalizeFirst(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

export function TopVulnerabilities() {
  const navigate = useNavigate();
  const { data: cveData, isLoading } = useCVEs({ limit: 5, severity: 'critical' });

  // Fall back to top 5 from any severity if no critical CVEs
  const { data: fallbackData } = useCVEs({ limit: 5 });

  const cves = cveData?.data && cveData.data.length > 0 ? cveData.data : (fallbackData?.data ?? []);
  const topCves = cves.slice(0, 5).sort((a, b) => (b.cvss_v3_score ?? 0) - (a.cvss_v3_score ?? 0));

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
          Top Vulnerabilities
        </span>
      </div>
      <div style={{ padding: '12px 20px 16px', flex: 1 }}>
        {isLoading ? (
          <div style={{ color: 'var(--text-faint)', fontSize: 12, padding: '9px 0' }}>
            Loading...
          </div>
        ) : topCves.length === 0 ? (
          <div style={{ color: 'var(--text-faint)', fontSize: 12, padding: '9px 0' }}>
            No CVEs found
          </div>
        ) : (
          topCves.map((cve, i) => {
            const severity = cve.severity ?? 'medium';
            const score = cve.cvss_v3_score ?? 0;
            return (
              <div
                key={cve.id}
                onClick={() => navigate(`/cves/${cve.id}`)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '9px 0',
                  borderBottom: i < topCves.length - 1 ? '1px solid var(--border-faint)' : 'none',
                  cursor: 'pointer',
                }}
              >
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 12,
                    color: 'var(--accent)',
                    minWidth: 120,
                    flexShrink: 0,
                  }}
                >
                  {cve.cve_id}
                </span>
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 13,
                    fontWeight: 600,
                    minWidth: 34,
                    textAlign: 'right',
                    flexShrink: 0,
                    color: scoreColors[severity] ?? 'var(--text-primary)',
                  }}
                >
                  {score.toFixed(1)}
                </span>
                <span
                  style={{
                    fontSize: 12,
                    minWidth: 60,
                    flexShrink: 0,
                    color: severityColors[severity] ?? 'var(--text-secondary)',
                  }}
                >
                  {capitalizeFirst(severity)}
                </span>
                <span
                  style={{
                    fontSize: 11,
                    color: 'var(--text-muted)',
                    marginLeft: 'auto',
                  }}
                >
                  {cve.affected_endpoint_count} endpoints
                </span>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
