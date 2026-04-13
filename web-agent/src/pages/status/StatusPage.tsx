import { useRef, useState } from 'react';
import { Activity, Server, ArrowUp, ArrowDown, Copy, Check } from 'lucide-react';
import { Skeleton, ErrorState } from '@patchiq/ui';
import { useAgentStatus } from '../../api/hooks/useStatus';
import { useMetrics } from '../../api/hooks/useMetrics';
import { CARD_STYLE, CARD_PAD_STYLE } from '../../lib/styles';

function formatUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}d ${h}h ${m}m`;
  return `${h}h ${m}m`;
}

function formatNetworkSpeed(bytesPerSec: number): string {
  if (bytesPerSec >= 1024 * 1024) return `${(bytesPerSec / 1024 / 1024).toFixed(1)} MB/s`;
  if (bytesPerSec >= 1024) return `${(bytesPerSec / 1024).toFixed(1)} KB/s`;
  return `${Math.round(bytesPerSec)} B/s`;
}

function formatOsVersion(osFamily: string, osVersion: string): string {
  if (osFamily === 'darwin') {
    if (osVersion.startsWith('macOS')) {
      if (osVersion.includes('arm64')) return `${osVersion} (Apple Silicon)`;
      if (osVersion.includes('amd64') || osVersion.includes('x86_64'))
        return `${osVersion} (Intel)`;
      return osVersion;
    }
    if (osVersion.includes('arm64')) return `macOS ${osVersion} (Apple Silicon)`;
    if (osVersion.includes('amd64') || osVersion.includes('x86_64'))
      return `macOS ${osVersion} (Intel)`;
    return osVersion;
  }
  return osVersion;
}

function formatRelativeTime(iso: string | null): string {
  if (!iso) return 'Pending first heartbeat';
  const diff = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  return `${Math.floor(diff / 3600)}h ago`;
}

/** A heartbeat within the last 3 minutes is considered connected. */
function isHeartbeatRecent(iso: string | null): boolean {
  if (!iso) return false;
  const diff = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  return diff < 180;
}

/** Hover effect handlers — sets border-color on mouse enter/leave. */
function hoverHandlers(el: HTMLDivElement | null) {
  if (!el) return;
  el.addEventListener('mouseenter', () => {
    el.style.borderColor = 'var(--text-faint)';
  });
  el.addEventListener('mouseleave', () => {
    el.style.borderColor = 'var(--border)';
  });
}

function WidgetCard({
  children,
  style,
}: {
  children: React.ReactNode;
  style?: React.CSSProperties;
}) {
  const ref = useRef<HTMLDivElement>(null);
  return (
    <div
      ref={(el) => {
        (ref as React.MutableRefObject<HTMLDivElement | null>).current = el;
        hoverHandlers(el);
      }}
      style={{ ...CARD_STYLE, ...style }}
    >
      <div style={CARD_PAD_STYLE}>{children}</div>
    </div>
  );
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '6px 0',
        borderBottom: '1px solid var(--border-faint)',
      }}
    >
      <span style={{ fontSize: '13px', color: 'var(--text-muted)' }}>{label}</span>
      <span style={{ fontSize: '13px', color: 'var(--text-emphasis)', fontWeight: 500 }}>
        {value}
      </span>
    </div>
  );
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <p
      style={{
        fontSize: 13,
        color: 'var(--text-muted)',
        fontWeight: 400,
        margin: '0 0 12px',
      }}
    >
      {children}
    </p>
  );
}

function MiniRing({ value, size = 36 }: { value: number; size?: number }) {
  const r = (size - 4) / 2;
  const circ = 2 * Math.PI * r;
  const offset = circ * (1 - Math.min(100, Math.max(0, value)) / 100);
  return (
    <svg width={size} height={size} style={{ transform: 'rotate(-90deg)', flexShrink: 0 }}>
      <circle
        cx={size / 2}
        cy={size / 2}
        r={r}
        fill="none"
        stroke="var(--ring-track, var(--border))"
        strokeWidth={3}
      />
      <circle
        cx={size / 2}
        cy={size / 2}
        r={r}
        fill="none"
        stroke="var(--accent)"
        strokeWidth={3}
        strokeDasharray={circ}
        strokeDashoffset={offset}
        strokeLinecap="round"
      />
    </svg>
  );
}

function ProgressBar({ value, label, detail }: { value: number; label: string; detail?: string }) {
  const pct = Math.min(100, Math.round(value));
  const barColor =
    pct > 85 ? 'var(--signal-critical)' : pct > 70 ? 'var(--signal-warning)' : 'var(--accent)';
  return (
    <div style={{ marginBottom: '16px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '6px' }}>
        <span style={{ fontSize: '12px', color: 'var(--text-muted)' }}>{label}</span>
        <span
          style={{
            fontSize: '12px',
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-emphasis)',
            fontWeight: 600,
          }}
        >
          {detail ?? `${pct}%`}
        </span>
      </div>
      <div
        style={{
          height: '6px',
          borderRadius: '3px',
          background: 'var(--border)',
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            height: '100%',
            borderRadius: '3px',
            background: barColor,
            width: `${pct}%`,
            transition: 'width 0.3s',
          }}
        />
      </div>
    </div>
  );
}

function CopyableAgentId({ agentId }: { agentId: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(agentId).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      title={copied ? 'Copied!' : 'Click to copy full Agent ID'}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '4px',
        fontFamily: 'var(--font-mono)',
        fontSize: '11px',
        color: 'var(--text-emphasis)',
        fontWeight: 500,
        background: 'none',
        border: 'none',
        cursor: 'pointer',
        padding: 0,
      }}
    >
      {agentId?.slice(0, 8)}…
      {copied ? (
        <Check style={{ width: '12px', height: '12px', color: 'var(--signal-healthy)' }} />
      ) : (
        <Copy style={{ width: '12px', height: '12px', color: 'var(--text-muted)' }} />
      )}
    </button>
  );
}

export function StatusPage() {
  const { data, isLoading, isError, refetch } = useAgentStatus();
  const metricsQuery = useMetrics();
  const liveMetrics = metricsQuery.data;
  const isHealthy = data?.enrollment_status === 'enrolled';

  const cpuPct = liveMetrics?.cpu_usage_pct ?? 0;
  const memPct = liveMetrics?.memory_used_pct ?? 0;
  // Use root filesystem usage if available, fall back to swap as last resort
  const rootFS = liveMetrics?.filesystems?.find((fs: { mount: string }) => fs.mount === '/');
  const diskPct = rootFS
    ? rootFS.use_pct
    : liveMetrics?.swap_total_bytes && liveMetrics.swap_total_bytes > 0
      ? (liveMetrics.swap_used_bytes / liveMetrics.swap_total_bytes) * 100
      : 0;
  const totalRxBps = liveMetrics?.network_io
    ? liveMetrics.network_io.reduce((sum, n) => sum + n.rx_bytes_per_sec, 0)
    : null;
  const totalTxBps = liveMetrics?.network_io
    ? liveMetrics.network_io.reduce((sum, n) => sum + n.tx_bytes_per_sec, 0)
    : null;
  const cpuCoreCount = liveMetrics?.cpu_per_core?.length ?? null;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {isLoading && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <Skeleton className="h-16 w-full" />
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
            <Skeleton className="h-48" />
            <Skeleton className="h-48" />
          </div>
        </div>
      )}

      {isError && (
        <div style={{ ...CARD_STYLE }}>
          <ErrorState message="Failed to load agent status." onRetry={() => refetch()} />
        </div>
      )}

      {data && (
        <>
          {/* Compact status strip — single line */}
          <div
            ref={(el) => hoverHandlers(el)}
            style={{
              ...CARD_STYLE,
              ...CARD_PAD_STYLE,
              flexDirection: 'row',
              alignItems: 'center',
              gap: '0',
              flexWrap: 'wrap',
            }}
          >
            {/* Status dot + label */}
            <div
              style={{ display: 'flex', alignItems: 'center', gap: '8px', paddingRight: '20px' }}
            >
              <span
                style={{
                  width: '8px',
                  height: '8px',
                  borderRadius: '50%',
                  background: isHealthy ? 'var(--signal-healthy)' : 'var(--signal-warning)',
                  flexShrink: 0,
                }}
              />
              <span
                style={{
                  fontSize: '14px',
                  fontWeight: 600,
                  color: isHealthy ? 'var(--signal-healthy)' : 'var(--signal-warning)',
                }}
              >
                {isHealthy ? 'Healthy' : 'Degraded'}
              </span>
            </div>

            <span style={{ color: 'var(--border)', marginRight: '20px' }}>│</span>

            {/* CPU ring */}
            <div
              style={{ display: 'flex', alignItems: 'center', gap: '8px', paddingRight: '20px' }}
            >
              <MiniRing value={cpuPct} size={36} />
              <div>
                <div
                  style={{
                    fontSize: '10px',
                    color: 'var(--text-muted)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.08em',
                  }}
                >
                  CPU
                </div>
                <div
                  style={{
                    fontSize: '13px',
                    fontWeight: 600,
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-emphasis)',
                  }}
                >
                  {Math.round(cpuPct)}%
                </div>
              </div>
            </div>

            <span style={{ color: 'var(--border)', marginRight: '20px' }}>│</span>

            {/* Memory ring */}
            <div
              style={{ display: 'flex', alignItems: 'center', gap: '8px', paddingRight: '20px' }}
            >
              <MiniRing value={memPct} size={36} />
              <div>
                <div
                  style={{
                    fontSize: '10px',
                    color: 'var(--text-muted)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.08em',
                  }}
                >
                  Memory
                </div>
                <div
                  style={{
                    fontSize: '13px',
                    fontWeight: 600,
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-emphasis)',
                  }}
                >
                  {Math.round(memPct)}%
                </div>
              </div>
            </div>

            <span style={{ color: 'var(--border)', marginRight: '20px' }}>│</span>

            {/* Disk ring */}
            <div
              style={{ display: 'flex', alignItems: 'center', gap: '8px', paddingRight: '20px' }}
            >
              <MiniRing value={diskPct} size={36} />
              <div>
                <div
                  style={{
                    fontSize: '10px',
                    color: 'var(--text-muted)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.08em',
                  }}
                >
                  Disk
                </div>
                <div
                  style={{
                    fontSize: '13px',
                    fontWeight: 600,
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-emphasis)',
                  }}
                >
                  {Math.round(diskPct)}%
                </div>
              </div>
            </div>

            <span style={{ color: 'var(--border)', marginRight: '20px' }}>│</span>

            {/* Network */}
            <div
              style={{ display: 'flex', alignItems: 'center', gap: '10px', paddingRight: '20px' }}
            >
              <div>
                <div
                  style={{
                    fontSize: '10px',
                    color: 'var(--text-muted)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.08em',
                    marginBottom: '2px',
                  }}
                >
                  Network
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <span
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '3px',
                      fontSize: '12px',
                      color: 'var(--text-emphasis)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    <ArrowUp
                      style={{ width: '11px', height: '11px', color: 'var(--signal-healthy)' }}
                    />
                    {totalTxBps !== null ? formatNetworkSpeed(totalTxBps) : '—'}
                  </span>
                  <span
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '3px',
                      fontSize: '12px',
                      color: 'var(--text-emphasis)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    <ArrowDown style={{ width: '11px', height: '11px', color: 'var(--accent)' }} />
                    {totalRxBps !== null ? formatNetworkSpeed(totalRxBps) : '—'}
                  </span>
                </div>
              </div>
            </div>

            <span style={{ color: 'var(--border)', marginRight: '20px' }}>│</span>

            {/* Patches */}
            <div>
              <div
                style={{
                  fontSize: '10px',
                  color: 'var(--text-muted)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.08em',
                }}
              >
                Patches
              </div>
              <div
                style={{
                  fontSize: '13px',
                  fontWeight: 700,
                  color:
                    (data as any).pending_patch_count > 0 ? 'var(--signal-warning)' : 'var(--text-muted)',
                  fontFamily: 'var(--font-mono)',
                }}
              >
                {(data as any).pending_patch_count}
              </div>
            </div>
          </div>

          {/* Two-column: Resources | Compliance */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
            {/* Left: Resources */}
            <WidgetCard>
              <SectionTitle>Resources</SectionTitle>

              {/* CPU section */}
              <div style={{ marginBottom: '20px' }}>
                <div
                  style={{
                    fontSize: '13px',
                    fontWeight: 600,
                    color: 'var(--text-emphasis)',
                    marginBottom: '2px',
                  }}
                >
                  CPU
                </div>
                <div style={{ fontSize: '12px', color: 'var(--text-muted)', marginBottom: '10px' }}>
                  {data.hostname}
                  {cpuCoreCount !== null ? ` • ${cpuCoreCount} cores` : ''}
                </div>
                <ProgressBar value={cpuPct} label="Usage" />
              </div>

              {/* Memory section */}
              <div style={{ marginBottom: '20px' }}>
                <div
                  style={{
                    fontSize: '13px',
                    fontWeight: 600,
                    color: 'var(--text-emphasis)',
                    marginBottom: '10px',
                  }}
                >
                  Memory
                </div>
                <ProgressBar
                  value={memPct}
                  label="Used"
                  detail={
                    liveMetrics
                      ? `${(liveMetrics.memory_used_bytes / 1024 / 1024 / 1024).toFixed(1)} / ${(liveMetrics.memory_total_bytes / 1024 / 1024 / 1024).toFixed(1)} GB`
                      : `${Math.round(memPct)}%`
                  }
                />
              </div>

              {/* Disk section */}
              <div>
                <div
                  style={{
                    fontSize: '13px',
                    fontWeight: 600,
                    color: 'var(--text-emphasis)',
                    marginBottom: '10px',
                  }}
                >
                  Disk
                </div>
                <ProgressBar
                  value={diskPct}
                  label="Used"
                  detail={
                    rootFS
                      ? `${(rootFS.used_bytes / 1024 / 1024 / 1024).toFixed(0)} / ${(rootFS.total_bytes / 1024 / 1024 / 1024).toFixed(0)} GB`
                      : `${Math.round(diskPct)}%`
                  }
                />
              </div>
            </WidgetCard>

            {/* Right: Compliance */}
            <WidgetCard>
              <SectionTitle>Compliance</SectionTitle>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  height: '120px',
                  borderRadius: '8px',
                  border: '1px dashed var(--border)',
                  padding: '16px',
                }}
              >
                <p
                  style={{
                    fontSize: '13px',
                    color: 'var(--text-muted)',
                    textAlign: 'center',
                    lineHeight: '1.6',
                  }}
                >
                  Compliance scores are managed by your Patch Manager.
                  <br />
                  Check your PM dashboard for compliance status.
                </p>
              </div>
            </WidgetCard>
          </div>

          {/* Two-column: Agent Health | Server Connection */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
            <WidgetCard>
              <div
                style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '12px' }}
              >
                <Activity className="h-4 w-4" style={{ color: 'var(--signal-healthy)' }} />
                <SectionTitle>Agent Health</SectionTitle>
              </div>
              <Row
                label="Enrollment"
                value={
                  <span
                    style={{
                      fontSize: '11px',
                      padding: '2px 8px',
                      borderRadius: '4px',
                      border: '1px solid var(--border)',
                      color: 'var(--text-emphasis)',
                    }}
                  >
                    {data.enrollment_status}
                  </span>
                }
              />
              <Row label="Uptime" value={formatUptime(data.uptime_seconds)} />
              <Row
                label="Version"
                value={<span style={{ fontFamily: 'var(--font-mono)' }}>{data.agent_version}</span>}
              />
              <Row
                label="Installed Patches"
                value={
                  <span style={{ fontFamily: 'var(--font-mono)' }}>{(data as any).installed_count}</span>
                }
              />
              <Row
                label="Failed"
                value={
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      color: (data as any).failed_count > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
                    }}
                  >
                    {(data as any).failed_count}
                  </span>
                }
              />
              <Row label="Agent ID" value={<CopyableAgentId agentId={data.agent_id} />} />
            </WidgetCard>

            <WidgetCard>
              <div
                style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '12px' }}
              >
                <Server className="h-4 w-4" style={{ color: 'var(--accent)' }} />
                <SectionTitle>Server Connection</SectionTitle>
              </div>
              <Row
                label="Server URL"
                value={
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: '11px',
                      wordBreak: 'break-all',
                    }}
                  >
                    {data.server_url}
                  </span>
                }
              />
              <Row label="Last Heartbeat" value={formatRelativeTime(data.last_heartbeat)} />
              <Row
                label="Status"
                value={(() => {
                  const connected = isHeartbeatRecent(data.last_heartbeat);
                  const neverReceived = !data.last_heartbeat;
                  const color = connected
                    ? 'var(--signal-healthy)'
                    : neverReceived
                      ? 'var(--text-muted)'
                      : 'var(--signal-warning)';
                  const label = connected
                    ? 'Connected'
                    : neverReceived
                      ? 'Awaiting heartbeat'
                      : 'Heartbeat stale';
                  return (
                    <span style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                      <span
                        style={{
                          width: '8px',
                          height: '8px',
                          borderRadius: '50%',
                          background: color,
                          display: 'inline-block',
                        }}
                      />
                      <span style={{ color }}>{label}</span>
                    </span>
                  );
                })()}
              />
            </WidgetCard>
          </div>

          {/* System Info — full width */}
          <WidgetCard>
            <SectionTitle>System Info</SectionTitle>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '8px' }}>
              <Row
                label="OS Family"
                value={
                  data.os_family === 'darwin'
                    ? 'macOS'
                    : data.os_family === 'windows'
                      ? 'Windows'
                      : data.os_family
                }
              />
              <Row
                label="OS Version"
                value={
                  <span style={{ fontFamily: 'var(--font-mono)', fontSize: '11px' }}>
                    {formatOsVersion(data.os_family, data.os_version)}
                  </span>
                }
              />
              <Row
                label="Hostname"
                value={<span style={{ fontFamily: 'var(--font-mono)' }}>{data.hostname}</span>}
              />
            </div>
          </WidgetCard>
        </>
      )}
    </div>
  );
}
