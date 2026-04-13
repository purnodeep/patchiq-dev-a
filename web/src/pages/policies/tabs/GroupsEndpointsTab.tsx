import { Link } from 'react-router';
import type { components } from '../../../api/types';

type PolicyDetail = components['schemas']['PolicyDetail'];
type MatchedEndpoint = NonNullable<PolicyDetail['matched_endpoints']>[number];

interface TargetsTabProps {
  policy: PolicyDetail;
}

const TH: React.CSSProperties = {
  padding: '9px 12px',
  textAlign: 'left',
  fontFamily: 'var(--font-mono)',
  fontSize: 9,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  whiteSpace: 'nowrap',
  background: 'var(--bg-inset)',
  borderBottom: '1px solid var(--border)',
};

const TD: React.CSSProperties = {
  padding: '9px 12px',
  verticalAlign: 'middle',
  fontSize: 12,
  color: 'var(--text-primary)',
  borderBottom: '1px solid var(--border)',
};

export const GroupsEndpointsTab = ({ policy }: TargetsTabProps) => {
  const endpoints = policy.matched_endpoints ?? [];
  const lastEval = (policy.recent_evaluations ?? [])[0];
  const compliancePct =
    lastEval && lastEval.in_scope_endpoints > 0
      ? Math.round((lastEval.compliant_count / lastEval.in_scope_endpoints) * 100)
      : null;
  const totalEndpoints = policy.target_endpoints_count ?? endpoints.length;
  const selector = policy.target_selector;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(2, 1fr)',
          gap: 10,
        }}
      >
        {[
          {
            label: 'Total Endpoints',
            value: totalEndpoints,
            color: 'var(--accent)',
          },
          {
            label: 'Compliance',
            value: compliancePct != null ? `${compliancePct}%` : '—',
            color:
              compliancePct != null && compliancePct >= 80
                ? 'var(--signal-healthy)'
                : 'var(--signal-warning)',
          },
        ].map(({ label, value, color }) => (
          <div
            key={label}
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              padding: '14px 16px',
              textAlign: 'center',
            }}
          >
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 24,
                fontWeight: 700,
                color,
                lineHeight: 1,
                marginBottom: 4,
              }}
            >
              {value}
            </div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 9,
                fontWeight: 600,
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                color: 'var(--text-muted)',
              }}
            >
              {label}
            </div>
          </div>
        ))}
      </div>

      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          padding: 14,
        }}
      >
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 9,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            color: 'var(--text-muted)',
            marginBottom: 8,
          }}
        >
          Tag Selector
        </div>
        {selector ? (
          <pre
            style={{
              margin: 0,
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--text-primary)',
              whiteSpace: 'pre-wrap',
            }}
          >
            {JSON.stringify(selector, null, 2)}
          </pre>
        ) : (
          <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
            No tag selector configured — this policy matches zero endpoints.
          </div>
        )}
      </div>

      {endpoints.length > 0 && (
        <div
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            overflow: 'hidden',
          }}
        >
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={TH}>Hostname</th>
                <th style={TH}>OS</th>
                <th style={TH}>Status</th>
              </tr>
            </thead>
            <tbody>
              {endpoints.map((ep: MatchedEndpoint) => {
                const statusColor =
                  ep.status === 'online' ? 'var(--signal-healthy)' : 'var(--text-faint)';
                return (
                  <tr key={ep.id}>
                    <td style={TD}>
                      {ep.id ? (
                        <Link
                          to={`/endpoints/${ep.id}`}
                          style={{
                            color: 'var(--accent)',
                            textDecoration: 'none',
                            fontFamily: 'var(--font-mono)',
                            fontSize: 12,
                          }}
                        >
                          {ep.hostname}
                        </Link>
                      ) : (
                        <span style={{ color: 'var(--text-primary)' }}>{ep.hostname}</span>
                      )}
                    </td>
                    <td style={{ ...TD, color: 'var(--text-muted)' }}>{ep.os_family ?? '—'}</td>
                    <td style={TD}>
                      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                        <span
                          style={{
                            width: 6,
                            height: 6,
                            borderRadius: '50%',
                            background: statusColor,
                            flexShrink: 0,
                          }}
                        />
                        <span style={{ fontSize: 11, color: statusColor }}>
                          {ep.status ?? '—'}
                        </span>
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
