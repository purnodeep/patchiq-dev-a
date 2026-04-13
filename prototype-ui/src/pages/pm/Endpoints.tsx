import { useState, useMemo } from 'react';
import { motion } from 'framer-motion';
import { Monitor, Search, ChevronDown, ChevronRight, Plus, Download } from 'lucide-react';
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
type EndpointStatus = 'online' | 'offline' | 'pending';

interface Endpoint {
  hostname: string;
  os: string;
  status: EndpointStatus;
  agent: string;
  lastSeen: string;
  critical: number;
  high: number;
  medium: number;
  riskScore: number;
  compliance: number;
  groups: string[];
}

const ENDPOINTS: Endpoint[] = [
  {
    hostname: 'prod-web-01',
    os: 'Ubuntu 22.04',
    status: 'online',
    agent: '2.1.4',
    lastSeen: '1 min ago',
    critical: 3,
    high: 7,
    medium: 12,
    riskScore: 72,
    compliance: 91,
    groups: ['Production', 'Web'],
  },
  {
    hostname: 'prod-web-02',
    os: 'Ubuntu 22.04',
    status: 'online',
    agent: '2.1.4',
    lastSeen: '1 min ago',
    critical: 2,
    high: 5,
    medium: 9,
    riskScore: 65,
    compliance: 87,
    groups: ['Production', 'Web'],
  },
  {
    hostname: 'win-app-01',
    os: 'Windows Server 2022',
    status: 'online',
    agent: '2.1.3',
    lastSeen: '3 min ago',
    critical: 5,
    high: 9,
    medium: 15,
    riskScore: 88,
    compliance: 72,
    groups: ['App Servers'],
  },
  {
    hostname: 'db-primary-01',
    os: 'RHEL 9.2',
    status: 'online',
    agent: '2.1.4',
    lastSeen: '2 min ago',
    critical: 4,
    high: 8,
    medium: 11,
    riskScore: 82,
    compliance: 68,
    groups: ['Database', 'Critical'],
  },
  {
    hostname: 'k8s-node-01',
    os: 'Ubuntu 22.04',
    status: 'online',
    agent: '2.1.4',
    lastSeen: '1 min ago',
    critical: 1,
    high: 3,
    medium: 5,
    riskScore: 41,
    compliance: 95,
    groups: ['Kubernetes'],
  },
  {
    hostname: 'infra-vpn-01',
    os: 'FortiOS 7.4',
    status: 'online',
    agent: '2.0.9',
    lastSeen: '5 min ago',
    critical: 7,
    high: 12,
    medium: 18,
    riskScore: 95,
    compliance: 45,
    groups: ['Network', 'Critical'],
  },
  {
    hostname: 'dev-build-01',
    os: 'Ubuntu 20.04',
    status: 'offline',
    agent: '2.1.2',
    lastSeen: '2h ago',
    critical: 0,
    high: 2,
    medium: 6,
    riskScore: 28,
    compliance: 88,
    groups: ['Dev'],
  },
  {
    hostname: 'win-dc-01',
    os: 'Windows Server 2019',
    status: 'online',
    agent: '2.1.3',
    lastSeen: '4 min ago',
    critical: 3,
    high: 6,
    medium: 10,
    riskScore: 70,
    compliance: 79,
    groups: ['Infrastructure', 'AD'],
  },
  {
    hostname: 'macos-dev-01',
    os: 'macOS Sonoma',
    status: 'pending',
    agent: '2.1.1',
    lastSeen: '15 min ago',
    critical: 2,
    high: 4,
    medium: 7,
    riskScore: 58,
    compliance: 83,
    groups: ['Dev'],
  },
  {
    hostname: 'infra-jump-01',
    os: 'Ubuntu 22.04',
    status: 'online',
    agent: '2.1.4',
    lastSeen: '1 min ago',
    critical: 0,
    high: 1,
    medium: 3,
    riskScore: 18,
    compliance: 97,
    groups: ['Infrastructure'],
  },
];

const FILTER_PILLS = [
  { label: 'All', count: 247 },
  { label: 'Online', count: 189 },
  { label: 'Offline', count: 34 },
  { label: 'Pending', count: 18 },
  { label: 'Decommissioned', count: 6 },
];

const TABLE_COLS = [
  { label: 'Hostname', width: '18%' },
  { label: 'OS', width: '14%' },
  { label: 'Status', width: '8%' },
  { label: 'Agent Ver', width: '7%' },
  { label: 'Risk Score', width: '9%' },
  { label: 'Last Seen', width: '9%' },
  { label: 'Patches', width: '12%' },
  { label: 'Groups', width: '15%' },
  { label: '', width: '4%' },
];

// ── Sub-components ────────────────────────────────────────────────────────────

function StatusDot({ status }: { status: EndpointStatus }) {
  if (status === 'online') {
    return (
      <div style={{ position: 'relative', width: 10, height: 10, flexShrink: 0 }}>
        <div
          style={{
            position: 'absolute',
            inset: -3,
            borderRadius: '50%',
            border: '2px solid var(--color-success)',
            animation: 'pulse-ring 1.6s ease-out infinite',
          }}
        />
        <div
          style={{
            width: 10,
            height: 10,
            borderRadius: '50%',
            background: 'var(--color-success)',
            position: 'relative',
            zIndex: 1,
          }}
        />
      </div>
    );
  }
  if (status === 'pending') {
    return (
      <div style={{ position: 'relative', width: 10, height: 10, flexShrink: 0 }}>
        <div
          style={{
            position: 'absolute',
            inset: -3,
            borderRadius: '50%',
            border: '2px solid var(--color-warning)',
            animation: 'pulse-ring 1.6s ease-out infinite',
          }}
        />
        <div
          style={{
            width: 10,
            height: 10,
            borderRadius: '50%',
            background: 'var(--color-warning)',
            position: 'relative',
            zIndex: 1,
          }}
        />
      </div>
    );
  }
  // offline
  return (
    <div
      style={{
        width: 10,
        height: 10,
        borderRadius: '50%',
        background: 'var(--color-muted)',
        flexShrink: 0,
      }}
    />
  );
}

function RiskScore({ value }: { value: number }) {
  const color =
    value >= 70
      ? 'var(--color-danger)'
      : value >= 40
        ? 'var(--color-warning)'
        : 'var(--color-success)';
  const label = value >= 70 ? 'High' : value >= 40 ? 'Med' : 'Low';
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
      <div style={{ width: 7, height: 7, borderRadius: '50%', background: color, flexShrink: 0 }} />
      <span style={{ fontSize: 12, fontWeight: 600, color }}>{value}</span>
      <span style={{ fontSize: 10, color: 'var(--color-muted)' }}>{label}</span>
    </div>
  );
}

function PatchChips({
  critical,
  high,
  medium,
}: {
  critical: number;
  high: number;
  medium: number;
}) {
  if (critical === 0 && high === 0 && medium === 0) {
    return <span style={{ fontSize: 11, color: 'var(--color-success)' }}>✓ Up to date</span>;
  }
  return (
    <div style={{ display: 'flex', gap: 4 }}>
      {critical > 0 && (
        <span
          style={{
            fontSize: 10,
            fontWeight: 700,
            padding: '2px 6px',
            borderRadius: 4,
            background: 'color-mix(in srgb, var(--color-danger) 15%, transparent)',
            color: 'var(--color-danger)',
            border: '1px solid color-mix(in srgb, var(--color-danger) 30%, transparent)',
          }}
        >
          {critical}C
        </span>
      )}
      {high > 0 && (
        <span
          style={{
            fontSize: 10,
            fontWeight: 700,
            padding: '2px 6px',
            borderRadius: 4,
            background: 'color-mix(in srgb, var(--color-warning) 15%, transparent)',
            color: 'var(--color-warning)',
            border: '1px solid color-mix(in srgb, var(--color-warning) 30%, transparent)',
          }}
        >
          {high}H
        </span>
      )}
      {medium > 0 && (
        <span
          style={{
            fontSize: 10,
            fontWeight: 700,
            padding: '2px 6px',
            borderRadius: 4,
            background: 'color-mix(in srgb, var(--color-muted) 15%, transparent)',
            color: 'var(--color-muted)',
            border: '1px solid color-mix(in srgb, var(--color-muted) 30%, transparent)',
          }}
        >
          {medium}M
        </span>
      )}
    </div>
  );
}

function OsIcon({ os }: { os: string }) {
  const lower = os.toLowerCase();
  const [bg, label] =
    lower.includes('ubuntu') || lower.includes('debian')
      ? ['#f97316', 'U']
      : lower.includes('rhel') || lower.includes('centos')
        ? ['#dc2626', 'R']
        : lower.includes('windows')
          ? ['#2563eb', 'W']
          : lower.includes('macos') || lower.includes('mac')
            ? ['#6b7280', 'M']
            : lower.includes('fortios') || lower.includes('forti')
              ? ['#d97706', 'F']
              : ['#475569', os.slice(0, 1).toUpperCase()];
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: 18,
        height: 18,
        borderRadius: 4,
        background: bg,
        color: '#fff',
        fontSize: 9,
        fontWeight: 700,
        marginRight: 6,
        flexShrink: 0,
      }}
    >
      {label}
    </span>
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

// ── Endpoints page ────────────────────────────────────────────────────────────
export default function Endpoints() {
  useHotkeys();

  const [activeFilter, setActiveFilter] = useState<string>('All');
  const [search, setSearch] = useState('');
  const [osFilter, setOsFilter] = useState('');
  const [groupFilter, setGroupFilter] = useState('');
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [showRegister, setShowRegister] = useState(false);

  const toggleRow = (hostname: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(hostname)) next.delete(hostname);
      else next.add(hostname);
      return next;
    });
  };

  const filtered = useMemo(() => {
    return ENDPOINTS.filter((ep) => {
      if (
        search &&
        !ep.hostname.toLowerCase().includes(search.toLowerCase()) &&
        !ep.os.toLowerCase().includes(search.toLowerCase())
      )
        return false;
      if (activeFilter !== 'All' && ep.status !== activeFilter.toLowerCase()) return false;
      if (osFilter && !ep.os.toLowerCase().includes(osFilter.toLowerCase())) return false;
      if (groupFilter && !ep.groups.includes(groupFilter)) return false;
      return true;
    });
  }, [search, activeFilter, osFilter, groupFilter]);

  const selectStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: 6,
    padding: '6px 12px',
    borderRadius: 8,
    border: '1px solid var(--color-separator)',
    background: 'var(--color-card)',
    cursor: 'pointer',
    fontSize: 12,
    color: 'var(--color-muted)',
    colorScheme: 'dark' as React.CSSProperties['colorScheme'],
    outline: 'none',
  };

  return (
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
          title="Endpoints"
          subtitle="247 managed endpoints across all groups"
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
                onClick={() => setShowRegister(true)}
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
                <Plus size={13} />
                Register Endpoint
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
          icon={<Monitor size={16} color="var(--color-success)" />}
          iconColor="var(--color-success)"
          value="189"
          valueColor="var(--color-success)"
          label="Online"
          trend={{ value: '+3', positive: true }}
          trendText="since yesterday"
        />
        <StatCard
          icon={<Monitor size={16} color="var(--color-danger)" />}
          iconColor="var(--color-danger)"
          value="34"
          valueColor="var(--color-danger)"
          label="Offline"
          trend={{ value: '+2', positive: false }}
          trendText="since yesterday"
        />
        <StatCard
          icon={<Monitor size={16} color="var(--color-warning)" />}
          iconColor="var(--color-warning)"
          value="18"
          valueColor="var(--color-warning)"
          label="Pending"
        />
        <StatCard
          icon={<Monitor size={16} color="var(--color-cyan)" />}
          iconColor="var(--color-cyan)"
          value="76%"
          valueColor="var(--color-cyan)"
          label="Avg Compliance"
          trend={{ value: '2.1%', positive: false }}
          trendText="vs last week"
        />
      </motion.div>

      {/* Row 3 — Filter pills + search */}
      <motion.div variants={fadeUp} style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        {/* Filter pills */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          {FILTER_PILLS.map((pill) => (
            <FilterPill
              key={pill.label}
              label={pill.label}
              count={pill.count}
              active={activeFilter === pill.label}
              onClick={() => setActiveFilter(pill.label)}
            />
          ))}
        </div>
        {/* Search + dropdowns */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <div
            style={{
              flex: 1,
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '6px 12px',
              borderRadius: 8,
              border: '1px solid var(--color-separator)',
              background: 'transparent',
            }}
          >
            <Search size={13} color="var(--color-muted)" />
            <input
              type="text"
              placeholder="Search endpoints…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              style={{
                flex: 1,
                background: 'transparent',
                border: 'none',
                outline: 'none',
                fontSize: 12,
                color: 'var(--color-foreground)',
              }}
            />
            <span
              style={{
                marginLeft: 'auto',
                fontSize: 10,
                fontWeight: 600,
                padding: '1px 5px',
                borderRadius: 4,
                border: '1px solid var(--color-separator)',
                color: 'var(--color-muted)',
                flexShrink: 0,
              }}
            >
              /
            </span>
          </div>
          <select
            value={osFilter}
            onChange={(e) => setOsFilter(e.target.value)}
            style={selectStyle}
          >
            <option value="">All OS</option>
            <option value="ubuntu">Ubuntu</option>
            <option value="rhel">RHEL</option>
            <option value="windows">Windows</option>
            <option value="macos">macOS</option>
            <option value="debian">Debian</option>
            <option value="fortios">FortiOS</option>
          </select>
          <select
            value={groupFilter}
            onChange={(e) => setGroupFilter(e.target.value)}
            style={selectStyle}
          >
            <option value="">All Groups</option>
            <option value="Production">Production</option>
            <option value="Database">Database</option>
            <option value="Kubernetes">Kubernetes</option>
            <option value="Network">Network</option>
            <option value="Infrastructure">Infrastructure</option>
            <option value="Web">Web</option>
            <option value="App Servers">App Servers</option>
            <option value="Dev">Dev</option>
            <option value="Critical">Critical</option>
            <option value="AD">AD</option>
          </select>
        </div>
      </motion.div>

      {/* Row 4 — Endpoint table */}
      <motion.div variants={fadeUp}>
        <GlassCard className="p-5" hover={false}>
          <SectionHeader
            title="All Endpoints"
            action={
              <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>
                Showing {filtered.length} of 247
              </span>
            }
          />

          {/* Table */}
          <div style={{ marginTop: 16, overflowX: 'auto' }}>
            {/* Header */}
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: TABLE_COLS.map((c) => c.width).join(' '),
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
            {filtered.map((ep, idx) => (
              <div key={ep.hostname}>
                {/* Main row */}
                <div
                  style={{
                    display: 'grid',
                    gridTemplateColumns: TABLE_COLS.map((c) => c.width).join(' '),
                    gap: 0,
                    padding: '10px 0',
                    borderBottom:
                      idx < filtered.length - 1 && !expandedRows.has(ep.hostname)
                        ? '1px solid var(--color-separator)'
                        : 'none',
                    alignItems: 'center',
                    cursor: 'pointer',
                    transition: 'background 0.12s ease',
                    borderRadius: 6,
                  }}
                  onClick={() => toggleRow(ep.hostname)}
                  onMouseEnter={(e) => {
                    (e.currentTarget as HTMLDivElement).style.background =
                      'color-mix(in srgb, var(--color-primary) 4%, transparent)';
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLDivElement).style.background = 'transparent';
                  }}
                >
                  {/* Hostname */}
                  <div style={{ paddingRight: 8 }}>
                    <span
                      style={{
                        fontSize: 12,
                        fontWeight: 600,
                        color: 'var(--color-foreground)',
                        fontFamily: 'monospace',
                      }}
                    >
                      {ep.hostname}
                    </span>
                  </div>

                  {/* OS */}
                  <div style={{ display: 'flex', alignItems: 'center', paddingRight: 8 }}>
                    <OsIcon os={ep.os} />
                    <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>{ep.os}</span>
                  </div>

                  {/* Status */}
                  <div style={{ paddingRight: 8 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <StatusDot status={ep.status} />
                      <span
                        style={{
                          fontSize: 11,
                          fontWeight: 500,
                          color:
                            ep.status === 'online'
                              ? 'var(--color-success)'
                              : ep.status === 'pending'
                                ? 'var(--color-warning)'
                                : 'var(--color-muted)',
                          textTransform: 'capitalize',
                        }}
                      >
                        {ep.status}
                      </span>
                    </div>
                  </div>

                  {/* Agent Ver */}
                  <div style={{ paddingRight: 8 }}>
                    <span
                      style={{
                        fontSize: 11,
                        fontFamily: 'monospace',
                        color: 'var(--color-muted)',
                      }}
                    >
                      v{ep.agent}
                    </span>
                  </div>

                  {/* Risk Score */}
                  <div style={{ paddingRight: 8 }}>
                    <RiskScore value={ep.riskScore} />
                  </div>

                  {/* Last Seen */}
                  <div style={{ paddingRight: 8 }}>
                    <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>{ep.lastSeen}</span>
                  </div>

                  {/* Patches */}
                  <div style={{ paddingRight: 8 }}>
                    <PatchChips critical={ep.critical} high={ep.high} medium={ep.medium} />
                  </div>

                  {/* Groups */}
                  <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', paddingRight: 8 }}>
                    {ep.groups.map((g) => (
                      <span
                        key={g}
                        style={{
                          fontSize: 10,
                          fontWeight: 500,
                          padding: '2px 7px',
                          borderRadius: 10,
                          background: 'var(--color-separator)',
                          color: 'var(--color-muted)',
                        }}
                      >
                        {g}
                      </span>
                    ))}
                  </div>

                  {/* Expand chevron */}
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        toggleRow(ep.hostname);
                      }}
                      style={{
                        background: 'none',
                        border: 'none',
                        cursor: 'pointer',
                        color: 'var(--color-muted)',
                        display: 'flex',
                        alignItems: 'center',
                        padding: 4,
                        borderRadius: 4,
                      }}
                    >
                      {expandedRows.has(ep.hostname) ? (
                        <ChevronDown size={14} />
                      ) : (
                        <ChevronRight size={14} />
                      )}
                    </button>
                  </div>
                </div>

                {/* Expanded panel */}
                {expandedRows.has(ep.hostname) && (
                  <div
                    style={{
                      padding: '12px 16px',
                      background: 'color-mix(in srgb, var(--color-primary) 3%, transparent)',
                      borderBottom:
                        idx < filtered.length - 1 ? '1px solid var(--color-separator)' : 'none',
                      borderTop: '1px solid var(--color-separator)',
                      display: 'grid',
                      gridTemplateColumns: '1fr 1fr 1fr',
                      gap: 16,
                    }}
                  >
                    {/* Left: System info */}
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
                        System
                      </div>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                        {(
                          [
                            ['OS', ep.os],
                            ['Agent', `v${ep.agent}`],
                            ['Compliance', `${ep.compliance}%`],
                          ] as [string, string][]
                        ).map(([k, v]) => (
                          <div
                            key={k}
                            style={{
                              display: 'flex',
                              justifyContent: 'space-between',
                              fontSize: 11,
                            }}
                          >
                            <span style={{ color: 'var(--color-muted)' }}>{k}</span>
                            <span style={{ color: 'var(--color-foreground)', fontWeight: 500 }}>
                              {v}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>

                    {/* Middle: Patch summary */}
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
                        Pending Patches
                      </div>
                      <div style={{ display: 'flex', gap: 8 }}>
                        {ep.critical > 0 && (
                          <div style={{ textAlign: 'center' }}>
                            <div
                              style={{
                                fontSize: 18,
                                fontWeight: 700,
                                color: 'var(--color-danger)',
                              }}
                            >
                              {ep.critical}
                            </div>
                            <div style={{ fontSize: 10, color: 'var(--color-muted)' }}>
                              Critical
                            </div>
                          </div>
                        )}
                        {ep.high > 0 && (
                          <div style={{ textAlign: 'center' }}>
                            <div
                              style={{
                                fontSize: 18,
                                fontWeight: 700,
                                color: 'var(--color-warning)',
                              }}
                            >
                              {ep.high}
                            </div>
                            <div style={{ fontSize: 10, color: 'var(--color-muted)' }}>High</div>
                          </div>
                        )}
                        {ep.medium > 0 && (
                          <div style={{ textAlign: 'center' }}>
                            <div
                              style={{ fontSize: 18, fontWeight: 700, color: 'var(--color-muted)' }}
                            >
                              {ep.medium}
                            </div>
                            <div style={{ fontSize: 10, color: 'var(--color-muted)' }}>Medium</div>
                          </div>
                        )}
                        {ep.critical === 0 && ep.high === 0 && ep.medium === 0 && (
                          <span style={{ fontSize: 12, color: 'var(--color-success)' }}>
                            ✓ Up to date
                          </span>
                        )}
                      </div>
                    </div>

                    {/* Right: Quick actions */}
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
                        Quick Actions
                      </div>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                        {[
                          { label: 'Scan Now', color: 'var(--color-primary)' },
                          { label: 'Deploy Patches', color: 'var(--color-foreground)' },
                          { label: 'Assign Group', color: 'var(--color-foreground)' },
                        ].map((btn) => (
                          <button
                            key={btn.label}
                            onClick={(e) => e.stopPropagation()}
                            style={{
                              padding: '5px 10px',
                              borderRadius: 6,
                              fontSize: 11,
                              fontWeight: 500,
                              border: '1px solid var(--color-separator)',
                              background: 'transparent',
                              color: btn.color,
                              cursor: 'pointer',
                              textAlign: 'left',
                            }}
                          >
                            {btn.label}
                          </button>
                        ))}
                      </div>
                    </div>
                  </div>
                )}
              </div>
            ))}

            {/* Pagination */}
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                paddingTop: 12,
                borderTop: '1px solid var(--color-separator)',
                marginTop: 4,
              }}
            >
              <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>
                Showing {filtered.length} of 247 endpoints
              </span>
              <div style={{ display: 'flex', gap: 4 }}>
                {(['←', '1', '2', '3', '...', '13', '→'] as string[]).map((p, i) => (
                  <button
                    key={i}
                    style={{
                      padding: '4px 8px',
                      borderRadius: 5,
                      fontSize: 11,
                      fontWeight: p === '1' ? 600 : 400,
                      border:
                        p === '1'
                          ? '1px solid var(--color-primary)'
                          : '1px solid var(--color-separator)',
                      background:
                        p === '1'
                          ? 'color-mix(in srgb, var(--color-primary) 12%, transparent)'
                          : 'transparent',
                      color: p === '1' ? 'var(--color-primary)' : 'var(--color-muted)',
                      cursor: p === '←' || p === '→' || p === '...' ? 'default' : 'pointer',
                    }}
                  >
                    {p}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </GlassCard>
      </motion.div>

      {/* Register Endpoint Modal */}
      {showRegister && (
        <div
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(0,0,0,0.6)',
            backdropFilter: 'blur(4px)',
            zIndex: 1000,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <div
            style={{
              background: 'var(--color-card)',
              border: '1px solid var(--color-separator)',
              borderRadius: 12,
              width: 480,
              padding: 24,
              boxShadow: '0 20px 60px rgba(0,0,0,0.4)',
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                marginBottom: 20,
              }}
            >
              <h2 style={{ fontSize: 16, fontWeight: 700, margin: 0 }}>Register New Endpoint</h2>
              <button
                onClick={() => setShowRegister(false)}
                style={{
                  background: 'none',
                  border: 'none',
                  color: 'var(--color-muted)',
                  cursor: 'pointer',
                  fontSize: 20,
                  lineHeight: 1,
                }}
              >
                ✕
              </button>
            </div>

            {[
              {
                id: 'reg-hostname',
                label: 'Hostname',
                placeholder: 'e.g. prod-web-01',
                required: true,
              },
              {
                id: 'reg-ip',
                label: 'IP Address',
                placeholder: 'e.g. 192.168.1.100',
                required: true,
              },
            ].map((field) => (
              <div key={field.id} style={{ marginBottom: 14 }}>
                <label
                  style={{
                    display: 'block',
                    fontSize: 12,
                    fontWeight: 600,
                    color: 'var(--color-muted)',
                    marginBottom: 6,
                  }}
                >
                  {field.label}{' '}
                  {field.required && <span style={{ color: 'var(--color-danger)' }}>*</span>}
                </label>
                <input
                  type="text"
                  placeholder={field.placeholder}
                  style={{
                    width: '100%',
                    padding: '8px 10px',
                    background: 'color-mix(in srgb, var(--color-foreground) 5%, transparent)',
                    border: '1px solid var(--color-separator)',
                    borderRadius: 6,
                    color: 'var(--color-foreground)',
                    fontSize: 12,
                    outline: 'none',
                    boxSizing: 'border-box',
                  }}
                />
              </div>
            ))}

            <div style={{ marginBottom: 14 }}>
              <label
                style={{
                  display: 'block',
                  fontSize: 12,
                  fontWeight: 600,
                  color: 'var(--color-muted)',
                  marginBottom: 6,
                }}
              >
                OS Type
              </label>
              <select
                style={{
                  width: '100%',
                  padding: '8px 10px',
                  background: 'var(--color-card)',
                  border: '1px solid var(--color-separator)',
                  borderRadius: 6,
                  color: 'var(--color-foreground)',
                  fontSize: 12,
                  outline: 'none',
                  colorScheme: 'dark' as React.CSSProperties['colorScheme'],
                }}
              >
                <option value="">Select OS...</option>
                <option>Ubuntu</option>
                <option>RHEL</option>
                <option>Windows Server</option>
                <option>macOS</option>
                <option>Debian</option>
                <option>CentOS</option>
                <option>Other</option>
              </select>
            </div>

            <div style={{ marginBottom: 14 }}>
              <label
                style={{
                  display: 'block',
                  fontSize: 12,
                  fontWeight: 600,
                  color: 'var(--color-muted)',
                  marginBottom: 6,
                }}
              >
                Agent Token
              </label>
              <input
                type="text"
                readOnly
                value="tok_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
                style={{
                  width: '100%',
                  padding: '8px 10px',
                  background: 'color-mix(in srgb, var(--color-muted) 10%, transparent)',
                  border: '1px solid var(--color-separator)',
                  borderRadius: 6,
                  color: 'var(--color-muted)',
                  fontSize: 11,
                  fontFamily: 'monospace',
                  outline: 'none',
                  boxSizing: 'border-box',
                  cursor: 'default',
                }}
              />
            </div>

            <div style={{ marginBottom: 20 }}>
              <label
                style={{
                  display: 'block',
                  fontSize: 12,
                  fontWeight: 600,
                  color: 'var(--color-muted)',
                  marginBottom: 6,
                }}
              >
                Group
              </label>
              <select
                style={{
                  width: '100%',
                  padding: '8px 10px',
                  background: 'var(--color-card)',
                  border: '1px solid var(--color-separator)',
                  borderRadius: 6,
                  color: 'var(--color-foreground)',
                  fontSize: 12,
                  outline: 'none',
                  colorScheme: 'dark' as React.CSSProperties['colorScheme'],
                }}
              >
                <option value="">Select Group...</option>
                <option>Production</option>
                <option>Staging</option>
                <option>Database</option>
                <option>Kubernetes</option>
                <option>CI/CD</option>
                <option>Development</option>
                <option>Network</option>
                <option>Infrastructure</option>
              </select>
            </div>

            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <button
                onClick={() => setShowRegister(false)}
                style={{
                  padding: '8px 16px',
                  borderRadius: 6,
                  border: '1px solid var(--color-separator)',
                  background: 'transparent',
                  color: 'var(--color-muted)',
                  cursor: 'pointer',
                  fontSize: 12,
                  fontWeight: 500,
                }}
              >
                Cancel
              </button>
              <button
                onClick={() => setShowRegister(false)}
                style={{
                  padding: '8px 16px',
                  borderRadius: 6,
                  border: 'none',
                  background: 'var(--color-primary)',
                  color: '#fff',
                  cursor: 'pointer',
                  fontSize: 12,
                  fontWeight: 600,
                }}
              >
                Register
              </button>
            </div>
          </div>
        </div>
      )}
    </motion.div>
  );
}
