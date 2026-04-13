import {
  Clock,
  Loader2,
  Play,
  Zap,
  CircleCheckBig,
  Copy,
  MoreHorizontal,
  Pencil,
} from 'lucide-react';
import { Link } from 'react-router';
import { timeAgo } from '../../lib/time';
import { nodeTypeIcon, defaultPipelineTypes } from './workflow-node-styles';
import type { WorkflowListItem } from '../../flows/policy-workflow/types';

// Status config using design-system tokens only
const statusConfig: Record<string, { label: string; color: string }> = {
  published: { label: 'Published', color: 'var(--accent)' },
  draft: { label: 'Draft', color: 'var(--signal-warning)' },
  archived: { label: 'Archived', color: 'var(--text-muted)' },
};

function LastRunIndicator({ status }: { status: string | null }) {
  if (!status || status === '') return null;
  switch (status) {
    case 'completed':
      return <span style={{ color: 'var(--signal-healthy)', fontSize: 10 }}>✓</span>;
    case 'failed':
      return <span style={{ color: 'var(--signal-critical)', fontSize: 10 }}>✗</span>;
    case 'running':
      return (
        <Loader2
          style={{ width: 10, height: 10, color: 'var(--signal-warning)' }}
          className="animate-spin"
        />
      );
    default:
      return null;
  }
}

interface WorkflowCardProps {
  workflow: WorkflowListItem;
  onClick: () => void;
  selected?: boolean;
}

export function WorkflowCard({ workflow, onClick, selected }: WorkflowCardProps) {
  const status = statusConfig[workflow.current_status] ?? statusConfig.draft;

  return (
    <div
      onClick={onClick}
      style={{
        background: 'var(--bg-card)',
        border: `1px solid ${selected ? 'var(--accent)' : 'var(--border)'}`,
        borderRadius: 10,
        padding: '16px',
        cursor: 'pointer',
        position: 'relative',
        overflow: 'hidden',
        boxShadow: selected
          ? '0 0 0 1px var(--accent), 0 0 12px var(--accent-border)'
          : 'var(--shadow-sm)',
        transition: 'border-color 150ms, box-shadow 150ms, transform 150ms',
      }}
      onMouseEnter={(e) => {
        if (!selected) {
          (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border-hover)';
          (e.currentTarget as HTMLDivElement).style.transform = 'translateY(-1px)';
        }
      }}
      onMouseLeave={(e) => {
        if (!selected) {
          (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)';
          (e.currentTarget as HTMLDivElement).style.transform = 'translateY(0)';
        }
      }}
    >
      {/* Top accent line */}
      <div
        style={{
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          height: 2,
          background: status.color,
          opacity: 0.6,
        }}
      />

      {/* Header row */}
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          justifyContent: 'space-between',
          gap: 8,
          marginBottom: 12,
        }}
      >
        <div style={{ minWidth: 0, flex: 1 }}>
          <div
            style={{
              fontSize: 13,
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              marginBottom: 4,
            }}
          >
            {workflow.name}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <Zap style={{ width: 10, height: 10, color: 'var(--text-muted)' }} />
            <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
              {workflow.node_count} node{workflow.node_count !== 1 ? 's' : ''}
            </span>
          </div>
        </div>
        <div
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            padding: '2px 0',
            fontSize: 10,
            fontFamily: 'var(--font-mono)',
            color: status.color,
            flexShrink: 0,
          }}
        >
          {status.label}
        </div>
      </div>

      {/* Mini pipeline */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 0,
          marginBottom: 12,
          overflow: 'hidden',
        }}
      >
        {workflow.node_count > 0 ? (
          Array.from({ length: Math.min(workflow.node_count, 7) }).map((_, i) => {
            const typeKey = defaultPipelineTypes[i % defaultPipelineTypes.length];
            const Icon = nodeTypeIcon[typeKey] ?? CircleCheckBig;
            return (
              <div key={i} style={{ display: 'contents' }}>
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    width: 28,
                    height: 28,
                    borderRadius: 7,
                    border: '1px solid var(--border)',
                    background: 'var(--bg-inset)',
                    flexShrink: 0,
                  }}
                >
                  <Icon style={{ width: 11, height: 11, color: 'var(--text-muted)' }} />
                </div>
                {i < Math.min(workflow.node_count, 7) - 1 && (
                  <div
                    style={{
                      height: 1,
                      minWidth: 8,
                      maxWidth: 24,
                      flex: 1,
                      background: 'var(--border)',
                    }}
                  />
                )}
              </div>
            );
          })
        ) : (
          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>No nodes defined</span>
        )}
      </div>

      {/* Stats row */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          borderTop: '1px solid var(--border)',
          paddingTop: 10,
          fontSize: 11,
          color: 'var(--text-muted)',
        }}
      >
        <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <Play style={{ width: 10, height: 10 }} />
          {workflow.total_runs} runs
        </span>
        <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <Clock style={{ width: 10, height: 10 }} />
          {workflow.last_run_at ? timeAgo(workflow.last_run_at) : 'Never'}
          {workflow.last_run_status && <LastRunIndicator status={workflow.last_run_status} />}
        </span>
        <span style={{ marginLeft: 'auto', fontFamily: 'var(--font-mono)' }}>
          v{workflow.current_version}
        </span>
      </div>

      {/* Action buttons on hover */}
      <div
        className="workflow-card-actions"
        style={{
          position: 'absolute',
          bottom: 12,
          right: 12,
          display: 'flex',
          gap: 4,
          opacity: 0,
          transition: 'opacity 150ms',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <Link
          to={`/workflows/${workflow.id}/edit`}
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: 26,
            height: 26,
            borderRadius: 6,
            border: '1px solid var(--border)',
            background: 'var(--bg-card)',
            color: 'var(--text-secondary)',
            transition: 'border-color 100ms',
          }}
          title="Edit"
        >
          <Pencil style={{ width: 11, height: 11 }} />
        </Link>
        <button
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: 26,
            height: 26,
            borderRadius: 6,
            border: '1px solid var(--border)',
            background: 'var(--bg-card)',
            color: 'var(--text-secondary)',
            cursor: 'pointer',
          }}
          title="Duplicate"
        >
          <Copy style={{ width: 11, height: 11 }} />
        </button>
        <button
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: 26,
            height: 26,
            borderRadius: 6,
            border: '1px solid var(--border)',
            background: 'var(--bg-card)',
            color: 'var(--text-secondary)',
            cursor: 'pointer',
          }}
          title="More"
        >
          <MoreHorizontal style={{ width: 11, height: 11 }} />
        </button>
      </div>

      <style>{`
        .workflow-card-actions { opacity: 0; }
        div:hover > .workflow-card-actions { opacity: 1; }
      `}</style>
    </div>
  );
}
