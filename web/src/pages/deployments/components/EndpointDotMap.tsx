import { Link } from 'react-router';
import type { components } from '../../../api/types';

type DeploymentTarget = components['schemas']['DeploymentTarget'];

interface EndpointDotMapProps {
  targets: DeploymentTarget[];
  className?: string;
}

const dotConfig: Record<string, { bg: string; glow: string }> = {
  succeeded: {
    bg: 'var(--signal-healthy)',
    glow: 'color-mix(in srgb, var(--signal-healthy) 40%, transparent)',
  },
  failed: {
    bg: 'var(--signal-critical)',
    glow: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
  },
  executing: { bg: 'var(--accent)', glow: 'color-mix(in srgb, var(--accent) 40%, transparent)' },
  sent: { bg: 'var(--accent)', glow: 'color-mix(in srgb, var(--accent) 30%, transparent)' },
  pending: { bg: 'var(--text-faint)', glow: 'none' },
};

// Summary stat strip above the dot grid
function StatusSummary({ targets }: { targets: DeploymentTarget[] }) {
  const counts = targets.reduce(
    (acc, t) => {
      const s = t.status as string;
      if (s === 'succeeded') acc.succeeded++;
      else if (s === 'failed') acc.failed++;
      else if (s === 'executing' || s === 'sent') acc.active++;
      else acc.pending++;
      return acc;
    },
    { succeeded: 0, failed: 0, active: 0, pending: 0 },
  );

  const total = targets.length;
  const stats = [
    { label: 'succeeded', count: counts.succeeded, color: 'var(--signal-healthy)' },
    { label: 'active', count: counts.active, color: 'var(--accent)' },
    { label: 'failed', count: counts.failed, color: 'var(--signal-critical)' },
    { label: 'pending', count: counts.pending, color: 'var(--text-faint)' },
  ].filter((s) => s.count > 0);

  if (total === 0) return null;

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 12 }}>
      {/* Segmented bar */}
      <div
        style={{
          flex: 1,
          height: 4,
          borderRadius: 2,
          background: 'var(--bg-inset)',
          display: 'flex',
          overflow: 'hidden',
          gap: 1,
        }}
      >
        {counts.succeeded > 0 && (
          <div
            style={{
              flex: counts.succeeded,
              background: 'var(--signal-healthy)',
              transition: 'flex 0.5s',
            }}
          />
        )}
        {counts.active > 0 && (
          <div
            style={{ flex: counts.active, background: 'var(--accent)', transition: 'flex 0.5s' }}
          />
        )}
        {counts.failed > 0 && (
          <div
            style={{
              flex: counts.failed,
              background: 'var(--signal-critical)',
              transition: 'flex 0.5s',
            }}
          />
        )}
        {counts.pending > 0 && (
          <div
            style={{ flex: counts.pending, background: 'var(--border)', transition: 'flex 0.5s' }}
          />
        )}
      </div>

      {/* Labels */}
      <div style={{ display: 'flex', gap: 10, flexShrink: 0 }}>
        {stats.map((s) => (
          <span
            key={s.label}
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 10,
              color: s.color,
              display: 'flex',
              alignItems: 'center',
              gap: 4,
            }}
          >
            <span
              style={{
                width: 6,
                height: 6,
                borderRadius: '50%',
                background: s.color,
                display: 'inline-block',
                flexShrink: 0,
              }}
            />
            {s.count} {s.label}
          </span>
        ))}
      </div>
    </div>
  );
}

export function EndpointDotMap({ targets }: EndpointDotMapProps) {
  return (
    <div>
      <StatusSummary targets={targets} />

      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: 5,
          padding: '4px 0',
        }}
      >
        {targets.map((t) => {
          const cfg = dotConfig[t.status as string] ?? dotConfig.pending;
          const isActive = t.status === 'executing' || t.status === 'sent';

          const dot = (
            <div
              key={t.id}
              title={`${t.hostname} — ${t.status}`}
              style={{
                width: 10,
                height: 10,
                borderRadius: '50%',
                background: cfg.bg,
                flexShrink: 0,
                cursor: t.endpoint_id ? 'pointer' : 'default',
                boxShadow: cfg.glow !== 'none' ? `0 0 0 2px ${cfg.glow}` : undefined,
                animation: isActive ? 'pulse-dot 1.5s ease-in-out infinite' : undefined,
                transition: 'transform 0.1s',
              }}
            />
          );

          if (t.endpoint_id) {
            return (
              <Link
                key={t.id}
                to={`/endpoints/${t.endpoint_id}`}
                style={{ display: 'inline-flex', lineHeight: 0 }}
                title={`${t.hostname} — ${t.status}`}
              >
                {dot}
              </Link>
            );
          }

          return dot;
        })}
      </div>
    </div>
  );
}
