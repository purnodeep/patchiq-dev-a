import { useState, useMemo } from 'react';
import { motion } from 'framer-motion';
import { Shield, ChevronDown, ChevronRight, Download, Rocket, Search } from 'lucide-react';
import { useHotkeys } from '@/hooks/useHotkeys';
import { GlassCard } from '@/components/shared/GlassCard';
import { SectionHeader } from '@/components/shared/SectionHeader';
import { StatCard } from '@/components/shared/StatCard';
import { PageHeader } from '@/components/shared/PageHeader';

// ── Framer Motion variants ────────────────────────────────────────────────────
const stagger = {
  hidden: {},
  show: { transition: { staggerChildren: 0.06 } },
};

const fadeUp = {
  hidden: { opacity: 0, y: 12 },
  show: { opacity: 1, y: 0, transition: { duration: 0.4, ease: 'easeOut' } },
};

// ── Mock data ─────────────────────────────────────────────────────────────────
type Severity = 'critical' | 'high' | 'medium' | 'low';
type PatchStatus = 'pending' | 'partial' | 'deployed';

interface CveEntry {
  id: string;
  score: number;
}

interface Patch {
  name: string;
  ver: string;
  title: string;
  severity: Severity;
  os: string;
  cveCount: number;
  cves: CveEntry[];
  cvss: number;
  affected: number;
  deployed: number;
  released: string;
  status: PatchStatus;
}

const PATCHES: Patch[] = [
  {
    name: 'KB5034441',
    ver: '2024.01.B',
    title: 'Windows Recovery Env Update',
    severity: 'critical',
    os: 'Windows',
    cveCount: 3,
    cves: [
      { id: 'CVE-2024-20666', score: 9.8 },
      { id: 'CVE-2024-20653', score: 8.1 },
      { id: 'CVE-2024-20652', score: 7.5 },
    ],
    cvss: 9.8,
    affected: 89,
    deployed: 0,
    released: 'Jan 9, 2024',
    status: 'pending',
  },
  {
    name: 'KB5035853',
    ver: '2024.02.C',
    title: 'Cumulative Update Win 11 22H2',
    severity: 'high',
    os: 'Windows',
    cveCount: 8,
    cves: [
      { id: 'CVE-2024-21338', score: 7.8 },
      { id: 'CVE-2024-21305', score: 6.2 },
      { id: 'CVE-2024-21341', score: 7.1 },
    ],
    cvss: 7.8,
    affected: 136,
    deployed: 89,
    released: 'Feb 13, 2024',
    status: 'partial',
  },
  {
    name: 'USN-6587-1',
    ver: 'openssl 3.0.13',
    title: 'OpenSSL Vulnerability Fix',
    severity: 'critical',
    os: 'Ubuntu',
    cveCount: 2,
    cves: [
      { id: 'CVE-2024-0727', score: 9.1 },
      { id: 'CVE-2023-6237', score: 7.4 },
    ],
    cvss: 9.1,
    affected: 45,
    deployed: 0,
    released: 'Jan 25, 2024',
    status: 'pending',
  },
  {
    name: 'RHSA-2024:0012',
    ver: 'kernel-6.5.7',
    title: 'Kernel Security Update',
    severity: 'high',
    os: 'RHEL',
    cveCount: 5,
    cves: [
      { id: 'CVE-2024-1086', score: 7.4 },
      { id: 'CVE-2024-0646', score: 7.0 },
      { id: 'CVE-2023-6817', score: 5.9 },
    ],
    cvss: 7.4,
    affected: 18,
    deployed: 12,
    released: 'Jan 15, 2024',
    status: 'partial',
  },
  {
    name: 'KB5036893',
    ver: '2024.03.A',
    title: 'SharePoint Server 2019 CU',
    severity: 'high',
    os: 'Windows',
    cveCount: 4,
    cves: [
      { id: 'CVE-2024-26251', score: 8.8 },
      { id: 'CVE-2024-26254', score: 7.2 },
    ],
    cvss: 8.8,
    affected: 4,
    deployed: 0,
    released: 'Mar 12, 2024',
    status: 'pending',
  },
  {
    name: 'USN-6648-1',
    ver: 'linux-kernel 6.5.0',
    title: 'Linux Kernel 6.5 Update',
    severity: 'high',
    os: 'Ubuntu',
    cveCount: 3,
    cves: [
      { id: 'CVE-2024-1085', score: 7.8 },
      { id: 'CVE-2024-1086', score: 7.0 },
      { id: 'CVE-2023-6931', score: 6.1 },
    ],
    cvss: 7.8,
    affected: 27,
    deployed: 27,
    released: 'Feb 28, 2024',
    status: 'deployed',
  },
  {
    name: 'KB5034122',
    ver: '.NET 4.8.1 2024.01',
    title: '.NET Framework 4.8.1 Update',
    severity: 'medium',
    os: 'Windows',
    cveCount: 2,
    cves: [
      { id: 'CVE-2024-21312', score: 6.5 },
      { id: 'CVE-2024-21318', score: 5.4 },
    ],
    cvss: 6.5,
    affected: 22,
    deployed: 14,
    released: 'Jan 9, 2024',
    status: 'partial',
  },
  {
    name: 'DSA-5620',
    ver: 'chromium 121.0.6167.85',
    title: 'Chromium Security Update',
    severity: 'medium',
    os: 'Debian',
    cveCount: 12,
    cves: [
      { id: 'CVE-2024-0807', score: 6.3 },
      { id: 'CVE-2024-0808', score: 5.9 },
      { id: 'CVE-2024-0812', score: 5.1 },
    ],
    cvss: 6.3,
    affected: 15,
    deployed: 15,
    released: 'Feb 6, 2024',
    status: 'deployed',
  },
  {
    name: 'RHSA-2024:1891',
    ver: 'openssl 3.0.x',
    title: 'OpenSSL 3.0.x Critical',
    severity: 'critical',
    os: 'RHEL',
    cveCount: 1,
    cves: [{ id: 'CVE-2024-2511', score: 9.4 }],
    cvss: 9.4,
    affected: 18,
    deployed: 0,
    released: 'Mar 20, 2024',
    status: 'pending',
  },
  {
    name: 'KB5037849',
    ver: '2024.04.A',
    title: 'Windows Defender Update',
    severity: 'low',
    os: 'Windows',
    cveCount: 1,
    cves: [{ id: 'CVE-2024-26234', score: 3.3 }],
    cvss: 3.3,
    affected: 247,
    deployed: 247,
    released: 'Apr 2, 2024',
    status: 'deployed',
  },
];

const SEVERITY_PILLS = [
  { label: 'All', count: 156 },
  { label: 'Critical', count: 12 },
  { label: 'High', count: 23 },
  { label: 'Medium', count: 45 },
  { label: 'Low', count: 76 },
];

// ── Helpers ───────────────────────────────────────────────────────────────────

const SEVERITY_COLORS: Record<Severity, string> = {
  critical: 'var(--color-danger)',
  high: 'var(--color-warning)',
  medium: 'var(--color-caution, #f59e0b)',
  low: 'var(--color-success)',
};

const STATUS_COLORS: Record<PatchStatus, string> = {
  pending: 'var(--color-danger)',
  partial: 'var(--color-warning)',
  deployed: 'var(--color-success)',
};

const OS_COLORS: Record<string, { bg: string; color: string; letter: string }> = {
  Windows: { bg: 'color-mix(in srgb, #0078d4 18%, transparent)', color: '#0078d4', letter: 'W' },
  Ubuntu: { bg: 'color-mix(in srgb, #e95420 18%, transparent)', color: '#e95420', letter: 'U' },
  RHEL: { bg: 'color-mix(in srgb, #cc0000 18%, transparent)', color: '#cc0000', letter: 'R' },
  Debian: { bg: 'color-mix(in srgb, #d70a53 18%, transparent)', color: '#d70a53', letter: 'D' },
};

function cvssColor(score: number): string {
  if (score >= 9.0) return 'var(--color-danger)';
  if (score >= 7.0) return 'var(--color-warning)';
  return 'var(--color-caution, #f59e0b)';
}

// ── Sub-components ────────────────────────────────────────────────────────────

function SeverityBadge({ severity }: { severity: Severity }) {
  const color = SEVERITY_COLORS[severity];
  return (
    <span
      style={{
        fontSize: 10,
        fontWeight: 700,
        padding: '2px 8px',
        borderRadius: 4,
        background: `color-mix(in srgb, ${color} 14%, transparent)`,
        color,
        border: `1px solid color-mix(in srgb, ${color} 28%, transparent)`,
        textTransform: 'capitalize',
        letterSpacing: '0.03em',
      }}
    >
      {severity}
    </span>
  );
}

function StatusBadge({ status }: { status: PatchStatus }) {
  const color = STATUS_COLORS[status];
  return (
    <span
      style={{
        fontSize: 10,
        fontWeight: 700,
        padding: '2px 8px',
        borderRadius: 4,
        background: `color-mix(in srgb, ${color} 14%, transparent)`,
        color,
        border: `1px solid color-mix(in srgb, ${color} 28%, transparent)`,
        textTransform: 'capitalize',
        letterSpacing: '0.03em',
      }}
    >
      {status}
    </span>
  );
}

function OsIcon({ os }: { os: string }) {
  const cfg = OS_COLORS[os] ?? {
    bg: 'var(--color-separator)',
    color: 'var(--color-muted)',
    letter: os[0] ?? '?',
  };
  return (
    <div
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: 18,
        height: 18,
        borderRadius: 4,
        background: cfg.bg,
        color: cfg.color,
        fontSize: 9,
        fontWeight: 700,
        flexShrink: 0,
      }}
    >
      {cfg.letter}
    </div>
  );
}

function RemediationBar({ deployed, affected }: { deployed: number; affected: number }) {
  const pct = affected > 0 ? (deployed / affected) * 100 : 0;
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
      <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--color-foreground)' }}>
        {deployed}/{affected}
      </span>
      <div
        style={{
          height: 4,
          width: 64,
          borderRadius: 3,
          background: 'var(--color-separator)',
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            height: '100%',
            width: `${pct}%`,
            borderRadius: 3,
            background:
              pct >= 100
                ? 'var(--color-success)'
                : pct > 0
                  ? 'var(--color-warning)'
                  : 'var(--color-danger)',
            transition: 'width 0.6s ease',
          }}
        />
      </div>
    </div>
  );
}

function FilterPill({
  label,
  count,
  active,
  onClick,
}: {
  label: string;
  count: number;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 6,
        padding: '5px 12px',
        borderRadius: 20,
        border: active ? '1px solid var(--color-primary)' : '1px solid var(--color-separator)',
        background: active
          ? 'color-mix(in srgb, var(--color-primary) 12%, transparent)'
          : 'transparent',
        cursor: 'pointer',
        fontSize: 12,
        fontWeight: active ? 600 : 400,
        color: active ? 'var(--color-primary)' : 'var(--color-muted)',
        transition: 'all 0.15s ease',
      }}
    >
      {label}
      <span
        style={{
          fontSize: 10,
          fontWeight: 700,
          padding: '1px 5px',
          borderRadius: 8,
          background: active
            ? 'color-mix(in srgb, var(--color-primary) 20%, transparent)'
            : 'var(--color-separator)',
          color: active ? 'var(--color-primary)' : 'var(--color-muted)',
        }}
      >
        {count}
      </span>
    </button>
  );
}

const SELECT_STYLE: React.CSSProperties = {
  background: 'var(--color-surface, #111827)',
  border: '1px solid var(--color-separator)',
  borderRadius: 7,
  padding: '5px 10px',
  color: 'var(--color-foreground)',
  fontSize: 11.5,
  outline: 'none',
  cursor: 'pointer',
  colorScheme: 'dark',
};

const TABLE_COLS = [
  { label: 'Patch Name', width: '15%' },
  { label: 'Version', width: '10%' },
  { label: 'Severity', width: '8%' },
  { label: 'OS', width: '8%' },
  { label: 'CVEs', width: '5%' },
  { label: 'CVSS', width: '7%' },
  { label: 'Affected', width: '7%' },
  { label: 'Remediation', width: '13%' },
  { label: 'Released', width: '10%' },
  { label: 'Status', width: '8%' },
  { label: '', width: '4%' },
];

const GRID_TEMPLATE = TABLE_COLS.map((c) => c.width).join(' ');

// ── Create Deployment Modal ───────────────────────────────────────────────────

interface DeployModalProps {
  patch: Patch | null;
  onClose: () => void;
}

function DeployModal({ patch, onClose }: DeployModalProps) {
  const [depName, setDepName] = useState(patch ? `${patch.name} Deployment` : '');
  const [depDesc, setDepDesc] = useState('');
  const [configType, setConfigType] = useState<'install' | 'rollback'>('install');
  const [scope, setScope] = useState('');
  const [endpoints, setEndpoints] = useState('');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');

  function handleBackdrop(e: React.MouseEvent<HTMLDivElement>) {
    if (e.target === e.currentTarget) onClose();
  }

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '8px 12px',
    background: 'rgba(255,255,255,0.05)',
    border: '1px solid var(--color-separator)',
    borderRadius: 6,
    color: 'var(--color-foreground)',
    fontSize: 12,
    outline: 'none',
    colorScheme: 'dark',
  };

  const labelStyle: React.CSSProperties = {
    display: 'block',
    fontSize: 12,
    fontWeight: 600,
    marginBottom: 6,
    color: 'var(--color-foreground)',
  };

  return (
    <div
      onClick={handleBackdrop}
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        backdropFilter: 'blur(4px)',
        zIndex: 50,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <div
        style={{
          position: 'fixed',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%,-50%)',
          zIndex: 51,
          background: 'var(--color-card, #0f172a)',
          border: '1px solid var(--color-separator)',
          borderRadius: 12,
          width: 680,
          maxHeight: '85vh',
          overflowY: 'auto',
          boxShadow: '0 20px 60px rgba(0,0,0,0.6)',
        }}
      >
        {/* Modal header */}
        <div
          style={{
            padding: '16px 20px',
            borderBottom: '1px solid var(--color-separator)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            flexShrink: 0,
          }}
        >
          <h2
            style={{ fontSize: 16, fontWeight: 700, margin: 0, color: 'var(--color-foreground)' }}
          >
            Create Patch Deployment
          </h2>
          <button
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              fontSize: 20,
              cursor: 'pointer',
              color: 'var(--color-muted)',
              padding: 0,
              width: 24,
              height: 24,
              lineHeight: 1,
            }}
          >
            ×
          </button>
        </div>

        {/* Modal body */}
        <div style={{ padding: 20 }}>
          {/* Deployment Name */}
          <div style={{ marginBottom: 16 }}>
            <label style={labelStyle}>
              Deployment Name <span style={{ color: 'var(--color-danger)' }}>*</span>
            </label>
            <input
              type="text"
              value={depName}
              onChange={(e) => setDepName(e.target.value)}
              placeholder="e.g., KB5034441 - Critical Patch"
              style={inputStyle}
            />
          </div>

          {/* Description */}
          <div style={{ marginBottom: 16 }}>
            <label style={labelStyle}>Description</label>
            <textarea
              value={depDesc}
              onChange={(e) => setDepDesc(e.target.value)}
              placeholder="Optional: Deployment notes, approval info..."
              style={{ ...inputStyle, minHeight: 70, resize: 'vertical' }}
            />
          </div>

          {/* Config Type */}
          <div style={{ marginBottom: 16 }}>
            <label style={labelStyle}>
              Configuration Type <span style={{ color: 'var(--color-danger)' }}>*</span>
            </label>
            <div style={{ display: 'flex', gap: 16 }}>
              {(['install', 'rollback'] as const).map((val) => (
                <label
                  key={val}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    cursor: 'pointer',
                    fontSize: 12,
                  }}
                >
                  <input
                    type="radio"
                    name="depConfig"
                    value={val}
                    checked={configType === val}
                    onChange={() => setConfigType(val)}
                    style={{ cursor: 'pointer', width: 16, height: 16 }}
                  />
                  <span style={{ color: 'var(--color-foreground)', textTransform: 'capitalize' }}>
                    {val}
                  </span>
                </label>
              ))}
            </div>
          </div>

          {/* Scope & Endpoints */}
          <div
            style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 16 }}
          >
            <div>
              <label style={labelStyle}>
                Scope <span style={{ color: 'var(--color-danger)' }}>*</span>
              </label>
              <select
                value={scope}
                onChange={(e) => setScope(e.target.value)}
                style={{ ...inputStyle, colorScheme: 'dark' }}
              >
                <option value="">Select Scope</option>
                <option value="prod">Production</option>
                <option value="staging">Staging</option>
                <option value="dev">Development</option>
              </select>
            </div>
            <div>
              <label style={labelStyle}>
                Target Endpoints <span style={{ color: 'var(--color-danger)' }}>*</span>
              </label>
              <select
                value={endpoints}
                onChange={(e) => setEndpoints(e.target.value)}
                style={{ ...inputStyle, colorScheme: 'dark' }}
              >
                <option value="">Please Select Endpoints</option>
                <option value="all">All Eligible</option>
                <option value="windows">Windows Only</option>
                <option value="linux">Linux Only</option>
                <option value="critical">Critical</option>
              </select>
            </div>
          </div>

          {/* Patches table */}
          {patch && (
            <div style={{ marginBottom: 16 }}>
              <label style={labelStyle}>
                Patches to Deploy <span style={{ color: 'var(--color-danger)' }}>*</span>
              </label>
              <div
                style={{
                  background: 'rgba(0,0,0,0.2)',
                  border: '1px solid var(--color-separator)',
                  borderRadius: 6,
                  overflow: 'hidden',
                }}
              >
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11 }}>
                  <thead>
                    <tr
                      style={{
                        background: 'rgba(0,0,0,0.2)',
                        borderBottom: '1px solid var(--color-separator)',
                      }}
                    >
                      {['ID', 'Version', 'Severity', 'OS'].map((h) => (
                        <th
                          key={h}
                          style={{
                            padding: '8px 10px',
                            textAlign: 'left',
                            color: 'var(--color-muted)',
                            fontWeight: 600,
                          }}
                        >
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    <tr>
                      <td
                        style={{
                          padding: '8px 10px',
                          fontFamily: 'monospace',
                          color: 'var(--color-primary)',
                          fontWeight: 600,
                        }}
                      >
                        {patch.name}
                      </td>
                      <td style={{ padding: '8px 10px', color: 'var(--color-muted)' }}>
                        {patch.ver}
                      </td>
                      <td style={{ padding: '8px 10px' }}>
                        <SeverityBadge severity={patch.severity} />
                      </td>
                      <td style={{ padding: '8px 10px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
                          <OsIcon os={patch.os} />
                          <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>
                            {patch.os}
                          </span>
                        </div>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Schedule */}
          <div style={{ marginBottom: 16 }}>
            <label style={labelStyle}>Schedule Deployment (Optional)</label>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              <div>
                <label
                  style={{
                    display: 'block',
                    fontSize: 10,
                    color: 'var(--color-muted)',
                    marginBottom: 4,
                  }}
                >
                  Start Date
                </label>
                <input
                  type="date"
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                  style={{ ...inputStyle, colorScheme: 'dark' }}
                />
              </div>
              <div>
                <label
                  style={{
                    display: 'block',
                    fontSize: 10,
                    color: 'var(--color-muted)',
                    marginBottom: 4,
                  }}
                >
                  End Date
                </label>
                <input
                  type="date"
                  value={endDate}
                  onChange={(e) => setEndDate(e.target.value)}
                  style={{ ...inputStyle, colorScheme: 'dark' }}
                />
              </div>
            </div>
          </div>
        </div>

        {/* Modal footer */}
        <div
          style={{
            padding: '14px 20px',
            borderTop: '1px solid var(--color-separator)',
            display: 'flex',
            gap: 8,
            justifyContent: 'flex-end',
            flexShrink: 0,
          }}
        >
          <button
            onClick={onClose}
            style={{
              padding: '6px 14px',
              borderRadius: 6,
              border: '1px solid var(--color-separator)',
              background: 'transparent',
              cursor: 'pointer',
              fontSize: 12,
              fontWeight: 500,
              color: 'var(--color-foreground)',
            }}
          >
            Cancel
          </button>
          <button
            onClick={onClose}
            style={{
              padding: '6px 14px',
              borderRadius: 6,
              border: '1px solid var(--color-separator)',
              background: 'transparent',
              cursor: 'pointer',
              fontSize: 12,
              fontWeight: 500,
              color: 'var(--color-muted)',
            }}
          >
            Save as Draft
          </button>
          <button
            onClick={onClose}
            style={{
              padding: '6px 14px',
              borderRadius: 6,
              border: 'none',
              background: 'var(--color-primary)',
              cursor: 'pointer',
              fontSize: 12,
              fontWeight: 600,
              color: '#fff',
            }}
          >
            Publish
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Expanded row panel ────────────────────────────────────────────────────────

function ExpandedPanel({ patch, onDeploy }: { patch: Patch; onDeploy: (p: Patch) => void }) {
  const pending = patch.affected - patch.deployed;
  const successRate = patch.affected > 0 ? Math.round((patch.deployed / patch.affected) * 100) : 0;

  return (
    <div
      style={{
        gridColumn: `1 / -1`,
        padding: '16px 20px',
        background: 'rgba(0,0,0,0.18)',
        borderTop: '1px solid var(--color-separator)',
        display: 'grid',
        gridTemplateColumns: '1fr 1fr 1fr',
        gap: 20,
        animation: 'fadeInDown 0.15s ease',
      }}
    >
      {/* Col 1 — CVE Details */}
      <div>
        <div
          style={{
            fontSize: 10,
            fontWeight: 600,
            color: 'var(--color-muted)',
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            marginBottom: 8,
          }}
        >
          CVE Details
        </div>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <tbody>
            {patch.cves.map((cve) => (
              <tr key={cve.id}>
                <td
                  style={{
                    padding: '4px 6px',
                    fontSize: 11,
                    fontFamily: 'monospace',
                    color: 'var(--color-primary)',
                    borderBottom:
                      '1px solid color-mix(in srgb, var(--color-separator) 40%, transparent)',
                  }}
                >
                  {cve.id}
                </td>
                <td
                  style={{
                    padding: '4px 6px',
                    fontSize: 11,
                    fontWeight: 700,
                    color: cvssColor(cve.score),
                    fontFamily: 'monospace',
                    borderBottom:
                      '1px solid color-mix(in srgb, var(--color-separator) 40%, transparent)',
                    textAlign: 'right',
                  }}
                >
                  {cve.score.toFixed(1)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Col 2 — Deployment Stats */}
      <div>
        <div
          style={{
            fontSize: 10,
            fontWeight: 600,
            color: 'var(--color-muted)',
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            marginBottom: 8,
          }}
        >
          Deployment Stats
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 7, fontSize: 11 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: 'var(--color-muted)' }}>Affected Endpoints</span>
            <span style={{ fontWeight: 700, color: 'var(--color-foreground)' }}>
              {patch.affected}
            </span>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: 'var(--color-muted)' }}>Deployed</span>
            <span style={{ fontWeight: 600, color: 'var(--color-success)' }}>{patch.deployed}</span>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: 'var(--color-muted)' }}>Pending</span>
            <span style={{ fontWeight: 600, color: 'var(--color-warning)' }}>{pending}</span>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: 'var(--color-muted)' }}>Success Rate</span>
            <span style={{ fontWeight: 600, color: 'var(--color-success)' }}>{successRate}%</span>
          </div>
        </div>
      </div>

      {/* Col 3 — Actions */}
      <div style={{ display: 'flex', alignItems: 'flex-end' }}>
        <button
          onClick={() => onDeploy(patch)}
          style={{
            width: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 6,
            padding: '8px 14px',
            borderRadius: 6,
            border: 'none',
            background: 'var(--color-primary)',
            color: '#fff',
            cursor: 'pointer',
            fontSize: 12,
            fontWeight: 600,
          }}
        >
          <Rocket size={13} />
          Deploy Now
        </button>
      </div>
    </div>
  );
}

// ── Patches page ──────────────────────────────────────────────────────────────
export default function Patches() {
  useHotkeys();

  const [activeFilter, setActiveFilter] = useState('All');
  const [searchQuery, setSearchQuery] = useState('');
  const [osFilter, setOsFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [showDeployModal, setShowDeployModal] = useState(false);
  const [selectedPatch, setSelectedPatch] = useState<Patch | null>(null);
  const [currentPage, setCurrentPage] = useState(1);

  function toggleRow(name: string) {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }

  function openDeploy(patch: Patch | null) {
    setSelectedPatch(patch);
    setShowDeployModal(true);
  }

  function closeDeploy() {
    setShowDeployModal(false);
    setSelectedPatch(null);
  }

  const filtered = useMemo(() => {
    return PATCHES.filter((p) => {
      if (activeFilter !== 'All' && p.severity !== activeFilter.toLowerCase()) return false;
      if (searchQuery) {
        const q = searchQuery.toLowerCase();
        if (!p.name.toLowerCase().includes(q) && !p.title.toLowerCase().includes(q)) return false;
      }
      if (osFilter && p.os !== osFilter) return false;
      if (statusFilter && p.status !== statusFilter.toLowerCase()) return false;
      return true;
    });
  }, [activeFilter, searchQuery, osFilter, statusFilter]);

  const TOTAL = 156;

  return (
    <>
      {showDeployModal && <DeployModal patch={selectedPatch} onClose={closeDeploy} />}

      <motion.div
        variants={stagger}
        initial="hidden"
        animate="show"
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
          padding: '20px 24px',
          overflowY: 'auto',
          height: '100%',
        }}
      >
        {/* Row 1 — Page header */}
        <motion.div variants={fadeUp}>
          <PageHeader
            title="Patches"
            subtitle="156 patches tracked across all OS families"
            actions={
              <>
                <button
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    padding: '6px 12px',
                    borderRadius: 6,
                    border: '1px solid var(--color-separator)',
                    background: 'transparent',
                    cursor: 'pointer',
                    fontSize: 12,
                    fontWeight: 500,
                    color: 'var(--color-muted)',
                  }}
                >
                  <Download size={13} />
                  Export
                </button>
                <button
                  onClick={() => openDeploy(null)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    padding: '6px 12px',
                    borderRadius: 6,
                    border: 'none',
                    background: 'var(--color-primary)',
                    cursor: 'pointer',
                    fontSize: 12,
                    fontWeight: 600,
                    color: '#fff',
                  }}
                >
                  <Rocket size={13} />
                  Create Deployment
                </button>
              </>
            }
          />
        </motion.div>

        {/* Row 2 — Stat cards */}
        <motion.div
          variants={fadeUp}
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(4, 1fr)',
            gap: 12,
          }}
        >
          <StatCard
            icon={<Shield size={16} color="var(--color-danger)" />}
            iconColor="var(--color-danger)"
            value="12"
            valueColor="var(--color-danger)"
            label="Critical Patches"
            trend={{ value: '+3', positive: false }}
            trendText="this week"
          />
          <StatCard
            icon={<Shield size={16} color="var(--color-warning)" />}
            iconColor="var(--color-warning)"
            value="23"
            valueColor="var(--color-warning)"
            label="High Severity"
            trend={{ value: '+1', positive: false }}
            trendText="this week"
          />
          <StatCard
            icon={<Shield size={16} color="var(--color-success)" />}
            iconColor="var(--color-success)"
            value="45%"
            valueColor="var(--color-success)"
            label="Avg Remediation"
            trend={{ value: '8%', positive: true }}
            trendText="vs last month"
          />
          <StatCard
            icon={<Shield size={16} color="var(--color-cyan)" />}
            iconColor="var(--color-cyan)"
            value="35"
            valueColor="var(--color-cyan)"
            label="Pending Deploy"
          />
        </motion.div>

        {/* Row 3 — Filter pills */}
        <motion.div
          variants={fadeUp}
          style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}
        >
          {SEVERITY_PILLS.map((pill) => (
            <FilterPill
              key={pill.label}
              label={pill.label}
              count={pill.count}
              active={activeFilter === pill.label}
              onClick={() => {
                setActiveFilter(pill.label);
                setCurrentPage(1);
              }}
            />
          ))}
          <div style={{ marginLeft: 'auto', display: 'flex', gap: 8 }}>
            <select
              value={osFilter}
              onChange={(e) => {
                setOsFilter(e.target.value);
                setCurrentPage(1);
              }}
              style={SELECT_STYLE}
            >
              <option value="">All OS</option>
              <option value="Windows">Windows</option>
              <option value="Ubuntu">Ubuntu</option>
              <option value="RHEL">RHEL</option>
              <option value="Debian">Debian</option>
            </select>
            <select
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setCurrentPage(1);
              }}
              style={SELECT_STYLE}
            >
              <option value="">All Status</option>
              <option value="pending">Pending</option>
              <option value="partial">Partial</option>
              <option value="deployed">Deployed</option>
            </select>
          </div>
        </motion.div>

        {/* Row 4 — Search bar */}
        <motion.div variants={fadeUp}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '7px 10px',
              borderRadius: 8,
              border: '1px solid var(--color-separator)',
              background: 'rgba(255,255,255,0.03)',
            }}
          >
            <Search size={13} color="var(--color-muted)" style={{ flexShrink: 0 }} />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                setCurrentPage(1);
              }}
              placeholder="Search patches (KB, USN, RHSA…)"
              style={{
                width: '100%',
                padding: 0,
                borderRadius: 0,
                border: 'none',
                background: 'transparent',
                color: 'var(--color-foreground)',
                fontSize: 12,
                outline: 'none',
              }}
            />
          </div>
        </motion.div>

        {/* Row 5 — Patches table */}
        <motion.div variants={fadeUp}>
          <GlassCard className="p-5" hover={false}>
            <SectionHeader
              title="Patch Catalog"
              action={
                <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>
                  Showing {filtered.length} of {TOTAL}
                </span>
              }
            />

            {/* Table */}
            <div style={{ marginTop: 16, overflowX: 'auto' }}>
              {/* Header */}
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: GRID_TEMPLATE,
                  gap: 0,
                  paddingBottom: 8,
                  borderBottom: '1px solid var(--color-separator)',
                }}
              >
                {TABLE_COLS.map((col) => (
                  <div
                    key={col.label}
                    style={{
                      fontSize: 10,
                      fontWeight: 600,
                      color: 'var(--color-muted)',
                      letterSpacing: '0.06em',
                      textTransform: 'uppercase',
                      paddingRight: 8,
                    }}
                  >
                    {col.label}
                  </div>
                ))}
              </div>

              {/* Rows */}
              {filtered.map((patch, idx) => {
                const isExpanded = expandedRows.has(patch.name);
                return (
                  <div key={patch.name}>
                    {/* Main row */}
                    <div
                      style={{
                        display: 'grid',
                        gridTemplateColumns: GRID_TEMPLATE,
                        gap: 0,
                        padding: '10px 0',
                        borderBottom:
                          !isExpanded && idx < filtered.length - 1
                            ? '1px solid var(--color-separator)'
                            : isExpanded
                              ? '1px solid var(--color-separator)'
                              : 'none',
                        alignItems: 'center',
                        cursor: 'pointer',
                        transition: 'background 0.12s ease',
                        borderRadius: 6,
                      }}
                      onMouseEnter={(e) => {
                        (e.currentTarget as HTMLDivElement).style.background =
                          'color-mix(in srgb, var(--color-primary) 4%, transparent)';
                      }}
                      onMouseLeave={(e) => {
                        (e.currentTarget as HTMLDivElement).style.background = 'transparent';
                      }}
                    >
                      {/* Patch Name */}
                      <div style={{ paddingRight: 8 }}>
                        <span
                          style={{
                            fontSize: 12,
                            fontWeight: 700,
                            color: 'var(--color-foreground)',
                            fontFamily: 'monospace',
                            display: 'block',
                          }}
                        >
                          {patch.name}
                        </span>
                        <span
                          style={{
                            fontSize: 10,
                            color: 'var(--color-muted)',
                            display: 'block',
                            marginTop: 1,
                          }}
                        >
                          {patch.title}
                        </span>
                      </div>

                      {/* Version */}
                      <div style={{ paddingRight: 8 }}>
                        <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>
                          {patch.ver}
                        </span>
                      </div>

                      {/* Severity */}
                      <div style={{ paddingRight: 8 }}>
                        <SeverityBadge severity={patch.severity} />
                      </div>

                      {/* OS */}
                      <div
                        style={{ paddingRight: 8, display: 'flex', alignItems: 'center', gap: 5 }}
                      >
                        <OsIcon os={patch.os} />
                        <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>
                          {patch.os}
                        </span>
                      </div>

                      {/* CVE Count */}
                      <div style={{ paddingRight: 8 }}>
                        <span
                          style={{
                            fontSize: 12,
                            fontWeight: 600,
                            color:
                              patch.cveCount > 5
                                ? 'var(--color-warning)'
                                : 'var(--color-foreground)',
                          }}
                        >
                          {patch.cveCount}
                        </span>
                      </div>

                      {/* CVSS */}
                      <div style={{ paddingRight: 8 }}>
                        <span
                          style={{
                            fontSize: 13,
                            fontWeight: 700,
                            color: cvssColor(patch.cvss),
                            fontFamily: 'monospace',
                          }}
                        >
                          {patch.cvss.toFixed(1)}
                        </span>
                      </div>

                      {/* Affected */}
                      <div style={{ paddingRight: 8 }}>
                        <span
                          style={{
                            fontSize: 12,
                            fontWeight: 600,
                            color: 'var(--color-foreground)',
                          }}
                        >
                          {patch.affected}
                        </span>
                      </div>

                      {/* Remediation */}
                      <div style={{ paddingRight: 8 }}>
                        <RemediationBar deployed={patch.deployed} affected={patch.affected} />
                      </div>

                      {/* Released */}
                      <div style={{ paddingRight: 8 }}>
                        <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>
                          {patch.released}
                        </span>
                      </div>

                      {/* Status */}
                      <div>
                        <StatusBadge status={patch.status} />
                      </div>

                      {/* Expand toggle */}
                      <div
                        style={{ display: 'flex', justifyContent: 'center' }}
                        onClick={(e) => {
                          e.stopPropagation();
                          toggleRow(patch.name);
                        }}
                      >
                        <button
                          style={{
                            background: 'none',
                            border: 'none',
                            cursor: 'pointer',
                            color: 'var(--color-muted)',
                            padding: '2px 4px',
                            borderRadius: 3,
                            display: 'flex',
                            alignItems: 'center',
                            transition: 'color 0.15s ease',
                          }}
                          onMouseEnter={(e) => {
                            (e.currentTarget as HTMLButtonElement).style.color =
                              'var(--color-foreground)';
                          }}
                          onMouseLeave={(e) => {
                            (e.currentTarget as HTMLButtonElement).style.color =
                              'var(--color-muted)';
                          }}
                        >
                          {isExpanded ? <ChevronDown size={13} /> : <ChevronRight size={13} />}
                        </button>
                      </div>
                    </div>

                    {/* Expanded panel */}
                    {isExpanded && (
                      <div
                        style={{
                          borderBottom:
                            idx < filtered.length - 1 ? '1px solid var(--color-separator)' : 'none',
                        }}
                      >
                        <ExpandedPanel patch={patch} onDeploy={openDeploy} />
                      </div>
                    )}
                  </div>
                );
              })}

              {filtered.length === 0 && (
                <div
                  style={{
                    padding: '32px 0',
                    textAlign: 'center',
                    color: 'var(--color-muted)',
                    fontSize: 13,
                  }}
                >
                  No patches match your filters.
                </div>
              )}
            </div>

            {/* Pagination */}
            <div
              style={{
                marginTop: 0,
                padding: '12px 0 0',
                borderTop: '1px solid var(--color-separator)',
                display: 'flex',
                alignItems: 'center',
                gap: 8,
              }}
            >
              <span style={{ fontSize: 11, color: 'var(--color-muted)', flex: 1 }}>
                Showing {filtered.length} of {TOTAL} patches
              </span>
              <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
                <button
                  disabled={currentPage === 1}
                  onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                  style={{
                    padding: '4px 10px',
                    borderRadius: 5,
                    fontSize: 11,
                    border: '1px solid var(--color-separator)',
                    background: 'transparent',
                    color: currentPage === 1 ? 'var(--color-muted)' : 'var(--color-foreground)',
                    cursor: currentPage === 1 ? 'not-allowed' : 'pointer',
                    opacity: currentPage === 1 ? 0.4 : 1,
                  }}
                >
                  ← Prev
                </button>
                {[1, 2, 3, 4, 5].map((n) => (
                  <button
                    key={n}
                    onClick={() => setCurrentPage(n)}
                    style={{
                      padding: '4px 10px',
                      borderRadius: 5,
                      fontSize: 11,
                      border: '1px solid var(--color-separator)',
                      background: currentPage === n ? 'var(--color-primary)' : 'transparent',
                      color: currentPage === n ? '#fff' : 'var(--color-foreground)',
                      cursor: 'pointer',
                      transition: 'all 0.12s ease',
                    }}
                  >
                    {n}
                  </button>
                ))}
                <span style={{ color: 'var(--color-muted)', fontSize: 12, padding: '0 4px' }}>
                  …
                </span>
                <button
                  onClick={() => setCurrentPage(11)}
                  style={{
                    padding: '4px 10px',
                    borderRadius: 5,
                    fontSize: 11,
                    border: '1px solid var(--color-separator)',
                    background: currentPage === 11 ? 'var(--color-primary)' : 'transparent',
                    color: currentPage === 11 ? '#fff' : 'var(--color-foreground)',
                    cursor: 'pointer',
                  }}
                >
                  11
                </button>
                <button
                  onClick={() => setCurrentPage((p) => Math.min(11, p + 1))}
                  style={{
                    padding: '4px 10px',
                    borderRadius: 5,
                    fontSize: 11,
                    border: '1px solid var(--color-separator)',
                    background: 'transparent',
                    color: 'var(--color-foreground)',
                    cursor: 'pointer',
                  }}
                >
                  Next →
                </button>
              </div>
            </div>
          </GlassCard>
        </motion.div>
      </motion.div>
    </>
  );
}
