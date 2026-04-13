import { useState } from 'react';
import { DEPLOYMENT_TIMELINE_DATA, type TimelineEvent } from '@/data/mock-data';

// ── Status Orb ─────────────────────────────────────────────────────────────────
// Shape: circle = standard, diamond = wave, hexagon = workflow
interface StatusOrbProps {
  status: TimelineEvent['status'];
  deploymentType: TimelineEvent['deploymentType'];
}

function StatusOrb({ status, deploymentType }: StatusOrbProps) {
  const color =
    status === 'complete'
      ? '#34c759'
      : status === 'running'
        ? '#3b82f6'
        : status === 'failed'
          ? '#ff3b30'
          : '#ff9500';

  const isPulsing = status === 'running';

  // Diamond transform for wave, hexagon clip for workflow, circle for standard
  if (deploymentType === 'wave') {
    return (
      <div
        style={{
          position: 'relative',
          width: 22,
          height: 22,
          flexShrink: 0,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        {isPulsing && (
          <div
            style={{
              position: 'absolute',
              inset: -3,
              background: color,
              opacity: 0.25,
              transform: 'rotate(45deg)',
              borderRadius: 3,
              animation: 'pulse-ring-diamond 1.8s ease-out infinite',
            }}
          />
        )}
        <div
          style={{
            width: 18,
            height: 18,
            background: color,
            opacity: status === 'pending' ? 0.4 : 0.9,
            transform: 'rotate(45deg)',
            borderRadius: 3,
            boxShadow: `0 0 8px ${color}55`,
          }}
        />
      </div>
    );
  }

  if (deploymentType === 'workflow') {
    // Hexagon via CSS clip-path
    return (
      <div
        style={{
          position: 'relative',
          width: 24,
          height: 24,
          flexShrink: 0,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        {isPulsing && (
          <div
            style={{
              position: 'absolute',
              inset: -3,
              background: color,
              opacity: 0.2,
              clipPath: 'polygon(50% 0%, 93.3% 25%, 93.3% 75%, 50% 100%, 6.7% 75%, 6.7% 25%)',
              animation: 'pulse-ring-hex 1.8s ease-out infinite',
            }}
          />
        )}
        <div
          style={{
            width: 20,
            height: 20,
            background: color,
            opacity: status === 'pending' ? 0.4 : 0.9,
            clipPath: 'polygon(50% 0%, 93.3% 25%, 93.3% 75%, 50% 100%, 6.7% 75%, 6.7% 25%)',
            boxShadow: `0 0 8px ${color}55`,
          }}
        />
      </div>
    );
  }

  // Standard — circle
  return (
    <div
      style={{
        position: 'relative',
        width: 22,
        height: 22,
        flexShrink: 0,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      {isPulsing && (
        <div
          style={{
            position: 'absolute',
            inset: -3,
            borderRadius: '50%',
            background: color,
            opacity: 0,
            animation: 'pulse-ring 1.8s ease-out infinite',
          }}
        />
      )}
      <div
        style={{
          width: 16,
          height: 16,
          borderRadius: '50%',
          background: color,
          opacity: status === 'pending' ? 0.35 : 0.9,
          boxShadow: `0 0 8px ${color}55`,
          border: `2px solid ${color}44`,
        }}
      />
    </div>
  );
}

// ── Type Badge ─────────────────────────────────────────────────────────────────
const TYPE_LABELS: Record<TimelineEvent['deploymentType'], string> = {
  standard: 'Standard',
  wave: 'Wave',
  workflow: 'Workflow',
};

const TYPE_COLORS: Record<TimelineEvent['deploymentType'], string> = {
  standard: 'var(--color-primary)',
  wave: 'var(--color-cyan)',
  workflow: 'var(--color-purple)',
};

// ── Progress Bar ───────────────────────────────────────────────────────────────
function ProgressBar({ progress }: { progress: number }) {
  return (
    <div
      style={{
        height: 4,
        width: '100%',
        background: 'var(--color-separator)',
        borderRadius: 2,
        overflow: 'hidden',
        marginTop: 6,
      }}
    >
      <div
        style={{
          height: '100%',
          width: `${progress}%`,
          borderRadius: 2,
          background: 'linear-gradient(90deg, #34c759, #5ac8fa)',
          boxShadow: '0 0 6px #34c75988',
          transition: 'width 0.6s cubic-bezier(0.4,0,0.2,1)',
          animation: 'progress-shimmer 2s ease infinite',
        }}
      />
    </div>
  );
}

// ── Timeline Item ──────────────────────────────────────────────────────────────
interface TimelineItemProps {
  event: TimelineEvent;
  isLast: boolean;
  index: number;
}

function TimelineItem({ event, isLast, index }: TimelineItemProps) {
  const [expanded, setExpanded] = useState(false);

  const lineColor =
    event.status === 'complete'
      ? '#34c75940'
      : event.status === 'running'
        ? '#3b82f640'
        : event.status === 'failed'
          ? '#ff3b3040'
          : 'var(--color-separator)';

  const statusColor =
    event.status === 'complete'
      ? '#34c759'
      : event.status === 'running'
        ? '#3b82f6'
        : event.status === 'failed'
          ? '#ff3b30'
          : '#ff9500';

  const statusLabel =
    event.status === 'complete'
      ? 'Complete'
      : event.status === 'running'
        ? 'Running'
        : event.status === 'failed'
          ? 'Failed'
          : 'Pending';

  return (
    <div
      style={{
        display: 'flex',
        gap: 0,
        animation: `fade-in-up 0.4s ease both`,
        animationDelay: `${index * 60}ms`,
      }}
    >
      {/* Orb + connecting line column */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          width: 36,
          flexShrink: 0,
        }}
      >
        <StatusOrb status={event.status} deploymentType={event.deploymentType} />
        {!isLast && (
          <div
            style={{
              flex: 1,
              width: 2,
              background: lineColor,
              borderRadius: 1,
              minHeight: 24,
              marginTop: 4,
            }}
          />
        )}
      </div>

      {/* Content */}
      <div
        style={{
          flex: 1,
          paddingBottom: isLast ? 0 : 16,
          paddingLeft: 4,
        }}
      >
        {/* Header row */}
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            justifyContent: 'space-between',
            gap: 8,
            cursor: 'pointer',
          }}
          onClick={() => setExpanded((v) => !v)}
        >
          <div style={{ flex: 1, minWidth: 0 }}>
            <div
              style={{
                fontSize: 12,
                fontWeight: 600,
                color: 'var(--color-foreground)',
                lineHeight: 1.3,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {event.title}
            </div>
            <div
              style={{
                fontSize: 10,
                color: 'var(--color-muted)',
                marginTop: 2,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {event.subtitle}
            </div>
          </div>

          {/* Right meta */}
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'flex-end',
              gap: 3,
              flexShrink: 0,
            }}
          >
            <div
              style={{
                fontSize: 9,
                fontWeight: 600,
                color: statusColor,
                padding: '1px 6px',
                borderRadius: 999,
                background: `${statusColor}18`,
                border: `1px solid ${statusColor}30`,
                letterSpacing: '0.03em',
                textTransform: 'uppercase',
              }}
            >
              {statusLabel}
            </div>
            <div
              style={{
                fontSize: 9,
                color: 'var(--color-muted)',
                display: 'flex',
                alignItems: 'center',
                gap: 3,
              }}
            >
              {/* Type shape mini-icon */}
              <span
                style={{
                  fontSize: 9,
                  color: TYPE_COLORS[event.deploymentType],
                  fontWeight: 600,
                }}
              >
                {TYPE_LABELS[event.deploymentType]}
              </span>
              <span>·</span>
              <span>{event.time}</span>
            </div>
          </div>
        </div>

        {/* Progress bar for running items */}
        {event.status === 'running' && event.progress !== undefined && (
          <div style={{ marginTop: 4 }}>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                fontSize: 9,
                color: 'var(--color-muted)',
                marginBottom: 2,
              }}
            >
              <span>Progress</span>
              <span style={{ fontWeight: 700, color: '#3b82f6' }}>{event.progress}%</span>
            </div>
            <ProgressBar progress={event.progress} />
          </div>
        )}

        {/* Expandable detail section */}
        <div
          style={{
            overflow: 'hidden',
            maxHeight: expanded ? 80 : 0,
            transition: 'max-height 0.28s cubic-bezier(0.4,0,0.2,1), opacity 0.2s ease',
            opacity: expanded ? 1 : 0,
          }}
        >
          {event.details && (
            <div
              style={{
                marginTop: 8,
                padding: '6px 10px',
                background: 'var(--color-separator)',
                borderRadius: 6,
                fontSize: 10,
                color: 'var(--color-muted)',
                lineHeight: 1.5,
                borderLeft: `2px solid ${statusColor}60`,
              }}
            >
              {event.details}
            </div>
          )}
        </div>

        {/* Expand toggle hint */}
        {event.details && (
          <button
            onClick={() => setExpanded((v) => !v)}
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              fontSize: 9,
              color: 'var(--color-muted)',
              padding: '2px 0',
              marginTop: 2,
              display: 'flex',
              alignItems: 'center',
              gap: 3,
              transition: 'color 0.15s ease',
            }}
          >
            <svg
              width={8}
              height={8}
              viewBox="0 0 8 8"
              style={{
                transform: expanded ? 'rotate(180deg)' : 'rotate(0deg)',
                transition: 'transform 0.2s ease',
              }}
            >
              <polyline
                points="1,2 4,6 7,2"
                fill="none"
                stroke="currentColor"
                strokeWidth={1.5}
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
            {expanded ? 'Hide details' : 'Show details'}
          </button>
        )}
      </div>
    </div>
  );
}

// ── Main Component ─────────────────────────────────────────────────────────────
export function DeploymentTimeline() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column' }}>
      <style>{`
        @keyframes pulse-ring-diamond {
          0% { transform: rotate(45deg) scale(0.8); opacity: 0.6; }
          100% { transform: rotate(45deg) scale(1.8); opacity: 0; }
        }
        @keyframes pulse-ring-hex {
          0% { opacity: 0.5; transform: scale(0.8); }
          100% { opacity: 0; transform: scale(1.9); }
        }
        @keyframes progress-shimmer {
          0% { background-position: -200% center; }
          100% { background-position: 200% center; }
        }
      `}</style>

      {DEPLOYMENT_TIMELINE_DATA.map((event, i) => (
        <TimelineItem
          key={event.id}
          event={event}
          isLast={i === DEPLOYMENT_TIMELINE_DATA.length - 1}
          index={i}
        />
      ))}
    </div>
  );
}
