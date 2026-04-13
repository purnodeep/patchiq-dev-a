import { useState, useEffect, Fragment, useCallback, useMemo } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router';
import { Search, ShieldX, Eye, Rocket, Copy, ExternalLink } from 'lucide-react';
import { useCVEs, useCVE, useCVESummary } from '../../api/hooks/useCVEs';
import { EmptyState } from '../../components/EmptyState';
import { DeploymentWizard } from '../../components/DeploymentWizard';
import type { DeploymentWizardInitialState } from '../../types/deployment-wizard';
import type { CVEListItem } from '../../types/cves';

// ─── Helpers ──────────────────────────────────────────────────────────────────

function cvssColor(score: number | null): string {
  if (score === null) return 'var(--text-muted)';
  if (score >= 9) return 'var(--signal-critical)';
  if (score >= 7) return 'var(--signal-warning)';
  return 'var(--text-secondary)';
}

function severityColor(severity: string): string {
  switch (severity.toLowerCase()) {
    case 'critical':
      return 'var(--signal-critical)';
    case 'high':
      return 'var(--signal-warning)';
    default:
      return 'var(--text-secondary)';
  }
}

function displaySeverity(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function relativeTime(dateStr: string | undefined | null): string {
  if (!dateStr) return '—';
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diff = Math.floor((now - then) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 86400 * 30) return `${Math.floor(diff / 86400)}d ago`;
  if (diff < 86400 * 365) return `${Math.floor(diff / (86400 * 30))}mo ago`;
  return `${Math.floor(diff / (86400 * 365))}y ago`;
}

// ─── Skeleton Rows ────────────────────────────────────────────────────────────

function SkeletonRows({ cols, rows = 8 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }).map((_, i) => (
        <tr key={i}>
          {Array.from({ length: cols }).map((__, j) => (
            <td key={j} style={{ padding: '10px 12px' }}>
              <div
                style={{
                  height: 14,
                  borderRadius: 4,
                  background: 'var(--bg-inset)',
                  width: j === 0 ? '55%' : j === 1 ? '30%' : '50%',
                  animation: 'pulse 1.5s ease-in-out infinite',
                }}
              />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

// ─── Skeleton Cards ───────────────────────────────────────────────────────────

function SkeletonCards({ count = 8 }: { count?: number }) {
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
        gap: 12,
        padding: 16,
      }}
    >
      {Array.from({ length: count }).map((_, i) => (
        <div
          key={i}
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 10,
            padding: 16,
            height: 180,
            animation: 'pulse 1.5s ease-in-out infinite',
          }}
        />
      ))}
    </div>
  );
}

// ─── Filter Toggle Pill ───────────────────────────────────────────────────────

interface FilterToggleProps {
  label: string;
  count?: number;
  active: boolean;
  onClick: () => void;
  activeColor?: string;
}

function FilterToggle({
  label,
  count,
  active,
  onClick,
  activeColor = 'var(--signal-critical)',
}: FilterToggleProps) {
  const [hovered, setHovered] = useState(false);
  return (
    <button
      type="button"
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 5,
        padding: '4px 10px',
        borderRadius: 20,
        border: '1px solid',
        cursor: 'pointer',
        fontSize: 11.5,
        fontWeight: 500,
        transition: 'all 0.15s',
        background: active ? `color-mix(in srgb, ${activeColor} 9%, transparent)` : 'transparent',
        borderColor: active ? activeColor : hovered ? 'var(--border-hover)' : 'var(--border)',
        color: active ? activeColor : hovered ? 'var(--text-primary)' : 'var(--text-muted)',
        outline: 'none',
      }}
    >
      {label}
      {count !== undefined && (
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 700,
            color: active ? activeColor : 'var(--text-faint)',
          }}
        >
          {count}
        </span>
      )}
    </button>
  );
}

// ─── Pagination Button ────────────────────────────────────────────────────────

interface PaginationButtonProps {
  children: React.ReactNode;
  active?: boolean;
  disabled?: boolean;
  onClick?: () => void;
}

function PaginationButton({ children, active, disabled, onClick }: PaginationButtonProps) {
  const [hovered, setHovered] = useState(false);
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        padding: '4px 9px',
        fontSize: 11,
        fontFamily: 'var(--font-mono)',
        borderRadius: 5,
        border: '1px solid',
        cursor: disabled ? 'not-allowed' : 'pointer',
        transition: 'all 0.15s',
        background: active
          ? 'color-mix(in srgb, var(--accent) 12%, transparent)'
          : hovered && !disabled
            ? 'color-mix(in srgb, white 4%, transparent)'
            : 'transparent',
        borderColor: active ? 'var(--accent)' : 'var(--border)',
        color: active ? 'var(--accent)' : disabled ? 'var(--text-faint)' : 'var(--text-secondary)',
        opacity: disabled && !active ? 0.45 : 1,
      }}
    >
      {children}
    </button>
  );
}

// ─── Expanded CVE Row ─────────────────────────────────────────────────────────

const EXPANDED_CARD: React.CSSProperties = {
  background: 'var(--bg-inset)',
  border: '1px solid var(--border)',
  borderRadius: 6,
  padding: '14px 16px',
};

const EXPANDED_LABEL: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 9,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.07em',
  color: 'var(--text-muted)',
  marginBottom: 10,
};

const EXPANDED_BTN_BASE: React.CSSProperties = {
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
  letterSpacing: '0.01em',
  width: '100%',
};

function ExpandedCVERow({
  cve,
  onDeploy,
}: {
  cve: CVEListItem;
  onDeploy: (patchIds: string[]) => void;
}) {
  const navigate = useNavigate();
  const { data: detail } = useCVE(cve.id);
  const patchIds = detail?.patches?.map((p) => p.id) ?? [];
  return (
    <tr>
      <td colSpan={10} style={{ padding: 0, borderBottom: '1px solid var(--border)' }}>
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
          {/* Description card — wider, more detail */}
          <div style={{ ...EXPANDED_CARD, flex: '0 0 500px' }}>
            <div style={EXPANDED_LABEL}>Description</div>
            <p
              style={{
                fontSize: 15,
                color: 'var(--text-secondary)',
                lineHeight: 1.6,
                margin: '0 0 14px',
              }}
            >
              {cve.description ?? 'No description available.'}
            </p>
            <div
              style={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: '10px 24px',
                borderTop: '1px solid var(--border)',
                paddingTop: 12,
              }}
            >
              {[
                {
                  label: 'Severity',
                  value: displaySeverity(cve.severity),
                  color: severityColor(cve.severity),
                },
                {
                  label: 'CVSS Score',
                  value: cve.cvss_v3_score != null ? String(cve.cvss_v3_score) : '—',
                  color: cvssColor(cve.cvss_v3_score),
                },
                {
                  label: 'Published',
                  value: relativeTime(cve.published_at),
                  color: 'var(--text-muted)',
                },
                {
                  label: 'Last Modified',
                  value: relativeTime(cve.nvd_last_modified),
                  color: 'var(--text-muted)',
                },
              ].map(({ label, value, color }) => (
                <div key={label} style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 10,
                      textTransform: 'uppercase',
                      letterSpacing: '0.06em',
                      color: 'var(--text-faint)',
                    }}
                  >
                    {label}
                  </span>
                  <span
                    style={{ fontFamily: 'var(--font-mono)', fontSize: 15, fontWeight: 700, color }}
                  >
                    {value}
                  </span>
                </div>
              ))}
            </div>
            {cve.cvss_v3_vector && (
              <div
                style={{
                  marginTop: 8,
                  padding: '5px 8px',
                  background: 'color-mix(in srgb, white 4%, transparent)',
                  border: '1px solid var(--border)',
                  borderRadius: 4,
                  fontFamily: 'var(--font-mono)',
                  fontSize: 9.5,
                  color: 'var(--text-muted)',
                  wordBreak: 'break-all',
                }}
              >
                {cve.cvss_v3_vector}
              </div>
            )}
          </div>

          {/* Threat Detail card */}
          <div style={{ ...EXPANDED_CARD, flex: '0 0 480px', marginLeft: 24 }}>
            <div style={EXPANDED_LABEL}>Threat Detail</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {[
                {
                  label: 'Attack Vector',
                  value: cve.attack_vector ?? '—',
                  color: 'var(--text-primary)',
                },
                {
                  label: 'Exploit Available',
                  value: cve.exploit_available ? 'Yes' : 'No',
                  color: cve.exploit_available ? 'var(--signal-critical)' : 'var(--text-secondary)',
                },
                {
                  label: 'KEV Listed',
                  value: cve.cisa_kev_due_date ? 'Yes' : 'No',
                  color: cve.cisa_kev_due_date ? 'var(--signal-warning)' : 'var(--text-secondary)',
                },
                {
                  label: 'Endpoints Affected',
                  value: String(cve.affected_endpoint_count),
                  color: 'var(--text-primary)',
                },
                {
                  label: 'Patches Available',
                  value: cve.patch_available
                    ? `${cve.patch_count} patch${cve.patch_count !== 1 ? 'es' : ''}`
                    : 'None',
                  color: cve.patch_available ? 'var(--signal-healthy)' : 'var(--text-muted)',
                },
              ].map(({ label, value, color }) => (
                <div
                  key={label}
                  style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}
                >
                  <span style={{ color: 'var(--text-secondary)' }}>{label}</span>
                  <span
                    style={{ fontWeight: 600, color, fontFamily: 'var(--font-mono)', fontSize: 11 }}
                  >
                    {value}
                  </span>
                </div>
              ))}
            </div>
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
            <button
              type="button"
              style={{
                ...EXPANDED_BTN_BASE,
                color: 'var(--btn-accent-text, #000)',
                borderColor: 'var(--accent)',
                background: 'var(--accent)',
              }}
              onClick={(e) => {
                e.stopPropagation();
                onDeploy(patchIds);
              }}
            >
              ⎌ Deploy Fix
            </button>
            <button
              type="button"
              style={EXPANDED_BTN_BASE}
              onClick={(e) => {
                e.stopPropagation();
                navigate(`/cves/${cve.id}`);
              }}
            >
              View Details →
            </button>
          </div>
        </div>
      </td>
    </tr>
  );
}

// ─── Stat Card (inline, no icon needed) ──────────────────────────────────────

interface InlineStatCardProps {
  value: number;
  label: string;
  valueColor: string;
  active?: boolean;
  onClick?: () => void;
}

function InlineStatCard({ value, label, valueColor, active, onClick }: InlineStatCardProps) {
  const [hovered, setHovered] = useState(false);
  return (
    <div
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        background: active
          ? 'var(--bg-card-hover)'
          : hovered
            ? 'var(--bg-card-hover)'
            : 'var(--bg-card)',
        border: `1px solid ${active ? 'var(--border-strong)' : 'var(--border)'}`,
        borderRadius: 8,
        boxShadow: 'var(--shadow-sm)',
        padding: '12px 14px',
        cursor: onClick ? 'pointer' : 'default',
        transition: 'all 0.15s',
      }}
    >
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 22,
          fontWeight: 700,
          color: valueColor,
          lineHeight: 1,
          letterSpacing: '-0.02em',
        }}
      >
        {value}
      </div>
      <div
        style={{
          fontSize: 10,
          fontFamily: 'var(--font-mono)',
          color: 'var(--text-muted)',
          marginTop: 4,
          fontWeight: 500,
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
        }}
      >
        {label}
      </div>
    </div>
  );
}

// ─── Kebab Menu ──────────────────────────────────────────────────────────────

interface KebabMenuItem {
  label: string;
  icon?: React.ReactNode;
  action: () => void;
  destructive?: boolean;
}

function KebabMenu({ items, ariaLabel }: { items: KebabMenuItem[]; ariaLabel?: string }) {
  const [open, setOpen] = useState(false);
  return (
    <div style={{ position: 'relative' }}>
      <button
        type="button"
        aria-label={ariaLabel ?? 'Actions menu'}
        onClick={(e) => {
          e.stopPropagation();
          setOpen((p) => !p);
        }}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
        style={{
          width: 24,
          height: 24,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: open ? 'color-mix(in srgb, white 6%, transparent)' : 'transparent',
          border: 'none',
          borderRadius: 4,
          cursor: 'pointer',
          color: open ? 'var(--text-primary)' : 'var(--text-muted)',
          padding: 0,
          transition: 'color 0.15s, background 0.15s',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.color = 'var(--text-primary)';
          e.currentTarget.style.background = 'color-mix(in srgb, white 6%, transparent)';
        }}
        onMouseLeave={(e) => {
          if (!open) {
            e.currentTarget.style.color = 'var(--text-muted)';
            e.currentTarget.style.background = 'transparent';
          }
        }}
      >
        <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor">
          <circle cx="6" cy="2" r="1.2" />
          <circle cx="6" cy="6" r="1.2" />
          <circle cx="6" cy="10" r="1.2" />
        </svg>
      </button>
      {open && (
        <div
          style={{
            position: 'absolute',
            right: 0,
            top: '100%',
            marginTop: 4,
            minWidth: 180,
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            boxShadow: 'var(--shadow-lg, 0 8px 24px rgba(0,0,0,.25))',
            zIndex: 50,
            padding: '4px 0',
            overflow: 'hidden',
          }}
        >
          {items.map((item) => (
            <Fragment key={item.label}>
              {item.destructive && (
                <div style={{ height: 1, background: 'var(--border)', margin: '4px 0' }} />
              )}
              <button
                type="button"
                onMouseDown={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  item.action();
                  setOpen(false);
                }}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  width: '100%',
                  padding: '7px 14px',
                  fontSize: 12,
                  fontFamily: 'var(--font-sans)',
                  color: item.destructive ? 'var(--signal-critical)' : 'var(--text-secondary)',
                  background: 'transparent',
                  border: 'none',
                  cursor: 'pointer',
                  textAlign: 'left',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--bg-inset)';
                  if (!item.destructive) e.currentTarget.style.color = 'var(--text-primary)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'transparent';
                  e.currentTarget.style.color = item.destructive
                    ? 'var(--signal-critical)'
                    : 'var(--text-secondary)';
                }}
              >
                {item.icon}
                {item.label}
              </button>
            </Fragment>
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Sort Header ─────────────────────────────────────────────────────────────

const TH: React.CSSProperties = {
  padding: '9px 12px',
  textAlign: 'left' as const,
  fontFamily: 'var(--font-mono)',
  fontSize: 11,
  fontWeight: 600,
  letterSpacing: '0.05em',
  color: 'var(--text-muted)',
  textTransform: 'uppercase' as const,
  whiteSpace: 'nowrap' as const,
};

function SortHeader({
  label,
  colKey,
  sortCol,
  sortDir,
  onSort,
}: {
  label: string;
  colKey: string;
  sortCol: string | null;
  sortDir: 'asc' | 'desc';
  onSort: (col: string) => void;
}) {
  const active = sortCol === colKey;
  const [hovered, setHovered] = useState(false);
  return (
    <th
      style={{ ...TH, cursor: 'pointer', userSelect: 'none' }}
      aria-label={`Sort by ${label}`}
      onClick={() => onSort(colKey)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        <span style={{ color: active ? 'var(--text-emphasis)' : undefined }}>{label}</span>
        <svg
          width="10"
          height="10"
          viewBox="0 0 10 10"
          fill="none"
          style={{
            opacity: active ? 1 : hovered ? 0.5 : 0,
            transition: 'opacity 0.15s',
            flexShrink: 0,
          }}
        >
          {(!active || sortDir === 'asc') && (
            <path
              d="M5 2L8 5.5H2L5 2Z"
              fill={active ? 'var(--text-emphasis)' : 'var(--text-muted)'}
            />
          )}
          {(!active || sortDir === 'desc') && (
            <path
              d="M5 8L2 4.5H8L5 8Z"
              fill={active ? 'var(--text-emphasis)' : 'var(--text-muted)'}
            />
          )}
        </svg>
      </div>
    </th>
  );
}

// ─── View Toggle Button ──────────────────────────────────────────────────────

function ViewToggleButton({
  active,
  onClick,
  children,
  title,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
  title: string;
}) {
  const [hovered, setHovered] = useState(false);
  return (
    <button
      type="button"
      title={title}
      aria-label={title}
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        width: 30,
        height: 30,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: active
          ? 'color-mix(in srgb, var(--accent) 12%, transparent)'
          : hovered
            ? 'color-mix(in srgb, white 4%, transparent)'
            : 'transparent',
        border: `1px solid ${active ? 'var(--accent)' : 'var(--border)'}`,
        borderRadius: 6,
        cursor: 'pointer',
        color: active ? 'var(--accent)' : 'var(--text-muted)',
        padding: 0,
        transition: 'all 0.15s',
      }}
    >
      {children}
    </button>
  );
}

// ─── CVE Card ────────────────────────────────────────────────────────────────

function CVECard({ cve }: { cve: CVEListItem }) {
  const navigate = useNavigate();
  const score = cve.cvss_v3_score;
  const scoreNorm = score !== null ? Math.min(score / 10, 1) : 0;
  const circumference = 2 * Math.PI * 16;
  const strokeDashoffset = circumference * (1 - scoreNorm);

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 10,
        padding: 16,
        display: 'flex',
        flexDirection: 'column',
        gap: 12,
        cursor: 'pointer',
        transition: 'border-color 0.15s, box-shadow 0.15s',
      }}
      onClick={() => navigate(`/cves/${cve.id}`)}
      onMouseEnter={(e) => {
        e.currentTarget.style.borderColor = 'var(--border-hover)';
        e.currentTarget.style.boxShadow = 'var(--shadow-md, 0 4px 12px rgba(0,0,0,.15))';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.borderColor = 'var(--border)';
        e.currentTarget.style.boxShadow = 'none';
      }}
    >
      {/* Top: CVE ID + CVSS ring */}
      <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
        <div>
          <Link
            to={`/cves/${cve.id}`}
            onClick={(e) => e.stopPropagation()}
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 13,
              fontWeight: 700,
              color: 'var(--accent)',
              textDecoration: 'none',
              letterSpacing: '0.01em',
            }}
          >
            {cve.cve_id}
          </Link>
          <div style={{ marginTop: 4 }}>
            <span
              style={{
                display: 'inline-block',
                padding: '2px 8px',
                borderRadius: 10,
                fontSize: 10,
                fontWeight: 600,
                fontFamily: 'var(--font-mono)',
                background: `color-mix(in srgb, ${severityColor(cve.severity)} 12%, transparent)`,
                color: severityColor(cve.severity),
                border: `1px solid color-mix(in srgb, ${severityColor(cve.severity)} 25%, transparent)`,
              }}
            >
              {displaySeverity(cve.severity)}
            </span>
          </div>
        </div>

        {/* CVSS ring gauge */}
        <div style={{ position: 'relative', width: 44, height: 44, flexShrink: 0 }}>
          <svg width="44" height="44" viewBox="0 0 44 44" style={{ transform: 'rotate(-90deg)' }}>
            <circle cx="22" cy="22" r="16" fill="none" stroke="var(--border)" strokeWidth="3" />
            <circle
              cx="22"
              cy="22"
              r="16"
              fill="none"
              stroke={cvssColor(score)}
              strokeWidth="3"
              strokeDasharray={circumference}
              strokeDashoffset={strokeDashoffset}
              strokeLinecap="round"
            />
          </svg>
          <div
            style={{
              position: 'absolute',
              inset: 0,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontFamily: 'var(--font-mono)',
              fontSize: 12,
              fontWeight: 700,
              color: cvssColor(score),
            }}
          >
            {score !== null ? score.toFixed(1) : '—'}
          </div>
        </div>
      </div>

      {/* Middle: metadata row */}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, alignItems: 'center' }}>
        {/* Attack Vector */}
        <span
          style={{
            fontSize: 10,
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-secondary)',
            padding: '2px 6px',
            borderRadius: 4,
            background: 'var(--bg-inset)',
          }}
        >
          {cve.attack_vector ?? '—'}
        </span>

        {/* Exploit badge */}
        {cve.exploit_available && (
          <span
            style={{
              fontSize: 10,
              fontWeight: 600,
              fontFamily: 'var(--font-mono)',
              color: 'var(--signal-critical)',
              padding: '2px 6px',
              borderRadius: 4,
              background: 'color-mix(in srgb, var(--signal-critical) 10%, transparent)',
              border: '1px solid color-mix(in srgb, var(--signal-critical) 25%, transparent)',
            }}
          >
            Exploit
          </span>
        )}

        {/* KEV badge */}
        {cve.cisa_kev_due_date && (
          <span
            style={{
              fontSize: 10,
              fontWeight: 600,
              fontFamily: 'var(--font-mono)',
              color: 'var(--signal-warning)',
              padding: '2px 6px',
              borderRadius: 4,
              background: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
              border: '1px solid color-mix(in srgb, var(--signal-warning) 25%, transparent)',
            }}
          >
            KEV
          </span>
        )}
      </div>

      {/* Bottom: endpoints + published + actions */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginTop: 'auto',
          paddingTop: 8,
          borderTop: '1px solid var(--border)',
        }}
      >
        <div style={{ display: 'flex', gap: 12 }}>
          <div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 14,
                fontWeight: 700,
                color:
                  cve.affected_endpoint_count > 0 ? 'var(--text-primary)' : 'var(--text-faint)',
              }}
            >
              {cve.affected_endpoint_count}
            </div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 9,
                color: 'var(--text-muted)',
                textTransform: 'uppercase',
                letterSpacing: '0.04em',
              }}
            >
              Endpoints
            </div>
          </div>
          <div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                color: 'var(--text-muted)',
              }}
            >
              {relativeTime(cve.published_at)}
            </div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 9,
                color: 'var(--text-muted)',
                textTransform: 'uppercase',
                letterSpacing: '0.04em',
              }}
            >
              Published
            </div>
          </div>
        </div>

        <div style={{ display: 'flex', gap: 6 }} onClick={(e) => e.stopPropagation()}>
          <button
            type="button"
            onClick={() => navigate(`/cves/${cve.id}`)}
            style={{
              padding: '5px 10px',
              background: 'var(--accent)',
              color: 'var(--btn-accent-text, #000)',
              border: 'none',
              borderRadius: 5,
              fontSize: 11,
              fontWeight: 600,
              cursor: 'pointer',
              transition: 'opacity 0.15s',
            }}
            onMouseEnter={(e) => (e.currentTarget.style.opacity = '0.85')}
            onMouseLeave={(e) => (e.currentTarget.style.opacity = '1')}
          >
            Detail
          </button>
          <button
            type="button"
            onClick={() => navigate('/deployments/new')}
            style={{
              padding: '5px 10px',
              background: 'transparent',
              color: 'var(--text-secondary)',
              border: '1px solid var(--border)',
              borderRadius: 5,
              fontSize: 11,
              fontWeight: 500,
              cursor: 'pointer',
              transition: 'all 0.15s',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--border-hover)';
              e.currentTarget.style.color = 'var(--text-primary)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--border)';
              e.currentTarget.style.color = 'var(--text-secondary)';
            }}
          >
            Deploy
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export const CVEsPage = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const viewMode = searchParams.get('view') === 'card' ? 'card' : 'list';
  const [search, setSearch] = useState(searchParams.get('search') ?? '');
  const [severityFilter, setSeverityFilter] = useState('');
  const [kevActive, setKevActive] = useState(false);
  const [exploitActive, setExploitActive] = useState(false);
  const [attackVectorFilter, setAttackVectorFilter] = useState('');
  const [dateRangeFilter, setDateRangeFilter] = useState('');
  const [remediationFilter, setRemediationFilter] = useState('');
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [cursors, setCursors] = useState<string[]>([]);
  const [sortCol, setSortCol] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const [actionsOpen, setActionsOpen] = useState(false);
  const [wizardOpen, setWizardOpen] = useState(false);
  const [wizardInit, setWizardInit] = useState<DeploymentWizardInitialState | undefined>();
  const navigate = useNavigate();

  const openDeploy = useCallback((patchIds: string[]) => {
    setWizardInit(patchIds.length > 0 ? { sourceType: 'catalog', patchIds } : undefined);
    setWizardOpen(true);
  }, []);

  const hasActiveFilters =
    search !== '' ||
    severityFilter !== '' ||
    kevActive ||
    exploitActive ||
    attackVectorFilter !== '' ||
    dateRangeFilter !== '' ||
    remediationFilter !== '';

  const clearAllFilters = useCallback(() => {
    setSearch('');
    setSeverityFilter('');
    setKevActive(false);
    setExploitActive(false);
    setAttackVectorFilter('');
    setDateRangeFilter('');
    setRemediationFilter('');
    setCursors([]);
  }, []);

  const setViewMode = useCallback(
    (mode: 'list' | 'card') => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        if (mode === 'card') next.set('view', 'card');
        else next.delete('view');
        return next;
      });
    },
    [setSearchParams],
  );

  const currentCursor = cursors[cursors.length - 1];

  // Debounced search
  const [debouncedSearch, setDebouncedSearch] = useState(search);
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setCursors([]);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  // Wrap filter setters to reset cursor synchronously (avoids stale cursor in the same render tick)
  const setSeverityFilterAndReset = useCallback((v: string) => {
    setSeverityFilter(v);
    setCursors([]);
  }, []);
  const setKevActiveAndReset = useCallback((v: boolean | ((prev: boolean) => boolean)) => {
    setKevActive(v);
    setCursors([]);
  }, []);
  const setExploitActiveAndReset = useCallback((v: boolean | ((prev: boolean) => boolean)) => {
    setExploitActive(v);
    setCursors([]);
  }, []);
  const setAttackVectorFilterAndReset = useCallback((v: string) => {
    setAttackVectorFilter(v);
    setCursors([]);
  }, []);

  const { data: summary, isLoading: summaryLoading } = useCVESummary();

  const publishedAfter = (() => {
    if (!dateRangeFilter) return undefined;
    const days = parseInt(dateRangeFilter, 10);
    if (isNaN(days)) return undefined;
    const d = new Date();
    d.setDate(d.getDate() - days);
    return d.toISOString();
  })();

  const hasPatch = (() => {
    if (remediationFilter === 'yes') return 'true';
    if (remediationFilter === 'none' || remediationFilter === 'workaround') return 'false';
    return undefined;
  })();

  const { data, isLoading, isError, refetch } = useCVEs({
    cursor: currentCursor,
    limit: 25,
    search: debouncedSearch || undefined,
    severity: severityFilter || undefined,
    cisa_kev: kevActive ? 'true' : undefined,
    exploit_available: exploitActive ? 'true' : undefined,
    attack_vector: attackVectorFilter || undefined,
    published_after: publishedAfter,
    has_patch: hasPatch,
  });

  const rows = data?.data ?? [];
  const totalCount = data?.total_count ?? 0;
  const pageSize = rows.length;
  const pageStart = pageSize > 0 ? (cursors.length > 0 ? cursors.length * 25 + 1 : 1) : 0;
  const pageEnd = pageSize > 0 ? Math.min(pageStart + pageSize - 1, totalCount) : 0;
  const currentPage = cursors.length + 1;
  const totalPages = Math.max(1, Math.ceil(totalCount / 25));
  const hasNext = !!data?.next_cursor;
  const navigableTo = hasNext ? currentPage + 1 : currentPage;

  const pageNums: number[] = [];
  for (let p = 1; p <= Math.min(navigableTo, totalPages); p++) {
    pageNums.push(p);
  }
  const showLastPage = totalPages > navigableTo;

  const toggleExpand = (id: string) => setExpanded((prev) => ({ ...prev, [id]: !prev[id] }));

  const toggleSort = useCallback(
    (col: string) => {
      if (sortCol === col) {
        if (sortDir === 'asc') setSortDir('desc');
        else if (sortDir === 'desc') {
          setSortCol(null);
          setSortDir('asc');
        }
      } else {
        setSortCol(col);
        setSortDir('asc');
      }
    },
    [sortCol, sortDir],
  );

  const sortedData = useMemo(() => {
    if (!sortCol) return rows;
    const sorted = [...rows].sort((a, b) => {
      switch (sortCol) {
        case 'cve_id':
          return a.cve_id.localeCompare(b.cve_id);
        case 'cvss':
          return (a.cvss_v3_score ?? -1) - (b.cvss_v3_score ?? -1);
        case 'severity': {
          const order: Record<string, number> = { critical: 4, high: 3, medium: 2, low: 1 };
          return (order[a.severity.toLowerCase()] ?? 0) - (order[b.severity.toLowerCase()] ?? 0);
        }
        case 'vector':
          return (a.attack_vector ?? '').localeCompare(b.attack_vector ?? '');
        case 'exploit':
          return (a.exploit_available ? 1 : 0) - (b.exploit_available ? 1 : 0);
        case 'kev':
          return (a.cisa_kev_due_date ? 1 : 0) - (b.cisa_kev_due_date ? 1 : 0);
        case 'endpoints':
          return a.affected_endpoint_count - b.affected_endpoint_count;
        case 'published':
          return (a.published_at ?? '').localeCompare(b.published_at ?? '');
        default:
          return 0;
      }
    });
    return sortDir === 'desc' ? sorted.reverse() : sorted;
  }, [rows, sortCol, sortDir]);

  const COL_COUNT = 10;

  const thStyle: React.CSSProperties = {
    ...TH,
    background: 'var(--bg-inset)',
    borderBottom: '1px solid var(--border)',
  };

  return (
    <div
      style={{
        padding: '24px',
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        minHeight: '100%',
        background: 'var(--bg-page)',
      }}
    >
      {/* Stat Cards */}
      {summaryLoading ? (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 8 }}>
          {Array.from({ length: 4 }).map((_, i) => (
            <div
              key={i}
              style={{
                height: 80,
                borderRadius: 8,
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                animation: 'pulse 1.5s ease-in-out infinite',
              }}
            />
          ))}
        </div>
      ) : summary ? (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(6, 1fr)', gap: 8 }}>
          <InlineStatCard
            value={hasActiveFilters ? totalCount : (summary.total ?? 0)}
            label={hasActiveFilters ? 'Filtered' : 'Total'}
            valueColor="var(--text-primary)"
          />
          <InlineStatCard
            value={summary.by_severity.critical ?? 0}
            label="Critical"
            valueColor="var(--signal-critical)"
            active={severityFilter === 'critical'}
            onClick={() =>
              setSeverityFilterAndReset(severityFilter === 'critical' ? '' : 'critical')
            }
          />
          <InlineStatCard
            value={summary.by_severity.high ?? 0}
            label="High"
            valueColor="var(--signal-warning)"
            active={severityFilter === 'high'}
            onClick={() => setSeverityFilterAndReset(severityFilter === 'high' ? '' : 'high')}
          />
          <InlineStatCard
            value={summary.by_severity.medium ?? 0}
            label="Medium"
            valueColor="var(--text-secondary)"
            active={severityFilter === 'medium'}
            onClick={() => setSeverityFilterAndReset(severityFilter === 'medium' ? '' : 'medium')}
          />
          <InlineStatCard
            value={summary.by_severity.low ?? 0}
            label="Low"
            valueColor="var(--signal-healthy)"
            active={severityFilter === 'low'}
            onClick={() => setSeverityFilterAndReset(severityFilter === 'low' ? '' : 'low')}
          />
          <InlineStatCard
            value={summary.kev_count ?? 0}
            label="KEV Listed"
            valueColor="var(--signal-warning)"
            active={kevActive}
            onClick={() => setKevActiveAndReset((v) => !v)}
          />
        </div>
      ) : null}

      {/* Filter Bar + Actions */}
      <div style={{ display: 'flex', alignItems: 'stretch', gap: 8 }}>
        {/* Filter Bar */}
        <div
          style={{
            flex: 1,
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: '10px 14px',
            boxShadow: 'var(--shadow-sm)',
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            flexWrap: 'wrap',
          }}
        >
          {/* Search */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '5px 10px',
              border: '1px solid var(--border)',
              borderRadius: 6,
              background: 'var(--bg-inset)',
              minWidth: 220,
              transition: 'border-color 0.15s',
            }}
            onFocusCapture={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlurCapture={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          >
            <svg
              width="12"
              height="12"
              viewBox="0 0 24 24"
              fill="none"
              stroke="var(--text-muted)"
              strokeWidth="2.5"
              style={{ flexShrink: 0 }}
            >
              <circle cx="11" cy="11" r="8" />
              <path d="M21 21l-4.35-4.35" />
            </svg>
            <input
              type="text"
              id="cve-search"
              name="search"
              aria-label="Search CVEs by ID"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search CVE-YYYY-NNNNN..."
              style={{
                background: 'transparent',
                border: 'none',
                outline: 'none',
                fontSize: 12,
                color: 'var(--text-primary)',
                width: '100%',
              }}
            />
          </div>

          {/* Divider */}
          <div
            style={{
              width: 1,
              height: 20,
              background: 'var(--border)',
              margin: '0 4px',
              flexShrink: 0,
            }}
          />

          {/* Toggle Filters */}
          <FilterToggle
            label="Exploit Available"
            count={summary?.exploit_count ?? undefined}
            active={exploitActive}
            onClick={() => setExploitActive((v) => !v)}
            activeColor="var(--signal-critical)"
          />
          <FilterToggle
            label="KEV Only"
            count={summary?.kev_count ?? undefined}
            active={kevActive}
            onClick={() => setKevActive((v) => !v)}
            activeColor="var(--signal-warning)"
          />

          {/* Divider */}
          <div
            style={{
              width: 1,
              height: 20,
              background: 'var(--border)',
              margin: '0 4px',
              flexShrink: 0,
            }}
          />

          {/* Attack Vector Select */}
          <select
            id="attack-vector-filter"
            name="attack_vector"
            aria-label="Filter by attack vector"
            value={attackVectorFilter}
            onChange={(e) => setAttackVectorFilter(e.target.value)}
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 10px',
              fontSize: 11.5,
              color: attackVectorFilter ? 'var(--text-primary)' : 'var(--text-secondary)',
              outline: 'none',
              cursor: 'pointer',
            }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          >
            <option value="">Attack Vector</option>
            <option value="Network">Network</option>
            <option value="Adjacent">Adjacent</option>
            <option value="Local">Local</option>
            <option value="Physical">Physical</option>
          </select>

          {/* Date Range Select */}
          <select
            id="date-range-filter"
            name="date_range"
            aria-label="Filter by date range"
            value={dateRangeFilter}
            onChange={(e) => setDateRangeFilter(e.target.value)}
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 10px',
              fontSize: 11.5,
              color: dateRangeFilter ? 'var(--text-primary)' : 'var(--text-secondary)',
              outline: 'none',
              cursor: 'pointer',
            }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          >
            <option value="">Date Range</option>
            <option value="7">Last 7 days</option>
            <option value="30">Last 30 days</option>
            <option value="90">Last 90 days</option>
          </select>

          {/* Remediation Select */}
          <select
            id="patch-available-filter"
            name="has_patch"
            aria-label="Filter by patch availability"
            value={remediationFilter}
            onChange={(e) => setRemediationFilter(e.target.value)}
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 10px',
              fontSize: 11.5,
              color: remediationFilter ? 'var(--text-primary)' : 'var(--text-secondary)',
              outline: 'none',
              cursor: 'pointer',
            }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          >
            <option value="">Patch Available</option>
            <option value="yes">Yes</option>
            <option value="workaround">Workaround Only</option>
            <option value="none">No</option>
          </select>

          {/* Spacer */}
          <div style={{ flex: 1 }} />

          {/* View Toggle */}
          <div style={{ display: 'flex', gap: 4 }}>
            <ViewToggleButton
              active={viewMode === 'list'}
              onClick={() => setViewMode('list')}
              title="Switch to table view"
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 14 14"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
              >
                <path d="M2 3.5h10M2 7h10M2 10.5h10" />
              </svg>
            </ViewToggleButton>
            <ViewToggleButton
              active={viewMode === 'card'}
              onClick={() => setViewMode('card')}
              title="Switch to grid view"
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 14 14"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
              >
                <rect x="1.5" y="1.5" width="4.5" height="4.5" rx="1" />
                <rect x="8" y="1.5" width="4.5" height="4.5" rx="1" />
                <rect x="1.5" y="8" width="4.5" height="4.5" rx="1" />
                <rect x="8" y="8" width="4.5" height="4.5" rx="1" />
              </svg>
            </ViewToggleButton>
          </div>
        </div>

        {/* Actions Card */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: '10px 12px',
            boxShadow: 'var(--shadow-sm)',
          }}
        >
          <div style={{ position: 'relative' }}>
            <button
              type="button"
              onClick={() => setActionsOpen((o) => !o)}
              style={{
                padding: '5px 12px',
                fontSize: 11.5,
                fontWeight: 600,
                borderRadius: 6,
                border: 'none',
                background: 'var(--accent)',
                color: 'var(--btn-accent-text, #000)',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                transition: 'opacity 0.15s',
              }}
              onMouseEnter={(e) => (e.currentTarget.style.opacity = '0.85')}
              onMouseLeave={(e) => (e.currentTarget.style.opacity = '1')}
            >
              Actions
              <svg
                width="10"
                height="10"
                viewBox="0 0 10 10"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                style={{ marginLeft: 2 }}
              >
                <path d="M2 3.5l3 3 3-3" />
              </svg>
            </button>
            {actionsOpen && (
              <>
                <div
                  style={{ position: 'fixed', inset: 0, zIndex: 49 }}
                  onClick={() => setActionsOpen(false)}
                />
                <div
                  style={{
                    position: 'absolute',
                    top: 'calc(100% + 4px)',
                    right: 0,
                    zIndex: 50,
                    background: 'var(--bg-card)',
                    border: '1px solid var(--border)',
                    borderRadius: 8,
                    boxShadow: '0 4px 16px rgba(0,0,0,0.18)',
                    minWidth: 180,
                    overflow: 'hidden',
                  }}
                >
                  <div style={{ padding: '4px 0' }}>
                    <button
                      type="button"
                      onClick={() => {
                        setActionsOpen(false);
                        const rows = data?.data ?? [];
                        if (!rows.length) return;
                        const headers = [
                          'CVE ID',
                          'CVSS',
                          'Severity',
                          'Attack Vector',
                          'Exploit',
                          'KEV Due Date',
                          'Affected Endpoints',
                          'Published',
                          'Patches Available',
                        ];
                        const csvRows = rows.map((c) =>
                          [
                            c.cve_id,
                            c.cvss_v3_score ?? '',
                            c.severity,
                            c.attack_vector ?? '',
                            c.exploit_available ? 'Yes' : 'No',
                            c.cisa_kev_due_date ?? '',
                            c.affected_endpoint_count,
                            c.published_at ? new Date(c.published_at).toLocaleDateString() : '',
                            c.patch_available ? 'Yes' : 'No',
                          ].join(','),
                        );
                        const csv = [headers.join(','), ...csvRows].join('\n');
                        const blob = new Blob([csv], { type: 'text/csv' });
                        const url = URL.createObjectURL(blob);
                        const a = document.createElement('a');
                        a.href = url;
                        a.download = 'cves-export.csv';
                        a.click();
                        URL.revokeObjectURL(url);
                      }}
                      style={{
                        width: '100%',
                        padding: '8px 14px',
                        fontSize: 12,
                        textAlign: 'left',
                        background: 'transparent',
                        border: 'none',
                        cursor: 'pointer',
                        color: 'var(--text-primary)',
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                      }}
                      onMouseEnter={(e) =>
                        (e.currentTarget.style.background = 'var(--bg-card-hover)')
                      }
                      onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                    >
                      <svg
                        width="12"
                        height="12"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                      >
                        <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4" />
                        <polyline points="7 10 12 15 17 10" />
                        <line x1="12" y1="15" x2="12" y2="3" />
                      </svg>
                      Export CSV
                    </button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Content Area */}
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          boxShadow: 'var(--shadow-sm)',
          overflow: 'hidden',
        }}
      >
        {isError ? (
          <div style={{ padding: 24 }}>
            <div
              style={{
                padding: '12px 16px',
                borderRadius: 6,
                border: '1px solid color-mix(in srgb, var(--signal-critical) 30%, transparent)',
                background: 'color-mix(in srgb, var(--signal-critical) 6%, transparent)',
                color: 'var(--signal-critical)',
                fontSize: 13,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
              }}
            >
              <span>Failed to load CVEs.</span>
              <button
                type="button"
                onClick={() => refetch()}
                style={{
                  fontSize: 12,
                  color: 'var(--signal-critical)',
                  background: 'transparent',
                  border: '1px solid color-mix(in srgb, var(--signal-critical) 40%, transparent)',
                  borderRadius: 4,
                  padding: '3px 10px',
                  cursor: 'pointer',
                }}
              >
                Retry
              </button>
            </div>
          </div>
        ) : viewMode === 'card' ? (
          /* ─── Card/Grid View ─── */
          <>
            {isLoading ? (
              <SkeletonCards count={8} />
            ) : rows.length === 0 ? (
              <EmptyState
                icon={debouncedSearch ? Search : ShieldX}
                title={debouncedSearch ? 'No CVEs match your search' : 'No CVEs found'}
                description={
                  debouncedSearch
                    ? 'Try a different CVE ID or adjust your filters.'
                    : 'No CVEs match your current filters. Try adjusting your search or date range.'
                }
                action={
                  hasActiveFilters
                    ? { label: 'Clear Filters', onClick: clearAllFilters }
                    : undefined
                }
              />
            ) : (
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
                  gap: 12,
                  padding: 16,
                }}
              >
                {sortedData.map((cve) => (
                  <CVECard key={cve.id} cve={cve} />
                ))}
              </div>
            )}

            {/* Pagination (card view) */}
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                padding: '10px 14px',
                borderTop: '1px solid var(--border)',
                background: 'var(--bg-inset)',
                gap: 6,
              }}
            >
              <span
                style={{
                  flex: 1,
                  fontSize: 11,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-muted)',
                }}
              >
                {pageSize > 0
                  ? hasActiveFilters && (summary?.total ?? 0) > totalCount
                    ? `Showing ${pageStart}–${pageEnd} of ${totalCount} CVEs (filtered from ${summary?.total ?? 0} total)`
                    : `Showing ${pageStart}–${pageEnd} of ${totalCount} CVEs`
                  : totalCount > 0
                    ? `${totalCount} CVEs`
                    : 'No CVEs found'}
              </span>

              <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
                <PaginationButton
                  disabled={cursors.length === 0}
                  onClick={() => setCursors((prev) => prev.slice(0, -1))}
                >
                  ← Prev
                </PaginationButton>

                {pageNums.map((p) => {
                  const isActive = p === currentPage;
                  const isDisabled = p > navigableTo;
                  return (
                    <PaginationButton
                      key={p}
                      active={isActive}
                      disabled={isDisabled}
                      onClick={() => {
                        if (p === 1) {
                          setCursors([]);
                        } else if (p === currentPage + 1 && hasNext) {
                          setCursors((prev) => [...prev, data!.next_cursor!]);
                        } else {
                          setCursors((prev) => prev.slice(0, p - 1));
                        }
                      }}
                    >
                      {p}
                    </PaginationButton>
                  );
                })}

                {totalPages > navigableTo + 1 && (
                  <span
                    style={{
                      fontSize: 11,
                      color: 'var(--text-faint)',
                      padding: '0 2px',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    ...
                  </span>
                )}

                {showLastPage && <PaginationButton disabled>{totalPages}</PaginationButton>}

                <PaginationButton
                  disabled={!data?.next_cursor}
                  onClick={() => {
                    if (data?.next_cursor) setCursors((prev) => [...prev, data.next_cursor!]);
                  }}
                >
                  Next →
                </PaginationButton>
              </div>
            </div>
          </>
        ) : (
          /* ─── List/Table View ─── */
          <>
            <div style={{ overflowX: 'auto' }}>
              <table
                style={{
                  width: '100%',
                  borderCollapse: 'collapse',
                  minWidth: 860,
                }}
              >
                <thead>
                  <tr>
                    <th style={{ ...thStyle, width: 36 }} />
                    <SortHeader
                      label="CVE ID"
                      colKey="cve_id"
                      sortCol={sortCol}
                      sortDir={sortDir}
                      onSort={toggleSort}
                    />
                    <SortHeader
                      label="CVSS"
                      colKey="cvss"
                      sortCol={sortCol}
                      sortDir={sortDir}
                      onSort={toggleSort}
                    />
                    <SortHeader
                      label="Severity"
                      colKey="severity"
                      sortCol={sortCol}
                      sortDir={sortDir}
                      onSort={toggleSort}
                    />
                    <SortHeader
                      label="Vector"
                      colKey="vector"
                      sortCol={sortCol}
                      sortDir={sortDir}
                      onSort={toggleSort}
                    />
                    <SortHeader
                      label="Exploit"
                      colKey="exploit"
                      sortCol={sortCol}
                      sortDir={sortDir}
                      onSort={toggleSort}
                    />
                    <SortHeader
                      label="KEV"
                      colKey="kev"
                      sortCol={sortCol}
                      sortDir={sortDir}
                      onSort={toggleSort}
                    />
                    <SortHeader
                      label="Endpoints"
                      colKey="endpoints"
                      sortCol={sortCol}
                      sortDir={sortDir}
                      onSort={toggleSort}
                    />
                    <SortHeader
                      label="Published"
                      colKey="published"
                      sortCol={sortCol}
                      sortDir={sortDir}
                      onSort={toggleSort}
                    />
                    <th style={{ ...thStyle, width: 40 }} />
                  </tr>
                </thead>
                <tbody>
                  {isLoading ? (
                    <SkeletonRows cols={COL_COUNT} rows={8} />
                  ) : rows.length === 0 ? (
                    <tr>
                      <td colSpan={COL_COUNT}>
                        <EmptyState
                          icon={debouncedSearch ? Search : ShieldX}
                          title={debouncedSearch ? 'No CVEs match your search' : 'No CVEs found'}
                          description={
                            debouncedSearch
                              ? 'Try a different CVE ID or adjust your filters.'
                              : 'No CVEs match your current filters. Try adjusting your search or date range.'
                          }
                          action={
                            hasActiveFilters
                              ? { label: 'Clear Filters', onClick: clearAllFilters }
                              : undefined
                          }
                        />
                      </td>
                    </tr>
                  ) : (
                    sortedData.map((cve) => {
                      const isExpanded = !!expanded[cve.id];
                      const score = cve.cvss_v3_score;
                      return (
                        <Fragment key={cve.id}>
                          <tr
                            onClick={() => navigate(`/cves/${cve.id}`)}
                            style={{
                              borderBottom: isExpanded ? 'none' : '1px solid var(--border)',
                              cursor: 'pointer',
                              transition: 'background 0.1s',
                            }}
                            onMouseEnter={(e) =>
                              (e.currentTarget.style.background = 'var(--bg-card-hover)')
                            }
                            onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                          >
                            {/* Expand toggle */}
                            <td
                              style={{ padding: '10px 12px', verticalAlign: 'middle', width: 36 }}
                              onClick={(e) => {
                                e.stopPropagation();
                                toggleExpand(cve.id);
                              }}
                            >
                              <button
                                type="button"
                                aria-label={`${isExpanded ? 'Collapse' : 'Expand'} details for ${cve.cve_id}`}
                                style={{
                                  width: 24,
                                  height: 24,
                                  display: 'flex',
                                  alignItems: 'center',
                                  justifyContent: 'center',
                                  background: 'transparent',
                                  border: 'none',
                                  borderRadius: 4,
                                  cursor: 'pointer',
                                  color: 'var(--text-muted)',
                                  padding: 0,
                                }}
                              >
                                <svg
                                  width="12"
                                  height="12"
                                  viewBox="0 0 12 12"
                                  fill="none"
                                  stroke="currentColor"
                                  strokeWidth="2"
                                  style={{
                                    transform: isExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                                    transition: 'transform 0.2s',
                                  }}
                                >
                                  <path d="M4.5 2.5L8.5 6L4.5 9.5" />
                                </svg>
                              </button>
                            </td>

                            {/* CVE ID */}
                            <td
                              style={{
                                padding: '10px 12px',
                                verticalAlign: 'middle',
                              }}
                              onClick={(e) => e.stopPropagation()}
                            >
                              <Link
                                to={`/cves/${cve.id}`}
                                onClick={(e) => e.stopPropagation()}
                                style={{
                                  fontFamily: 'var(--font-mono)',
                                  fontSize: 12,
                                  fontWeight: 600,
                                  color: 'var(--accent)',
                                  textDecoration: 'none',
                                  letterSpacing: '0.01em',
                                }}
                              >
                                {cve.cve_id}
                              </Link>
                            </td>

                            {/* CVSS Score */}
                            <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                              <span
                                style={{
                                  fontFamily: 'var(--font-mono)',
                                  fontSize: 13,
                                  fontWeight: 700,
                                  color: cvssColor(score),
                                  lineHeight: 1,
                                }}
                              >
                                {score !== null ? `${score.toFixed(1)}/10` : '—'}
                              </span>
                            </td>

                            {/* Severity — click to filter */}
                            <td
                              style={{ padding: '10px 12px', verticalAlign: 'middle' }}
                              onClick={(e) => {
                                e.stopPropagation();
                                setSeverityFilterAndReset(
                                  severityFilter === cve.severity.toLowerCase()
                                    ? ''
                                    : cve.severity.toLowerCase(),
                                );
                              }}
                            >
                              <span
                                style={{
                                  color: severityColor(cve.severity),
                                  fontSize: 11,
                                  fontWeight: 500,
                                  fontFamily: 'var(--font-mono)',
                                  cursor: 'pointer',
                                }}
                              >
                                {displaySeverity(cve.severity)}
                              </span>
                            </td>

                            {/* Attack Vector — click to filter */}
                            <td
                              style={{ padding: '10px 12px', verticalAlign: 'middle' }}
                              onClick={(e) => {
                                e.stopPropagation();
                                if (cve.attack_vector) {
                                  setAttackVectorFilterAndReset(
                                    attackVectorFilter === cve.attack_vector
                                      ? ''
                                      : cve.attack_vector,
                                  );
                                }
                              }}
                            >
                              <span
                                style={{
                                  fontSize: 11,
                                  color: 'var(--text-secondary)',
                                  cursor: cve.attack_vector ? 'pointer' : 'default',
                                }}
                              >
                                {cve.attack_vector ?? '—'}
                              </span>
                            </td>

                            {/* Exploit — click to filter (only on "Yes" cells) */}
                            <td
                              style={{ padding: '10px 12px', verticalAlign: 'middle' }}
                              onClick={(e) => {
                                e.stopPropagation();
                                if (cve.exploit_available) {
                                  setExploitActiveAndReset(!exploitActive);
                                }
                              }}
                            >
                              {cve.exploit_available ? (
                                <span
                                  style={{
                                    display: 'inline-flex',
                                    alignItems: 'center',
                                    gap: 5,
                                    fontFamily: 'var(--font-mono)',
                                    fontSize: 11,
                                    fontWeight: 700,
                                    color: 'var(--signal-critical)',
                                    cursor: 'pointer',
                                  }}
                                >
                                  <span
                                    style={{
                                      width: 6,
                                      height: 6,
                                      borderRadius: '50%',
                                      background: 'var(--signal-critical)',
                                      display: 'inline-block',
                                      flexShrink: 0,
                                    }}
                                  />
                                  Yes
                                </span>
                              ) : (
                                <span
                                  style={{
                                    fontFamily: 'var(--font-mono)',
                                    fontSize: 11,
                                    color: 'var(--text-faint)',
                                  }}
                                >
                                  —
                                </span>
                              )}
                            </td>

                            {/* KEV — click to filter (only on "Yes" cells) */}
                            <td
                              style={{ padding: '10px 12px', verticalAlign: 'middle' }}
                              onClick={(e) => {
                                e.stopPropagation();
                                if (cve.cisa_kev_due_date) {
                                  setKevActiveAndReset(!kevActive);
                                }
                              }}
                            >
                              {cve.cisa_kev_due_date ? (
                                <span
                                  style={{
                                    display: 'inline-flex',
                                    alignItems: 'center',
                                    gap: 5,
                                    fontFamily: 'var(--font-mono)',
                                    fontSize: 11,
                                    fontWeight: 700,
                                    color: 'var(--signal-warning)',
                                    cursor: 'pointer',
                                  }}
                                >
                                  <span
                                    style={{
                                      width: 6,
                                      height: 6,
                                      borderRadius: '50%',
                                      background: 'var(--signal-warning)',
                                      display: 'inline-block',
                                      flexShrink: 0,
                                    }}
                                  />
                                  Yes
                                </span>
                              ) : (
                                <span
                                  style={{
                                    fontFamily: 'var(--font-mono)',
                                    fontSize: 11,
                                    color: 'var(--text-faint)',
                                  }}
                                >
                                  —
                                </span>
                              )}
                            </td>

                            {/* Endpoints */}
                            <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                              <span
                                style={{
                                  fontFamily: 'var(--font-mono)',
                                  fontSize: 13,
                                  fontWeight: 600,
                                  color:
                                    cve.affected_endpoint_count > 0
                                      ? 'var(--text-primary)'
                                      : 'var(--text-faint)',
                                  cursor: cve.affected_endpoint_count > 0 ? 'pointer' : 'default',
                                }}
                              >
                                {cve.affected_endpoint_count}
                              </span>
                            </td>

                            {/* Published */}
                            <td style={{ padding: '10px 12px', verticalAlign: 'middle' }}>
                              <span
                                style={{
                                  fontFamily: 'var(--font-mono)',
                                  fontSize: 11,
                                  color: 'var(--text-muted)',
                                }}
                                title={
                                  cve.published_at
                                    ? new Date(cve.published_at).toLocaleString('en-US', {
                                        year: 'numeric',
                                        month: 'short',
                                        day: 'numeric',
                                        hour: '2-digit',
                                        minute: '2-digit',
                                        timeZoneName: 'short',
                                      })
                                    : undefined
                                }
                              >
                                {relativeTime(cve.published_at)}
                              </span>
                            </td>

                            {/* Actions */}
                            <td
                              style={{ padding: '10px 12px', verticalAlign: 'middle', width: 40 }}
                              onClick={(e) => e.stopPropagation()}
                            >
                              {!isExpanded && (
                                <KebabMenu
                                  ariaLabel={`Actions for ${cve.cve_id}`}
                                  items={[
                                    {
                                      label: 'View Details',
                                      icon: <Eye size={12} />,
                                      action: () => navigate(`/cves/${cve.id}`),
                                    },
                                    {
                                      label: 'Copy CVE ID',
                                      icon: <Copy size={12} />,
                                      action: () => navigator.clipboard.writeText(cve.cve_id),
                                    },
                                    {
                                      label: 'Open in NVD',
                                      icon: <ExternalLink size={12} />,
                                      action: () =>
                                        window.open(
                                          `https://nvd.nist.gov/vuln/detail/${cve.cve_id}`,
                                          '_blank',
                                          'noopener',
                                        ),
                                    },
                                    {
                                      label: 'Deploy Fix',
                                      icon: <Rocket size={12} />,
                                      action: () => openDeploy([]),
                                    },
                                  ]}
                                />
                              )}
                            </td>
                          </tr>
                          {isExpanded && <ExpandedCVERow cve={cve} onDeploy={openDeploy} />}
                        </Fragment>
                      );
                    })
                  )}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                padding: '10px 14px',
                borderTop: '1px solid var(--border)',
                background: 'var(--bg-inset)',
                gap: 6,
              }}
            >
              <span
                style={{
                  flex: 1,
                  fontSize: 11,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-muted)',
                }}
              >
                {pageSize > 0
                  ? hasActiveFilters && (summary?.total ?? 0) > totalCount
                    ? `Showing ${pageStart}–${pageEnd} of ${totalCount} CVEs (filtered from ${summary?.total ?? 0} total)`
                    : `Showing ${pageStart}–${pageEnd} of ${totalCount} CVEs`
                  : totalCount > 0
                    ? `${totalCount} CVEs`
                    : 'No CVEs found'}
              </span>

              <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
                <PaginationButton
                  disabled={cursors.length === 0}
                  onClick={() => setCursors((prev) => prev.slice(0, -1))}
                >
                  ← Prev
                </PaginationButton>

                {pageNums.map((p) => {
                  const isActive = p === currentPage;
                  const isDisabled = p > navigableTo;
                  return (
                    <PaginationButton
                      key={p}
                      active={isActive}
                      disabled={isDisabled}
                      onClick={() => {
                        if (p === 1) {
                          setCursors([]);
                        } else if (p === currentPage + 1 && hasNext) {
                          setCursors((prev) => [...prev, data!.next_cursor!]);
                        } else {
                          setCursors((prev) => prev.slice(0, p - 1));
                        }
                      }}
                    >
                      {p}
                    </PaginationButton>
                  );
                })}

                {totalPages > navigableTo + 1 && (
                  <span
                    style={{
                      fontSize: 11,
                      color: 'var(--text-faint)',
                      padding: '0 2px',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    ...
                  </span>
                )}

                {showLastPage && <PaginationButton disabled>{totalPages}</PaginationButton>}

                <PaginationButton
                  disabled={!data?.next_cursor}
                  onClick={() => {
                    if (data?.next_cursor) setCursors((prev) => [...prev, data.next_cursor!]);
                  }}
                >
                  Next →
                </PaginationButton>
              </div>
            </div>
          </>
        )}
      </div>

      <DeploymentWizard open={wizardOpen} onOpenChange={setWizardOpen} initialState={wizardInit} />
    </div>
  );
};
