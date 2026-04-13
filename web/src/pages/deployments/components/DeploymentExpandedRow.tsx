import { Link } from 'react-router';
import type { components } from '../../../api/types';

type Deployment = components['schemas']['Deployment'];

interface DeploymentExpandedRowProps {
  deployment: Deployment;
  onCancel?: () => void;
  onRetry?: () => void;
  onRollback?: () => void;
}

const CARD: React.CSSProperties = {
  background: 'var(--bg-inset)',
  border: '1px solid var(--border)',
  borderRadius: 6,
  padding: '14px 16px',
};

const SECTION_LABEL: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 9,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.07em',
  color: 'var(--text-muted)',
  marginBottom: 10,
};

const BTN_BASE: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  gap: 5,
  padding: '7px 14px',
  borderRadius: 5,
  fontSize: 13,
  fontWeight: 500,
  cursor: 'pointer',
  border: '1px solid var(--border)',
  background: 'transparent',
  color: 'var(--text-secondary)',
  transition: 'background 0.12s, color 0.12s',
  textDecoration: 'none',
  letterSpacing: '0.01em',
  width: '100%',
  fontFamily: 'var(--font-sans)',
};

export function DeploymentExpandedRow({
  deployment: d,
  onCancel,
  onRetry,
  onRollback,
}: DeploymentExpandedRowProps) {
  const succeeded = d.success_count;
  const failed = d.failed_count;
  const active = Math.max(
    0,
    d.status === 'running' ? d.completed_count - d.success_count - d.failed_count : 0,
  );
  const pending = Math.max(0, d.target_count - d.completed_count);
  const total = d.target_count;

  const successPct = total > 0 ? Math.round((succeeded / total) * 100) : 0;
  const failPct = total > 0 ? Math.round((failed / total) * 100) : 0;

  return (
    <div
      style={{
        padding: '8px 10px',
        background: 'var(--bg-page)',
        borderTop: '1px solid var(--border)',
        display: 'flex',
        gap: 8,
        alignItems: 'stretch',
      }}
    >
      {/* Endpoint breakdown */}
      <div style={{ ...CARD, flex: '0 0 500px' }}>
        <div style={SECTION_LABEL}>Endpoint Breakdown</div>
        {total > 0 ? (
          <>
            {/* Segmented bar */}
            <div
              style={{
                display: 'flex',
                height: 5,
                borderRadius: 3,
                overflow: 'hidden',
                background: 'var(--bg-card)',
                gap: 1,
                marginBottom: 10,
              }}
            >
              {succeeded > 0 && (
                <div
                  style={{
                    flex: succeeded,
                    background: 'var(--signal-healthy)',
                    borderRadius: 3,
                    transition: 'flex 0.4s',
                  }}
                />
              )}
              {active > 0 && (
                <div
                  style={{
                    flex: active,
                    background: 'var(--accent)',
                    borderRadius: 3,
                    transition: 'flex 0.4s',
                  }}
                />
              )}
              {failed > 0 && (
                <div
                  style={{
                    flex: failed,
                    background: 'var(--signal-critical)',
                    borderRadius: 3,
                    transition: 'flex 0.4s',
                  }}
                />
              )}
              {pending > 0 && (
                <div style={{ flex: pending, background: 'var(--border)', borderRadius: 3 }} />
              )}
            </div>

            {/* Legend */}
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px 14px' }}>
              {succeeded > 0 && (
                <span
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 4,
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                    color: 'var(--signal-healthy)',
                  }}
                >
                  <span
                    style={{
                      width: 6,
                      height: 6,
                      borderRadius: '50%',
                      background: 'var(--signal-healthy)',
                      display: 'inline-block',
                    }}
                  />
                  {succeeded} done
                </span>
              )}
              {active > 0 && (
                <span
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 4,
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                    color: 'var(--accent)',
                  }}
                >
                  <span
                    style={{
                      width: 6,
                      height: 6,
                      borderRadius: '50%',
                      background: 'var(--accent)',
                      display: 'inline-block',
                    }}
                  />
                  {active} active
                </span>
              )}
              {failed > 0 && (
                <span
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 4,
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                    color: 'var(--signal-critical)',
                  }}
                >
                  <span
                    style={{
                      width: 6,
                      height: 6,
                      borderRadius: '50%',
                      background: 'var(--signal-critical)',
                      display: 'inline-block',
                    }}
                  />
                  {failed} failed
                </span>
              )}
              {pending > 0 && (
                <span
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 4,
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                    color: 'var(--text-muted)',
                  }}
                >
                  <span
                    style={{
                      width: 6,
                      height: 6,
                      borderRadius: '50%',
                      background: 'var(--border)',
                      display: 'inline-block',
                    }}
                  />
                  {pending} pending
                </span>
              )}
            </div>
          </>
        ) : (
          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>No target data</span>
        )}
      </div>

      {/* Quick stats */}
      <div style={{ ...CARD, flex: '0 0 480px', marginLeft: 24 }}>
        <div style={SECTION_LABEL}>Quick Stats</div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px 14px' }}>
          {[
            { label: 'Total', value: total, color: 'var(--text-emphasis)' },
            { label: 'Succeeded', value: succeeded, color: 'var(--signal-healthy)' },
            {
              label: 'Failed',
              value: failed,
              color: failed > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
            },
            { label: 'Pending', value: pending, color: 'var(--text-muted)' },
          ].map(({ label, value, color }) => (
            <div key={label}>
              <div
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 18,
                  fontWeight: 700,
                  color,
                  lineHeight: 1,
                }}
              >
                {value}
              </div>
              <div
                style={{
                  fontSize: 9,
                  color: 'var(--text-muted)',
                  marginTop: 2,
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                  fontFamily: 'var(--font-mono)',
                }}
              >
                {label}
              </div>
            </div>
          ))}
        </div>
        {total > 0 && (
          <div style={{ marginTop: 10, display: 'flex', alignItems: 'center', gap: 8 }}>
            <div
              style={{
                flex: 1,
                height: 3,
                borderRadius: 2,
                background: 'var(--bg-card)',
                overflow: 'hidden',
              }}
            >
              <div
                style={{
                  width: `${successPct}%`,
                  height: '100%',
                  background:
                    successPct >= 80
                      ? 'var(--signal-healthy)'
                      : failPct > 20
                        ? 'var(--signal-critical)'
                        : 'var(--signal-warning)',
                  borderRadius: 2,
                  transition: 'width 0.4s',
                }}
              />
            </div>
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                color: 'var(--text-muted)',
                flexShrink: 0,
              }}
            >
              {successPct}% success
            </span>
          </div>
        )}
      </div>

      {/* Action buttons — outside cards */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 6,
          flexShrink: 0,
          width: 160,
          alignSelf: 'start',
          marginLeft: 40,
        }}
      >
        {d.status === 'running' && onCancel && (
          <button
            type="button"
            style={{
              ...BTN_BASE,
              color: 'var(--btn-accent-text, #fff)',
              borderColor: 'var(--signal-critical)',
              background: 'var(--signal-critical)',
            }}
            onClick={(e) => {
              e.stopPropagation();
              onCancel();
            }}
          >
            ✗ Cancel Deployment
          </button>
        )}
        {d.status === 'failed' && onRetry && (
          <button
            type="button"
            style={{
              ...BTN_BASE,
              color: 'var(--btn-accent-text, #000)',
              borderColor: 'var(--accent)',
              background: 'var(--accent)',
            }}
            onClick={(e) => {
              e.stopPropagation();
              onRetry();
            }}
          >
            ↺ Retry Failed ({failed})
          </button>
        )}
        {(d.status === 'completed' || d.status === 'failed') && onRollback && (
          <button
            type="button"
            style={BTN_BASE}
            onClick={(e) => {
              e.stopPropagation();
              onRollback();
            }}
          >
            ⎌ Rollback
          </button>
        )}
        <Link to={`/deployments/${d.id}`} style={BTN_BASE} onClick={(e) => e.stopPropagation()}>
          View Details →
        </Link>
      </div>
    </div>
  );
}
