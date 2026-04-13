import { timeAgo } from '../../../lib/time';
import type { components } from '../../../api/types';

type PolicyDetail = components['schemas']['PolicyDetail'];

interface OverviewTabProps {
  policy: PolicyDetail;
  DonutGauge?: React.ComponentType<{ pct: number; color: string; size: number }>;
}

const CARD: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  padding: '20px 24px',
};

// ── Pipeline node ─────────────────────────────────────────────────────────────

function PipelineNode({
  label,
  value,
  sub,
  color,
  isLast = false,
}: {
  label: string;
  value: string | number;
  sub: string;
  color: string;
  isLast?: boolean;
}) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', flex: 1 }}>
      <div
        style={{
          flex: 1,
          background: 'var(--bg-inset)',
          border: `1px solid color-mix(in srgb, ${color} 13%, transparent)`,
          borderRadius: 6,
          padding: '14px 12px',
          textAlign: 'center',
        }}
      >
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 9,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            color,
            marginBottom: 6,
          }}
        >
          {label}
        </div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 22,
            fontWeight: 700,
            color: 'var(--text-emphasis)',
            lineHeight: 1,
            marginBottom: 4,
          }}
        >
          {value}
        </div>
        <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>{sub}</div>
      </div>
      {!isLast && (
        <div
          style={{
            padding: '0 8px',
            fontSize: 16,
            color: 'var(--text-muted)',
            flexShrink: 0,
          }}
        >
          &rarr;
        </div>
      )}
    </div>
  );
}

// ── Main tab ──────────────────────────────────────────────────────────────────

export const OverviewTab = ({ policy, DonutGauge }: OverviewTabProps) => {
  const lastEval = policy.recent_evaluations?.[0];
  const lastDeploy = policy.recent_deployments?.[0];
  const endpoints = policy.matched_endpoints ?? [];
  const endpointCount = policy.target_endpoints_count ?? endpoints.length;
  const deployCount = policy.deployment_count ?? 0;
  const hasSelector = !!policy.target_selector;

  const compliancePct =
    lastEval && lastEval.in_scope_endpoints > 0
      ? Math.round((lastEval.compliant_count / lastEval.in_scope_endpoints) * 100)
      : null;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* ── Hero row: Pipeline (60%) + Effectiveness (40%) ──────────────── */}
      <div style={{ display: 'grid', gridTemplateColumns: '60fr 40fr', gap: 12 }}>
        {/* Policy scope pipeline */}
        <div style={CARD}>
          <div
            style={{
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              fontWeight: 600,
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
              color: 'var(--text-muted)',
              marginBottom: 16,
            }}
          >
            Policy Scope Pipeline
          </div>

          {/* Pipeline nodes */}
          <div style={{ display: 'flex', alignItems: 'stretch', overflowX: 'auto' }}>
            <PipelineNode
              label="Selector"
              value={hasSelector ? 'tag' : '—'}
              sub={hasSelector ? 'tag predicates' : 'none'}
              color="var(--accent)"
            />
            <PipelineNode
              label="Endpoints"
              value={endpointCount}
              sub="in scope"
              color="var(--text-secondary)"
            />
            <PipelineNode
              label="Criteria"
              value={policy.selection_mode ?? '—'}
              sub={policy.min_severity ? `≥ ${policy.min_severity}` : 'all severities'}
              color="var(--signal-warning)"
            />
            <PipelineNode
              label="Patches"
              value={lastEval?.matched_patches ?? '—'}
              sub="matched"
              color="var(--signal-warning)"
            />
            <PipelineNode
              label="Deploy"
              value={policy.mode ?? '—'}
              sub={policy.schedule_type === 'recurring' ? 'Scheduled' : 'On demand'}
              color="var(--accent)"
              isLast
            />
          </div>

          {/* Bottom stats grid */}
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(3, 1fr)',
              gap: 10,
              marginTop: 16,
            }}
          >
            {/* Last evaluation */}
            <div
              style={{
                background: 'color-mix(in srgb, white 2%, transparent)',
                borderRadius: 6,
                padding: 12,
              }}
            >
              <div
                style={{
                  fontSize: 10,
                  color: 'var(--text-muted)',
                  marginBottom: 6,
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  fontFamily: 'var(--font-mono)',
                }}
              >
                Last Eval
              </div>
              {lastEval ? (
                <>
                  <div
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 13,
                      fontWeight: 600,
                      color: lastEval.pass ? 'var(--signal-healthy)' : 'var(--signal-critical)',
                      marginBottom: 3,
                    }}
                  >
                    {lastEval.pass ? '\u2713 Pass' : '\u2717 Fail'}
                  </div>
                  <div
                    style={{
                      fontSize: 10,
                      color: 'var(--text-muted)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    {timeAgo(lastEval.evaluated_at)}
                  </div>
                </>
              ) : (
                <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>No evaluations yet</div>
              )}
            </div>

            {/* Last deployment */}
            <div
              style={{
                background: 'color-mix(in srgb, white 2%, transparent)',
                borderRadius: 6,
                padding: 12,
              }}
            >
              <div
                style={{
                  fontSize: 10,
                  color: 'var(--text-muted)',
                  marginBottom: 6,
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  fontFamily: 'var(--font-mono)',
                }}
              >
                Last Deploy
              </div>
              {lastDeploy ? (
                <>
                  <div
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 13,
                      fontWeight: 600,
                      color:
                        lastDeploy.status === 'completed'
                          ? 'var(--signal-healthy)'
                          : 'var(--text-secondary)',
                      marginBottom: 3,
                    }}
                  >
                    {lastDeploy.status === 'completed' ? '\u2713 Success' : lastDeploy.status}
                  </div>
                  <div
                    style={{
                      fontSize: 10,
                      color: 'var(--text-muted)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    {timeAgo(lastDeploy.created_at)}
                  </div>
                </>
              ) : (
                <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>No deployments yet</div>
              )}
            </div>

            {/* Total deployments */}
            <div
              style={{
                background: 'color-mix(in srgb, white 2%, transparent)',
                borderRadius: 6,
                padding: 12,
              }}
            >
              <div
                style={{
                  fontSize: 10,
                  color: 'var(--text-muted)',
                  marginBottom: 6,
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  fontFamily: 'var(--font-mono)',
                }}
              >
                Deployments
              </div>
              <div
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 22,
                  fontWeight: 700,
                  color: 'var(--text-emphasis)',
                  lineHeight: 1,
                }}
              >
                {deployCount}
              </div>
            </div>
          </div>
        </div>

        {/* Policy effectiveness panel */}
        <div style={{ ...CARD, display: 'flex', flexDirection: 'column' }}>
          <div
            style={{
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              fontWeight: 600,
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
              color: 'var(--text-muted)',
              marginBottom: 16,
            }}
          >
            Policy Effectiveness
          </div>

          {/* Donut gauge */}
          {DonutGauge ? (
            <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 20 }}>
              <div style={{ position: 'relative', width: 80, height: 80, flexShrink: 0 }}>
                <DonutGauge
                  pct={compliancePct ?? 0}
                  color={
                    compliancePct != null && compliancePct >= 80
                      ? 'var(--signal-healthy)'
                      : compliancePct != null
                        ? 'var(--signal-warning)'
                        : 'var(--text-faint)'
                  }
                  size={80}
                />
                <div
                  style={{
                    position: 'absolute',
                    inset: 0,
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 15,
                      fontWeight: 700,
                      color: 'var(--text-emphasis)',
                      lineHeight: 1,
                    }}
                  >
                    {compliancePct != null ? `${compliancePct}%` : '\u2014'}
                  </span>
                </div>
              </div>
              <div>
                <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 2 }}>
                  Pass Rate
                </div>
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 22,
                    fontWeight: 700,
                    color:
                      compliancePct != null && compliancePct >= 80
                        ? 'var(--signal-healthy)'
                        : compliancePct != null
                          ? 'var(--signal-warning)'
                          : 'var(--text-muted)',
                  }}
                >
                  {compliancePct != null ? `${compliancePct}%` : '\u2014'}
                </div>
              </div>
            </div>
          ) : (
            <div style={{ marginBottom: 20 }}>
              <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4 }}>
                Pass Rate
              </div>
              <div
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 28,
                  fontWeight: 700,
                  color: 'var(--signal-healthy)',
                }}
              >
                {compliancePct != null ? `${compliancePct}%` : '\u2014'}
              </div>
            </div>
          )}

          {/* Stats list */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10, flex: 1 }}>
            {[
              {
                label: 'In Scope Endpoints',
                value: String(endpointCount),
                color: 'var(--text-primary)',
              },
              {
                label: 'Compliant',
                value:
                  lastEval?.compliant_count != null ? String(lastEval.compliant_count) : '\u2014',
                color: 'var(--signal-healthy)',
              },
              {
                label: 'Non-compliant',
                value:
                  lastEval?.compliant_count != null && lastEval?.in_scope_endpoints != null
                    ? String(lastEval.in_scope_endpoints - lastEval.compliant_count)
                    : '\u2014',
                color:
                  lastEval?.compliant_count != null &&
                  lastEval?.in_scope_endpoints != null &&
                  lastEval.in_scope_endpoints - lastEval.compliant_count > 0
                    ? 'var(--signal-critical)'
                    : 'var(--text-muted)',
              },
              {
                label: 'Matched Patches',
                value:
                  lastEval?.matched_patches != null ? String(lastEval.matched_patches) : '\u2014',
                color: 'var(--signal-warning)',
              },
              {
                label: 'Total Deployments',
                value: String(deployCount),
                color: 'var(--text-primary)',
              },
            ].map((stat) => (
              <div
                key={stat.label}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  paddingBottom: 8,
                  borderBottom: '1px solid var(--border)',
                }}
              >
                <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{stat.label}</span>
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 13,
                    fontWeight: 600,
                    color: stat.color,
                  }}
                >
                  {stat.value}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};
