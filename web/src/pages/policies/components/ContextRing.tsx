import type { components } from '../../../api/types';

type PolicyDetail = components['schemas']['PolicyDetail'];

interface ContextRingProps {
  policy: PolicyDetail;
}

const CARD: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  padding: 16,
  textAlign: 'center',
};

const VAL: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 22,
  fontWeight: 700,
  color: 'var(--text-emphasis)',
  lineHeight: 1,
};

const LABEL: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 10,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  marginTop: 4,
};

const SUB: React.CSSProperties = {
  fontSize: 10,
  color: 'var(--text-muted)',
  marginTop: 3,
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
};

export const ContextRing = ({ policy }: ContextRingProps) => {
  const endpoints = policy.matched_endpoints ?? [];
  const evals = policy.recent_evaluations ?? [];
  const lastEval = evals[0];
  const passRate =
    lastEval && lastEval.in_scope_endpoints > 0
      ? Math.round((lastEval.compliant_count / lastEval.in_scope_endpoints) * 100)
      : null;

  const hasSelector = !!policy.target_selector;
  const endpointCount = policy.target_endpoints_count ?? endpoints.length;
  const patchCount = lastEval?.matched_patches ?? '\u2014';
  const deployCount = policy.deployment_count ?? 0;

  // Simple sparkline bars
  const sparkBars = [3, 5, 4, 7, 6, 8, 5, 9, 7, 4, 6, 8];

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(6, 1fr)',
        gap: 10,
      }}
    >
      {/* Target selector */}
      <div style={CARD}>
        <div style={VAL}>{hasSelector ? 'tag' : '\u2014'}</div>
        <div style={LABEL}>Selector</div>
        <div style={SUB}>{hasSelector ? 'tag predicates' : 'no predicates'}</div>
      </div>

      {/* Endpoints */}
      <div style={CARD}>
        <div style={VAL}>{endpointCount}</div>
        <div style={LABEL}>Endpoints</div>
        <div style={SUB}>in scope</div>
      </div>

      {/* Patches */}
      <div style={CARD}>
        <div style={{ ...VAL, color: 'var(--signal-warning)' }}>{patchCount}</div>
        <div style={LABEL}>Matching Patches</div>
        <div style={{ display: 'flex', justifyContent: 'center', gap: 3, marginTop: 5 }}>
          {['var(--signal-critical)', 'var(--signal-warning)', 'var(--text-muted)'].map((bg, i) => (
            <div
              key={i}
              title={['Critical', 'High', 'Medium'][i]}
              style={{ width: 6, height: 6, borderRadius: '50%', background: bg }}
            />
          ))}
        </div>
      </div>

      {/* Pass rate */}
      <div style={CARD}>
        <div style={{ ...VAL, color: 'var(--signal-healthy)' }}>
          {passRate != null ? `${passRate}%` : '\u2014'}
        </div>
        <div style={LABEL}>Pass Rate</div>
        {passRate != null && (
          <div
            style={{
              marginTop: 6,
              height: 3,
              borderRadius: 2,
              background: 'var(--bg-inset)',
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                width: `${passRate}%`,
                height: '100%',
                background: 'var(--signal-healthy)',
                borderRadius: 2,
                transition: 'width 0.3s',
              }}
            />
          </div>
        )}
      </div>

      {/* Next run */}
      <div style={CARD}>
        <div
          style={{
            ...VAL,
            fontSize: 13,
            color: policy.schedule_type === 'recurring' ? 'var(--accent)' : 'var(--text-muted)',
          }}
        >
          {policy.schedule_type === 'recurring' ? 'Scheduled' : 'On demand'}
        </div>
        <div style={LABEL}>Next Run</div>
        {policy.schedule_type === 'recurring' && policy.schedule_cron && (
          <div
            style={{
              marginTop: 5,
              fontFamily: 'var(--font-mono)',
              fontSize: 9,
              color: 'var(--text-muted)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {policy.schedule_cron}
          </div>
        )}
      </div>

      {/* Deployments */}
      <div style={CARD}>
        <div style={VAL}>{deployCount}</div>
        <div style={LABEL}>Deployments</div>
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-end',
            gap: 2,
            height: 24,
            justifyContent: 'center',
            marginTop: 4,
          }}
        >
          {sparkBars.map((h, i) => (
            <div
              key={i}
              style={{
                width: 4,
                borderRadius: 1,
                height: `${(h / 9) * 100}%`,
                background: 'var(--accent)',
                opacity: 0.5,
              }}
            />
          ))}
        </div>
      </div>
    </div>
  );
};
