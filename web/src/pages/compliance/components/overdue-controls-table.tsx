import { useState } from 'react';
import { useNavigate } from 'react-router';
import { useOverdueControls, type OverdueControl } from '../../../api/hooks/useCompliance';

const thStyle: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 10,
  fontWeight: 500,
  textTransform: 'uppercase' as const,
  letterSpacing: '0.05em',
  color: 'var(--text-muted)',
  padding: '8px 12px',
  textAlign: 'left' as const,
  background: 'var(--bg-inset)',
  borderBottom: '1px solid var(--border)',
  whiteSpace: 'nowrap' as const,
};

function getStatusColor(status: string): string {
  if (status === 'fail') return 'var(--signal-critical)';
  if (status === 'partial') return 'var(--signal-warning)';
  return 'var(--text-muted)';
}

const PAGE_SIZE = 10;

export function OverdueControlsTable() {
  const { data: rawControls, isLoading } = useOverdueControls();
  const navigate = useNavigate();
  const [page, setPage] = useState(0);

  // Deduplicate by framework_id + control_id (backend may return duplicates from multiple eval runs)
  const controls = (() => {
    if (!rawControls) return [];
    const seen = new Map<string, OverdueControl>();
    for (const c of rawControls) {
      const key = `${c.framework_id}-${c.control_id}`;
      if (!seen.has(key)) seen.set(key, c);
    }
    return Array.from(seen.values());
  })();

  const totalPages = Math.ceil(controls.length / PAGE_SIZE);
  const pagedControls = controls.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  if (isLoading || controls.length === 0) return null;

  return (
    <div>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          marginBottom: 12,
        }}
      >
        <div
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 14,
            fontWeight: 600,
            color: 'var(--text-primary)',
          }}
        >
          Overdue Controls
        </div>
        <span
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            minWidth: 22,
            height: 18,
            padding: '0 6px',
            background: 'var(--bg-card-hover)',
            border: '1px solid var(--border-strong)',
            borderRadius: 999,
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 700,
            color: 'var(--signal-critical)',
          }}
        >
          {controls.length}
        </span>
      </div>

      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          overflow: 'hidden',
        }}
      >
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', minWidth: 900, borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={thStyle}>Framework</th>
                <th style={thStyle}>Control Id</th>
                <th style={thStyle}>Control Name</th>
                <th style={thStyle}>Status</th>
                <th style={thStyle}>Sla Deadline</th>
                <th style={thStyle}>Overdue By</th>
                <th style={thStyle}>Affected Endpoints</th>
                <th style={thStyle}>Action</th>
              </tr>
            </thead>
            <tbody>
              {pagedControls.map((c: OverdueControl, index: number) => (
                <tr
                  key={`${c.framework_id}-${c.control_id}-${index}`}
                  style={{ cursor: 'pointer' }}
                  onClick={() =>
                    navigate(`/compliance/frameworks/${c.framework_id}#${c.control_id}`)
                  }
                  onMouseEnter={(e) =>
                    ((e.currentTarget as HTMLTableRowElement).style.background =
                      'color-mix(in srgb, var(--signal-critical) 1%, transparent)')
                  }
                  onMouseLeave={(e) =>
                    ((e.currentTarget as HTMLTableRowElement).style.background = '')
                  }
                >
                  <td
                    style={{
                      padding: '9px 12px',
                      borderBottom: '1px solid var(--border)',
                    }}
                  >
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 11,
                        color: 'var(--text-secondary)',
                      }}
                    >
                      {c.framework_name}
                    </span>
                  </td>
                  <td
                    style={{
                      padding: '9px 12px',
                      borderBottom: '1px solid var(--border)',
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      color: 'var(--text-muted)',
                    }}
                  >
                    {c.control_id}
                  </td>
                  <td
                    style={{
                      padding: '9px 12px',
                      borderBottom: '1px solid var(--border)',
                      fontFamily: 'var(--font-sans)',
                      fontSize: 12,
                      color: 'var(--text-primary)',
                    }}
                  >
                    {c.control_name}
                  </td>
                  <td
                    style={{
                      padding: '9px 12px',
                      borderBottom: '1px solid var(--border)',
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      fontWeight: 600,
                      color: getStatusColor(c.status),
                      textTransform: 'uppercase',
                    }}
                  >
                    {c.status}
                  </td>
                  <td
                    style={{
                      padding: '9px 12px',
                      borderBottom: '1px solid var(--border)',
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      color: 'var(--text-muted)',
                    }}
                  >
                    {new Date(c.sla_deadline_at).toLocaleDateString('en', {
                      month: 'short',
                      day: 'numeric',
                      year: 'numeric',
                    })}
                  </td>
                  <td
                    style={{
                      padding: '9px 12px',
                      borderBottom: '1px solid var(--border)',
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      fontWeight: 700,
                      color: 'var(--signal-critical)',
                    }}
                  >
                    <span aria-label={`${c.days_overdue} days overdue`}>{c.days_overdue}d</span>
                  </td>
                  <td
                    style={{
                      padding: '9px 12px',
                      borderBottom: '1px solid var(--border)',
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      color: 'var(--text-secondary)',
                    }}
                  >
                    {c.affected_endpoints}
                  </td>
                  <td style={{ padding: '9px 12px', borderBottom: '1px solid var(--border)' }}>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        navigate(`/compliance/frameworks/${c.framework_id}#${c.control_id}`);
                      }}
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 10,
                        fontWeight: 600,
                        color: 'var(--accent)',
                        background: 'none',
                        border: 'none',
                        cursor: 'pointer',
                        textDecoration: 'underline',
                        textUnderlineOffset: 2,
                      }}
                      aria-label={`View control ${c.control_id} in ${c.framework_name}`}
                    >
                      View →
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Pagination footer */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '8px 12px',
            borderTop: '1px solid var(--border)',
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            color: 'var(--text-muted)',
          }}
        >
          <span>
            Showing {page * PAGE_SIZE + 1}–{Math.min((page + 1) * PAGE_SIZE, controls.length)} of{' '}
            {controls.length}
          </span>
          {totalPages > 1 && (
            <div style={{ display: 'flex', gap: 4 }}>
              <button
                type="button"
                disabled={page === 0}
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                style={{
                  padding: '3px 8px',
                  borderRadius: 4,
                  border: '1px solid var(--border)',
                  background: 'transparent',
                  color: page === 0 ? 'var(--text-faint)' : 'var(--text-secondary)',
                  cursor: page === 0 ? 'default' : 'pointer',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                }}
              >
                Prev
              </button>
              <button
                type="button"
                disabled={page >= totalPages - 1}
                onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                style={{
                  padding: '3px 8px',
                  borderRadius: 4,
                  border: '1px solid var(--border)',
                  background: 'transparent',
                  color: page >= totalPages - 1 ? 'var(--text-faint)' : 'var(--text-secondary)',
                  cursor: page >= totalPages - 1 ? 'default' : 'pointer',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                }}
              >
                Next
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
