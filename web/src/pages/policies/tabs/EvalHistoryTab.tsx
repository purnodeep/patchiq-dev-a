import { timeAgo } from '../../../lib/time';
import type { components } from '../../../api/types';

type PolicyDetail = components['schemas']['PolicyDetail'];
type PolicyEvaluation = components['schemas']['PolicyEvaluation'];

interface EvalHistoryTabProps {
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
  borderBottom: '1px solid var(--border)',
  fontSize: 12,
};

// Horizontal timeline dots
function EvalTimeline({ evaluations }: { evaluations: PolicyEvaluation[] }) {
  if (evaluations.length === 0) return null;

  const recent = [...evaluations].slice(0, 20).reverse();

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        padding: '16px 20px',
        marginBottom: 14,
      }}
    >
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 9,
          fontWeight: 600,
          textTransform: 'uppercase',
          letterSpacing: '0.07em',
          color: 'var(--text-muted)',
          marginBottom: 14,
        }}
      >
        Evaluation History — {evaluations.length} runs
      </div>

      {/* Horizontal timeline */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 4,
          overflowX: 'auto',
          paddingBottom: 4,
        }}
      >
        {recent.map((ev: PolicyEvaluation, i: number) => {
          const isPass = ev.pass;
          const color = isPass ? 'var(--signal-healthy)' : 'var(--signal-critical)';
          const isLast = i === recent.length - 1;

          return (
            <div
              key={ev.id}
              style={{ display: 'flex', alignItems: 'center', gap: 4, flexShrink: 0 }}
            >
              <div
                title={`${timeAgo(ev.evaluated_at)} — ${isPass ? 'Pass' : 'Fail'} — ${ev.compliant_count}/${ev.in_scope_endpoints} compliant`}
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  gap: 4,
                  cursor: 'default',
                }}
              >
                <div
                  style={{
                    width: 10,
                    height: 10,
                    borderRadius: '50%',
                    background: color,
                    boxShadow: `0 0 0 2px color-mix(in srgb, ${color} 13%, transparent)`,
                    flexShrink: 0,
                  }}
                />
                {/* Score beneath dot */}
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 8,
                    color: 'var(--text-faint)',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {ev.in_scope_endpoints > 0
                    ? `${Math.round((ev.compliant_count / ev.in_scope_endpoints) * 100)}%`
                    : '—'}
                </div>
              </div>
              {!isLast && (
                <div style={{ width: 16, height: 1, background: 'var(--border)', flexShrink: 0 }} />
              )}
            </div>
          );
        })}
      </div>

      {/* Trend stats */}
      <div
        style={{
          display: 'flex',
          gap: 20,
          marginTop: 12,
          paddingTop: 10,
          borderTop: '1px solid var(--border)',
        }}
      >
        {(() => {
          const passes = evaluations.filter((ev) => ev.pass).length;
          const passRate =
            evaluations.length > 0 ? Math.round((passes / evaluations.length) * 100) : 0;
          const avgCompliance =
            evaluations.length > 0
              ? Math.round(
                  evaluations.reduce((sum: number, ev: PolicyEvaluation) => {
                    return (
                      sum +
                      (ev.in_scope_endpoints > 0
                        ? (ev.compliant_count / ev.in_scope_endpoints) * 100
                        : 0)
                    );
                  }, 0) / evaluations.length,
                )
              : 0;

          return [
            {
              label: 'Pass Rate',
              value: `${passRate}%`,
              color: passRate >= 80 ? 'var(--signal-healthy)' : 'var(--signal-warning)',
            },
            {
              label: 'Avg Compliance',
              value: `${avgCompliance}%`,
              color: avgCompliance >= 80 ? 'var(--signal-healthy)' : 'var(--signal-warning)',
            },
            {
              label: 'Total Runs',
              value: String(evaluations.length),
              color: 'var(--text-primary)',
            },
            {
              label: 'Failures',
              value: String(evaluations.length - passes),
              color:
                evaluations.length - passes > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
            },
          ].map(({ label, value, color }) => (
            <div key={label}>
              <div style={{ fontFamily: 'var(--font-mono)', fontSize: 14, fontWeight: 700, color }}>
                {value}
              </div>
              <div
                style={{
                  fontSize: 9,
                  color: 'var(--text-muted)',
                  marginTop: 2,
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  fontFamily: 'var(--font-mono)',
                }}
              >
                {label}
              </div>
            </div>
          ));
        })()}
      </div>
    </div>
  );
}

export const EvalHistoryTab = ({ policy }: EvalHistoryTabProps) => {
  const evaluations = policy.recent_evaluations ?? [];

  if (evaluations.length === 0) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: 140,
          borderRadius: 8,
          border: '1px dashed var(--border)',
          background: 'var(--bg-card)',
          fontSize: 12,
          color: 'var(--text-muted)',
        }}
      >
        No evaluations recorded yet.
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
      <EvalTimeline evaluations={evaluations} />

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
              <th style={TH}>Date</th>
              <th style={TH}>Matched</th>
              <th style={TH}>In Scope</th>
              <th style={TH}>Compliant</th>
              <th style={TH}>Non-Compliant</th>
              <th style={TH}>Duration</th>
              <th style={TH}>Result</th>
            </tr>
          </thead>
          <tbody>
            {evaluations.map((ev: PolicyEvaluation) => {
              const passRate =
                ev.in_scope_endpoints > 0
                  ? Math.round((ev.compliant_count / ev.in_scope_endpoints) * 100)
                  : null;

              return (
                <tr
                  key={ev.id}
                  style={{ transition: 'background 0.1s' }}
                  onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--bg-card-hover)')}
                  onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                >
                  <td
                    style={{
                      ...TD,
                      fontFamily: 'var(--font-mono)',
                      fontSize: 10,
                      color: 'var(--text-muted)',
                    }}
                  >
                    {timeAgo(ev.evaluated_at)}
                  </td>
                  <td
                    style={{
                      ...TD,
                      fontFamily: 'var(--font-mono)',
                      fontSize: 12,
                      color: 'var(--text-primary)',
                    }}
                  >
                    {ev.matched_patches}
                  </td>
                  <td
                    style={{
                      ...TD,
                      fontFamily: 'var(--font-mono)',
                      fontSize: 12,
                      color: 'var(--text-primary)',
                    }}
                  >
                    {ev.in_scope_endpoints}
                  </td>
                  <td
                    style={{
                      ...TD,
                      fontFamily: 'var(--font-mono)',
                      fontSize: 12,
                      color: 'var(--signal-healthy)',
                    }}
                  >
                    {ev.compliant_count}
                  </td>
                  <td
                    style={{
                      ...TD,
                      fontFamily: 'var(--font-mono)',
                      fontSize: 12,
                      color:
                        ev.non_compliant_count > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
                    }}
                  >
                    {ev.non_compliant_count}
                  </td>
                  <td
                    style={{
                      ...TD,
                      fontFamily: 'var(--font-mono)',
                      fontSize: 10,
                      color: 'var(--text-muted)',
                    }}
                  >
                    {ev.duration_ms}ms
                  </td>
                  <td style={TD}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      {/* Pass/fail dot */}
                      <span
                        style={{
                          width: 6,
                          height: 6,
                          borderRadius: '50%',
                          background: ev.pass ? 'var(--signal-healthy)' : 'var(--signal-critical)',
                          display: 'inline-block',
                          flexShrink: 0,
                        }}
                      />
                      <span
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          fontWeight: 600,
                          color: ev.pass ? 'var(--signal-healthy)' : 'var(--signal-critical)',
                        }}
                      >
                        {ev.pass ? '✓ Pass' : '✗ Fail'}
                      </span>
                      {/* Compliance rate */}
                      {passRate != null && (
                        <div
                          style={{
                            width: 36,
                            height: 3,
                            borderRadius: 2,
                            background: 'var(--bg-inset)',
                            overflow: 'hidden',
                            flexShrink: 0,
                          }}
                        >
                          <div
                            style={{
                              width: `${passRate}%`,
                              height: '100%',
                              background:
                                passRate >= 80 ? 'var(--signal-healthy)' : 'var(--signal-warning)',
                              borderRadius: 2,
                            }}
                          />
                        </div>
                      )}
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
};
