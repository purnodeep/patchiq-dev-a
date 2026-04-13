import React, { useState } from 'react';
import { Skeleton, RingGauge } from '@patchiq/ui';
import { RefreshCw } from 'lucide-react';
import { useEndpoint } from '../../../api/hooks/useEndpoints';
import { ProgressBar } from '../../../components/ProgressBar';
import { timeAgo, formatUptime } from '../../../lib/time';
import type { HardwareInfo, NetworkInfo } from '../../../types/hardware';

interface HardwareTabProps {
  endpointId: string;
}

// ── design tokens ──────────────────────────────────────────────
const S = {
  card: {
    background: 'var(--bg-card)',
    border: '1px solid var(--border)',
    borderRadius: 8,
    boxShadow: 'var(--shadow-sm)',
    overflow: 'hidden' as const,
  },
  cardTitle: {
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 600,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.06em',
    color: 'var(--text-muted)',
    padding: '12px 16px 8px',
    borderBottom: '1px solid var(--border)',
    background: 'var(--bg-inset)',
  },
  cardBody: { padding: '16px' },
  th: {
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 600,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.06em',
    color: 'var(--text-muted)',
    padding: '9px 12px',
    background: 'var(--bg-inset)',
    borderBottom: '1px solid var(--border)',
    textAlign: 'left' as const,
  },
  td: {
    padding: '10px 12px',
    borderBottom: '1px solid var(--border)',
    color: 'var(--text-primary)',
    fontSize: 13,
  },
  mono: { fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-primary)' },
  label: { fontSize: 11, color: 'var(--text-muted)', marginBottom: 2 },
  value: { fontFamily: 'var(--font-mono)', fontSize: 13, color: 'var(--text-primary)' },
};

const KEY_CPU_FLAGS = new Set([
  'avx',
  'avx2',
  'avx512f',
  'avx-512',
  'aes',
  'aes-ni',
  'sse4_1',
  'sse4_2',
  'sha_ni',
  'rdrand',
  'vmx',
  'svm',
  'vt-x',
  'amd-v',
  'nx',
  'smep',
  'smap',
  'sgx',
  'pku',
  'bmi1',
  'bmi2',
  'fma',
  'f16c',
  'popcnt',
  'rdtscp',
]);

// ── helpers ────────────────────────────────────────────────────
function formatDate(dateStr: string | null | undefined): string {
  if (!dateStr) return '\u2014';
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return '\u2014';
  return d.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' });
}

function certExpiryWarning(dateStr: string | null | undefined): boolean {
  if (!dateStr) return false;
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return false;
  return (d.getTime() - Date.now()) / (1000 * 60 * 60 * 24) < 30;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

function formatMHz(mhz: number | null | undefined): string {
  if (mhz == null || mhz <= 0) return '—';
  if (mhz >= 1000) return `${(mhz / 1000).toFixed(2)} GHz`;
  return `${mhz} MHz`;
}

// ── ring gauge ────────────────────────────────────────────────
function HardwareRingGauge({
  pct,
  label,
  sub,
  size = 80,
}: {
  pct: number;
  label: string;
  sub: string;
  color?: string; // kept for API compatibility, colorByValue used instead
  size?: number;
}) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6 }}>
      <RingGauge value={pct} size={size} strokeWidth={6} colorByValue label={label} />
      <div
        style={{
          fontSize: 10,
          color: 'var(--text-muted)',
          fontFamily: 'var(--font-mono)',
          textAlign: 'center',
        }}
      >
        {sub}
      </div>
    </div>
  );
}

// ── cpu capabilities (collapsible) ────────────────────────────
function CpuCapabilities({ flags }: { flags: string[] }) {
  const [expanded, setExpanded] = useState(false);
  const keyFlags = flags.filter((f) => KEY_CPU_FLAGS.has(f.toLowerCase()));
  const otherFlags = flags.filter((f) => !KEY_CPU_FLAGS.has(f.toLowerCase()));
  const badge = (flag: string, isKey: boolean) => (
    <span
      key={flag}
      style={{
        display: 'inline-block',
        padding: '2px 6px',
        borderRadius: 4,
        border: isKey
          ? '1px solid color-mix(in srgb, var(--accent) 30%, transparent)'
          : '1px solid var(--border)',
        background: isKey
          ? 'color-mix(in srgb, var(--accent) 15%, transparent)'
          : 'var(--bg-inset)',
        fontSize: 10,
        fontFamily: 'var(--font-mono)',
        color: isKey ? 'var(--accent)' : 'var(--text-secondary)',
        whiteSpace: 'nowrap',
      }}
    >
      {flag}
    </span>
  );

  return (
    <div style={{ marginTop: 12, borderTop: '1px solid var(--border)', paddingTop: 10 }}>
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          fontWeight: 600,
          color: 'var(--text-muted)',
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          padding: '0 0 8px',
        }}
      >
        <span
          style={{
            display: 'inline-block',
            transition: 'transform 0.15s',
            transform: expanded ? 'rotate(90deg)' : 'rotate(0deg)',
            fontSize: 8,
          }}
        >
          {'\u25B6'}
        </span>
        Capabilities ({keyFlags.length} key · {flags.length} total)
      </button>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
        {expanded
          ? flags.map((f) => badge(f, KEY_CPU_FLAGS.has(f.toLowerCase())))
          : keyFlags.map((f) => badge(f, true))}
        {!expanded && otherFlags.length > 0 && (
          <button
            type="button"
            onClick={() => setExpanded(true)}
            style={{
              display: 'inline-block',
              padding: '2px 8px',
              borderRadius: 4,
              border: '1px dashed var(--border)',
              background: 'transparent',
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-faint)',
              cursor: 'pointer',
              whiteSpace: 'nowrap',
            }}
          >
            +{otherFlags.length} more
          </button>
        )}
      </div>
    </div>
  );
}

// ── memory slot details (collapsible) ─────────────────────────
function MemorySlotDetails({
  memory,
}: {
  memory?: import('../../../types/hardware').MemoryInfo | null;
}) {
  const [expanded, setExpanded] = useState(false);
  if (!memory) return null;
  const hasSlots = memory.num_slots != null && memory.num_slots > 0;
  const hasDimms = memory.dimms && memory.dimms.length > 0;
  if (!hasSlots && !hasDimms) return null;

  const populatedCount = memory.dimms?.filter((d) => d.size_mb > 0).length ?? 0;

  return (
    <div style={{ marginTop: 12, borderTop: '1px solid var(--border)', paddingTop: 10 }}>
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          fontWeight: 600,
          color: 'var(--text-muted)',
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          padding: '0 0 8px',
        }}
      >
        <span
          style={{
            display: 'inline-block',
            transition: 'transform 0.15s',
            transform: expanded ? 'rotate(90deg)' : 'rotate(0deg)',
            fontSize: 8,
          }}
        >
          {'\u25B6'}
        </span>
        Slot Layout ({populatedCount}/{memory.num_slots ?? 0} populated)
      </button>
      {expanded && (
        <>
          {/* Visual slots */}
          {hasSlots && (
            <div style={{ display: 'flex', gap: 8, alignItems: 'flex-end', marginBottom: 12 }}>
              {Array.from({ length: memory.num_slots! }, (_, i) => {
                const dimm = memory.dimms?.[i];
                const filled = dimm && dimm.size_mb > 0;
                const slotName = dimm?.locator || `Slot ${i + 1}`;
                const shortName = slotName
                  .replace('Controller', 'C')
                  .replace('DIMM', 'D')
                  .replace(/-/g, '');
                return (
                  <div
                    key={i}
                    style={{
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      gap: 3,
                    }}
                  >
                    <div
                      title={
                        filled
                          ? `${slotName}: ${(dimm.size_mb / 1024).toFixed(0)}GB ${dimm.type} ${dimm.speed_mhz}MHz`
                          : `${slotName}: Empty`
                      }
                      style={{
                        width: 56,
                        height: 72,
                        borderRadius: 4,
                        border: filled
                          ? '2px solid var(--accent)'
                          : '2px dashed color-mix(in srgb, var(--border) 60%, transparent)',
                        background: filled
                          ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
                          : 'transparent',
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: 'center',
                        justifyContent: 'center',
                        gap: 2,
                      }}
                    >
                      {filled ? (
                        <>
                          <span
                            style={{
                              fontSize: 15,
                              fontFamily: 'var(--font-mono)',
                              color: 'var(--accent)',
                              fontWeight: 700,
                            }}
                          >
                            {(dimm.size_mb / 1024).toFixed(0)}G
                          </span>
                          <span
                            style={{
                              fontSize: 10,
                              fontFamily: 'var(--font-mono)',
                              color: 'var(--text-faint)',
                            }}
                          >
                            {dimm.type || ''}
                          </span>
                          <span
                            style={{
                              fontSize: 8,
                              fontFamily: 'var(--font-mono)',
                              color: 'var(--text-faint)',
                            }}
                          >
                            {dimm.speed_mhz ? String(dimm.speed_mhz) : ''}
                          </span>
                        </>
                      ) : (
                        <span style={{ fontSize: 14, color: 'var(--text-faint)' }}>—</span>
                      )}
                    </div>
                    <span
                      style={{
                        fontSize: 9,
                        fontFamily: 'var(--font-mono)',
                        color: 'var(--text-faint)',
                        textAlign: 'center',
                        maxWidth: 56,
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {shortName}
                    </span>
                  </div>
                );
              })}
            </div>
          )}
          {/* DIMM table */}
          {hasDimms && (
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  {['Slot', 'Size', 'Type', 'Speed', 'Mfr'].map((h) => (
                    <th key={h} style={{ ...S.th, padding: '6px 8px', fontSize: 9 }}>
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {memory.dimms!.map((dimm, i) => (
                  <tr key={dimm.locator || i}>
                    <td style={{ ...S.td, padding: '6px 8px', fontSize: 11 }}>
                      {dimm.locator || `Slot ${i}`}
                    </td>
                    <td style={{ ...S.td, padding: '6px 8px', fontSize: 11 }}>
                      {dimm.size_mb > 0 ? `${dimm.size_mb} MB` : 'Empty'}
                    </td>
                    <td style={{ ...S.td, padding: '6px 8px', fontSize: 11 }}>
                      {dimm.type || '—'}
                    </td>
                    <td style={{ ...S.td, padding: '6px 8px', fontSize: 11 }}>
                      {dimm.speed_mhz > 0 ? `${dimm.speed_mhz} MHz` : '—'}
                    </td>
                    <td style={{ ...S.td, padding: '6px 8px', fontSize: 11, borderRight: 'none' }}>
                      {dimm.manufacturer || '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </>
      )}
    </div>
  );
}

// ── kv row ────────────────────────────────────────────────────
function KVRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '7px 0',
        borderBottom: '1px solid var(--border)',
      }}
    >
      <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{label}</span>
      <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-primary)' }}>
        {value}
      </span>
    </div>
  );
}

function hasHardwareDetails(hw: HardwareInfo | null | undefined): hw is HardwareInfo {
  if (!hw) return false;
  return !!(hw.cpu?.model_name || hw.memory?.total_bytes || (hw.storage && hw.storage.length > 0));
}

// ── network interfaces ───────────────────────────────────────
const VIRTUAL_PREFIXES = ['docker', 'br-', 'veth', 'virbr', 'lo', 'vnet', 'tun', 'tap'];
const isVirtualInterface = (name: string) => VIRTUAL_PREFIXES.some((p) => name.startsWith(p));

function NetworkInterfaceCard({ iface }: { iface: NetworkInfo }) {
  return (
    <div style={{ ...S.card, padding: 16 }}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 12,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: iface.state === 'up' ? 'var(--signal-healthy)' : 'var(--text-faint)',
            }}
          />
          <span
            style={{
              fontSize: 14,
              fontWeight: 600,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-primary)',
            }}
          >
            {iface.name}
          </span>
        </div>
        {iface.type && (
          <span
            style={{
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-muted)',
              padding: '2px 6px',
              borderRadius: 4,
              border: '1px solid var(--border)',
              background: 'var(--bg-inset)',
            }}
          >
            {iface.type}
          </span>
        )}
      </div>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))',
          gap: 8,
          fontSize: 12,
          fontFamily: 'var(--font-mono)',
        }}
      >
        <div>
          <div
            style={{
              color: 'var(--text-muted)',
              fontSize: 10,
              textTransform: 'uppercase',
              marginBottom: 2,
            }}
          >
            MAC
          </div>
          <div style={{ color: 'var(--text-secondary)' }}>{iface.mac_address || '\u2014'}</div>
        </div>
        <div>
          <div
            style={{
              color: 'var(--text-muted)',
              fontSize: 10,
              textTransform: 'uppercase',
              marginBottom: 2,
            }}
          >
            Speed
          </div>
          <div style={{ color: 'var(--text-secondary)' }}>
            {iface.speed_mbps ? `${iface.speed_mbps} Mbps` : '\u2014'}
          </div>
        </div>
        <div>
          <div
            style={{
              color: 'var(--text-muted)',
              fontSize: 10,
              textTransform: 'uppercase',
              marginBottom: 2,
            }}
          >
            MTU
          </div>
          <div style={{ color: 'var(--text-secondary)' }}>{iface.mtu || '\u2014'}</div>
        </div>
        <div>
          <div
            style={{
              color: 'var(--text-muted)',
              fontSize: 10,
              textTransform: 'uppercase',
              marginBottom: 2,
            }}
          >
            Driver
          </div>
          <div style={{ color: 'var(--text-secondary)' }}>{iface.driver || '\u2014'}</div>
        </div>
      </div>
      {iface.ipv4_addresses?.length > 0 && (
        <div style={{ marginTop: 8 }}>
          <div
            style={{
              color: 'var(--text-muted)',
              fontSize: 10,
              textTransform: 'uppercase',
              fontFamily: 'var(--font-mono)',
              marginBottom: 2,
            }}
          >
            IPv4
          </div>
          {iface.ipv4_addresses.map((ip, j) => (
            <div
              key={j}
              style={{
                fontSize: 12,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-primary)',
              }}
            >
              {ip.address}/{ip.prefix_len}
            </div>
          ))}
        </div>
      )}
      {iface.ipv6_addresses?.length > 0 && (
        <div style={{ marginTop: 8 }}>
          <div
            style={{
              color: 'var(--text-muted)',
              fontSize: 10,
              textTransform: 'uppercase',
              fontFamily: 'var(--font-mono)',
              marginBottom: 2,
            }}
          >
            IPv6
          </div>
          {iface.ipv6_addresses.map((ip, j) => (
            <div
              key={j}
              style={{
                fontSize: 12,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-secondary)',
                wordBreak: 'break-all',
              }}
            >
              {ip.address}/{ip.prefix_len}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function VirtualInterfacesTable({ interfaces }: { interfaces: NetworkInfo[] }) {
  const [expanded, setExpanded] = useState(false);
  if (interfaces.length === 0) return null;

  return (
    <div style={{ marginTop: 12 }}>
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          fontFamily: 'var(--font-mono)',
          fontSize: 11,
          fontWeight: 600,
          color: 'var(--text-muted)',
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          padding: '4px 0',
        }}
      >
        <span
          style={{
            display: 'inline-block',
            transition: 'transform 0.15s',
            transform: expanded ? 'rotate(90deg)' : 'rotate(0deg)',
          }}
        >
          {'\u25B6'}
        </span>
        Virtual Interfaces ({interfaces.length})
      </button>
      {expanded && (
        <div style={{ overflowX: 'auto', marginTop: 4 }}>
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                {['Interface', 'Type', 'MAC Address', 'IPv4', 'IPv6', 'State'].map((h) => (
                  <th key={h} style={S.th}>
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {interfaces.map((iface) => (
                <tr
                  key={iface.name}
                  style={{ cursor: 'default' }}
                  onMouseEnter={(e) => {
                    (e.currentTarget as HTMLTableRowElement).style.background =
                      'var(--bg-card-hover)';
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLTableRowElement).style.background = '';
                  }}
                >
                  <td style={{ ...S.td, fontFamily: 'var(--font-mono)', fontSize: 12 }}>
                    {iface.name}
                  </td>
                  <td style={S.td}>{iface.type || '\u2014'}</td>
                  <td style={{ ...S.td, fontFamily: 'var(--font-mono)', fontSize: 11 }}>
                    {iface.mac_address || '\u2014'}
                  </td>
                  <td style={{ ...S.td, fontFamily: 'var(--font-mono)', fontSize: 11 }}>
                    {iface.ipv4_addresses?.length > 0
                      ? iface.ipv4_addresses
                          .map((ip) => `${ip.address}/${ip.prefix_len}`)
                          .join(', ')
                      : '\u2014'}
                  </td>
                  <td style={{ ...S.td, fontFamily: 'var(--font-mono)', fontSize: 11 }}>
                    {iface.ipv6_addresses?.length > 0
                      ? iface.ipv6_addresses
                          .map((ip) => `${ip.address}/${ip.prefix_len}`)
                          .join(', ')
                      : '\u2014'}
                  </td>
                  <td style={S.td}>
                    <span
                      style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: 5,
                        fontSize: 12,
                        color:
                          iface.state === 'up' ? 'var(--signal-healthy)' : 'var(--signal-critical)',
                      }}
                    >
                      <span
                        style={{
                          width: 6,
                          height: 6,
                          borderRadius: '50%',
                          background:
                            iface.state === 'up'
                              ? 'var(--signal-healthy)'
                              : 'var(--signal-critical)',
                          flexShrink: 0,
                        }}
                      />
                      {iface.state || 'unknown'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function NetworkInterfacesSection({ interfaces }: { interfaces: NetworkInfo[] }) {
  const physical = interfaces.filter((i) => !isVirtualInterface(i.name));
  const virtual = interfaces.filter((i) => isVirtualInterface(i.name));

  return (
    <div style={S.card}>
      <div style={S.cardTitle}>Network Interfaces</div>
      <div style={S.cardBody}>
        {physical.length > 0 && (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))',
              gap: 12,
            }}
          >
            {physical.map((iface) => (
              <NetworkInterfaceCard key={iface.name} iface={iface} />
            ))}
          </div>
        )}
        <VirtualInterfacesTable interfaces={virtual} />
      </div>
    </div>
  );
}

// ── rich view ─────────────────────────────────────────────────
function RichHardwareView({ hw, endpoint }: { hw: HardwareInfo; endpoint: EndpointFields }) {
  const memTotalGb = (hw.memory?.total_bytes ?? 0) / (1024 * 1024 * 1024);
  const memAvailGb = (hw.memory?.available_bytes ?? 0) / (1024 * 1024 * 1024);
  const memUsedGb = memTotalGb - memAvailGb;
  const memPct = memTotalGb > 0 ? (memUsedGb / memTotalGb) * 100 : 0;

  const diskTotal = hw.storage?.reduce((a, d) => a + (d.size_bytes ?? 0), 0) ?? 0;
  const diskUsed =
    hw.storage?.reduce(
      (a, d) =>
        a +
        (d.partitions?.reduce(
          (pa, p) => pa + (p.size_bytes ?? 0) * ((p.usage_pct ?? 0) / 100),
          0,
        ) ?? 0),
      0,
    ) ?? 0;
  const diskPct = diskTotal > 0 ? (diskUsed / diskTotal) * 100 : 0;

  const cpuPct: number = endpoint.cpu_usage_percent ?? 0;

  const gaugeColor = (pct: number) => {
    if (pct >= 90) return 'var(--signal-critical)';
    if (pct >= 70) return 'var(--signal-warning)';
    return 'var(--signal-healthy)';
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Hero row: resource gauges + system info side by side */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))',
          gap: 16,
        }}
      >
        {/* System Resources */}
        <div style={{ ...S.card }}>
          <div style={S.cardTitle}>System Resources</div>
          <div
            style={{
              ...S.cardBody,
              display: 'flex',
              gap: 24,
              flexWrap: 'wrap' as const,
              alignItems: 'center',
            }}
          >
            <HardwareRingGauge
              pct={cpuPct}
              label="CPU"
              sub={`${hw.cpu?.total_logical_cpus ?? '?'} cores`}
              color={gaugeColor(cpuPct)}
            />
            <HardwareRingGauge
              pct={memPct}
              label="Memory"
              sub={`${memUsedGb.toFixed(1)}/${memTotalGb.toFixed(1)} GB`}
              color={gaugeColor(memPct)}
            />
            <HardwareRingGauge
              pct={diskPct}
              label="Disk"
              sub={diskTotal > 0 ? `${formatBytes(diskUsed)}/${formatBytes(diskTotal)}` : '—'}
              color={gaugeColor(diskPct)}
            />
            {/* GPU — shows utilization ring when available, dash otherwise */}
            {hw.gpu &&
              hw.gpu.length > 0 &&
              (() => {
                const gpu = hw.gpu[0];
                const vramText =
                  gpu.vram_mb >= 1024
                    ? `${(gpu.vram_mb / 1024).toFixed(0)} GB`
                    : `${gpu.vram_mb} MB`;
                const usagePct = gpu.usage_pct ?? 0;
                if (gpu.usage_pct !== undefined) {
                  return (
                    <HardwareRingGauge
                      pct={usagePct}
                      label="GPU"
                      sub={`${vramText} VRAM`}
                      color={gaugeColor(usagePct)}
                    />
                  );
                }
                const size = 80;
                const sw = 6;
                const r = (size - sw) / 2;
                const c = size / 2;
                return (
                  <div
                    title="GPU utilization monitoring not available — showing 0% as default"
                    style={{
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      gap: 6,
                    }}
                  >
                    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
                      <circle
                        cx={c}
                        cy={c}
                        r={r}
                        fill="none"
                        stroke="var(--border)"
                        strokeWidth={sw}
                        opacity={0.4}
                      />
                      {/* Center text: 0% since GPU utilization monitoring not yet available */}
                      <text
                        x={c}
                        y={c}
                        textAnchor="middle"
                        dominantBaseline="central"
                        fill="var(--signal-healthy)"
                        fontFamily="var(--font-sans)"
                        fontSize={size * 0.18}
                        fontWeight={600}
                      >
                        0%
                      </text>
                    </svg>
                    <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>GPU</span>
                    <div
                      style={{
                        fontSize: 10,
                        color: 'var(--text-muted)',
                        fontFamily: 'var(--font-mono)',
                        textAlign: 'center',
                      }}
                    >
                      {vramText} VRAM
                    </div>
                  </div>
                );
              })()}
            <div
              style={{
                marginLeft: 'auto',
                display: 'flex',
                flexDirection: 'column' as const,
                gap: 6,
              }}
            >
              <div>
                <span style={{ fontSize: 10, color: 'var(--text-muted)' }}>Uptime</span>
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 13,
                    color: 'var(--text-primary)',
                  }}
                >
                  {formatUptime(endpoint.uptime_seconds)}
                </div>
              </div>
              <div>
                <span style={{ fontSize: 10, color: 'var(--text-muted)' }}>Last Heartbeat</span>
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 13,
                    color: 'var(--text-primary)',
                  }}
                >
                  {timeAgo(endpoint.last_heartbeat)}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* System Info */}
        <div style={{ ...S.card }}>
          <div style={S.cardTitle}>System</div>
          <div
            style={{
              ...S.cardBody,
              display: 'grid',
              gridTemplateColumns: 'repeat(2, 1fr)',
              gap: 14,
            }}
          >
            {[
              { label: 'OS', value: endpoint.os_version ?? '—' },
              { label: 'Architecture', value: hw.cpu?.architecture || endpoint.arch || '—' },
              { label: 'Kernel', value: endpoint.kernel_version ?? '—' },
              {
                label: 'Virtualization',
                value: hw.virtualization?.is_virtual
                  ? hw.virtualization.hypervisor_type || 'Virtual'
                  : 'Bare Metal',
              },
              { label: 'Agent Version', value: endpoint.agent_version ?? '—' },
              { label: 'Enrolled', value: formatDate(endpoint.enrolled_at) },
            ].map(({ label, value }) => (
              <div key={label}>
                <div style={S.label}>{label}</div>
                <div style={S.value}>{value}</div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* CPU + Memory side by side */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))',
          gap: 16,
        }}
      >
        {/* CPU */}
        <div style={S.card}>
          <div style={S.cardTitle}>{endpoint?.os_family === 'darwin' ? 'SoC' : 'CPU'}</div>
          <div style={S.cardBody}>
            {/* Hero: model + vendor */}
            <div style={{ marginBottom: 12 }}>
              <div
                style={{
                  fontSize: 15,
                  fontWeight: 600,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-primary)',
                  lineHeight: 1.3,
                }}
              >
                {hw.cpu?.model_name || '—'}
              </div>
              <div
                style={{
                  fontSize: 11,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-muted)',
                  marginTop: 2,
                }}
              >
                {hw.cpu?.vendor || ''}
              </div>
            </div>

            {/* Stats grid: 2×2 */}
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))',
                gap: 8,
                marginBottom: 12,
              }}
            >
              {(
                [
                  {
                    label: 'Topology',
                    value: `${hw.cpu?.sockets ?? '—'}S / ${hw.cpu?.cores_per_socket ?? '—'}C / ${hw.cpu?.threads_per_core ?? '—'}T`,
                  },
                  { label: 'Logical CPUs', value: hw.cpu?.total_logical_cpus ?? '—' },
                  ...(endpoint.os_family !== 'darwin'
                    ? [
                        {
                          label: 'BogoMIPS',
                          value: hw.cpu?.bogomips ? hw.cpu.bogomips.toFixed(0) : '—',
                        },
                        { label: 'Virt Type', value: hw.cpu?.virtualization_type || '—' },
                      ]
                    : []),
                ] as { label: string; value: string | number }[]
              ).map(({ label, value }) => (
                <div
                  key={label}
                  style={{ padding: '6px 0', borderBottom: '1px solid var(--border)' }}
                >
                  <div
                    style={{
                      fontSize: 9,
                      fontFamily: 'var(--font-mono)',
                      color: 'var(--text-muted)',
                      textTransform: 'uppercase',
                      letterSpacing: '0.04em',
                    }}
                  >
                    {label}
                  </div>
                  <div
                    style={{
                      fontSize: 12,
                      fontFamily: 'var(--font-mono)',
                      color: 'var(--text-primary)',
                      marginTop: 2,
                    }}
                  >
                    {value}
                  </div>
                </div>
              ))}
            </div>

            {/* Frequency bar */}
            {hw.cpu?.min_mhz != null && hw.cpu?.max_mhz != null && hw.cpu.max_mhz > 0 && (
              <div style={{ marginBottom: 12 }}>
                <KVRow
                  label="Frequency"
                  value={`${formatMHz(hw.cpu?.min_mhz ?? 0)} – ${formatMHz(hw.cpu.max_mhz)}`}
                />
                <div style={{ margin: '4px 0 0', display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span
                    style={{
                      fontSize: 9,
                      fontFamily: 'var(--font-mono)',
                      color: 'var(--text-faint)',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {hw.cpu.min_mhz} MHz
                  </span>
                  <div
                    style={{
                      flex: 1,
                      height: 4,
                      borderRadius: 2,
                      background: 'var(--border)',
                      position: 'relative',
                      overflow: 'hidden',
                    }}
                  >
                    <div
                      style={{
                        position: 'absolute',
                        left: 0,
                        top: 0,
                        height: '100%',
                        borderRadius: 2,
                        background:
                          'linear-gradient(90deg, var(--signal-healthy), var(--accent), var(--signal-warning))',
                        width: '100%',
                      }}
                    />
                  </div>
                  <span
                    style={{
                      fontSize: 9,
                      fontFamily: 'var(--font-mono)',
                      color: 'var(--text-faint)',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {(hw.cpu.max_mhz / 1000).toFixed(2)} GHz
                  </span>
                </div>
              </div>
            )}

            {/* Cache: compact horizontal row */}
            {(hw.cpu?.cache_l1d || hw.cpu?.cache_l2 || hw.cpu?.cache_l3) && (
              <div
                style={{
                  display: 'flex',
                  gap: 16,
                  padding: '8px 0',
                  borderTop: '1px solid var(--border)',
                  marginTop: 4,
                }}
              >
                {[
                  { label: 'L1d', value: hw.cpu?.cache_l1d },
                  { label: 'L1i', value: hw.cpu?.cache_l1i },
                  { label: 'L2', value: hw.cpu?.cache_l2 },
                  { label: 'L3', value: hw.cpu?.cache_l3 },
                ]
                  .filter((c) => c.value)
                  .map(({ label, value }) => (
                    <div key={label}>
                      <span
                        style={{
                          fontSize: 9,
                          fontFamily: 'var(--font-mono)',
                          color: 'var(--text-muted)',
                          textTransform: 'uppercase',
                        }}
                      >
                        {label}{' '}
                      </span>
                      <span
                        style={{
                          fontSize: 11,
                          fontFamily: 'var(--font-mono)',
                          color: 'var(--text-secondary)',
                        }}
                      >
                        {value}
                      </span>
                    </div>
                  ))}
              </div>
            )}

            {/* CPU Capabilities — collapsed by default, shows key flags only */}
            {hw.cpu?.flags && hw.cpu.flags.length > 0 && <CpuCapabilities flags={hw.cpu.flags} />}
          </div>
        </div>

        {/* Memory */}
        <div style={S.card}>
          <div style={S.cardTitle}>Memory</div>
          <div style={S.cardBody}>
            <KVRow label="Total" value={`${memTotalGb.toFixed(1)} GB`} />
            {(() => {
              const dimms = hw.memory?.dimms?.filter((d) => d.size_mb > 0) ?? [];
              if (dimms.length === 0) return null;
              const type = dimms[0]?.type || '';
              const speed = dimms[0]?.speed_mhz || 0;
              if (!type && !speed) return null;
              return (
                <div
                  style={{
                    fontSize: 11,
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-muted)',
                    margin: '2px 0 4px',
                    paddingLeft: 2,
                  }}
                >
                  {type}
                  {speed ? `-${speed}` : ''}
                </div>
              );
            })()}
            <KVRow label="Used" value={`${memUsedGb.toFixed(1)} GB`} />
            <KVRow label="Available" value={`${memAvailGb.toFixed(1)} GB`} />
            {memTotalGb > 0 &&
              (() => {
                const memPct = memTotalGb > 0 ? (memUsedGb / memTotalGb) * 100 : 0;
                const memColor =
                  memPct >= 80
                    ? 'var(--signal-critical)'
                    : memPct >= 60
                      ? 'var(--signal-warning)'
                      : 'var(--signal-healthy)';
                return (
                  <div
                    style={{
                      margin: '8px 0',
                      height: 6,
                      borderRadius: 3,
                      background: 'var(--border)',
                      overflow: 'hidden',
                    }}
                  >
                    <div
                      style={{
                        height: '100%',
                        borderRadius: 3,
                        background: memColor,
                        width: `${Math.min(100, memPct)}%`,
                        transition: 'width 0.3s',
                      }}
                    />
                  </div>
                );
              })()}
            {hw.memory?.max_capacity && (
              <KVRow label="Max Capacity" value={hw.memory.max_capacity} />
            )}
            {hw.memory?.num_slots && hw.memory.num_slots > 0 && (
              <KVRow label="Slots" value={hw.memory.num_slots} />
            )}
            {hw.memory?.error_correction && (
              <KVRow label="ECC" value={hw.memory.error_correction} />
            )}
            {(() => {
              const populatedDimms = hw.memory?.dimms?.filter((d) => d.size_mb > 0) ?? [];
              const channelLabel =
                populatedDimms.length === 0
                  ? '—'
                  : populatedDimms.length === 1
                    ? 'Single Channel'
                    : populatedDimms.length === 2
                      ? 'Dual Channel'
                      : populatedDimms.length === 3
                        ? 'Triple Channel'
                        : 'Quad Channel';
              return <KVRow label="Channel" value={channelLabel} />;
            })()}

            {/* Slot Layout + DIMMs — collapsible */}
            <MemorySlotDetails memory={hw.memory} />
          </div>
        </div>
      </div>

      {/* Storage */}
      {hw.storage && hw.storage.length > 0 && (
        <div style={S.card}>
          <div style={S.cardTitle}>Storage</div>
          <div style={S.cardBody}>
            {hw.storage.map((disk, i) => (
              <div
                key={disk.name || i}
                style={
                  i > 0
                    ? { borderTop: '1px solid var(--border)', paddingTop: 14, marginTop: 14 }
                    : {}
                }
              >
                {/* Header row */}
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 14,
                      fontWeight: 700,
                      color: 'var(--text-primary)',
                    }}
                  >
                    {disk.name}
                  </span>
                  {disk.type && (
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 9,
                        color: 'var(--text-muted)',
                        border: '1px solid var(--border)',
                        borderRadius: 3,
                        padding: '1px 5px',
                        textTransform: 'uppercase',
                      }}
                    >
                      {disk.type}
                    </span>
                  )}
                  {disk.transport && (
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 9,
                        color: 'var(--text-muted)',
                        border: '1px solid var(--border)',
                        borderRadius: 3,
                        padding: '1px 5px',
                        textTransform: 'uppercase',
                      }}
                    >
                      {disk.transport}
                    </span>
                  )}
                  {disk.smart_status && (
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 9,
                        padding: '1px 5px',
                        borderRadius: 3,
                        border: '1px solid var(--border)',
                        color:
                          disk.smart_status.toLowerCase() === 'passed'
                            ? 'var(--signal-healthy)'
                            : 'var(--signal-critical)',
                      }}
                    >
                      SMART: {disk.smart_status}
                    </span>
                  )}
                  {disk.temperature_celsius > 0 && (
                    <span
                      style={{
                        fontSize: 9,
                        fontFamily: 'var(--font-mono)',
                        padding: '1px 5px',
                        borderRadius: 3,
                        border: '1px solid var(--border)',
                        background:
                          disk.temperature_celsius > 60
                            ? 'color-mix(in srgb, var(--signal-critical) 15%, transparent)'
                            : 'transparent',
                        color:
                          disk.temperature_celsius > 60
                            ? 'var(--signal-critical)'
                            : disk.temperature_celsius > 45
                              ? 'var(--signal-warning)'
                              : 'var(--text-secondary)',
                      }}
                    >
                      {disk.temperature_celsius}°C
                    </span>
                  )}
                  <span
                    style={{
                      marginLeft: 'auto',
                      fontFamily: 'var(--font-mono)',
                      fontSize: 13,
                      fontWeight: 600,
                      color: 'var(--text-secondary)',
                    }}
                  >
                    {disk.size_bytes > 0 ? formatBytes(disk.size_bytes) : '—'}
                  </span>
                </div>

                {/* Subtitle: model + serial */}
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 11,
                    color: 'var(--text-muted)',
                    marginBottom: 10,
                  }}
                >
                  {disk.model || 'Unknown model'}
                  {disk.serial && (
                    <span style={{ color: 'var(--text-faint)', marginLeft: 8 }}>
                      S/N {disk.serial}
                    </span>
                  )}
                  {disk.firmware_version && (
                    <span style={{ color: 'var(--text-faint)', marginLeft: 8 }}>
                      FW {disk.firmware_version}
                    </span>
                  )}
                </div>

                {/* Partition bar + table */}
                {disk.partitions && disk.partitions.length > 0 && (
                  <>
                    <div
                      style={{
                        display: 'flex',
                        height: 20,
                        borderRadius: 4,
                        overflow: 'hidden',
                        border: '1px solid var(--border)',
                        marginBottom: 8,
                      }}
                    >
                      {disk.partitions.map((part, pi) => {
                        const pct =
                          disk.size_bytes > 0 ? (part.size_bytes / disk.size_bytes) * 100 : 0;
                        if (pct < 0.5) return null;
                        const isRoot = part.mountpoint === '/';
                        const colors = [
                          'var(--accent)',
                          'var(--signal-healthy)',
                          'var(--signal-warning)',
                          'var(--text-muted)',
                          'var(--signal-critical)',
                        ];
                        const color = isRoot ? 'var(--accent)' : colors[pi % colors.length];
                        return (
                          <div
                            key={pi}
                            title={`${part.name} (${part.mountpoint || 'unmounted'}) — ${(part.size_bytes / 1024 ** 3).toFixed(1)} GB — ${part.usage_pct ?? 0}% used`}
                            style={{
                              width: `${pct}%`,
                              background: color,
                              opacity: 0.6,
                              minWidth: 2,
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'center',
                              overflow: 'hidden',
                            }}
                          >
                            {pct > 8 && (
                              <span
                                style={{
                                  fontSize: 8,
                                  fontFamily: 'var(--font-mono)',
                                  color: '#fff',
                                  fontWeight: 600,
                                  whiteSpace: 'nowrap',
                                }}
                              >
                                {part.mountpoint || part.name}
                              </span>
                            )}
                          </div>
                        );
                      })}
                    </div>

                    <div
                      style={{
                        display: 'grid',
                        gridTemplateColumns: '1fr 1fr 80px 80px 100px',
                        gap: 0,
                      }}
                    >
                      {/* Header */}
                      {['Partition', 'Mount', 'FS', 'Size', 'Usage'].map((h) => (
                        <div
                          key={h}
                          style={{
                            fontSize: 9,
                            fontFamily: 'var(--font-mono)',
                            color: 'var(--text-faint)',
                            textTransform: 'uppercase',
                            letterSpacing: '0.04em',
                            padding: '4px 6px',
                            borderBottom: '1px solid var(--border)',
                          }}
                        >
                          {h}
                        </div>
                      ))}
                      {/* Rows */}
                      {disk.partitions.map((part, pi) => {
                        const isRoot = part.mountpoint === '/';
                        return (
                          <React.Fragment key={pi}>
                            <div
                              style={{
                                padding: '5px 6px',
                                fontSize: 11,
                                fontFamily: 'var(--font-mono)',
                                color: 'var(--text-primary)',
                                borderBottom: '1px solid var(--border)',
                                borderLeft: isRoot ? '3px solid var(--accent)' : 'none',
                              }}
                            >
                              {part.name}
                            </div>
                            <div
                              style={{
                                padding: '5px 6px',
                                fontSize: 11,
                                fontFamily: 'var(--font-mono)',
                                color: part.mountpoint
                                  ? 'var(--text-secondary)'
                                  : 'var(--text-faint)',
                                borderBottom: '1px solid var(--border)',
                              }}
                            >
                              {part.mountpoint || '—'}
                            </div>
                            <div
                              style={{
                                padding: '5px 6px',
                                fontSize: 11,
                                fontFamily: 'var(--font-mono)',
                                color: 'var(--text-muted)',
                                borderBottom: '1px solid var(--border)',
                              }}
                            >
                              {part.fstype || '—'}
                            </div>
                            <div
                              style={{
                                padding: '5px 6px',
                                fontSize: 11,
                                fontFamily: 'var(--font-mono)',
                                color: 'var(--text-secondary)',
                                borderBottom: '1px solid var(--border)',
                              }}
                            >
                              {formatBytes(part.size_bytes)}
                            </div>
                            <div
                              style={{
                                padding: '5px 6px',
                                fontSize: 11,
                                fontFamily: 'var(--font-mono)',
                                borderBottom: '1px solid var(--border)',
                                display: 'flex',
                                alignItems: 'center',
                                gap: 4,
                              }}
                            >
                              {part.usage_pct != null && part.usage_pct > 0 ? (
                                <>
                                  <div
                                    style={{
                                      flex: 1,
                                      height: 3,
                                      borderRadius: 2,
                                      background: 'var(--border)',
                                      overflow: 'hidden',
                                    }}
                                  >
                                    <div
                                      style={{
                                        height: '100%',
                                        borderRadius: 2,
                                        background:
                                          part.usage_pct > 80
                                            ? 'var(--signal-critical)'
                                            : part.usage_pct > 60
                                              ? 'var(--signal-warning)'
                                              : 'var(--signal-healthy)',
                                        width: `${part.usage_pct}%`,
                                      }}
                                    />
                                  </div>
                                  <span
                                    style={{
                                      fontSize: 10,
                                      color: 'var(--text-muted)',
                                      minWidth: 24,
                                      textAlign: 'right',
                                    }}
                                  >
                                    {part.usage_pct}%
                                  </span>
                                </>
                              ) : (
                                <span style={{ color: 'var(--text-faint)' }}>—</span>
                              )}
                            </div>
                          </React.Fragment>
                        );
                      })}
                    </div>
                  </>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Network Interfaces */}
      {hw.network && hw.network.length > 0 && <NetworkInterfacesSection interfaces={hw.network} />}

      {/* GPU */}
      {hw.gpu && hw.gpu.length > 0 && (
        <div style={S.card}>
          <div style={S.cardTitle}>GPU</div>
          <div style={S.cardBody}>
            {hw.gpu.map((gpu, i) => (
              <div
                key={gpu.pci_slot || i}
                style={
                  i > 0
                    ? { borderTop: '1px solid var(--border)', paddingTop: 12, marginTop: 12 }
                    : {}
                }
              >
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: 12 }}>
                  {[
                    { label: 'Model', value: gpu.model || '—' },
                    {
                      label: 'VRAM',
                      value:
                        gpu.vram_mb > 0
                          ? gpu.vram_mb >= 1024
                            ? `${(gpu.vram_mb / 1024).toFixed(1)} GB`
                            : `${gpu.vram_mb} MB`
                          : '—',
                    },
                    {
                      label: 'Usage',
                      value: gpu.usage_pct !== undefined ? `${gpu.usage_pct}%` : '—',
                    },
                    { label: 'Driver', value: gpu.driver_version || '—' },
                    { label: 'PCI Slot', value: gpu.pci_slot || '—' },
                  ].map(({ label, value }) => (
                    <div key={label}>
                      <div style={S.label}>{label}</div>
                      <div style={S.value}>{value}</div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Motherboard + BIOS */}
      {hw.motherboard && Object.values(hw.motherboard).some((v) => v && v !== '' && v !== '—') && (
        <div style={S.card}>
          <div style={S.cardTitle}>Motherboard &amp; BIOS</div>
          <div
            style={{
              ...S.cardBody,
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))',
              gap: 16,
            }}
          >
            {[
              { label: 'Board Manufacturer', value: hw.motherboard.board_manufacturer || '—' },
              { label: 'Board Product', value: hw.motherboard.board_product || '—' },
              { label: 'Board Version', value: hw.motherboard.board_version || '—' },
              { label: 'Board Serial', value: hw.motherboard.board_serial || '—' },
              { label: 'BIOS Vendor', value: hw.motherboard.bios_vendor || '—' },
              { label: 'BIOS Version', value: hw.motherboard.bios_version || '—' },
              { label: 'BIOS Release', value: hw.motherboard.bios_release_date || '—' },
            ].map(({ label, value }) => (
              <div key={label}>
                <div style={S.label}>{label}</div>
                <div style={S.value}>{value}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* TPM */}
      {hw.tpm?.present && (
        <div style={S.card}>
          <div style={S.cardTitle}>TPM</div>
          <div style={S.cardBody}>
            <KVRow label="Version" value={hw.tpm.version || '—'} />
          </div>
        </div>
      )}

      {/* Battery */}
      {hw.battery?.present && (
        <div style={S.card}>
          <div style={S.cardTitle}>Battery</div>
          <div style={S.cardBody}>
            <KVRow label="Status" value={hw.battery.status || '—'} />
            {hw.battery.capacity_pct != null && (
              <>
                <KVRow label="Capacity" value={`${hw.battery.capacity_pct}%`} />
                <div style={{ margin: '8px 0' }}>
                  <ProgressBar
                    value={hw.battery.capacity_pct}
                    max={100}
                    color={
                      hw.battery.capacity_pct < 20
                        ? 'var(--signal-critical)'
                        : hw.battery.capacity_pct < 50
                          ? 'var(--signal-warning)'
                          : 'var(--signal-healthy)'
                    }
                  />
                </div>
              </>
            )}
            {hw.battery.health_pct != null && (
              <KVRow label="Health" value={`${hw.battery.health_pct}%`} />
            )}
            {hw.battery.cycle_count != null && hw.battery.cycle_count > 0 && (
              <KVRow label="Cycle Count" value={hw.battery.cycle_count} />
            )}
            {hw.battery.technology && <KVRow label="Technology" value={hw.battery.technology} />}
          </div>
        </div>
      )}

      {/* USB Devices */}
      {hw.usb && hw.usb.length > 0 && (
        <div style={S.card}>
          <div style={S.cardTitle}>USB Devices</div>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  {['Description', 'Bus', 'Device', 'Vendor ID', 'Product ID'].map((h) => (
                    <th key={h} style={S.th}>
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {hw.usb.map((device, i) => (
                  <tr key={`${device.vendor_id}-${device.product_id}-${i}`}>
                    <td style={S.td}>{device.description || '—'}</td>
                    <td style={{ ...S.td, fontFamily: 'var(--font-mono)', fontSize: 11 }}>
                      {device.bus}
                    </td>
                    <td style={{ ...S.td, fontFamily: 'var(--font-mono)', fontSize: 11 }}>
                      {device.device_num}
                    </td>
                    <td style={{ ...S.td, fontFamily: 'var(--font-mono)', fontSize: 11 }}>
                      {device.vendor_id}
                    </td>
                    <td style={{ ...S.td, fontFamily: 'var(--font-mono)', fontSize: 11 }}>
                      {device.product_id}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Certificate */}
      {endpoint.cert_expiry && (
        <div style={S.card}>
          <div style={S.cardTitle}>Certificate</div>
          <div style={S.cardBody}>
            <KVRow
              label="Cert Expiry"
              value={
                <span
                  style={
                    certExpiryWarning(endpoint.cert_expiry)
                      ? { color: 'var(--signal-warning)' }
                      : {}
                  }
                >
                  {formatDate(endpoint.cert_expiry)}
                  {certExpiryWarning(endpoint.cert_expiry) && ' (expiring soon)'}
                </span>
              }
            />
          </div>
        </div>
      )}
    </div>
  );
}

// ── flat fallback view ─────────────────────────────────────────
function FlatHardwareView({ endpoint }: { endpoint: EndpointFields }) {
  const memTotalGb = endpoint.memory_total_mb ? endpoint.memory_total_mb / 1024 : 0;
  const memUsedGb = endpoint.memory_used_mb ? endpoint.memory_used_mb / 1024 : 0;
  const memPct = memTotalGb > 0 ? (memUsedGb / memTotalGb) * 100 : 0;
  const diskPct =
    endpoint.disk_total_gb && endpoint.disk_used_gb && endpoint.disk_total_gb > 0
      ? (endpoint.disk_used_gb / endpoint.disk_total_gb) * 100
      : 0;

  const gaugeColor = (pct: number) => {
    if (pct >= 90) return 'var(--signal-critical)';
    if (pct >= 70) return 'var(--signal-warning)';
    return 'var(--signal-healthy)';
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Hero gauges */}
      <div style={S.card}>
        <div style={S.cardTitle}>System Resources</div>
        <div
          style={{
            ...S.cardBody,
            display: 'flex',
            gap: 48,
            flexWrap: 'wrap' as const,
            alignItems: 'center',
          }}
        >
          <HardwareRingGauge
            pct={memPct}
            label="Memory"
            sub={memTotalGb > 0 ? `${memUsedGb.toFixed(1)}/${memTotalGb.toFixed(1)} GB` : '—'}
            color={gaugeColor(memPct)}
          />
          <HardwareRingGauge
            pct={diskPct}
            label="Disk"
            sub={
              endpoint.disk_total_gb
                ? `${endpoint.disk_used_gb?.toFixed(0) ?? '?'}/${endpoint.disk_total_gb} GB`
                : '—'
            }
            color={gaugeColor(diskPct)}
          />
          <div style={{ marginLeft: 'auto' }}>
            <div style={S.label}>Uptime</div>
            <div style={S.value}>{formatUptime(endpoint.uptime_seconds)}</div>
          </div>
        </div>
      </div>

      {/* System details */}
      <div style={S.card}>
        <div style={S.cardTitle}>System Info</div>
        <div
          style={{
            ...S.cardBody,
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))',
            gap: 16,
          }}
        >
          {[
            { label: 'OS', value: endpoint.os_version ?? '—' },
            { label: 'Kernel', value: endpoint.kernel_version ?? '—' },
            { label: 'Architecture', value: endpoint.arch ?? '—' },
            { label: 'CPU Model', value: endpoint.cpu_model ?? '—' },
            { label: 'CPU Cores', value: endpoint.cpu_cores ?? '—' },
            { label: 'GPU', value: endpoint.gpu_model ?? '—' },
            { label: 'Agent Version', value: endpoint.agent_version ?? '—' },
            { label: 'Enrolled', value: formatDate(endpoint.enrolled_at) },
            { label: 'Last Heartbeat', value: timeAgo(endpoint.last_heartbeat) },
          ].map(({ label, value }) => (
            <div key={label}>
              <div style={S.label}>{label}</div>
              <div style={S.value}>{value}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Network interfaces (flat) */}
      {endpoint.network_interfaces && endpoint.network_interfaces.length > 0 && (
        <div style={S.card}>
          <div style={S.cardTitle}>Network Interfaces</div>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  {['Interface', 'IP Address', 'MAC Address', 'Status'].map((h) => (
                    <th key={h} style={S.th}>
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {endpoint.network_interfaces.map((iface) => (
                  <tr
                    key={iface.id}
                    onMouseEnter={(e) => {
                      (e.currentTarget as HTMLTableRowElement).style.background =
                        'var(--bg-card-hover)';
                    }}
                    onMouseLeave={(e) => {
                      (e.currentTarget as HTMLTableRowElement).style.background = '';
                    }}
                  >
                    <td style={{ ...S.td, fontFamily: 'var(--font-mono)', fontSize: 12 }}>
                      {iface.name}
                    </td>
                    <td style={{ ...S.td, fontFamily: 'var(--font-mono)', fontSize: 11 }}>
                      {iface.ip_address ?? '—'}
                    </td>
                    <td style={{ ...S.td, fontFamily: 'var(--font-mono)', fontSize: 11 }}>
                      {iface.mac_address ?? '—'}
                    </td>
                    <td style={S.td}>
                      <span
                        style={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: 5,
                          fontSize: 12,
                          color:
                            iface.status === 'up' ? 'var(--signal-healthy)' : 'var(--text-faint)',
                        }}
                      >
                        <span
                          style={{
                            width: 6,
                            height: 6,
                            borderRadius: '50%',
                            background:
                              iface.status === 'up' ? 'var(--signal-healthy)' : 'var(--text-faint)',
                          }}
                        />
                        {iface.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {endpoint.cert_expiry && (
        <div style={S.card}>
          <div style={S.cardTitle}>Certificate</div>
          <div style={S.cardBody}>
            <KVRow
              label="Cert Expiry"
              value={
                <span
                  style={
                    certExpiryWarning(endpoint.cert_expiry)
                      ? { color: 'var(--signal-warning)' }
                      : {}
                  }
                >
                  {formatDate(endpoint.cert_expiry)}
                  {certExpiryWarning(endpoint.cert_expiry) && ' (expiring soon)'}
                </span>
              }
            />
          </div>
        </div>
      )}
    </div>
  );
}

// ── endpoint fields shape ──────────────────────────────────────
interface EndpointFields {
  id: string;
  os_version?: string;
  cpu_model?: string | null;
  cpu_cores?: number | null;
  cpu_usage_percent?: number | null;
  memory_total_mb?: number | null;
  memory_used_mb?: number | null;
  disk_total_gb?: number | null;
  disk_used_gb?: number | null;
  gpu_model?: string | null;
  uptime_seconds?: number | null;
  arch?: string | null;
  kernel_version?: string | null;
  agent_version?: string | null;
  os_family?: string | null;
  enrolled_at?: string | null;
  last_heartbeat?: string | null;
  cert_expiry?: string | null;
  network_interfaces?: {
    id: string;
    name: string;
    ip_address?: string | null;
    mac_address?: string | null;
    status: string;
  }[];
}

// ── main export ───────────────────────────────────────────────
export function HardwareTab({ endpointId }: HardwareTabProps) {
  const { data: endpoint, isLoading, error, refetch } = useEndpoint(endpointId);

  if (isLoading) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {[120, 200, 200, 120].map((h, i) => (
          <Skeleton
            key={i}
            className={`h-[${h}px] w-full rounded-lg`}
            style={{ height: h, borderRadius: 8 }}
          />
        ))}
      </div>
    );
  }

  if (error || !endpoint) {
    return (
      <div
        style={{
          ...S.card,
          padding: 16,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <span style={{ fontSize: 13, color: 'var(--signal-critical)' }}>
          Failed to load hardware data
        </span>
        <button
          onClick={() => void refetch()}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            fontSize: 12,
            color: 'var(--text-secondary)',
            background: 'none',
            border: '1px solid var(--border)',
            borderRadius: 6,
            padding: '4px 10px',
            cursor: 'pointer',
          }}
        >
          <RefreshCw size={12} />
          Retry
        </button>
      </div>
    );
  }

  const hw = endpoint.hardware_details as HardwareInfo | null | undefined;

  if (hasHardwareDetails(hw)) {
    return <RichHardwareView hw={hw!} endpoint={endpoint as unknown as EndpointFields} />;
  }

  return <FlatHardwareView endpoint={endpoint as unknown as EndpointFields} />;
}
