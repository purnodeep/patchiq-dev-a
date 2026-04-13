import { useState } from 'react';
import { motion } from 'framer-motion';
import { AlertTriangle, Shield, Zap } from 'lucide-react';
import { useHotkeys } from '@/hooks/useHotkeys';
import { StatCard } from '@/components/shared/StatCard';
import { GlassCard } from '@/components/shared/GlassCard';
import { SectionHeader } from '@/components/shared/SectionHeader';
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

interface CveRow {
  id: string;
  cvss: number;
  severity: Severity;
  vector: string;
  exploit: boolean;
  kev: boolean;
  affected: number;
  patch: string;
  published: string;
}

const CVE_DATA: CveRow[] = [
  {
    id: 'CVE-2024-21762',
    cvss: 9.8,
    severity: 'critical',
    vector: 'Network',
    exploit: true,
    kev: true,
    affected: 15,
    patch: 'KB5034441',
    published: 'Jan 2024',
  },
  {
    id: 'CVE-2024-38094',
    cvss: 9.0,
    severity: 'critical',
    vector: 'Network',
    exploit: true,
    kev: true,
    affected: 4,
    patch: 'KB5036893',
    published: 'Mar 2024',
  },
  {
    id: 'CVE-2024-1086',
    cvss: 7.8,
    severity: 'high',
    vector: 'Local',
    exploit: true,
    kev: false,
    affected: 27,
    patch: 'USN-6648-1',
    published: 'Feb 2024',
  },
  {
    id: 'CVE-2024-21338',
    cvss: 7.8,
    severity: 'high',
    vector: 'Local',
    exploit: false,
    kev: false,
    affected: 136,
    patch: 'KB5035853',
    published: 'Feb 2024',
  },
  {
    id: 'CVE-2024-20666',
    cvss: 9.1,
    severity: 'critical',
    vector: 'Network',
    exploit: false,
    kev: false,
    affected: 89,
    patch: 'KB5034441',
    published: 'Jan 2024',
  },
  {
    id: 'CVE-2024-0727',
    cvss: 9.4,
    severity: 'critical',
    vector: 'Network',
    exploit: false,
    kev: false,
    affected: 18,
    patch: 'RHSA-2024:1891',
    published: 'Mar 2024',
  },
  {
    id: 'CVE-2024-21412',
    cvss: 8.1,
    severity: 'high',
    vector: 'Network',
    exploit: true,
    kev: false,
    affected: 247,
    patch: 'KB5035853',
    published: 'Feb 2024',
  },
  {
    id: 'CVE-2024-49113',
    cvss: 7.5,
    severity: 'high',
    vector: 'Network',
    exploit: false,
    kev: false,
    affected: 12,
    patch: 'KB5035853',
    published: 'Dec 2024',
  },
  {
    id: 'CVE-2024-0553',
    cvss: 5.9,
    severity: 'medium',
    vector: 'Network',
    exploit: false,
    kev: false,
    affected: 45,
    patch: 'USN-6587-1',
    published: 'Jan 2024',
  },
  {
    id: 'CVE-2024-3094',
    cvss: 10.0,
    severity: 'critical',
    vector: 'Network',
    exploit: true,
    kev: true,
    affected: 18,
    patch: 'USN-6661-1',
    published: 'Mar 2024',
  },
  {
    id: 'CVE-2024-21386',
    cvss: 6.5,
    severity: 'medium',
    vector: 'Network',
    exploit: false,
    kev: false,
    affected: 22,
    patch: 'KB5034122',
    published: 'Jan 2024',
  },
  {
    id: 'CVE-2024-30103',
    cvss: 8.8,
    severity: 'high',
    vector: 'Network',
    exploit: true,
    kev: false,
    affected: 3,
    patch: 'KB5037849',
    published: 'Apr 2024',
  },
];

// ── Helpers ───────────────────────────────────────────────────────────────────
function cvssColor(cvss: number): string {
  if (cvss >= 9.0) return 'var(--color-danger)';
  if (cvss >= 7.0) return 'var(--color-warning)';
  if (cvss >= 4.0) return 'var(--color-caution, #f59e0b)';
  return 'var(--color-success)';
}

function severityColor(severity: Severity): string {
  switch (severity) {
    case 'critical':
      return 'var(--color-danger)';
    case 'high':
      return 'var(--color-warning)';
    case 'medium':
      return 'var(--color-caution, #f59e0b)';
    case 'low':
      return 'var(--color-success)';
  }
}

// ── Sub-components ────────────────────────────────────────────────────────────
function CvssCell({ cvss }: { cvss: number }) {
  const color = cvssColor(cvss);
  const barWidth = `${(cvss / 10) * 100}%`;
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
      <span style={{ fontSize: 12, fontWeight: 700, color, minWidth: 28 }}>{cvss.toFixed(1)}</span>
      <div
        style={{
          width: 48,
          height: 4,
          borderRadius: 2,
          background: 'var(--color-separator)',
          overflow: 'hidden',
          flexShrink: 0,
        }}
      >
        <div
          style={{
            height: '100%',
            width: barWidth,
            borderRadius: 2,
            background: color,
          }}
        />
      </div>
    </div>
  );
}

function SeverityBadge({ severity }: { severity: Severity }) {
  const color = severityColor(severity);
  return (
    <span
      style={{
        display: 'inline-block',
        padding: '2px 7px',
        borderRadius: 4,
        fontSize: 10,
        fontWeight: 700,
        letterSpacing: '0.04em',
        textTransform: 'uppercase',
        color,
        background: `color-mix(in srgb, ${color} 12%, transparent)`,
        border: `1px solid color-mix(in srgb, ${color} 30%, transparent)`,
      }}
    >
      {severity}
    </span>
  );
}

type FilterKey = 'all' | 'critical' | 'high' | 'medium' | 'low';

interface FilterPill {
  key: FilterKey;
  label: string;
  count: number;
}

const FILTER_PILLS: FilterPill[] = [
  { key: 'all', label: 'All', count: 324 },
  { key: 'critical', label: 'Critical', count: 42 },
  { key: 'high', label: 'High', count: 89 },
  { key: 'medium', label: 'Medium', count: 121 },
  { key: 'low', label: 'Low', count: 72 },
];

// ── Table ─────────────────────────────────────────────────────────────────────
const TH_STYLE: React.CSSProperties = {
  padding: '8px 10px',
  textAlign: 'left',
  fontSize: 10,
  fontWeight: 700,
  letterSpacing: '0.06em',
  textTransform: 'uppercase',
  color: 'var(--color-muted)',
  borderBottom: '1px solid var(--color-separator)',
  whiteSpace: 'nowrap',
};

const TD_STYLE: React.CSSProperties = {
  padding: '10px 10px',
  fontSize: 12,
  borderBottom: '1px solid color-mix(in srgb, var(--color-separator) 50%, transparent)',
  verticalAlign: 'middle',
};

function CveTable({ rows }: { rows: CveRow[] }) {
  return (
    <div style={{ overflowX: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            <th style={TH_STYLE}>CVE ID</th>
            <th style={TH_STYLE}>CVSS</th>
            <th style={TH_STYLE}>Severity</th>
            <th style={TH_STYLE}>Attack Vector</th>
            <th style={TH_STYLE}>Exploit</th>
            <th style={TH_STYLE}>KEV</th>
            <th style={{ ...TH_STYLE, textAlign: 'right' }}>Affected</th>
            <th style={TH_STYLE}>Patch</th>
            <th style={TH_STYLE}>Published</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr
              key={row.id}
              style={{
                background:
                  i % 2 === 0
                    ? 'transparent'
                    : 'color-mix(in srgb, var(--color-separator) 20%, transparent)',
                transition: 'background 0.15s',
              }}
            >
              <td style={TD_STYLE}>
                <span
                  style={{
                    fontFamily: 'monospace',
                    fontSize: 11,
                    fontWeight: 600,
                    color: 'var(--color-foreground)',
                  }}
                >
                  {row.id}
                </span>
              </td>
              <td style={TD_STYLE}>
                <CvssCell cvss={row.cvss} />
              </td>
              <td style={TD_STYLE}>
                <SeverityBadge severity={row.severity} />
              </td>
              <td style={{ ...TD_STYLE, color: 'var(--color-muted)' }}>{row.vector}</td>
              <td style={TD_STYLE}>
                {row.exploit ? (
                  <span style={{ fontSize: 11, fontWeight: 700, color: 'var(--color-danger)' }}>
                    YES
                  </span>
                ) : (
                  <span style={{ fontSize: 12, color: 'var(--color-muted)' }}>—</span>
                )}
              </td>
              <td style={TD_STYLE}>
                {row.kev ? (
                  <span
                    style={{
                      display: 'inline-block',
                      padding: '2px 6px',
                      borderRadius: 4,
                      fontSize: 9,
                      fontWeight: 700,
                      letterSpacing: '0.06em',
                      textTransform: 'uppercase',
                      color: 'var(--color-purple)',
                      background: 'color-mix(in srgb, var(--color-purple) 12%, transparent)',
                      border: '1px solid color-mix(in srgb, var(--color-purple) 30%, transparent)',
                    }}
                  >
                    KEV
                  </span>
                ) : (
                  <span style={{ fontSize: 12, color: 'var(--color-muted)' }}>—</span>
                )}
              </td>
              <td style={{ ...TD_STYLE, textAlign: 'right', fontWeight: 600 }}>{row.affected}</td>
              <td style={TD_STYLE}>
                <span
                  style={{
                    fontFamily: 'monospace',
                    fontSize: 10,
                    color: 'var(--color-muted)',
                  }}
                >
                  {row.patch}
                </span>
              </td>
              <td style={{ ...TD_STYLE, color: 'var(--color-muted)', whiteSpace: 'nowrap' }}>
                {row.published}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────
export default function CVEs() {
  useHotkeys();

  const [activeFilter, setActiveFilter] = useState<FilterKey>('all');
  const [exploitOnly, setExploitOnly] = useState(false);
  const [kevOnly, setKevOnly] = useState(false);

  const filteredRows = CVE_DATA.filter((row) => {
    if (activeFilter !== 'all' && row.severity !== activeFilter) return false;
    if (exploitOnly && !row.exploit) return false;
    if (kevOnly && !row.kev) return false;
    return true;
  });

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
      {/* Row 1 — Page title */}
      <motion.div variants={fadeUp}>
        <PageHeader
          title="CVE Intelligence"
          subtitle="324 vulnerabilities tracked · Last synced 4 min ago"
          actions={
            <button
              style={{
                padding: '6px 12px',
                borderRadius: 6,
                border: 'none',
                background: 'var(--color-primary)',
                color: '#fff',
                fontSize: 12,
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              Export Report
            </button>
          }
        />
      </motion.div>

      {/* Row 2 — KPI Stat Cards (4×1) */}
      <motion.div
        variants={fadeUp}
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gap: 12,
        }}
      >
        <StatCard
          icon={<AlertTriangle size={16} />}
          iconColor="var(--color-danger)"
          value={42}
          valueColor="var(--color-danger)"
          label="Critical CVEs"
          trend={{ value: '8 new', positive: false }}
          trendText="this week"
        />
        <StatCard
          icon={<AlertTriangle size={16} />}
          iconColor="var(--color-warning)"
          value={89}
          valueColor="var(--color-warning)"
          label="High CVEs"
          trend={{ value: '12 new', positive: false }}
          trendText="this week"
        />
        <StatCard
          icon={<Shield size={16} />}
          iconColor="var(--color-cyan)"
          value={121}
          label="Medium CVEs"
          trendText="no change"
        />
        <StatCard
          icon={<Zap size={16} />}
          iconColor="var(--color-purple)"
          value={28}
          valueColor="var(--color-purple)"
          label="KEV Listed"
          trendText="CISA Known Exploited"
        />
      </motion.div>

      {/* Row 3 — Filter pills + table */}
      <motion.div variants={fadeUp}>
        <GlassCard className="p-5" hover={false}>
          <SectionHeader
            title="Vulnerability Catalog"
            action={
              <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>
                {filteredRows.length} results
              </span>
            }
          />

          {/* Filter pills */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              marginTop: 14,
              flexWrap: 'wrap',
            }}
          >
            {FILTER_PILLS.map((pill) => {
              const active = activeFilter === pill.key;
              return (
                <button
                  key={pill.key}
                  onClick={() => setActiveFilter(pill.key)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 5,
                    padding: '4px 10px',
                    borderRadius: 20,
                    border: active
                      ? '1px solid var(--color-primary)'
                      : '1px solid var(--color-separator)',
                    background: active
                      ? 'color-mix(in srgb, var(--color-primary) 15%, transparent)'
                      : 'transparent',
                    color: active ? 'var(--color-primary)' : 'var(--color-muted)',
                    fontSize: 11,
                    fontWeight: active ? 700 : 500,
                    cursor: 'pointer',
                    transition: 'all 0.15s',
                  }}
                >
                  {pill.label}
                  <span
                    style={{
                      fontSize: 10,
                      opacity: 0.7,
                    }}
                  >
                    {pill.count}
                  </span>
                </button>
              );
            })}

            {/* Divider */}
            <div
              style={{
                width: 1,
                height: 18,
                background: 'var(--color-separator)',
                marginLeft: 4,
                marginRight: 4,
              }}
            />

            {/* Toggle pills */}
            {[
              {
                label: 'Exploit Available',
                active: exploitOnly,
                toggle: () => setExploitOnly((v) => !v),
              },
              { label: 'KEV Only', active: kevOnly, toggle: () => setKevOnly((v) => !v) },
            ].map(({ label, active, toggle }) => (
              <button
                key={label}
                onClick={toggle}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '4px 10px',
                  borderRadius: 20,
                  border: active
                    ? '1px solid var(--color-purple)'
                    : '1px solid var(--color-separator)',
                  background: active
                    ? 'color-mix(in srgb, var(--color-purple) 12%, transparent)'
                    : 'transparent',
                  color: active ? 'var(--color-purple)' : 'var(--color-muted)',
                  fontSize: 11,
                  fontWeight: active ? 700 : 500,
                  cursor: 'pointer',
                  transition: 'all 0.15s',
                }}
              >
                {label}
              </button>
            ))}
          </div>

          {/* Table */}
          <div style={{ marginTop: 16 }}>
            <CveTable rows={filteredRows} />
          </div>
        </GlassCard>
      </motion.div>
    </motion.div>
  );
}
