import { Fragment, useState, useCallback, useMemo, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router';
import { useCan } from '../../app/auth/AuthContext';
import {
  SkeletonCard,
  ErrorState,
  EmptyState,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  Button,
} from '@patchiq/ui';
import {
  useEndpoints,
  useTriggerScan,
  useCreateRegistration,
  useDecommissionEndpoint,
} from '../../api/hooks/useEndpoints';
import { buildEndpointCsv, downloadCsvString } from './export-csv';
import { computeRiskScore as computeRisk } from '../../lib/risk';
import { AssignTagsDialog } from './AssignTagsDialog';
import { ExportEndpointsDialog } from './ExportEndpointsDialog';
import { DeploymentWizard } from '../../components/DeploymentWizard';
import type { Endpoint } from '../../api/hooks/useEndpoints';
import { useAgentBinaries, type AgentBinaryInfo } from '../../api/hooks/useAgentBinaries';
import { deriveStatus } from './deriveStatus';

type StatusFilter = 'all' | 'online' | 'offline' | 'pending' | 'stale';

function osEmoji(f: string): string {
  const l = f.toLowerCase();
  if (/linux|ubuntu|debian|centos|rocky|rhel|fedora/.test(l)) return '\u{1F427}';
  if (/windows|win/.test(l)) return '\u{1FA9F}';
  if (/mac|darwin|apple/.test(l)) return '\u{1F34E}';
  return '\u{1F4BB}';
}
function dotColor(s: string) {
  if (s === 'online') return 'var(--signal-healthy)';
  if (s === 'offline') return 'var(--signal-critical)';
  if (s === 'stale') return 'var(--signal-warning)';
  if (s === 'pending') return 'var(--accent)';
  return 'var(--text-muted)';
}
function sLabel(s: string) {
  return s === 'pending' ? 'Patching' : s.charAt(0).toUpperCase() + s.slice(1);
}
function sColor(s: string) {
  if (s === 'online') return 'var(--signal-healthy)';
  if (s === 'offline') return 'var(--signal-critical)';
  if (s === 'stale') return 'var(--signal-warning)';
  return 'var(--text-secondary)';
}
function riskScore(ep: Endpoint): number {
  return computeRisk({
    criticalCves: ep.critical_cve_count ?? ep.critical_patch_count,
    highCves: ep.high_cve_count ?? ep.high_patch_count,
    mediumCves: ep.medium_cve_count ?? ep.medium_patch_count,
  });
}
function riskClr(s: number) {
  return s >= 7
    ? 'var(--signal-critical)'
    : s >= 3
      ? 'var(--signal-warning)'
      : 'var(--signal-healthy)';
}
function pendClr(n: number) {
  return n >= 7
    ? 'var(--signal-critical)'
    : n >= 3
      ? 'var(--signal-warning)'
      : 'var(--text-secondary)';
}
function relTime(d: string | null | undefined): string {
  if (!d) return 'Never';
  const ms = Date.now() - new Date(d).getTime();
  if (ms < 60000) return 'Just now';
  const m = Math.floor(ms / 60000);
  if (m < 60) return `${m} min ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h} hr${h > 1 ? 's' : ''} ago`;
  const dy = Math.floor(h / 24);
  return `${dy} day${dy > 1 ? 's' : ''} ago`;
}
function fmtN(n?: number) {
  return n == null ? '0' : n.toLocaleString();
}

interface StatCardProps {
  label: string;
  value: number | undefined;
  valueColor?: string;
  active?: boolean;
  onClick: () => void;
}

function StatCard({ label, value, valueColor, active, onClick }: StatCardProps) {
  const [hovered, setHovered] = useState(false);
  return (
    <button
      type="button"
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        flex: 1,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        padding: '12px 14px',
        background: active ? 'color-mix(in srgb, white 3%, transparent)' : 'var(--bg-card)',
        border: `1px solid ${active ? (valueColor ?? 'var(--accent)') : hovered ? 'var(--border-hover)' : 'var(--border)'}`,
        borderRadius: 8,
        cursor: 'pointer',
        transition: 'all 0.15s',
        outline: 'none',
        textAlign: 'left',
      }}
    >
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 22,
          fontWeight: 700,
          lineHeight: 1,
          color: valueColor ?? 'var(--text-emphasis)',
          letterSpacing: '-0.02em',
        }}
      >
        {value ?? '—'}
      </span>
      <span
        style={{
          fontSize: 10,
          fontFamily: 'var(--font-mono)',
          fontWeight: 500,
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
          color: active ? (valueColor ?? 'var(--accent)') : 'var(--text-muted)',
          marginTop: 4,
        }}
      >
        {label}
      </span>
    </button>
  );
}

function CB({ on, onClick }: { on: boolean; onClick: (e: React.MouseEvent) => void }) {
  return (
    <div
      onClick={onClick}
      style={{
        width: 14,
        height: 14,
        borderRadius: 3,
        border: `1.5px solid ${on ? 'var(--accent)' : 'var(--border-hover)'}`,
        background: on ? 'var(--accent)' : 'transparent',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        cursor: 'pointer',
        flexShrink: 0,
      }}
    >
      {on && (
        <svg width="8" height="6" viewBox="0 0 8 6" fill="none">
          <path
            d="M1 3L3 5L7 1"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      )}
    </div>
  );
}

interface KebabMenuProps {
  onView: () => void;
  onScan: () => void;
  onDeploy: () => void;
  onAssignTags: () => void;
  onDelete: () => void;
}

function KebabMenu({ onView, onScan, onDeploy, onAssignTags, onDelete }: KebabMenuProps) {
  const [open, setOpen] = useState(false);
  const can = useCan();
  const menuItems = [
    { label: 'View Details', action: onView },
    {
      label: 'Scan',
      action: onScan,
      disabled: !can('endpoints', 'scan'),
      disabledTitle: "You don't have permission",
    },
    {
      label: 'Deploy Patches',
      action: onDeploy,
      disabled: !can('deployments', 'create'),
      disabledTitle: "You don't have permission",
    },
    {
      label: 'Assign Tags',
      action: onAssignTags,
      disabled: !can('endpoints', 'create'),
      disabledTitle: "You don't have permission",
    },
    {
      label: 'Delete',
      action: onDelete,
      destructive: true,
      disabled: !can('endpoints', 'delete'),
      disabledTitle: "You don't have permission",
    },
  ];
  return (
    <div style={{ position: 'relative' }}>
      <button
        type="button"
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
            minWidth: 150,
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            boxShadow: 'var(--shadow-lg, 0 8px 24px rgba(0,0,0,.25))',
            zIndex: 50,
            padding: '4px 0',
            overflow: 'hidden',
          }}
        >
          {menuItems.map((item) => (
            <Fragment key={item.label}>
              {item.destructive && (
                <div style={{ height: 1, background: 'var(--border)', margin: '4px 0' }} />
              )}
              <button
                type="button"
                disabled={item.disabled}
                title={item.disabled ? item.disabledTitle : undefined}
                onMouseDown={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  if (item.disabled) return;
                  item.action();
                  setOpen(false);
                }}
                style={{
                  display: 'block',
                  width: '100%',
                  padding: '7px 14px',
                  fontSize: 12,
                  fontFamily: 'var(--font-sans)',
                  color: item.destructive ? 'var(--signal-critical)' : 'var(--text-secondary)',
                  background: 'transparent',
                  border: 'none',
                  cursor: item.disabled ? 'not-allowed' : 'pointer',
                  textAlign: 'left',
                  opacity: item.disabled ? 0.5 : 1,
                }}
                onMouseEnter={(e) => {
                  if (item.disabled) return;
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
                {item.label}
              </button>
            </Fragment>
          ))}
        </div>
      )}
    </div>
  );
}

const TH: React.CSSProperties = {
  padding: '9px 12px',
  textAlign: 'left',
  fontFamily: 'var(--font-mono)',
  fontSize: 11,
  fontWeight: 600,
  letterSpacing: '0.05em',
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  whiteSpace: 'nowrap',
};
const TD: React.CSSProperties = {
  padding: '12px 12px',
  borderBottom: '1px solid var(--border)',
  verticalAlign: 'middle',
};
const secLbl: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 10,
  fontWeight: 600,
  letterSpacing: '0.06em',
  textTransform: 'uppercase',
  color: 'var(--text-muted)',
  marginBottom: 10,
};

function ExpRow({
  ep,
  nav,
  onDeploy,
}: {
  ep: Endpoint;
  nav: (id: string) => void;
  onDeploy: () => void;
}) {
  const scan = useTriggerScan();
  const mu = ep.memory_used_mb ?? 0,
    mt = ep.memory_total_mb ?? 1;
  const du = ep.disk_used_gb ?? 0,
    dt = ep.disk_total_gb ?? 1;
  const cpu = ep.cpu_usage_percent ?? 0;
  const mem = mt > 0 ? Math.round((mu / mt) * 100) : 0;
  const disk = dt > 0 ? Math.round((du / dt) * 100) : 0;
  const bc = (p: number) => (p >= 70 ? 'var(--signal-warning)' : 'var(--signal-healthy)');
  const CARD: React.CSSProperties = {
    background: 'var(--bg-inset)',
    border: '1px solid var(--border)',
    borderRadius: 6,
    padding: '12px 14px',
  };
  const BTN: React.CSSProperties = {
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
    letterSpacing: '0.01em',
    width: '100%',
    textDecoration: 'none',
    fontFamily: 'var(--font-sans)',
  };
  return (
    <tr>
      <style>{`@keyframes expandRow{from{max-height:0;opacity:0}to{max-height:300px;opacity:1}}`}</style>
      <td colSpan={9} style={{ padding: 0, borderBottom: '1px solid var(--border)' }}>
        <div
          style={{
            padding: '8px 10px',
            background: 'var(--bg-page)',
            borderTop: '1px solid var(--border)',
            display: 'flex',
            gap: 8,
            alignItems: 'stretch',
            animation: 'expandRow 0.2s ease-out',
            overflow: 'hidden',
          }}
        >
          {/* System Health card */}
          <div style={{ ...CARD, flex: '0 0 500px' }}>
            <div style={secLbl}>System Health</div>
            {[
              { l: 'CPU', p: cpu, hasData: true },
              { l: 'Memory', p: mem, hasData: mt > 0 },
              { l: 'Disk', p: disk, hasData: dt > 0 },
            ].map((m) => (
              <div key={m.l} style={{ marginBottom: 8 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                  <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{m.l}</span>
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      color: 'var(--text-primary)',
                    }}
                  >
                    {m.hasData ? `${m.p}%` : 'N/A'}
                  </span>
                </div>
                {m.hasData && (
                  <div
                    style={{
                      height: 3,
                      background: 'var(--progress-track)',
                      borderRadius: 2,
                      overflow: 'hidden',
                    }}
                  >
                    <div
                      style={{
                        height: '100%',
                        borderRadius: 2,
                        width: `${m.p}%`,
                        background: bc(m.p),
                      }}
                    />
                  </div>
                )}
              </div>
            ))}
          </div>
          {/* Pending Patches card */}
          <div style={{ ...CARD, flex: '0 0 480px', marginLeft: 24 }}>
            <div style={secLbl}>Pending Patches</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 5 }}>
              {[
                { n: ep.critical_patch_count ?? 0, l: 'critical', c: 'var(--signal-critical)' },
                { n: ep.high_patch_count ?? 0, l: 'high', c: 'var(--signal-warning)' },
                { n: ep.medium_patch_count ?? 0, l: 'medium', c: 'var(--text-secondary)' },
              ].map(({ n, l, c }) => (
                <div
                  key={l}
                  style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12 }}
                >
                  <span style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, color: c }}>
                    {n}
                  </span>
                  <span style={{ color: 'var(--text-secondary)' }}>{l}</span>
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
                ...BTN,
                color: 'var(--btn-accent-text, #000)',
                borderColor: 'var(--accent)',
                background: 'var(--accent)',
              }}
              onClick={(e) => {
                e.stopPropagation();
                onDeploy();
              }}
            >
              ⎌ Deploy
            </button>
            <button
              type="button"
              style={BTN}
              onClick={(e) => {
                e.stopPropagation();
                scan.mutate(ep.id);
              }}
            >
              {scan.isPending ? 'Scanning…' : 'Scan'}
            </button>
            <button
              type="button"
              style={BTN}
              onClick={(e) => {
                e.stopPropagation();
                nav(ep.id);
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

function osToIconKey(f: string): string {
  const l = f.toLowerCase();
  if (/linux|ubuntu|debian|centos|rocky|rhel|fedora/.test(l)) return 'linux';
  if (/windows|win/.test(l)) return 'windows';
  if (/mac|darwin|apple/.test(l)) return 'darwin';
  return 'linux';
}

function EndpointCard({
  ep,
  selected,
  onSelect,
  onNavigate,
  onScan,
  onDeploy,
  onAssignTags,
  onDelete,
}: {
  ep: Endpoint;
  selected: boolean;
  onSelect: () => void;
  onNavigate: () => void;
  onScan: () => void;
  onDeploy: () => void;
  onAssignTags: () => void;
  onDelete: () => void;
}) {
  const displayStatus = deriveStatus(ep.status, ep.last_seen);
  const rs = computeRisk({
    criticalCves: ep.critical_cve_count ?? ep.critical_patch_count,
    highCves: ep.high_cve_count ?? ep.high_patch_count,
    mediumCves: ep.medium_cve_count ?? ep.medium_patch_count,
  });
  const pn = ep.pending_patches_count ?? 0;
  const tags = ep.tags ?? [];
  const [hovered, setHovered] = useState(false);

  return (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        background: selected
          ? 'color-mix(in srgb, var(--accent) 5%, var(--bg-card))'
          : 'var(--bg-card)',
        border: `1px solid ${selected ? 'var(--accent)' : hovered ? 'var(--border-hover)' : 'var(--border)'}`,
        borderRadius: 8,
        padding: 0,
        display: 'flex',
        flexDirection: 'column',
        transition: 'border-color 0.15s, background 0.15s',
        overflow: 'hidden',
      }}
    >
      {/* Header row */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px 14px',
          borderBottom: '1px solid var(--border)',
          background: 'var(--bg-inset)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, minWidth: 0 }}>
          <CB
            on={selected}
            onClick={(e) => {
              e.stopPropagation();
              onSelect();
            }}
          />
          <div
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              flexShrink: 0,
              background: dotColor(displayStatus),
              ...(displayStatus === 'pending'
                ? { animation: 'pulse-dot 1.8s ease-in-out infinite' }
                : {}),
            }}
          />
          <span
            onClick={(e) => {
              e.stopPropagation();
              onNavigate();
            }}
            style={{
              fontSize: 13,
              fontWeight: 700,
              color: 'var(--text-emphasis)',
              cursor: 'pointer',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {ep.hostname}
          </span>
        </div>
        <KebabMenu
          onView={onNavigate}
          onScan={onScan}
          onDeploy={onDeploy}
          onAssignTags={onAssignTags}
          onDelete={onDelete}
        />
      </div>

      {/* Body */}
      <div style={{ padding: '14px 14px 12px', display: 'flex', flexDirection: 'column', gap: 12 }}>
        {/* OS + Status row */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <span
            style={{
              fontSize: 12,
              color: 'var(--text-secondary)',
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
            }}
          >
            <OsIcon os={osToIconKey(ep.os_family)} size={14} /> {ep.os_version || ep.os_family}
          </span>
          <span
            style={{
              fontSize: 10,
              fontWeight: 600,
              fontFamily: 'var(--font-mono)',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
              color: sColor(displayStatus),
            }}
          >
            {sLabel(displayStatus)}
          </span>
        </div>

        {/* Stats grid: 3 columns */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr 1fr',
            gap: 8,
            textAlign: 'center',
          }}
        >
          <div>
            {(() => {
              const riskPct = Math.min(100, rs * 10);
              const r = 18;
              const circ = 2 * Math.PI * r;
              const fill = (riskPct / 100) * circ;
              return (
                <div style={{ position: 'relative', width: 44, height: 44, margin: '0 auto' }}>
                  <svg width="44" height="44" viewBox="0 0 44 44">
                    <circle
                      cx="22"
                      cy="22"
                      r={r}
                      fill="none"
                      stroke="var(--border)"
                      strokeWidth="3"
                    />
                    <circle
                      cx="22"
                      cy="22"
                      r={r}
                      fill="none"
                      stroke={riskClr(rs)}
                      strokeWidth="3"
                      strokeLinecap="round"
                      strokeDasharray={`${fill} ${circ}`}
                      transform="rotate(-90 22 22)"
                      style={{ transition: 'stroke-dasharray 0.4s ease' }}
                    />
                  </svg>
                  <div
                    style={{
                      position: 'absolute',
                      inset: 0,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 12,
                        fontWeight: 700,
                        color: riskClr(rs),
                      }}
                    >
                      {rs.toFixed(1)}
                    </span>
                  </div>
                </div>
              );
            })()}
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 9,
                color: 'var(--text-muted)',
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
              }}
            >
              Risk
            </div>
          </div>
          <div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 18,
                fontWeight: 700,
                color: pendClr(pn),
                lineHeight: 1,
                marginBottom: 3,
              }}
            >
              {pn}
            </div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 9,
                color: 'var(--text-muted)',
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
              }}
            >
              Pending
            </div>
            {(() => {
              const crit = ep.critical_patch_count ?? 0;
              const high = ep.high_patch_count ?? 0;
              const med = ep.medium_patch_count ?? 0;
              const total = crit + high + med;
              if (total <= 0) return null;
              return (
                <div
                  style={{
                    display: 'flex',
                    height: 3,
                    borderRadius: 2,
                    overflow: 'hidden',
                    marginTop: 4,
                    width: '100%',
                  }}
                >
                  {crit > 0 && <div style={{ flex: crit, background: 'var(--signal-critical)' }} />}
                  {high > 0 && <div style={{ flex: high, background: 'var(--signal-warning)' }} />}
                  {med > 0 && <div style={{ flex: med, background: 'var(--text-muted)' }} />}
                </div>
              );
            })()}
          </div>
          <div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 12,
                fontWeight: 600,
                color: 'var(--text-secondary)',
                lineHeight: 1,
                marginBottom: 3,
                marginTop: 3,
              }}
            >
              {relTime(ep.last_seen)}
            </div>
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 9,
                color: 'var(--text-muted)',
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
              }}
            >
              Last Seen
            </div>
          </div>
        </div>

        {/* Tags */}
        {tags.length > 0 && (
          <div
            style={{
              display: 'flex',
              flexWrap: 'wrap',
              gap: 4,
              borderTop: '1px solid var(--border)',
              paddingTop: 10,
            }}
          >
            {tags.slice(0, 3).map((t) => (
              <span
                key={t.id}
                style={{
                  display: 'inline-flex',
                  padding: '2px 7px',
                  background: 'var(--bg-inset)',
                  border: '1px solid var(--border-strong)',
                  borderRadius: 4,
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                  color: 'var(--text-secondary)',
                }}
              >
                {t.key}:{t.value}
              </span>
            ))}
            {tags.length > 3 && (
              <span
                style={{
                  fontSize: 10,
                  color: 'var(--text-faint)',
                  fontFamily: 'var(--font-mono)',
                  padding: '2px 4px',
                }}
              >
                +{tags.length - 3}
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

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
      style={{
        ...TH,
        cursor: 'pointer',
        userSelect: 'none',
      }}
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

export function EndpointsPage() {
  const nav = useNavigate();
  const can = useCan();
  const [search, setSearch] = useState('');
  const [sf, setSf] = useState<StatusFilter>('all');
  const [atOpen, setAtOpen] = useState(false);
  const [atIds, setAtIds] = useState<string[]>([]);
  const [deployOpen, setDeployOpen] = useState(false);
  const [sel, setSel] = useState<Set<string>>(new Set());
  const [exp, setExp] = useState<Set<string>>(new Set());
  const [curs, setCurs] = useState<string[]>([]);
  const pp = 15;
  const [regOpen, setRegOpen] = useState(false);
  const [expOpen, setExpOpen] = useState(false);

  const [tok, setTok] = useState<string | null>(null);
  const [cop, setCop] = useState(false);
  const crReg = useCreateRegistration();
  const tScan = useTriggerScan();
  const decommission = useDecommissionEndpoint();
  const [deleteConfirm, setDeleteConfirm] = useState(false);
  const [sortCol, setSortCol] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const [searchParams, setSearchParams] = useSearchParams();
  // Open register dialog from TopBar button via ?register=true
  useEffect(() => {
    if (searchParams.get('register') === 'true') {
      setRegOpen(true);
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.delete('register');
          return next;
        },
        { replace: true },
      );
    }
  }, [searchParams, setSearchParams]);

  // Drill-down from dashboard heatmap / other widgets via ?q=<hostname>
  useEffect(() => {
    const q = searchParams.get('q');
    if (q) {
      setSearch(q);
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.delete('q');
          return next;
        },
        { replace: true },
      );
    }
  }, [searchParams, setSearchParams]);

  const viewMode = (searchParams.get('view') === 'card' ? 'card' : 'list') as 'list' | 'card';
  const setViewMode = useCallback(
    (mode: 'list' | 'card') => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (mode === 'list') next.delete('view');
          else next.set('view', mode);
          return next;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  const cc = curs[curs.length - 1];
  const { data, isLoading, isError, refetch } = useEndpoints({
    cursor: cc,
    limit: pp,
    status: sf !== 'all' ? sf : undefined,
    search: search || undefined,
  });
  const eps = data?.data ?? [];
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
  const sortedEps = useMemo(() => {
    if (!sortCol) return eps;
    const sorted = [...eps].sort((a, b) => {
      let av: string | number = 0,
        bv: string | number = 0;
      switch (sortCol) {
        case 'hostname':
          av = a.hostname.toLowerCase();
          bv = b.hostname.toLowerCase();
          break;
        case 'os':
          av = (a.os_version || a.os_family).toLowerCase();
          bv = (b.os_version || b.os_family).toLowerCase();
          break;
        case 'status':
          av = a.status;
          bv = b.status;
          break;
        case 'risk':
          av = computeRisk({
            criticalCves: a.critical_cve_count ?? a.critical_patch_count,
            highCves: a.high_cve_count ?? a.high_patch_count,
            mediumCves: a.medium_cve_count ?? a.medium_patch_count,
          });
          bv = computeRisk({
            criticalCves: b.critical_cve_count ?? b.critical_patch_count,
            highCves: b.high_cve_count ?? b.high_patch_count,
            mediumCves: b.medium_cve_count ?? b.medium_patch_count,
          });
          break;
        case 'pending':
          av = a.pending_patches_count ?? 0;
          bv = b.pending_patches_count ?? 0;
          break;
        case 'last_seen':
          av = a.last_seen ? new Date(a.last_seen).getTime() : 0;
          bv = b.last_seen ? new Date(b.last_seen).getTime() : 0;
          break;
      }
      if (av < bv) return -1;
      if (av > bv) return 1;
      return 0;
    });
    return sortDir === 'desc' ? sorted.reverse() : sorted;
  }, [eps, sortCol, sortDir]);
  const sc = useMemo(() => {
    const c: Record<string, number> = {};
    for (const e of eps) c[e.status] = (c[e.status] ?? 0) + 1;
    return c;
  }, [eps]);
  const selN = sel.size;
  const togSel = useCallback((id: string) => {
    setSel((p) => {
      const n = new Set(p);
      if (n.has(id)) {
        n.delete(id);
      } else {
        n.add(id);
      }
      return n;
    });
  }, []);
  const togAll = useCallback(() => {
    setSel((p) => (p.size === eps.length ? new Set() : new Set(eps.map((e) => e.id))));
  }, [eps]);
  const togExp = useCallback((id: string) => {
    setExp((p) => {
      const n = new Set(p);
      if (n.has(id)) {
        n.delete(id);
      } else {
        n.add(id);
      }
      return n;
    });
  }, []);
  const allSel = eps.length > 0 && sel.size === eps.length;

  return (
    <div
      style={{
        background: 'var(--bg-page)',
        padding: 24,
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        minHeight: '100%',
      }}
    >
      {/* Stat Cards */}
      <div style={{ display: 'flex', gap: 8 }}>
        <StatCard
          label="Total"
          value={data?.total_count}
          active={sf === 'all'}
          onClick={() => {
            setSf('all');
            setCurs([]);
            setSel(new Set());
          }}
        />
        <StatCard
          label="Online"
          value={sc['online']}
          valueColor="var(--signal-healthy)"
          active={sf === 'online'}
          onClick={() => {
            setSf(sf === 'online' ? 'all' : 'online');
            setCurs([]);
            setSel(new Set());
          }}
        />
        <StatCard
          label="Offline"
          value={sc['offline']}
          valueColor="var(--signal-critical)"
          active={sf === 'offline'}
          onClick={() => {
            setSf(sf === 'offline' ? 'all' : 'offline');
            setCurs([]);
            setSel(new Set());
          }}
        />
        <StatCard
          label="Patching"
          value={sc['pending']}
          valueColor="var(--accent)"
          active={sf === 'pending'}
          onClick={() => {
            setSf(sf === 'pending' ? 'all' : 'pending');
            setCurs([]);
            setSel(new Set());
          }}
        />
      </div>

      {/* Filter Bar + Actions */}
      <div style={{ display: 'flex', alignItems: 'stretch', gap: 8 }}>
        <div
          style={{
            flex: 1,
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: '10px 14px',
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            flexWrap: 'wrap',
            boxShadow: 'var(--shadow-sm)',
          }}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '5px 10px',
              border: '1px solid var(--border)',
              borderRadius: 6,
              background: 'var(--bg-inset)',
              flex: 1,
              maxWidth: 360,
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
              aria-hidden="true"
            >
              <circle cx="11" cy="11" r="8" />
              <path d="M21 21l-4.35-4.35" />
            </svg>
            <input
              id="endpoints-search"
              name="search"
              type="text"
              aria-label="Search endpoints"
              placeholder="Search hostnames, OS, tags..."
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setCurs([]);
              }}
              style={{
                background: 'transparent',
                border: 'none',
                outline: 'none',
                fontSize: 12,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-sans)',
                width: '100%',
              }}
            />
            {search && (
              <button
                type="button"
                aria-label="Clear search"
                onClick={() => {
                  setSearch('');
                  setCurs([]);
                }}
                style={{
                  width: 16,
                  height: 16,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'transparent',
                  border: 'none',
                  cursor: 'pointer',
                  color: 'var(--text-muted)',
                  padding: 0,
                }}
              >
                <svg
                  width="10"
                  height="10"
                  viewBox="0 0 10 10"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                >
                  <path d="M2 2l6 6M8 2l-6 6" />
                </svg>
              </button>
            )}
          </div>
          <select
            id="endpoints-status"
            name="status"
            aria-label="Filter by status"
            value={sf}
            onChange={(e) => {
              setSf(e.target.value as StatusFilter);
              setCurs([]);
              setSel(new Set());
            }}
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 10px',
              fontSize: 11.5,
              color: sf !== 'all' ? 'var(--text-primary)' : 'var(--text-secondary)',
              outline: 'none',
              cursor: 'pointer',
            }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          >
            <option value="all">Status</option>
            <option value="online">Online</option>
            <option value="offline">Offline</option>
            <option value="pending">Patching</option>
            <option value="stale">Stale</option>
          </select>
          {(sf !== 'all' || search !== '') && (
            <button
              type="button"
              onClick={() => {
                setSf('all');
                setSearch('');
                setCurs([]);
              }}
              style={{
                padding: '5px 10px',
                fontSize: 11,
                borderRadius: 6,
                border: '1px solid var(--border)',
                background: 'transparent',
                color: 'var(--text-secondary)',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: 4,
              }}
            >
              <svg
                width="10"
                height="10"
                viewBox="0 0 10 10"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
              >
                <path d="M2 2l6 6M8 2l-6 6" />
              </svg>
              Clear filters
            </button>
          )}
          <div style={{ flex: 1 }} />
          <div
            style={{
              display: 'flex',
              border: '1px solid var(--border)',
              borderRadius: 6,
              overflow: 'hidden',
            }}
          >
            <button
              type="button"
              aria-label="List view"
              aria-pressed={viewMode === 'list'}
              onClick={() => setViewMode('list')}
              style={{
                padding: '5px 8px',
                background:
                  viewMode === 'list'
                    ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
                    : 'transparent',
                border: 'none',
                cursor: 'pointer',
                color: viewMode === 'list' ? 'var(--text-emphasis)' : 'var(--text-muted)',
                display: 'flex',
                alignItems: 'center',
              }}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path d="M2 3.5h10M2 7h10M2 10.5h10" stroke="currentColor" strokeWidth="1.2" />
              </svg>
            </button>
            <button
              type="button"
              aria-label="Grid view"
              aria-pressed={viewMode === 'card'}
              onClick={() => setViewMode('card')}
              style={{
                padding: '5px 8px',
                background:
                  viewMode === 'card'
                    ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
                    : 'transparent',
                border: 'none',
                borderLeft: '1px solid var(--border)',
                cursor: 'pointer',
                color: viewMode === 'card' ? 'var(--text-emphasis)' : 'var(--text-muted)',
                display: 'flex',
                alignItems: 'center',
              }}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <rect
                  x="2"
                  y="2"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
                <rect
                  x="8"
                  y="2"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
                <rect
                  x="2"
                  y="8"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
                <rect
                  x="8"
                  y="8"
                  width="4"
                  height="4"
                  rx="1"
                  stroke="currentColor"
                  strokeWidth="1.2"
                />
              </svg>
            </button>
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
          <button
            type="button"
            onClick={() => setExpOpen(true)}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 6,
              padding: '5px 12px',
              borderRadius: 6,
              fontSize: 12,
              fontWeight: 500,
              cursor: 'pointer',
              border: '1px solid var(--border)',
              background: 'transparent',
              color: 'var(--text-secondary)',
              fontFamily: 'var(--font-sans)',
              whiteSpace: 'nowrap',
            }}
          >
            Export
          </button>
        </div>
      </div>

      {/* Bulk Bar */}
      {selN > 0 && (
        <div
          style={{
            position: 'sticky',
            top: 0,
            zIndex: 10,
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: '10px 16px',
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            boxShadow: 'var(--shadow-sm)',
          }}
        >
          <span style={{ fontSize: 12, color: 'var(--text-secondary)', marginRight: 4 }}>
            <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{selN}</span> selected
          </span>
          <button
            disabled={!can('deployments', 'create')}
            title={!can('deployments', 'create') ? "You don't have permission" : undefined}
            onClick={() => setDeployOpen(true)}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 6,
              padding: '5px 12px',
              borderRadius: 6,
              fontSize: 12,
              fontWeight: 500,
              cursor: !can('deployments', 'create') ? 'not-allowed' : 'pointer',
              border: '1px solid var(--accent-border)',
              background: 'var(--accent-subtle)',
              color: 'var(--accent)',
              fontFamily: 'var(--font-sans)',
              opacity: !can('deployments', 'create') ? 0.5 : 1,
            }}
          >
            Deploy Patches
          </button>
          <button
            disabled={!can('endpoints', 'scan')}
            title={!can('endpoints', 'scan') ? "You don't have permission" : undefined}
            onClick={() => {
              for (const id of sel) tScan.mutate(id);
              setSel(new Set());
            }}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              padding: '5px 10px',
              borderRadius: 6,
              fontSize: 12,
              color: 'var(--text-secondary)',
              border: '1px solid var(--border)',
              background: 'transparent',
              cursor: !can('endpoints', 'scan') ? 'not-allowed' : 'pointer',
              fontFamily: 'var(--font-sans)',
              opacity: !can('endpoints', 'scan') ? 0.5 : 1,
            }}
          >
            Scan
          </button>
          <button
            disabled={!can('endpoints', 'create')}
            title={!can('endpoints', 'create') ? "You don't have permission" : undefined}
            onClick={() => {
              setAtIds([...sel]);
              setAtOpen(true);
            }}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              padding: '5px 10px',
              borderRadius: 6,
              fontSize: 12,
              color: 'var(--text-secondary)',
              border: '1px solid var(--border)',
              background: 'transparent',
              cursor: !can('endpoints', 'create') ? 'not-allowed' : 'pointer',
              fontFamily: 'var(--font-sans)',
              opacity: !can('endpoints', 'create') ? 0.5 : 1,
            }}
          >
            Assign Tags
          </button>
          <button
            disabled={!can('endpoints', 'delete')}
            title={!can('endpoints', 'delete') ? "You don't have permission" : undefined}
            onClick={() => setDeleteConfirm(true)}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              padding: '5px 10px',
              borderRadius: 6,
              fontSize: 12,
              color: 'var(--signal-critical)',
              border: '1px solid color-mix(in srgb, var(--signal-critical) 25%, transparent)',
              background: 'color-mix(in srgb, var(--signal-critical) 8%, transparent)',
              cursor: !can('endpoints', 'delete') ? 'not-allowed' : 'pointer',
              fontFamily: 'var(--font-sans)',
              opacity: !can('endpoints', 'delete') ? 0.5 : 1,
            }}
          >
            Delete
          </button>
          <button
            onClick={() => {
              const s = eps.filter((e) => sel.has(e.id));
              downloadCsvString(
                buildEndpointCsv(s),
                `endpoints-${new Date().toISOString().slice(0, 10)}.csv`,
              );
            }}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              padding: '5px 10px',
              borderRadius: 6,
              fontSize: 12,
              color: 'var(--text-secondary)',
              border: '1px solid var(--border)',
              background: 'transparent',
              cursor: 'pointer',
              fontFamily: 'var(--font-sans)',
            }}
          >
            Export
          </button>
          <button
            onClick={() => setSel(new Set())}
            style={{
              marginLeft: 'auto',
              fontSize: 12,
              color: 'var(--accent)',
              cursor: 'pointer',
              background: 'none',
              border: 'none',
              fontFamily: 'var(--font-sans)',
            }}
          >
            Clear
          </button>
        </div>
      )}

      {/* Table */}
      {isLoading ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {Array.from({ length: 6 }).map((_, i) => (
            <SkeletonCard key={i} lines={1} />
          ))}
        </div>
      ) : isError ? (
        <ErrorState message="Failed to load endpoints" onRetry={() => refetch()} />
      ) : viewMode === 'card' ? (
        <>
          {eps.length === 0 ? (
            data?.total_count === 0 && !search && sf === 'all' ? (
              <div
                style={{
                  padding: '32px 24px',
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  gap: 12,
                }}
              >
                <EmptyState
                  title="No endpoints enrolled yet"
                  description="Download and install the PatchIQ agent to get started."
                  action={{
                    label: 'Download Agent \u2192',
                    onClick: () => nav('/agent-downloads'),
                  }}
                />
              </div>
            ) : (
              <div
                style={{
                  height: 96,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: 'var(--text-muted)',
                  fontSize: 13,
                }}
              >
                No endpoints found.
              </div>
            )
          ) : (
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
                gap: 12,
              }}
            >
              {sortedEps.map((ep) => (
                <EndpointCard
                  key={ep.id}
                  ep={ep}
                  selected={sel.has(ep.id)}
                  onSelect={() => togSel(ep.id)}
                  onNavigate={() => nav(`/endpoints/${ep.id}`)}
                  onScan={() => tScan.mutate(ep.id)}
                  onDeploy={() => {
                    setSel(new Set([ep.id]));
                    setDeployOpen(true);
                  }}
                  onAssignTags={() => {
                    setAtIds([ep.id]);
                    setAtOpen(true);
                  }}
                  onDelete={() => {
                    setSel(new Set([ep.id]));
                    setDeleteConfirm(true);
                  }}
                />
              ))}
            </div>
          )}
          {/* Pagination */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '10px 14px',
              marginTop: 8,
              background: 'var(--bg-inset)',
              border: '1px solid var(--border)',
              borderRadius: 8,
            }}
          >
            <span
              style={{
                fontSize: 12,
                color: 'var(--text-muted)',
                fontFamily: 'var(--font-mono)',
              }}
            >
              {eps.length > 0
                ? `${curs.length * pp + 1}\u2013${curs.length * pp + eps.length} of ${fmtN(data?.total_count)}`
                : `0 of ${fmtN(data?.total_count)}`}
            </span>
            <div style={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <button
                disabled={curs.length === 0}
                onClick={() => setCurs((p) => p.slice(0, -1))}
                style={{
                  width: 28,
                  height: 28,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  borderRadius: 5,
                  cursor: curs.length > 0 ? 'pointer' : 'default',
                  color: curs.length > 0 ? 'var(--text-secondary)' : 'var(--text-faint)',
                  border: '1px solid transparent',
                  background: 'transparent',
                  opacity: curs.length === 0 ? 0.4 : 1,
                }}
              >
                <svg
                  viewBox="0 0 24 24"
                  width="12"
                  height="12"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <polyline points="15 18 9 12 15 6" />
                </svg>
              </button>
              <button
                disabled={!data?.next_cursor}
                onClick={() => {
                  if (data?.next_cursor) setCurs((p) => [...p, data.next_cursor!]);
                }}
                style={{
                  width: 28,
                  height: 28,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  borderRadius: 5,
                  cursor: data?.next_cursor ? 'pointer' : 'default',
                  color: data?.next_cursor ? 'var(--text-secondary)' : 'var(--text-faint)',
                  border: '1px solid transparent',
                  background: 'transparent',
                  opacity: !data?.next_cursor ? 0.4 : 1,
                }}
              >
                <svg
                  viewBox="0 0 24 24"
                  width="12"
                  height="12"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <polyline points="9 18 15 12 9 6" />
                </svg>
              </button>
            </div>
          </div>
        </>
      ) : (
        <div
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            overflow: 'hidden',
            boxShadow: 'var(--shadow-sm)',
          }}
        >
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr
                style={{
                  background: 'var(--bg-inset)',
                  borderBottom: '1px solid var(--border)',
                }}
              >
                <th style={{ width: 64, padding: '9px 12px 9px 12px', textAlign: 'left' }}>
                  <CB
                    on={allSel}
                    onClick={(e) => {
                      e.stopPropagation();
                      togAll();
                    }}
                  />
                </th>
                <SortHeader
                  label="Hostname"
                  colKey="hostname"
                  sortCol={sortCol}
                  sortDir={sortDir}
                  onSort={toggleSort}
                />
                <SortHeader
                  label="OS"
                  colKey="os"
                  sortCol={sortCol}
                  sortDir={sortDir}
                  onSort={toggleSort}
                />
                <SortHeader
                  label="Status"
                  colKey="status"
                  sortCol={sortCol}
                  sortDir={sortDir}
                  onSort={toggleSort}
                />
                <SortHeader
                  label="Risk Score"
                  colKey="risk"
                  sortCol={sortCol}
                  sortDir={sortDir}
                  onSort={toggleSort}
                />
                <SortHeader
                  label="Pending"
                  colKey="pending"
                  sortCol={sortCol}
                  sortDir={sortDir}
                  onSort={toggleSort}
                />
                <th style={TH}>Tags</th>
                <SortHeader
                  label="Last Seen"
                  colKey="last_seen"
                  sortCol={sortCol}
                  sortDir={sortDir}
                  onSort={toggleSort}
                />
                <th style={{ ...TH, width: 40 }} />
              </tr>
            </thead>
            <tbody>
              {eps.length === 0 ? (
                <tr>
                  <td colSpan={9}>
                    {data?.total_count === 0 && !search && sf === 'all' ? (
                      <div
                        style={{
                          padding: '32px 24px',
                          display: 'flex',
                          flexDirection: 'column',
                          alignItems: 'center',
                          gap: 12,
                        }}
                      >
                        <EmptyState
                          title="No endpoints enrolled yet"
                          description="Download and install the PatchIQ agent to get started."
                          action={{
                            label: 'Download Agent →',
                            onClick: () => nav('/agent-downloads'),
                          }}
                        />
                      </div>
                    ) : (
                      <div
                        style={{
                          height: 96,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          color: 'var(--text-muted)',
                          fontSize: 13,
                        }}
                      >
                        No endpoints found.
                      </div>
                    )}
                  </td>
                </tr>
              ) : (
                sortedEps.map((ep) => {
                  const isSel = sel.has(ep.id),
                    isExp = exp.has(ep.id);
                  const rs = riskScore(ep),
                    pn = ep.pending_patches_count ?? 0;
                  const epDisplayStatus = deriveStatus(ep.status, ep.last_seen);
                  const tags = ep.tags ?? [],
                    vt = tags.slice(0, 2),
                    ov = tags.length - 2;
                  return (
                    <Fragment key={ep.id}>
                      <tr
                        onClick={() => nav(`/endpoints/${ep.id}`)}
                        style={{
                          cursor: 'pointer',
                          borderLeft:
                            isSel || isExp ? '2px solid var(--accent)' : '2px solid transparent',
                          background: isSel
                            ? 'color-mix(in srgb, var(--accent) 5%, transparent)'
                            : isExp
                              ? 'var(--bg-inset)'
                              : 'transparent',
                        }}
                      >
                        <td style={{ ...TD, paddingLeft: 12 }}>
                          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                            <button
                              type="button"
                              aria-label={isExp ? 'Collapse' : 'Expand'}
                              onClick={(e) => {
                                e.stopPropagation();
                                togExp(ep.id);
                              }}
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
                                flexShrink: 0,
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
                                  transform: isExp ? 'rotate(90deg)' : 'rotate(0deg)',
                                  transition: 'transform 0.2s',
                                }}
                              >
                                <path d="M4.5 2.5L8.5 6L4.5 9.5" />
                              </svg>
                            </button>
                            <CB
                              on={isSel}
                              onClick={(e) => {
                                e.stopPropagation();
                                togSel(ep.id);
                              }}
                            />
                          </div>
                        </td>
                        <td style={TD}>
                          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                            <div
                              style={{
                                width: 6,
                                height: 6,
                                borderRadius: '50%',
                                flexShrink: 0,
                                background: dotColor(epDisplayStatus),
                                ...(epDisplayStatus === 'pending'
                                  ? { animation: 'pulse-dot 1.8s ease-in-out infinite' }
                                  : {}),
                              }}
                            />
                            <span
                              style={{
                                fontWeight: 600,
                                color: 'var(--text-emphasis)',
                                fontSize: 12.5,
                              }}
                            >
                              {ep.hostname}
                            </span>
                          </div>
                        </td>
                        <td style={TD}>
                          <span
                            style={{
                              color: 'var(--text-secondary)',
                              fontSize: 12,
                              whiteSpace: 'nowrap',
                            }}
                          >
                            {osEmoji(ep.os_family)} {ep.os_version || ep.os_family}
                          </span>
                        </td>
                        <td style={TD}>
                          <span
                            style={{
                              color: sColor(epDisplayStatus),
                              fontSize: 12,
                              fontWeight: 500,
                            }}
                          >
                            {sLabel(epDisplayStatus)}
                          </span>
                        </td>
                        <td style={TD}>
                          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                            <span
                              style={{
                                fontFamily: 'var(--font-mono)',
                                fontSize: 12,
                                fontWeight: 600,
                                width: 24,
                                textAlign: 'right',
                                flexShrink: 0,
                                color: riskClr(rs),
                              }}
                            >
                              {rs.toFixed(1)}
                            </span>
                            <div
                              style={{
                                width: 60,
                                height: 3,
                                background: 'var(--progress-track)',
                                borderRadius: 2,
                                overflow: 'hidden',
                                flexShrink: 0,
                              }}
                            >
                              <div
                                style={{
                                  height: '100%',
                                  borderRadius: 2,
                                  width: `${Math.min(100, rs * 10)}%`,
                                  background: riskClr(rs),
                                }}
                              />
                            </div>
                          </div>
                        </td>
                        <td style={TD}>
                          <span
                            style={{
                              fontFamily: 'var(--font-mono)',
                              fontSize: 12,
                              fontWeight: pn >= 3 ? 600 : 400,
                              color: pendClr(pn),
                            }}
                          >
                            {pn}
                          </span>
                        </td>
                        <td style={TD}>
                          <div
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 4,
                              flexWrap: 'nowrap',
                            }}
                          >
                            {vt.map((t) => (
                              <span
                                key={t.id}
                                style={{
                                  display: 'inline-flex',
                                  alignItems: 'center',
                                  padding: '2px 7px',
                                  background: 'var(--bg-inset)',
                                  border: '1px solid var(--border-strong)',
                                  borderRadius: 4,
                                  fontFamily: 'var(--font-mono)',
                                  fontSize: 10,
                                  color: 'var(--text-secondary)',
                                  whiteSpace: 'nowrap',
                                }}
                              >
                                {t.key}:{t.value}
                              </span>
                            ))}
                            {ov > 0 && (
                              <span
                                style={{
                                  fontSize: 10,
                                  color: 'var(--text-faint)',
                                  fontFamily: 'var(--font-mono)',
                                }}
                              >
                                +{ov}
                              </span>
                            )}
                          </div>
                        </td>
                        <td style={TD}>
                          <span
                            style={{
                              color: 'var(--text-muted)',
                              fontSize: 12,
                              whiteSpace: 'nowrap',
                            }}
                          >
                            {relTime(ep.last_seen)}
                          </span>
                        </td>
                        <td style={TD}>
                          <KebabMenu
                            onView={() => nav(`/endpoints/${ep.id}`)}
                            onScan={() => tScan.mutate(ep.id)}
                            onDeploy={() => {
                              setSel(new Set([ep.id]));
                              setDeployOpen(true);
                            }}
                            onAssignTags={() => {
                              setAtIds([ep.id]);
                              setAtOpen(true);
                            }}
                            onDelete={() => {
                              setSel(new Set([ep.id]));
                              setDeleteConfirm(true);
                            }}
                          />
                        </td>
                      </tr>
                      {isExp && (
                        <ExpRow
                          ep={ep}
                          nav={(id) => nav(`/endpoints/${id}`)}
                          onDeploy={() => {
                            setSel(new Set([ep.id]));
                            setDeployOpen(true);
                          }}
                        />
                      )}
                    </Fragment>
                  );
                })
              )}
            </tbody>
          </table>
          {/* Pagination */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '10px 14px',
              borderTop: '1px solid var(--border)',
              background: 'var(--bg-inset)',
            }}
          >
            <span
              style={{
                fontSize: 12,
                color: 'var(--text-muted)',
                fontFamily: 'var(--font-mono)',
              }}
            >
              {eps.length > 0
                ? `${curs.length * pp + 1}\u2013${curs.length * pp + eps.length} of ${fmtN(data?.total_count)}`
                : `0 of ${fmtN(data?.total_count)}`}
            </span>
            <div style={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <button
                disabled={curs.length === 0}
                onClick={() => setCurs((p) => p.slice(0, -1))}
                style={{
                  width: 28,
                  height: 28,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  borderRadius: 5,
                  cursor: curs.length > 0 ? 'pointer' : 'default',
                  color: curs.length > 0 ? 'var(--text-secondary)' : 'var(--text-faint)',
                  border: '1px solid transparent',
                  background: 'transparent',
                  opacity: curs.length === 0 ? 0.4 : 1,
                }}
              >
                <svg
                  viewBox="0 0 24 24"
                  width="12"
                  height="12"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <polyline points="15 18 9 12 15 6" />
                </svg>
              </button>
              <button
                disabled={!data?.next_cursor}
                onClick={() => {
                  if (data?.next_cursor) setCurs((p) => [...p, data.next_cursor!]);
                }}
                style={{
                  width: 28,
                  height: 28,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  borderRadius: 5,
                  cursor: data?.next_cursor ? 'pointer' : 'default',
                  color: data?.next_cursor ? 'var(--text-secondary)' : 'var(--text-faint)',
                  border: '1px solid transparent',
                  background: 'transparent',
                  opacity: !data?.next_cursor ? 0.4 : 1,
                }}
              >
                <svg
                  viewBox="0 0 24 24"
                  width="12"
                  height="12"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <polyline points="9 18 15 12 9 6" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      )}
      {/* Deploy Patches Wizard */}
      <DeploymentWizard open={deployOpen} onOpenChange={setDeployOpen} />

      {/* Assign Tags Dialog */}
      <AssignTagsDialog
        open={atOpen}
        onOpenChange={(o) => {
          setAtOpen(o);
          if (!o) {
            setAtIds([]);
            setSel(new Set());
          }
        }}
        selectedEndpointIds={atIds}
      />

      {/* Export Endpoints Dialog */}
      <ExportEndpointsDialog
        open={expOpen}
        onOpenChange={setExpOpen}
        filteredCount={eps.length}
        filters={{ status: sf !== 'all' ? sf : undefined, search: search || undefined }}
      />

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteConfirm} onOpenChange={setDeleteConfirm}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              Delete {sel.size} endpoint{sel.size > 1 ? 's' : ''}?
            </DialogTitle>
          </DialogHeader>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            This will permanently remove the selected endpoint{sel.size > 1 ? 's' : ''} and all
            associated data. This action cannot be undone.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteConfirm(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                for (const id of sel) decommission.mutate(id);
                setSel(new Set());
                setDeleteConfirm(false);
              }}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Register Endpoint Dialog */}
      <RegisterEndpointDialog
        open={regOpen}
        onOpenChange={setRegOpen}
        crReg={crReg}
        tok={tok}
        setTok={setTok}
        cop={cop}
        setCop={setCop}
      />
    </div>
  );
}

function OsIcon({ os, size = 24 }: { os: string; size?: number }) {
  const color = 'var(--text-secondary)';
  if (os === 'linux')
    return (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={color}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M12 2C9 2 7 5 7 9c0 2-2 3-3 4s-1 3 1 3c1 0 2-1 3-1s3 1 4 1 3-1 4-1 2 1 3 1c2 0 2-2 1-3s-3-2-3-4c0-4-2-7-5-7z" />
        <circle cx="9.5" cy="8" r="0.8" fill={color} stroke="none" />
        <circle cx="14.5" cy="8" r="0.8" fill={color} stroke="none" />
        <path d="M9 11c0 1 1.5 2 3 2s3-1 3-2" />
      </svg>
    );
  if (os === 'darwin')
    return (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={color}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M12 20.94c1.5 0 2.75 1.06 4 1.06 2.25 0 4-2.83 4-6.5 0-3-1.62-4.5-3.5-4.5-1.23 0-2.28.63-3 .63-.72 0-1.77-.63-3-.63-1.88 0-3.5 1.5-3.5 4.5 0 3.67 1.75 6.5 4 6.5 1.25 0 2.5-1.06 4-1.06z" />
        <path d="M15 5c0 1.66-1.34 3-3 3S9 6.66 9 5" />
        <path d="M12 2v6" />
      </svg>
    );
  if (os === 'windows')
    return (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={color}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M3 5.5l7-1v6.5H3zM12 4.2l9-1.2v8H12zM3 13h7v6.5l-7-1zM12 13h9v8l-9-1.2z" />
      </svg>
    );
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <path d="M9 9h6M9 12h6M9 15h4" />
    </svg>
  );
}
function osLabel(os: string): string {
  if (os === 'linux') return 'Linux';
  if (os === 'darwin') return 'macOS';
  if (os === 'windows') return 'Windows';
  return os;
}
function archLabel(arch: string, os: string): string {
  if (arch === 'amd64') return 'AMD64 (x86_64)';
  if (arch === 'arm64') return os === 'darwin' ? 'ARM64 (Apple Silicon)' : 'ARM64 (aarch64)';
  return arch;
}
function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

function RegisterEndpointDialog({
  open,
  onOpenChange,
  crReg,
  tok,
  setTok,
  cop,
  setCop,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  crReg: ReturnType<typeof useCreateRegistration>;
  tok: string | null;
  setTok: (t: string | null) => void;
  cop: boolean;
  setCop: (c: boolean) => void;
}) {
  const [selectedOs, setSelectedOs] = useState<string | null>(null);
  const [selectedBinary, setSelectedBinary] = useState<AgentBinaryInfo | null>(null);
  const [cmdCopied, setCmdCopied] = useState(false);
  const { data: binaries, isLoading: binariesLoading } = useAgentBinaries();

  const osList = useMemo(() => {
    if (!binaries) return [];
    return [...new Set(binaries.map((b) => b.os))];
  }, [binaries]);

  const archBinaries = useMemo(() => {
    if (!binaries || !selectedOs) return [];
    return binaries.filter((b) => b.os === selectedOs);
  }, [binaries, selectedOs]);

  useEffect(() => {
    if (!open) {
      setSelectedOs(null);
      setSelectedBinary(null);
      setTok(null);
      setCop(false);
      setCmdCopied(false);
    }
  }, [open, setTok, setCop]);

  const serverUrl = `${window.location.protocol}//${window.location.host}`;
  const isWindows = selectedBinary?.os === 'windows';
  const installCmd =
    selectedBinary && tok
      ? isWindows
        ? `.\\${selectedBinary.filename} install --server-url ${serverUrl} --token ${tok}`
        : `chmod +x ${selectedBinary.filename} && sudo ./${selectedBinary.filename} install --server-url ${serverUrl} --token ${tok}`
      : '';

  const handleClose = () => onOpenChange(false);

  const labelStyle: React.CSSProperties = {
    fontSize: 10,
    fontWeight: 600,
    color: 'var(--text-muted)',
    textTransform: 'uppercase',
    letterSpacing: '0.06em',
    marginBottom: 4,
  };

  const codeBlockStyle: React.CSSProperties = {
    fontFamily: 'var(--font-mono)',
    fontSize: 11,
    background: 'var(--bg-inset)',
    border: '1px solid var(--border)',
    borderRadius: 6,
    padding: '6px 8px',
    color: 'var(--accent)',
    wordBreak: 'break-all',
    flex: 1,
  };

  const copyBtnStyle: React.CSSProperties = {
    flexShrink: 0,
    padding: '4px 10px',
    borderRadius: 5,
    border: '1px solid var(--border)',
    background: 'transparent',
    color: 'var(--text-secondary)',
    fontSize: 11,
    cursor: 'pointer',
  };

  const sectionHeader = (num: number, title: string): React.ReactNode => (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 2 }}>
      <div
        style={{
          width: 20,
          height: 20,
          borderRadius: '50%',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 10,
          fontWeight: 600,
          fontFamily: 'var(--font-sans)',
          background: 'var(--accent)',
          color: '#fff',
          flexShrink: 0,
        }}
      >
        {num}
      </div>
      <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-emphasis)' }}>{title}</span>
    </div>
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent style={{ maxWidth: 440, overflow: 'hidden' }}>
        <DialogHeader>
          <DialogTitle>Register Endpoint</DialogTitle>
        </DialogHeader>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {/* Section 1: Select Platform */}
          <div>
            {sectionHeader(1, 'Select Platform')}
            {binariesLoading ? (
              <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                Loading available binaries...
              </p>
            ) : !binaries || binaries.length === 0 ? (
              <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                No agent binaries available. Upload binaries via the admin panel.
              </p>
            ) : (
              <>
                <div style={labelStyle}>Operating System</div>
                <div style={{ display: 'flex', gap: 8 }}>
                  {osList.map((os) => (
                    <button
                      key={os}
                      onClick={() => {
                        setSelectedOs(os);
                        setSelectedBinary(null);
                      }}
                      style={{
                        flex: 1,
                        padding: '6px 6px',
                        borderRadius: 6,
                        border: `1.5px solid ${selectedOs === os ? 'var(--accent)' : 'var(--border)'}`,
                        background: selectedOs === os ? 'var(--bg-inset)' : 'var(--bg-card)',
                        cursor: 'pointer',
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: 'center',
                        gap: 2,
                        color: 'var(--text-emphasis)',
                        fontSize: 12,
                        fontFamily: 'var(--font-sans)',
                      }}
                    >
                      <OsIcon os={os} size={18} />
                      <span>{osLabel(os)}</span>
                    </button>
                  ))}
                </div>
                {selectedOs && archBinaries.length > 0 && (
                  <>
                    <div style={{ ...labelStyle, marginTop: 4 }}>Architecture</div>
                    <div style={{ display: 'flex', gap: 6 }}>
                      {archBinaries.map((b) => (
                        <button
                          key={b.arch}
                          onClick={() => setSelectedBinary(b)}
                          style={{
                            flex: 1,
                            padding: '5px 6px',
                            borderRadius: 6,
                            border: `1.5px solid ${selectedBinary?.arch === b.arch ? 'var(--accent)' : 'var(--border)'}`,
                            background:
                              selectedBinary?.arch === b.arch
                                ? 'var(--bg-inset)'
                                : 'var(--bg-card)',
                            cursor: 'pointer',
                            display: 'flex',
                            flexDirection: 'column',
                            alignItems: 'center',
                            gap: 1,
                            color: 'var(--text-emphasis)',
                            fontSize: 11,
                            fontFamily: 'var(--font-sans)',
                          }}
                        >
                          <span style={{ fontWeight: 600 }}>{archLabel(b.arch, b.os)}</span>
                          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                            {formatSize(b.size)}
                          </span>
                        </button>
                      ))}
                    </div>
                  </>
                )}
              </>
            )}
          </div>

          {/* Section 2: Generate Token */}
          <div
            style={{
              opacity: selectedBinary ? 1 : 0.4,
              pointerEvents: selectedBinary ? 'auto' : 'none',
            }}
          >
            {sectionHeader(2, 'Generate Token')}
            <p style={{ fontSize: 11, color: 'var(--text-secondary)', margin: '0 0 6px 0' }}>
              Generate a one-time registration token. You'll need this when running the install command on the target machine.
            </p>
            {tok ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                <div style={labelStyle}>Registration Token</div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <code style={codeBlockStyle}>{tok}</code>
                  <button
                    onClick={() => {
                      void navigator.clipboard.writeText(tok);
                      setCop(true);
                      setTimeout(() => setCop(false), 2000);
                    }}
                    style={copyBtnStyle}
                  >
                    {cop ? 'Copied!' : 'Copy'}
                  </button>
                </div>
                <p style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 0 }}>
                  This token expires in 24 hours and can only be used once.
                </p>
              </div>
            ) : (
              <Button
                onClick={() =>
                  crReg.mutate(undefined, {
                    onSuccess: (data) =>
                      setTok((data as { registration_token: string }).registration_token),
                  })
                }
                disabled={crReg.isPending || !selectedBinary}
              >
                {crReg.isPending ? 'Generating...' : 'Generate Token'}
              </Button>
            )}
            {crReg.isError && (
              <p style={{ fontSize: 12, color: 'var(--signal-critical)', marginTop: 4 }}>
                Failed to generate token. Please try again.
              </p>
            )}
          </div>

          {/* Section 3: Download Agent */}
          <div
            style={{
              opacity: selectedBinary && tok ? 1 : 0.4,
              pointerEvents: selectedBinary && tok ? 'auto' : 'none',
            }}
          >
            {sectionHeader(3, 'Download Agent')}
            <p style={{ fontSize: 11, color: 'var(--text-secondary)', margin: '0 0 6px 0' }}>
              Download the installer and transfer it to the target endpoint.
            </p>
            {selectedBinary ? (
              <>
                <div
                  style={{
                    background: 'var(--bg-inset)',
                    border: '1px solid var(--border)',
                    borderRadius: 6,
                    padding: 8,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                  }}
                >
                  <div>
                    <div style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-emphasis)' }}>
                      {selectedBinary.filename}
                    </div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 1 }}>
                      {formatSize(selectedBinary.size)}
                    </div>
                  </div>
                  <a
                    href={`/api/v1/agent-binaries/${selectedBinary.filename}/download`}
                    download
                    style={{
                      padding: '6px 16px',
                      borderRadius: 6,
                      background: 'var(--accent)',
                      color: '#fff',
                      fontSize: 12,
                      fontWeight: 600,
                      textDecoration: 'none',
                      cursor: 'pointer',
                    }}
                  >
                    Download
                  </a>
                </div>
                <p style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 2 }}>
                  {selectedBinary.os === 'linux'
                    ? 'Extract the archive on the target machine before running the install command.'
                    : 'Copy this file to the target machine before running the install command.'}
                </p>
              </>
            ) : (
              <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                Select a platform and architecture above first.
              </p>
            )}
          </div>

          {/* Section 4: Install Command */}
          <div
            style={{
              opacity: selectedBinary && tok ? 1 : 0.4,
              pointerEvents: selectedBinary && tok ? 'auto' : 'none',
            }}
          >
            {sectionHeader(4, 'Install')}
            <p style={{ fontSize: 11, color: 'var(--text-secondary)', margin: '0 0 6px 0' }}>
              {selectedBinary?.os === 'windows'
                ? 'Open PowerShell as Administrator on the target machine and run this command.'
                : 'Open a terminal on the target machine and run this command as root.'}
            </p>
            {selectedBinary && tok ? (
              <>
                <div style={labelStyle}>Install Command</div>
                <div style={{ display: 'flex', alignItems: 'flex-start', gap: 8 }}>
                  <code
                    style={{
                      ...codeBlockStyle,
                      whiteSpace: 'pre-wrap',
                      fontSize: 10,
                    }}
                  >
                    {installCmd}
                  </code>
                  <button
                    onClick={() => {
                      void navigator.clipboard.writeText(installCmd);
                      setCmdCopied(true);
                      setTimeout(() => setCmdCopied(false), 2000);
                    }}
                    style={copyBtnStyle}
                  >
                    {cmdCopied ? 'Copied!' : 'Copy'}
                  </button>
                </div>
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    padding: '5px 8px',
                    borderRadius: 6,
                    background: 'var(--bg-inset)',
                    border: '1px solid var(--border)',
                    marginTop: 3,
                  }}
                >
                  <OsIcon os={selectedBinary.os} size={14} />
                  <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                    {osLabel(selectedBinary.os)} {archLabel(selectedBinary.arch, selectedBinary.os)}
                  </span>
                  <span
                    style={{
                      marginLeft: 'auto',
                      fontSize: 11,
                      color: 'var(--signal-healthy)',
                      fontWeight: 600,
                    }}
                  >
                    Ready to install
                  </span>
                </div>
              </>
            ) : (
              <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                Select a platform and generate a token to see the install command.
              </p>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
