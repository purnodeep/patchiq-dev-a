import type { components } from '../../../api/types';
import { timeAgo } from '../../../lib/time';
import { formatDeploymentId } from '../../../lib/format';

type Deployment = components['schemas']['Deployment'];
type DeploymentWave = components['schemas']['DeploymentWave'];
type DeploymentTarget = components['schemas']['DeploymentTarget'];

interface TimelineEvent {
  type: 'green' | 'accent' | 'red' | 'gray';
  text: string;
  sub: string;
  time: string;
  category: 'system' | 'wave' | 'endpoint';
}

interface DeploymentTimelineProps {
  deployment: Deployment;
  waves?: DeploymentWave[];
  targets?: DeploymentTarget[];
  className?: string;
}

const dotColor: Record<string, string> = {
  green: 'var(--signal-healthy)',
  accent: 'var(--accent)',
  red: 'var(--signal-critical)',
  gray: 'var(--text-faint)',
};

const categoryIcon: Record<string, string> = {
  system: '◈',
  wave: '◉',
  endpoint: '·',
};

function buildTimeline(
  deployment: Deployment,
  waves?: DeploymentWave[],
  targets?: DeploymentTarget[],
): TimelineEvent[] {
  const events: TimelineEvent[] = [];

  events.push({
    type: 'gray',
    text: `Deployment created`,
    sub: `ID: ${formatDeploymentId(deployment.id)} · ${deployment.target_count} targets queued`,
    time: timeAgo(deployment.created_at),
    category: 'system',
  });

  if (deployment.started_at) {
    events.push({
      type: 'accent',
      text: 'Deployment started',
      sub: '',
      time: timeAgo(deployment.started_at),
      category: 'system',
    });
  }

  const sortedWaves = waves ? [...waves].sort((a, b) => a.wave_number - b.wave_number) : [];

  for (const wave of sortedWaves) {
    if (wave.started_at) {
      events.push({
        type: 'accent',
        text: `Wave ${wave.wave_number} started`,
        sub: `${wave.target_count} targets · ${wave.percentage}% of fleet`,
        time: timeAgo(wave.started_at),
        category: 'wave',
      });
    }

    const waveTargets = targets?.filter((t) => t.wave_id === wave.id) ?? [];
    for (const t of waveTargets) {
      if ((t.status as string) === 'succeeded') {
        events.push({
          type: 'green',
          text: `${t.hostname}: patch applied`,
          sub: `exit code ${t.exit_code ?? 0}`,
          time: timeAgo(t.completed_at),
          category: 'endpoint',
        });
      } else if (t.status === 'failed') {
        events.push({
          type: 'red',
          text: `${t.hostname}: FAILED${t.exit_code != null ? ` — exit ${t.exit_code}` : ''}`,
          sub: t.error ?? '',
          time: timeAgo(t.completed_at),
          category: 'endpoint',
        });
      }
    }

    if (wave.completed_at) {
      const isSuccess = wave.status === 'completed';
      events.push({
        type: isSuccess ? 'green' : 'red',
        text: `Wave ${wave.wave_number} ${wave.status}`,
        sub: `${wave.success_count}/${wave.target_count} succeeded`,
        time: timeAgo(wave.completed_at),
        category: 'wave',
      });
    }
  }

  if (deployment.completed_at) {
    const isSuccess = deployment.status === 'completed';
    events.push({
      type: isSuccess ? 'green' : 'red',
      text: `Deployment ${deployment.status}`,
      sub: `${deployment.success_count}/${deployment.target_count} succeeded`,
      time: timeAgo(deployment.completed_at),
      category: 'system',
    });
  }

  return events;
}

export function DeploymentTimeline({ deployment, waves, targets }: DeploymentTimelineProps) {
  const events = buildTimeline(deployment, waves, targets);

  return (
    <div style={{ display: 'flex', flexDirection: 'column' }}>
      {events.map((event, i) => {
        const color = dotColor[event.type];
        const isLast = i === events.length - 1;
        const isEndpoint = event.category === 'endpoint';
        const dotSize = isEndpoint ? 16 : 22;
        const dotOffset = isEndpoint ? 3 : 0;

        return (
          <div
            key={i}
            style={{
              display: 'flex',
              gap: 14,
              position: 'relative',
              paddingBottom: isLast ? 0 : 2,
            }}
          >
            {/* Connector line */}
            {!isLast && (
              <div
                style={{
                  position: 'absolute',
                  left: 11,
                  top: dotSize,
                  bottom: 0,
                  width: 1,
                  background: isEndpoint ? 'var(--border)' : 'var(--border-hover)',
                  opacity: isEndpoint ? 0.4 : 1,
                }}
              />
            )}

            {/* Dot */}
            <div style={{ position: 'relative', zIndex: 1, flexShrink: 0, marginTop: dotOffset }}>
              <div
                style={{
                  width: dotSize,
                  height: dotSize,
                  borderRadius: '50%',
                  border: `${isEndpoint ? 1.5 : 2}px solid ${color}`,
                  background: `color-mix(in srgb, ${color} ${isEndpoint ? '5' : '9'}%, transparent)`,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <div
                  style={{
                    width: isEndpoint ? 4 : 6,
                    height: isEndpoint ? 4 : 6,
                    borderRadius: '50%',
                    background: color,
                    animation:
                      event.type === 'accent' ? 'pulse-dot 1.5s ease-in-out infinite' : undefined,
                  }}
                />
              </div>
            </div>

            {/* Content */}
            <div style={{ flex: 1, paddingBottom: isEndpoint ? 10 : 16 }}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'baseline',
                  gap: 6,
                  marginBottom: 2,
                }}
              >
                <span
                  style={{
                    fontSize: isEndpoint ? 11 : 12,
                    color: isEndpoint ? 'var(--text-secondary)' : 'var(--text-primary)',
                    fontFamily: isEndpoint ? 'var(--font-mono)' : 'var(--font-sans)',
                  }}
                >
                  {categoryIcon[event.category]}&ensp;{event.text}
                </span>
              </div>
              <div
                style={{
                  display: 'flex',
                  gap: 8,
                  fontSize: 10,
                  color: 'var(--text-muted)',
                  fontFamily: 'var(--font-mono)',
                }}
              >
                {event.sub && <span>{event.sub} ·</span>}
                <span>{event.time}</span>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
