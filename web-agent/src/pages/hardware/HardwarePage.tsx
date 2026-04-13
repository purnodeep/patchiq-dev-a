import { useState } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import {
  Skeleton,
  Badge,
  ErrorState,
  Collapsible,
  CollapsibleTrigger,
  CollapsibleContent,
} from '@patchiq/ui';
import { useAgentHardware } from '../../api/hooks/useHardware';
import { useAgentStatus } from '../../api/hooks/useStatus';
import { useMetrics } from '../../api/hooks/useMetrics';
import { CARD_STYLE, CARD_PAD_STYLE } from '../../lib/styles';
import type { StorageDevice, NetworkInfo, HardwareInfo } from '../../types/hardware';
import type { LiveMetrics } from '../../types/metrics';

// ---------------------------------------------------------------------------
// Local style helpers
// ---------------------------------------------------------------------------

const CARD_TITLE_STYLE: React.CSSProperties = {
  fontSize: 13,
  color: 'var(--text-muted)',
  fontWeight: 400,
  margin: '0 0 12px',
};

function hoverHandlers(el: HTMLDivElement | null) {
  if (!el) return;
  el.addEventListener('mouseenter', () => {
    el.style.borderColor = 'var(--text-faint)';
  });
  el.addEventListener('mouseleave', () => {
    el.style.borderColor = 'var(--border)';
  });
}

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 1 ? 1 : 0)} ${units[i]}`;
}

function formatRate(bytesPerSec: number): string {
  if (bytesPerSec < 1024) return `${bytesPerSec.toFixed(0)} B/s`;
  if (bytesPerSec < 1024 * 1024) return `${(bytesPerSec / 1024).toFixed(1)} KB/s`;
  return `${(bytesPerSec / (1024 * 1024)).toFixed(1)} MB/s`;
}

function formatUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const parts: string[] = [];
  if (d > 0) parts.push(`${d}d`);
  parts.push(`${h}h`);
  parts.push(`${m}m`);
  return parts.join(' ');
}

function usageColor(pct: number): string {
  if (pct < 30) return 'var(--signal-healthy)';
  if (pct < 70) return 'var(--signal-warning)';
  return 'var(--signal-critical)';
}

// Known important CPU flags to show as capability badges
const IMPORTANT_FLAGS = [
  'avx2',
  'avx512f',
  'aes',
  'sse4_2',
  'sha_ni',
  'vmx',
  'svm',
  'avx',
  'sse4_1',
  'pni',
  'rdrand',
  'bmi2',
  'fma',
] as const;

const FLAG_LABELS: Record<string, string> = {
  avx2: 'AVX2',
  avx512f: 'AVX-512',
  avx: 'AVX',
  aes: 'AES-NI',
  sse4_2: 'SSE4.2',
  sse4_1: 'SSE4.1',
  sha_ni: 'SHA-NI',
  vmx: 'VT-x',
  svm: 'AMD-V',
  pni: 'SSE3',
  rdrand: 'RDRAND',
  bmi2: 'BMI2',
  fma: 'FMA',
};

function isPhysicalInterface(name: string): boolean {
  // Virtual interface patterns (skip these)
  if (/^(utun|awdl|llw|gif|stf|ap\d|anpi|lo|veth|br-|docker|virbr|tun|tap)/.test(name)) {
    return false;
  }
  // Physical: en*, eth*, wl*, ww*, ib*, bond*, bridge*
  return /^(en|eth|wl|ww|ib|bond|bridge)[a-z0-9]*/i.test(name);
}

// ---------------------------------------------------------------------------
// Shared small components
// ---------------------------------------------------------------------------

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

function GaugeBar({ label, value, gradient }: { label: string; value: number; gradient: string }) {
  const pct = Math.min(100, Math.max(0, value));
  const barStyle =
    pct > 85
      ? 'linear-gradient(90deg, var(--signal-critical), var(--signal-critical))'
      : pct > 70
        ? 'linear-gradient(90deg, var(--signal-warning), var(--signal-warning))'
        : gradient;
  return (
    <div style={{ marginBottom: '12px' }}>
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
          {pct.toFixed(1)}%
        </span>
      </div>
      <div
        style={{
          height: '8px',
          borderRadius: '4px',
          background: 'var(--border)',
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            height: '100%',
            borderRadius: '4px',
            backgroundImage: barStyle,
            width: `${pct}%`,
            transition: 'width 0.6s ease',
          }}
        />
      </div>
    </div>
  );
}

function SmartBadge({ status }: { status: string }) {
  const passed = status.toLowerCase() === 'passed' || status.toLowerCase() === 'ok';
  return (
    <span
      style={{
        fontSize: '11px',
        padding: '2px 8px',
        borderRadius: '4px',
        border: `1px solid ${passed ? 'color-mix(in srgb, var(--signal-healthy) 30%, transparent)' : 'color-mix(in srgb, var(--signal-critical) 30%, transparent)'}`,
        background: passed
          ? 'color-mix(in srgb, var(--signal-healthy) 10%, transparent)'
          : 'color-mix(in srgb, var(--signal-critical) 10%, transparent)',
        color: passed ? 'var(--signal-healthy)' : 'var(--signal-critical)',
        fontWeight: 600,
      }}
    >
      {passed ? 'PASSED' : status.toUpperCase()}
    </span>
  );
}

function TypeBadge({ type }: { type: string }) {
  return (
    <span
      style={{
        fontSize: '10px',
        padding: '2px 6px',
        borderRadius: '4px',
        border: '1px solid var(--border)',
        color: 'var(--text-secondary)',
        fontFamily: 'var(--font-mono)',
      }}
    >
      {type}
    </span>
  );
}

function StateDot({ up }: { up: boolean }) {
  return (
    <span
      style={{
        width: '8px',
        height: '8px',
        borderRadius: '50%',
        background: up ? 'var(--signal-healthy)' : 'var(--signal-critical)',
        display: 'inline-block',
        flexShrink: 0,
      }}
    />
  );
}

// ---------------------------------------------------------------------------
// LIVE MONITORING SECTION
// ---------------------------------------------------------------------------

function LiveStatCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div ref={(el) => hoverHandlers(el)} style={{ ...CARD_STYLE, ...CARD_PAD_STYLE }}>
      <p style={{ ...CARD_TITLE_STYLE, marginBottom: '16px' }}>{title}</p>
      {children}
    </div>
  );
}

function CPUUsageCard({ metrics }: { metrics: LiveMetrics }) {
  const pct = metrics.cpu_usage_pct;
  const color = usageColor(pct);
  return (
    <LiveStatCard title="CPU Usage">
      <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
        <div style={{ position: 'relative', width: '80px', height: '80px', flexShrink: 0 }}>
          <svg width="80" height="80" viewBox="0 0 80 80">
            <circle cx="40" cy="40" r="34" fill="none" stroke="var(--ring-track)" strokeWidth="8" />
            <circle
              cx="40"
              cy="40"
              r="34"
              fill="none"
              stroke={color}
              strokeWidth="8"
              strokeDasharray={`${(pct / 100) * 2 * Math.PI * 34} ${2 * Math.PI * 34}`}
              strokeLinecap="round"
              transform="rotate(-90 40 40)"
              style={{ transition: 'stroke-dasharray 0.6s ease' }}
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
              style={{ fontSize: '18px', fontWeight: 700, color, fontFamily: 'var(--font-mono)' }}
            >
              {pct.toFixed(0)}%
            </span>
          </div>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
          {metrics.cpu_temp_celsius > 0 && (
            <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
              Temp:{' '}
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  color:
                    metrics.cpu_temp_celsius > 80
                      ? 'var(--signal-critical)'
                      : 'var(--text-primary)',
                }}
              >
                {metrics.cpu_temp_celsius}&deg;C
              </span>
            </span>
          )}
          <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
            Cores:{' '}
            <span style={{ color: 'var(--text-emphasis)', fontFamily: 'var(--font-mono)' }}>
              {metrics.cpu_per_core?.length ?? '—'}
            </span>
          </span>
        </div>
      </div>
    </LiveStatCard>
  );
}

function MemoryUsageCard({ metrics }: { metrics: LiveMetrics }) {
  const pct = metrics.memory_used_pct;
  const color = usageColor(pct);
  const total = metrics.memory_total_bytes;
  const compressed = metrics.memory_buffers_bytes; // macOS: compressed; Linux: buffers
  const cached = metrics.memory_cached_bytes;
  const isMac = compressed > 0 && total > 0;

  // macOS breakdown: Used = App + Wired + Compressed (already computed by backend)
  // App+Wired = Used - Compressed
  const appAndWired =
    metrics.memory_used_bytes > compressed
      ? metrics.memory_used_bytes - compressed
      : metrics.memory_used_bytes;

  // Memory pressure color (macOS style: green/yellow/red)
  const pressureColor =
    pct < 60
      ? 'var(--signal-healthy)'
      : pct < 80
        ? 'var(--signal-warning)'
        : 'var(--signal-critical)';
  const pressureLabel = pct < 60 ? 'Low' : pct < 80 ? 'Medium' : 'High';

  // Stacked bar segments as % of total
  const appPct = total > 0 ? (appAndWired / total) * 100 : 0;
  const compPct = total > 0 ? (compressed / total) * 100 : 0;
  const cachedPct = total > 0 ? (cached / total) * 100 : 0;

  return (
    <LiveStatCard title={isMac ? 'Memory Pressure' : 'Memory Usage'}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
        <div style={{ position: 'relative', width: '80px', height: '80px', flexShrink: 0 }}>
          <svg width="80" height="80" viewBox="0 0 80 80">
            <circle cx="40" cy="40" r="34" fill="none" stroke="var(--ring-track)" strokeWidth="8" />
            <circle
              cx="40"
              cy="40"
              r="34"
              fill="none"
              stroke={isMac ? pressureColor : color}
              strokeWidth="8"
              strokeDasharray={`${(pct / 100) * 2 * Math.PI * 34} ${2 * Math.PI * 34}`}
              strokeLinecap="round"
              transform="rotate(-90 40 40)"
              style={{ transition: 'stroke-dasharray 0.6s ease' }}
            />
          </svg>
          <div
            style={{
              position: 'absolute',
              inset: 0,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <span
              style={{
                fontSize: '18px',
                fontWeight: 700,
                color: isMac ? pressureColor : color,
                fontFamily: 'var(--font-mono)',
              }}
            >
              {pct.toFixed(0)}%
            </span>
            {isMac && (
              <span
                style={{
                  fontSize: '8px',
                  color: pressureColor,
                  fontWeight: 600,
                  textTransform: 'uppercase',
                }}
              >
                {pressureLabel}
              </span>
            )}
          </div>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', flex: 1, minWidth: 0 }}>
          {isMac ? (
            <>
              {/* macOS-style stacked memory bar */}
              <div
                style={{
                  height: '10px',
                  borderRadius: '5px',
                  overflow: 'hidden',
                  display: 'flex',
                  background: 'var(--border)',
                }}
              >
                <div
                  style={{
                    width: `${appPct}%`,
                    background: 'var(--signal-warning)',
                    transition: 'width 0.6s',
                  }}
                  title={`App+Wired: ${formatBytes(appAndWired)}`}
                />
                <div
                  style={{
                    width: `${compPct}%`,
                    background: 'var(--text-secondary)',
                    transition: 'width 0.6s',
                  }}
                  title={`Compressed: ${formatBytes(compressed)}`}
                />
                <div
                  style={{
                    width: `${cachedPct}%`,
                    background: 'var(--accent)',
                    transition: 'width 0.6s',
                    opacity: 0.5,
                  }}
                  title={`Cached: ${formatBytes(cached)}`}
                />
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px', marginTop: '2px' }}>
                <span
                  style={{
                    fontSize: '10px',
                    color: 'var(--signal-warning)',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '3px',
                  }}
                >
                  <span
                    style={{
                      width: '6px',
                      height: '6px',
                      borderRadius: '50%',
                      background: 'var(--signal-warning)',
                      display: 'inline-block',
                    }}
                  />
                  App {formatBytes(appAndWired)}
                </span>
                <span
                  style={{
                    fontSize: '10px',
                    color: 'var(--text-secondary)',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '3px',
                  }}
                >
                  <span
                    style={{
                      width: '6px',
                      height: '6px',
                      borderRadius: '50%',
                      background: 'var(--text-secondary)',
                      display: 'inline-block',
                    }}
                  />
                  Compressed {formatBytes(compressed)}
                </span>
                <span
                  style={{
                    fontSize: '10px',
                    color: 'var(--text-muted)',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '3px',
                  }}
                >
                  <span
                    style={{
                      width: '6px',
                      height: '6px',
                      borderRadius: '50%',
                      background: 'var(--accent)',
                      display: 'inline-block',
                      opacity: 0.5,
                    }}
                  />
                  Cached {formatBytes(cached)}
                </span>
              </div>
              <span style={{ fontSize: '11px', color: 'var(--text-muted)', marginTop: '2px' }}>
                {formatBytes(metrics.memory_used_bytes)} of {formatBytes(total)} used
              </span>
            </>
          ) : (
            <>
              <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
                Used:{' '}
                <span style={{ color: 'var(--text-emphasis)', fontFamily: 'var(--font-mono)' }}>
                  {formatBytes(metrics.memory_used_bytes)}
                </span>
              </span>
              <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
                Total:{' '}
                <span style={{ color: 'var(--text-emphasis)', fontFamily: 'var(--font-mono)' }}>
                  {formatBytes(total)}
                </span>
              </span>
              <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
                Available:{' '}
                <span style={{ color: 'var(--text-emphasis)', fontFamily: 'var(--font-mono)' }}>
                  {formatBytes(metrics.memory_available_bytes)}
                </span>
              </span>
            </>
          )}
        </div>
      </div>
    </LiveStatCard>
  );
}

function GPUUsageCard({ metrics }: { metrics: LiveMetrics }) {
  const pct = metrics.gpu_usage_pct ?? 0;
  if (pct <= 0) return null;
  const color = usageColor(pct);
  return (
    <LiveStatCard title="GPU Usage">
      <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
        <div style={{ position: 'relative', width: '80px', height: '80px', flexShrink: 0 }}>
          <svg width="80" height="80" viewBox="0 0 80 80">
            <circle cx="40" cy="40" r="34" fill="none" stroke="var(--ring-track)" strokeWidth="8" />
            <circle
              cx="40"
              cy="40"
              r="34"
              fill="none"
              stroke={color}
              strokeWidth="8"
              strokeDasharray={`${(pct / 100) * 2 * Math.PI * 34} ${2 * Math.PI * 34}`}
              strokeLinecap="round"
              transform="rotate(-90 40 40)"
              style={{ transition: 'stroke-dasharray 0.6s ease' }}
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
              style={{ fontSize: '18px', fontWeight: 700, color, fontFamily: 'var(--font-mono)' }}
            >
              {pct.toFixed(0)}%
            </span>
          </div>
        </div>
      </div>
    </LiveStatCard>
  );
}

function LoadCard({ metrics }: { metrics: LiveMetrics }) {
  const cores = metrics.cpu_per_core?.length || 1;
  const loads = [
    { label: '1m', value: metrics.load_avg_1 },
    { label: '5m', value: metrics.load_avg_5 },
    { label: '15m', value: metrics.load_avg_15 },
  ];
  return (
    <LiveStatCard title="System Load">
      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        {loads.map((l) => {
          const pct = Math.min(100, (l.value / cores) * 100);
          return (
            <div key={l.label}>
              <div
                style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '4px' }}
              >
                <span style={{ fontSize: '12px', color: 'var(--text-muted)' }}>{l.label}</span>
                <span
                  style={{
                    fontSize: '12px',
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-emphasis)',
                  }}
                >
                  {l.value.toFixed(2)}
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
                    background: usageColor(pct),
                    width: `${pct}%`,
                    transition: 'width 0.6s ease',
                  }}
                />
              </div>
            </div>
          );
        })}
        <span style={{ fontSize: '11px', color: 'var(--text-muted)', marginTop: '2px' }}>
          {cores} cores — load / cores shown
        </span>
      </div>
    </LiveStatCard>
  );
}

function SystemCard({ metrics }: { metrics: LiveMetrics }) {
  return (
    <LiveStatCard title="System">
      <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
        <div>
          <span
            style={{ fontSize: '11px', color: 'var(--text-muted)', textTransform: 'uppercase' }}
          >
            Uptime
          </span>
          <div
            style={{
              fontSize: '22px',
              fontWeight: 700,
              color: 'var(--text-emphasis)',
              fontFamily: 'var(--font-mono)',
            }}
          >
            {formatUptime(metrics.uptime_seconds)}
          </div>
        </div>
        <div>
          <span
            style={{ fontSize: '11px', color: 'var(--text-muted)', textTransform: 'uppercase' }}
          >
            Processes
          </span>
          <div
            style={{
              fontSize: '22px',
              fontWeight: 700,
              color: 'var(--text-emphasis)',
              fontFamily: 'var(--font-mono)',
            }}
          >
            {metrics.process_count}
          </div>
        </div>
        {metrics.swap_total_bytes > 0 && (
          <div>
            <span
              style={{ fontSize: '11px', color: 'var(--text-muted)', textTransform: 'uppercase' }}
            >
              Swap
            </span>
            <div
              style={{
                fontSize: '13px',
                color: 'var(--text-secondary)',
                fontFamily: 'var(--font-mono)',
              }}
            >
              {formatBytes(metrics.swap_used_bytes)} / {formatBytes(metrics.swap_total_bytes)}
            </div>
          </div>
        )}
      </div>
    </LiveStatCard>
  );
}

function CoreGrid({ metrics }: { metrics: LiveMetrics }) {
  const cores = metrics.cpu_per_core;
  if (!cores || cores.length === 0) return null;

  // Calculate columns: aim for roughly 8 columns, adjust for core count
  const cols =
    cores.length <= 8 ? cores.length : Math.min(16, Math.ceil(Math.sqrt(cores.length * 2)));

  return (
    <div ref={(el) => hoverHandlers(el)} style={{ ...CARD_STYLE, ...CARD_PAD_STYLE }}>
      <p style={CARD_TITLE_STYLE}>CPU Core Usage</p>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: `repeat(${cols}, 1fr)`,
          gap: '4px',
        }}
      >
        {cores.map((core) => {
          const color = usageColor(core.usage_pct);
          return (
            <div
              key={core.core_id}
              title={`Core ${core.core_id}: ${core.usage_pct.toFixed(1)}% @ ${core.freq_mhz} MHz`}
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                padding: '4px 2px',
                borderRadius: '4px',
                background: `${color}15`,
                border: `1px solid ${color}30`,
                cursor: 'default',
                minWidth: 0,
              }}
            >
              <span style={{ fontSize: '9px', color: 'var(--text-muted)' }}>{core.core_id}</span>
              <div
                style={{
                  width: '100%',
                  height: '16px',
                  borderRadius: '2px',
                  background: 'var(--border)',
                  overflow: 'hidden',
                  marginTop: '2px',
                  position: 'relative',
                }}
              >
                <div
                  style={{
                    position: 'absolute',
                    bottom: 0,
                    left: 0,
                    right: 0,
                    height: `${Math.max(1, core.usage_pct)}%`,
                    background: color,
                    transition: 'height 0.6s ease',
                    borderRadius: '2px',
                  }}
                />
              </div>
              <span
                style={{ fontSize: '9px', color, fontFamily: 'var(--font-mono)', marginTop: '2px' }}
              >
                {core.usage_pct.toFixed(0)}%
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function DiskIOCard({ metrics }: { metrics: LiveMetrics }) {
  const disks = metrics.disk_io;
  if (!disks || disks.length === 0) return null;

  return (
    <div ref={(el) => hoverHandlers(el)} style={{ ...CARD_STYLE, ...CARD_PAD_STYLE }}>
      <p style={CARD_TITLE_STYLE}>Disk I/O</p>
      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        {disks.map((d) => (
          <div key={d.device}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
              <span
                style={{
                  fontSize: '12px',
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-emphasis)',
                }}
              >
                {d.device}
              </span>
              {d.io_util_pct > 0 && (
                <span
                  style={{
                    fontSize: '10px',
                    fontFamily: 'var(--font-mono)',
                    color: usageColor(d.io_util_pct),
                    padding: '1px 4px',
                    borderRadius: '2px',
                    background: `${usageColor(d.io_util_pct)}15`,
                  }}
                >
                  {d.io_util_pct.toFixed(0)}% util
                </span>
              )}
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px' }}>
              <div>
                <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>Read</span>
                <div
                  style={{
                    fontSize: '13px',
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--accent)',
                  }}
                >
                  {formatRate(d.read_bytes_per_sec)}
                </div>
              </div>
              <div>
                <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>Write</span>
                <div
                  style={{
                    fontSize: '13px',
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--signal-warning)',
                  }}
                >
                  {formatRate(d.write_bytes_per_sec)}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function NetworkIOCard({ metrics }: { metrics: LiveMetrics }) {
  const nets = metrics.network_io;
  if (!nets || nets.length === 0) return null;

  // Filter out virtual Docker interfaces (veth*, br-*, docker*)
  const physical = nets.filter(
    (n) =>
      !n.interface.startsWith('veth') &&
      !n.interface.startsWith('br-') &&
      n.interface !== 'docker0' &&
      n.interface !== 'lo',
  );
  const virtualCount = nets.length - physical.length;

  return (
    <div ref={(el) => hoverHandlers(el)} style={{ ...CARD_STYLE, ...CARD_PAD_STYLE }}>
      <p style={CARD_TITLE_STYLE}>
        Network I/O
        {virtualCount > 0 && (
          <span
            style={{
              fontSize: '10px',
              color: 'var(--text-muted)',
              fontWeight: 400,
              marginLeft: '8px',
            }}
          >
            ({virtualCount} virtual hidden)
          </span>
        )}
      </p>
      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        {physical.map((n) => (
          <div key={n.interface}>
            <span
              style={{
                fontSize: '12px',
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-emphasis)',
                marginBottom: '6px',
                display: 'block',
              }}
            >
              {n.interface}
            </span>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px' }}>
              <div>
                <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>RX</span>
                <div
                  style={{
                    fontSize: '13px',
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--signal-healthy)',
                  }}
                >
                  {formatRate(n.rx_bytes_per_sec)}
                </div>
              </div>
              <div>
                <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>TX</span>
                <div
                  style={{
                    fontSize: '13px',
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-secondary)',
                  }}
                >
                  {formatRate(n.tx_bytes_per_sec)}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function LiveMonitorSection({ metrics }: { metrics: LiveMetrics }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <p style={{ ...CARD_TITLE_STYLE, marginBottom: 0, paddingLeft: '4px' }}>
          Live System Monitor
        </p>
        <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Auto-refresh 3s</span>
      </div>

      {/* Row 1: Stat cards — GPUUsageCard renders only when gpu_usage_pct > 0 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
        <CPUUsageCard metrics={metrics} />
        <MemoryUsageCard metrics={metrics} />
        <GPUUsageCard metrics={metrics} />
        <LoadCard metrics={metrics} />
        <SystemCard metrics={metrics} />
      </div>

      {/* Row 2: Core Grid */}
      <CoreGrid metrics={metrics} />

      {/* Row 3: I/O Monitoring */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
        <DiskIOCard metrics={metrics} />
        <NetworkIOCard metrics={metrics} />
      </div>
    </div>
  );
}

function MetricsError({ refetch }: { refetch: () => void }) {
  return (
    <div style={{ ...CARD_STYLE }}>
      <ErrorState
        title="Live metrics unavailable"
        message="Hardware inventory shown below."
        onRetry={refetch}
      />
    </div>
  );
}

function MetricsSkeleton() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
        <Skeleton className="h-36" />
        <Skeleton className="h-36" />
        <Skeleton className="h-36" />
        <Skeleton className="h-36" />
      </div>
      <Skeleton className="h-24 w-full" />
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
        <Skeleton className="h-28" />
        <Skeleton className="h-28" />
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// HARDWARE INVENTORY SECTION
// ---------------------------------------------------------------------------

function CPUCard({ data }: { data: HardwareInfo }) {
  const cpu = data.cpu;
  const presentFlags = (cpu.flags || []).map((f) => f.toLowerCase());
  const capBadges = IMPORTANT_FLAGS.filter((f) => presentFlags.includes(f));

  const isApple = data.motherboard.board_manufacturer === 'Apple';
  const virtLabel = data.virtualization.is_virtual
    ? data.virtualization.hypervisor_type || 'Virtual'
    : isApple
      ? 'Apple Silicon'
      : 'Bare Metal';

  const cacheEntries = [
    { level: 'L1d', val: cpu.cache_l1d, color: 'var(--accent)' },
    { level: 'L1i', val: cpu.cache_l1i, color: 'var(--signal-healthy)' },
    { level: 'L2', val: cpu.cache_l2, color: 'var(--text-secondary)' },
    { level: 'L3', val: cpu.cache_l3, color: 'var(--text-muted)' },
  ].filter((c) => c.val);

  // Frequency range bar calculation
  const freqMin = cpu.min_mhz > 0 ? cpu.min_mhz : 0;
  const freqMax = cpu.max_mhz > 0 ? cpu.max_mhz : 0;
  // Scale bar: min starts at some offset, max fills
  const freqBarMinPct = freqMax > 0 ? Math.round((freqMin / freqMax) * 100) : 0;

  return (
    <div ref={(el) => hoverHandlers(el)} style={{ ...CARD_STYLE, ...CARD_PAD_STYLE }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '12px' }}>
        <p style={{ ...CARD_TITLE_STYLE, marginBottom: 0 }}>CPU</p>
        {isApple && !data.virtualization.is_virtual ? (
          <span
            style={{
              fontSize: '10px',
              padding: '2px 8px',
              borderRadius: 'var(--radius-xl)',
              background: 'linear-gradient(135deg, var(--accent), var(--signal-healthy))',
              color: 'white',
              fontWeight: 600,
              letterSpacing: '0.05em',
            }}
          >
            Apple Silicon
          </span>
        ) : (
          <span
            style={{
              fontSize: '10px',
              padding: '2px 6px',
              borderRadius: '4px',
              border: data.virtualization.is_virtual
                ? 'color-mix(in srgb, var(--signal-warning) 30%, transparent)'
                : 'var(--border)',
              background: data.virtualization.is_virtual
                ? 'color-mix(in srgb, var(--signal-warning) 10%, transparent)'
                : 'transparent',
              color: data.virtualization.is_virtual
                ? 'var(--signal-warning)'
                : 'var(--text-secondary)',
            }}
          >
            {virtLabel}
          </span>
        )}
      </div>

      {/* Processor name */}
      <Row label="Processor" value={cpu.model_name || 'N/A'} />
      <Row label="Architecture" value={cpu.architecture || 'N/A'} />

      {/* Topology visualization */}
      <div
        style={{
          margin: '12px 0',
          padding: '12px',
          background: 'var(--bg-inset)',
          borderRadius: '8px',
        }}
      >
        <p
          style={{
            fontSize: '11px',
            color: 'var(--text-muted)',
            marginBottom: '10px',
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
          }}
        >
          Topology
        </p>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
          {[
            { label: 'Socket', value: cpu.sockets, icon: '\u25A3' },
            { label: 'Core', value: cpu.cores_per_socket, icon: '\u25C9' },
            { label: 'Thread', value: cpu.threads_per_core, icon: '\u2261' },
          ].map((item, idx) => (
            <div key={item.label} style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              {idx > 0 && (
                <span style={{ fontSize: '14px', color: 'var(--text-muted)', fontWeight: 700 }}>
                  &times;
                </span>
              )}
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '6px',
                  padding: '6px 12px',
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: '6px',
                }}
              >
                <span style={{ fontSize: '16px', color: 'var(--accent)' }}>{item.icon}</span>
                <div>
                  <span
                    style={{
                      fontSize: '18px',
                      fontWeight: 700,
                      color: 'var(--text-emphasis)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    {item.value}
                  </span>
                  <span style={{ fontSize: '11px', color: 'var(--text-muted)', marginLeft: '4px' }}>
                    {item.label}
                    {item.value !== 1 ? 's' : ''}
                  </span>
                </div>
              </div>
            </div>
          ))}
          <span style={{ fontSize: '14px', color: 'var(--text-muted)', fontWeight: 700 }}>=</span>
          <div
            style={{
              padding: '6px 12px',
              background: 'color-mix(in srgb, var(--accent) 15%, transparent)',
              border: 'color-mix(in srgb, var(--accent) 30%, transparent)',
              borderRadius: '6px',
            }}
          >
            <span
              style={{
                fontSize: '18px',
                fontWeight: 700,
                color: 'var(--accent)',
                fontFamily: 'var(--font-mono)',
              }}
            >
              {cpu.total_logical_cpus}
            </span>
            <span style={{ fontSize: '11px', color: 'var(--text-muted)', marginLeft: '4px' }}>
              Logical CPUs
            </span>
          </div>
        </div>
      </div>

      {/* Frequency range bar */}
      {freqMax > 0 && (
        <div style={{ margin: '12px 0' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '6px' }}>
            <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Frequency Range</span>
            <span
              style={{
                fontSize: '12px',
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-emphasis)',
              }}
            >
              {freqMin > 0 ? `${freqMin.toFixed(0)}` : '?'} - {freqMax.toFixed(0)} MHz
            </span>
          </div>
          <div
            style={{
              height: '10px',
              borderRadius: '5px',
              background: 'var(--border)',
              overflow: 'hidden',
              position: 'relative',
            }}
          >
            {/* Base frequency indicator */}
            {freqMin > 0 && (
              <div
                style={{
                  position: 'absolute',
                  left: `${freqBarMinPct}%`,
                  top: 0,
                  bottom: 0,
                  width: '2px',
                  background: 'var(--signal-warning)',
                  zIndex: 2,
                }}
                title={`Base: ${freqMin.toFixed(0)} MHz`}
              />
            )}
            <div
              style={{
                height: '100%',
                borderRadius: '5px',
                background: 'linear-gradient(90deg, var(--accent), var(--accent))',
                width: '100%',
              }}
            />
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '4px' }}>
            <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>0 MHz</span>
            {freqMin > 0 && (
              <span
                style={{
                  fontSize: '10px',
                  color: 'var(--signal-warning)',
                  position: 'relative',
                  left: `${freqBarMinPct - 50}%`,
                }}
              >
                Base {freqMin.toFixed(0)}
              </span>
            )}
            <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>
              Max {freqMax.toFixed(0)}
            </span>
          </div>
        </div>
      )}

      {/* Cache hierarchy visual diagram */}
      {cacheEntries.length > 0 && (
        <div style={{ margin: '12px 0' }}>
          <p
            style={{
              fontSize: '11px',
              color: 'var(--text-muted)',
              marginBottom: '8px',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
            }}
          >
            Cache Hierarchy
          </p>
          <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
            {cacheEntries.map((c, idx) => (
              <div key={c.level} style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                {idx > 0 && (
                  <span style={{ fontSize: '12px', color: 'var(--text-muted)' }}>&rarr;</span>
                )}
                <div
                  style={{
                    padding: '8px 14px',
                    background: `${c.color}12`,
                    border: `1px solid ${c.color}35`,
                    borderRadius: '6px',
                    textAlign: 'center',
                    minWidth: '70px',
                  }}
                >
                  <div
                    style={{
                      fontSize: '11px',
                      fontWeight: 600,
                      color: c.color,
                      marginBottom: '2px',
                    }}
                  >
                    {c.level}
                  </div>
                  <div
                    style={{
                      fontSize: '12px',
                      fontFamily: 'var(--font-mono)',
                      color: 'var(--text-emphasis)',
                    }}
                  >
                    {c.val}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Capability badges */}
      {capBadges.length > 0 && (
        <div style={{ marginTop: '12px' }}>
          <p
            style={{
              fontSize: '11px',
              color: 'var(--text-muted)',
              marginBottom: '6px',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
            }}
          >
            Capabilities
          </p>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
            {capBadges.map((flag) => (
              <span
                key={flag}
                style={{
                  fontSize: '11px',
                  padding: '3px 8px',
                  borderRadius: '4px',
                  border: 'color-mix(in srgb, var(--accent) 30%, transparent)',
                  background: 'color-mix(in srgb, var(--accent) 10%, transparent)',
                  color: 'var(--accent)',
                  fontFamily: 'var(--font-mono)',
                  fontWeight: 500,
                }}
              >
                {FLAG_LABELS[flag] || flag.toUpperCase()}
              </span>
            ))}
          </div>
        </div>
      )}

      {isApple && capBadges.length === 0 && (
        <div style={{ marginTop: '12px' }}>
          <p style={{ fontSize: '11px', color: 'var(--text-muted)', marginBottom: '6px' }}>
            Capabilities
          </p>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
            {[
              'Neural Engine',
              'GPU Cores',
              'Media Engine',
              'Secure Enclave',
              'ProRes',
              'Thunderbolt',
            ].map((cap) => (
              <span
                key={cap}
                style={{
                  fontSize: '11px',
                  padding: '3px 8px',
                  borderRadius: '4px',
                  border: 'color-mix(in srgb, var(--accent) 30%, transparent)',
                  background: 'color-mix(in srgb, var(--accent) 10%, transparent)',
                  color: 'var(--accent)',
                  fontFamily: 'var(--font-mono)',
                  fontWeight: 500,
                }}
              >
                {cap}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function MemoryCard({ data }: { data: HardwareInfo }) {
  const mem = data.memory;
  const allDimms = mem.dimms || [];
  const populatedDimms = allDimms.filter((d) => d.size_mb > 0 || (d.type && d.manufacturer));
  const totalSlots = mem.num_slots || allDimms.length || 0;

  // Build slot array: populated slots + empty slots to fill totalSlots
  const slotCards: Array<{ populated: boolean; dimm?: (typeof allDimms)[0]; index: number }> = [];
  for (let i = 0; i < Math.max(totalSlots, allDimms.length); i++) {
    if (
      i < allDimms.length &&
      (allDimms[i].size_mb > 0 || (allDimms[i].type && allDimms[i].manufacturer))
    ) {
      slotCards.push({ populated: true, dimm: allDimms[i], index: i });
    } else {
      slotCards.push({ populated: false, index: i });
    }
  }

  // Derive memory type summary
  const memType = populatedDimms.length > 0 ? populatedDimms[0].type : '';
  const memSpeed = populatedDimms.length > 0 ? populatedDimms[0].speed_mhz : 0;
  const channelCount = populatedDimms.length;
  const channelLabel =
    channelCount >= 4
      ? 'Quad Channel'
      : channelCount >= 2
        ? 'Dual Channel'
        : channelCount === 1
          ? 'Single Channel'
          : '';
  const typeSummary = [memType, memSpeed > 0 ? `${memSpeed} MHz` : '', channelLabel]
    .filter(Boolean)
    .join(' / ');

  // Memory usage bar from available_bytes
  const usedBytes =
    mem.total_bytes > 0 && mem.available_bytes > 0 ? mem.total_bytes - mem.available_bytes : 0;
  const usedPct = mem.total_bytes > 0 ? (usedBytes / mem.total_bytes) * 100 : 0;
  const availPct = mem.total_bytes > 0 ? (mem.available_bytes / mem.total_bytes) * 100 : 0;

  return (
    <div ref={(el) => hoverHandlers(el)} style={{ ...CARD_STYLE, ...CARD_PAD_STYLE }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '12px' }}>
        <p style={{ ...CARD_TITLE_STYLE, marginBottom: 0 }}>Memory</p>
        {data.motherboard.board_manufacturer === 'Apple' && (
          <span
            style={{
              fontSize: '10px',
              padding: '2px 8px',
              borderRadius: 'var(--radius-xl)',
              background: 'linear-gradient(135deg, var(--accent), var(--signal-healthy))',
              color: 'white',
              fontWeight: 600,
            }}
          >
            Unified Memory
          </span>
        )}
      </div>

      {/* Type summary line */}
      {typeSummary && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            marginBottom: '12px',
            padding: '8px 12px',
            background: 'var(--bg-inset)',
            borderRadius: '6px',
          }}
        >
          <span style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-emphasis)' }}>
            {formatBytes(mem.total_bytes)}
          </span>
          <span style={{ fontSize: '12px', color: 'var(--text-muted)' }}>&mdash;</span>
          <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>{typeSummary}</span>
        </div>
      )}

      {/* Memory usage bar */}
      {mem.total_bytes > 0 && mem.available_bytes > 0 && (
        <div style={{ marginBottom: '14px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '6px' }}>
            <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Memory Usage</span>
            <span
              style={{
                fontSize: '11px',
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-emphasis)',
              }}
            >
              {formatBytes(usedBytes)} used / {formatBytes(mem.available_bytes)} available
            </span>
          </div>
          <div
            style={{
              height: '12px',
              borderRadius: '6px',
              background: 'var(--border)',
              overflow: 'hidden',
              display: 'flex',
            }}
          >
            <div
              style={{
                height: '100%',
                width: `${usedPct}%`,
                background: usageColor(usedPct),
                transition: 'width 0.6s ease',
              }}
              title={`Used: ${usedPct.toFixed(1)}%`}
            />
            <div
              style={{
                height: '100%',
                width: `${availPct}%`,
                background: 'color-mix(in srgb, var(--signal-healthy) 25%, transparent)',
                transition: 'width 0.6s ease',
              }}
              title={`Available: ${availPct.toFixed(1)}%`}
            />
          </div>
          <div style={{ display: 'flex', gap: '16px', marginTop: '4px' }}>
            <span
              style={{
                fontSize: '10px',
                color: 'var(--text-muted)',
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
              }}
            >
              <span
                style={{
                  width: '8px',
                  height: '8px',
                  borderRadius: '2px',
                  background: usageColor(usedPct),
                  display: 'inline-block',
                }}
              />
              Used ({usedPct.toFixed(1)}%)
            </span>
            <span
              style={{
                fontSize: '10px',
                color: 'var(--text-muted)',
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
              }}
            >
              <span
                style={{
                  width: '8px',
                  height: '8px',
                  borderRadius: '2px',
                  background: 'color-mix(in srgb, var(--signal-healthy) 25%, transparent)',
                  display: 'inline-block',
                }}
              />
              Available ({availPct.toFixed(1)}%)
            </span>
          </div>
        </div>
      )}

      <Row
        label={data.motherboard.board_manufacturer === 'Apple' ? 'Memory Type' : 'Slots'}
        value={
          data.motherboard.board_manufacturer === 'Apple'
            ? populatedDimms.length > 0 && populatedDimms[0].manufacturer
              ? `${populatedDimms[0].manufacturer} ${populatedDimms[0].type || ''}`
              : 'Unified'
            : `${populatedDimms.length} populated / ${totalSlots} total`
        }
      />
      {mem.max_capacity && <Row label="Max Capacity" value={mem.max_capacity} />}
      {mem.error_correction && <Row label="ECC" value={mem.error_correction} />}

      {/* DIMM Slot visual diagram */}
      {totalSlots > 0 && (
        <div style={{ marginTop: '14px' }}>
          <p
            style={{
              fontSize: '11px',
              color: 'var(--text-muted)',
              marginBottom: '8px',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
            }}
          >
            DIMM Slots
          </p>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: `repeat(${Math.min(totalSlots, 4)}, 1fr)`,
              gap: '8px',
            }}
          >
            {slotCards.map((slot) => (
              <div
                key={slot.index}
                style={{
                  padding: '10px',
                  borderRadius: '8px',
                  border: slot.populated
                    ? 'color-mix(in srgb, var(--accent) 30%, transparent)'
                    : '1px dashed var(--border)',
                  background: slot.populated
                    ? 'color-mix(in srgb, var(--accent) 8%, transparent)'
                    : 'var(--bg-inset)',
                  minHeight: '90px',
                  display: 'flex',
                  flexDirection: 'column',
                  justifyContent: slot.populated ? 'flex-start' : 'center',
                  alignItems: slot.populated ? 'stretch' : 'center',
                }}
              >
                {slot.populated && slot.dimm ? (
                  <>
                    <div
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        marginBottom: '6px',
                      }}
                    >
                      <span
                        style={{
                          fontSize: '11px',
                          fontFamily: 'var(--font-mono)',
                          color: 'var(--text-secondary)',
                        }}
                      >
                        {slot.dimm.locator || `Slot ${slot.index + 1}`}
                      </span>
                      <span
                        style={{
                          fontSize: '12px',
                          fontWeight: 700,
                          color: 'var(--text-emphasis)',
                          fontFamily: 'var(--font-mono)',
                        }}
                      >
                        {slot.dimm.size_mb > 0
                          ? slot.dimm.size_mb >= 1024
                            ? `${(slot.dimm.size_mb / 1024).toFixed(0)} GB`
                            : `${slot.dimm.size_mb} MB`
                          : mem.total_bytes > 0
                            ? formatBytes(mem.total_bytes)
                            : 'Unified'}
                      </span>
                    </div>
                    <div
                      style={{
                        fontSize: '11px',
                        color: 'var(--text-secondary)',
                        display: 'flex',
                        flexDirection: 'column',
                        gap: '2px',
                      }}
                    >
                      {slot.dimm.type && <span>{slot.dimm.type}</span>}
                      {slot.dimm.speed_mhz > 0 && <span>{slot.dimm.speed_mhz} MHz</span>}
                      {slot.dimm.manufacturer && (
                        <span style={{ color: 'var(--text-muted)', fontSize: '10px' }}>
                          {slot.dimm.manufacturer}
                        </span>
                      )}
                    </div>
                  </>
                ) : (
                  <>
                    <div
                      style={{
                        width: '28px',
                        height: '28px',
                        borderRadius: '6px',
                        border: '1px dashed var(--border)',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        marginBottom: '4px',
                      }}
                    >
                      <span style={{ fontSize: '14px', color: 'var(--border)' }}>+</span>
                    </div>
                    <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Empty</span>
                  </>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function partitionUsageColor(pct: number): string {
  if (pct < 50) return 'var(--signal-healthy)';
  if (pct <= 80) return 'var(--signal-warning)';
  return 'var(--signal-critical)';
}

// Color palette for stacked partition bar (uses design tokens for consistency)
const PARTITION_COLORS = [
  'var(--accent)',
  'var(--signal-healthy)',
  'var(--signal-warning)',
  'var(--text-secondary)',
  'var(--text-muted)',
  'var(--text-faint)',
  'var(--border-strong)',
  'var(--border-hover)',
  'var(--border)',
  'var(--border-faint)',
];

function StorageCard({ device }: { device: StorageDevice }) {
  const partitions = device.partitions || [];
  const totalDiskBytes = device.size_bytes;

  return (
    <div ref={(el) => hoverHandlers(el)} style={{ ...CARD_STYLE, ...CARD_PAD_STYLE }}>
      {/* Drive header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          marginBottom: '4px',
          flexWrap: 'wrap',
        }}
      >
        <p style={{ ...CARD_TITLE_STYLE, marginBottom: 0 }}>
          {device.model || device.name || 'Storage Device'}
        </p>
        {device.type && <TypeBadge type={device.type} />}
        {device.smart_status && <SmartBadge status={device.smart_status} />}
        {device.transport && (
          <span
            style={{
              fontSize: '10px',
              padding: '2px 6px',
              borderRadius: '4px',
              border: '1px solid var(--border)',
              color: 'var(--text-secondary)',
              fontFamily: 'var(--font-mono)',
            }}
          >
            {device.transport}
          </span>
        )}
      </div>

      {/* Drive details row */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '16px',
          marginBottom: '12px',
          flexWrap: 'wrap',
        }}
      >
        <span
          style={{
            fontSize: '13px',
            color: 'var(--text-emphasis)',
            fontWeight: 600,
            fontFamily: 'var(--font-mono)',
          }}
        >
          {formatBytes(device.size_bytes)}
        </span>
        {device.serial && (
          <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
            S/N:{' '}
            <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)' }}>
              {device.serial}
            </span>
          </span>
        )}
        {device.firmware_version && (
          <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
            FW:{' '}
            <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)' }}>
              {device.firmware_version}
            </span>
          </span>
        )}
        {device.temperature_celsius > 0 && (
          <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
            Temp:{' '}
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                color:
                  device.temperature_celsius > 60
                    ? 'var(--signal-critical)'
                    : 'var(--text-primary)',
              }}
            >
              {device.temperature_celsius}&deg;C
            </span>
          </span>
        )}
      </div>

      {/* Stacked disk allocation bar */}
      {partitions.length > 0 && totalDiskBytes > 0 && (
        <div style={{ marginBottom: '14px' }}>
          <p
            style={{
              fontSize: '11px',
              color: 'var(--text-muted)',
              marginBottom: '6px',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
            }}
          >
            Disk Allocation
          </p>
          <div
            style={{
              height: '14px',
              borderRadius: '7px',
              background: 'var(--border)',
              overflow: 'hidden',
              display: 'flex',
            }}
          >
            {partitions.map((p, idx) => {
              const widthPct = Math.max(0.5, (p.size_bytes / totalDiskBytes) * 100);
              return (
                <div
                  key={p.name}
                  title={`${p.name}: ${formatBytes(p.size_bytes)} (${p.fstype || 'unknown'})`}
                  style={{
                    height: '100%',
                    width: `${widthPct}%`,
                    background: PARTITION_COLORS[idx % PARTITION_COLORS.length],
                    opacity: 0.8,
                    borderRight: idx < partitions.length - 1 ? '1px solid var(--bg-page)' : 'none',
                  }}
                />
              );
            })}
          </div>
          {/* Legend */}
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', marginTop: '6px' }}>
            {partitions.map((p, idx) => (
              <span
                key={p.name}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                  fontSize: '10px',
                  color: 'var(--text-secondary)',
                }}
              >
                <span
                  style={{
                    width: '8px',
                    height: '8px',
                    borderRadius: '2px',
                    background: PARTITION_COLORS[idx % PARTITION_COLORS.length],
                    display: 'inline-block',
                    flexShrink: 0,
                  }}
                />
                {p.name}
              </span>
            ))}
            {(() => {
              const allocatedBytes = partitions.reduce((sum, p) => sum + p.size_bytes, 0);
              const unallocated = totalDiskBytes - allocatedBytes;
              if (unallocated > totalDiskBytes * 0.01) {
                return (
                  <span
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '4px',
                      fontSize: '10px',
                      color: 'var(--text-muted)',
                    }}
                  >
                    <span
                      style={{
                        width: '8px',
                        height: '8px',
                        borderRadius: '2px',
                        background: 'var(--border)',
                        display: 'inline-block',
                        flexShrink: 0,
                      }}
                    />
                    Unallocated ({formatBytes(unallocated)})
                  </span>
                );
              }
              return null;
            })()}
          </div>
        </div>
      )}

      {/* Partition table */}
      {partitions.length > 0 && (
        <div>
          <p
            style={{
              fontSize: '11px',
              color: 'var(--text-muted)',
              marginBottom: '8px',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
            }}
          >
            Partitions
          </p>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '12px' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)' }}>
                  {['Partition', 'Filesystem', 'Mount Point', 'Size', 'Usage'].map((h) => (
                    <th
                      key={h}
                      style={{
                        textAlign: 'left',
                        padding: '6px 8px',
                        color: 'var(--text-muted)',
                        fontWeight: 600,
                        fontSize: '10px',
                        textTransform: 'uppercase',
                        letterSpacing: '0.05em',
                      }}
                    >
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {partitions.map((p) => (
                  <tr key={p.name} style={{ borderBottom: '1px solid var(--border-faint)' }}>
                    <td
                      style={{
                        padding: '6px 8px',
                        fontFamily: 'var(--font-mono)',
                        color: 'var(--text-emphasis)',
                        fontSize: '11px',
                      }}
                    >
                      {p.name}
                    </td>
                    <td style={{ padding: '6px 8px' }}>
                      {p.fstype ? (
                        <span
                          style={{
                            fontSize: '10px',
                            padding: '1px 6px',
                            borderRadius: '3px',
                            border: '1px solid var(--border)',
                            color: 'var(--text-secondary)',
                            fontFamily: 'var(--font-mono)',
                          }}
                        >
                          {p.fstype}
                        </span>
                      ) : (
                        <span style={{ color: 'var(--text-muted)', fontSize: '11px' }}>-</span>
                      )}
                    </td>
                    <td
                      style={{
                        padding: '6px 8px',
                        fontFamily: 'var(--font-mono)',
                        fontSize: '11px',
                      }}
                    >
                      {p.mountpoint ? (
                        <span style={{ color: 'var(--text-emphasis)' }}>{p.mountpoint}</span>
                      ) : (
                        <span style={{ color: 'var(--text-muted)', fontStyle: 'italic' }}>
                          not mounted
                        </span>
                      )}
                    </td>
                    <td
                      style={{
                        padding: '6px 8px',
                        fontFamily: 'var(--font-mono)',
                        color: 'var(--text-emphasis)',
                        fontSize: '11px',
                      }}
                    >
                      {formatBytes(p.size_bytes)}
                    </td>
                    <td style={{ padding: '6px 8px', minWidth: '140px' }}>
                      {p.usage_pct > 0 ? (
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <div
                            style={{
                              flex: 1,
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
                                width: `${p.usage_pct}%`,
                                background: partitionUsageColor(p.usage_pct),
                                transition: 'width 0.3s ease',
                              }}
                            />
                          </div>
                          <span
                            style={{
                              fontSize: '11px',
                              fontFamily: 'var(--font-mono)',
                              color: partitionUsageColor(p.usage_pct),
                              fontWeight: 600,
                              minWidth: '36px',
                              textAlign: 'right',
                            }}
                          >
                            {p.usage_pct.toFixed(0)}%
                          </span>
                        </div>
                      ) : (
                        <span style={{ color: 'var(--text-muted)', fontSize: '11px' }}>-</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

function GPUCard({ data }: { data: HardwareInfo }) {
  if (!data.gpu || data.gpu.length === 0) return null;
  return (
    <div ref={(el) => hoverHandlers(el)} style={{ ...CARD_STYLE, ...CARD_PAD_STYLE }}>
      <p style={CARD_TITLE_STYLE}>GPU</p>
      {data.gpu.map((gpu, i) => (
        <div
          key={gpu.pci_slot || i}
          style={{
            padding: '8px 0',
            borderBottom: i < data.gpu.length - 1 ? '1px solid var(--border-faint)' : 'none',
          }}
        >
          <Row label="Model" value={gpu.model || 'N/A'} />
          {gpu.vram_mb > 0 && (
            <Row
              label="VRAM"
              value={
                gpu.vram_mb >= 1024 ? `${(gpu.vram_mb / 1024).toFixed(1)} GB` : `${gpu.vram_mb} MB`
              }
            />
          )}
          {gpu.usage_pct !== undefined && <Row label="Usage" value={`${gpu.usage_pct}%`} />}
          {gpu.driver_version && <Row label="Driver" value={gpu.driver_version} />}
          {gpu.pci_slot && (
            <Row
              label="PCI Slot"
              value={
                <span style={{ fontFamily: 'var(--font-mono)', fontSize: '11px' }}>
                  {gpu.pci_slot}
                </span>
              }
            />
          )}
        </div>
      ))}
    </div>
  );
}

function MotherboardCard({ data }: { data: HardwareInfo }) {
  const mb = data.motherboard;
  return (
    <div ref={(el) => hoverHandlers(el)} style={{ ...CARD_STYLE, ...CARD_PAD_STYLE }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '12px' }}>
        <p style={{ ...CARD_TITLE_STYLE, marginBottom: 0 }}>Motherboard</p>
        {data.tpm.present ? (
          <span
            style={{
              fontSize: '10px',
              padding: '2px 6px',
              borderRadius: '4px',
              border: 'color-mix(in srgb, var(--signal-healthy) 30%, transparent)',
              background: 'color-mix(in srgb, var(--signal-healthy) 10%, transparent)',
              color: 'var(--signal-healthy)',
            }}
          >
            TPM {data.tpm.version || 'Present'}
          </span>
        ) : mb.board_manufacturer === 'Apple' ? (
          <span
            style={{
              fontSize: '10px',
              padding: '2px 8px',
              borderRadius: 'var(--radius-xl)',
              background: 'linear-gradient(135deg, var(--signal-healthy), var(--signal-healthy))',
              color: 'white',
              fontWeight: 600,
            }}
          >
            Secure Enclave
          </span>
        ) : null}
      </div>
      <Row label="Manufacturer" value={mb.board_manufacturer || 'N/A'} />
      <Row
        label={mb.board_manufacturer === 'Apple' ? 'Model Identifier' : 'Product'}
        value={mb.board_product || 'N/A'}
      />
      <Row
        label={mb.board_manufacturer === 'Apple' ? 'Firmware Vendor' : 'BIOS Vendor'}
        value={mb.bios_vendor || 'N/A'}
      />
      <Row
        label={mb.board_manufacturer === 'Apple' ? 'Boot ROM Version' : 'BIOS Version'}
        value={
          <span style={{ fontFamily: 'var(--font-mono)', fontSize: '11px' }}>
            {mb.bios_version || 'N/A'}
          </span>
        }
      />
      {mb.bios_release_date && (
        <Row
          label={mb.board_manufacturer === 'Apple' ? 'Firmware Date' : 'BIOS Date'}
          value={mb.bios_release_date}
        />
      )}
    </div>
  );
}

function PhysicalNetworkCard({ iface }: { iface: NetworkInfo }) {
  const isUp = ['up', 'UP', 'active', 'Active'].includes(iface.state);
  return (
    <div
      style={{
        padding: '12px',
        borderRadius: '8px',
        background: 'var(--bg-inset)',
        border: '1px solid var(--border-faint)',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '10px' }}>
        <StateDot up={isUp} />
        <span
          style={{
            fontSize: '13px',
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-emphasis)',
            fontWeight: 600,
          }}
        >
          {iface.name}
        </span>
        {iface.type && <TypeBadge type={iface.type} />}
      </div>
      <Row
        label="MAC"
        value={
          <span style={{ fontFamily: 'var(--font-mono)', fontSize: '11px' }}>
            {iface.mac_address || 'N/A'}
          </span>
        }
      />
      {iface.speed_mbps > 0 && <Row label="Speed" value={`${iface.speed_mbps} Mbps`} />}
      <Row label="MTU" value={iface.mtu} />
      {iface.driver && <Row label="Driver" value={iface.driver} />}
      {iface.ipv4_addresses && iface.ipv4_addresses.length > 0 && (
        <div style={{ marginTop: '6px' }}>
          <p style={{ fontSize: '11px', color: 'var(--text-muted)', marginBottom: '4px' }}>IPv4</p>
          {iface.ipv4_addresses.map((ip) => (
            <p
              key={ip.address}
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: '12px',
                color: 'var(--text-emphasis)',
              }}
            >
              {ip.address}/{ip.prefix_len}
            </p>
          ))}
        </div>
      )}
      {iface.ipv6_addresses && iface.ipv6_addresses.length > 0 && (
        <div style={{ marginTop: '6px' }}>
          <p style={{ fontSize: '11px', color: 'var(--text-muted)', marginBottom: '4px' }}>IPv6</p>
          {iface.ipv6_addresses.map((ip) => (
            <p
              key={ip.address}
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: '11px',
                color: 'var(--text-secondary)',
                wordBreak: 'break-all',
              }}
            >
              {ip.address}/{ip.prefix_len}
            </p>
          ))}
        </div>
      )}
    </div>
  );
}

function NetworkSection({ data }: { data: HardwareInfo }) {
  if (!data.network || data.network.length === 0) return null;

  const physical = data.network.filter((i) => isPhysicalInterface(i.name));
  const virtual = data.network.filter((i) => !isPhysicalInterface(i.name));

  const [virtualOpen, setVirtualOpen] = useState(false);

  return (
    <div ref={(el) => hoverHandlers(el)} style={{ ...CARD_STYLE, ...CARD_PAD_STYLE }}>
      <p style={CARD_TITLE_STYLE}>Network Interfaces</p>

      {/* Physical Interfaces */}
      {physical.length > 0 && (
        <div style={{ marginBottom: virtual.length > 0 ? '16px' : 0 }}>
          <p style={{ fontSize: '11px', color: 'var(--text-muted)', marginBottom: '8px' }}>
            Physical
          </p>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: physical.length === 1 ? '1fr' : '1fr 1fr',
              gap: '12px',
            }}
          >
            {physical.map((iface) => (
              <PhysicalNetworkCard key={iface.name} iface={iface} />
            ))}
          </div>
        </div>
      )}

      {/* Virtual Interfaces — collapsed */}
      {virtual.length > 0 && (
        <Collapsible open={virtualOpen} onOpenChange={setVirtualOpen}>
          <CollapsibleTrigger asChild>
            <button
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                padding: '8px 0',
                width: '100%',
              }}
            >
              {virtualOpen ? (
                <ChevronDown className="h-4 w-4" style={{ color: 'var(--text-muted)' }} />
              ) : (
                <ChevronRight className="h-4 w-4" style={{ color: 'var(--text-muted)' }} />
              )}
              <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Virtual</span>
              <Badge variant="secondary" className="text-[10px] px-2 py-0">
                {virtual.length} virtual interfaces
              </Badge>
            </button>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div
              style={{
                marginTop: '8px',
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))',
                gap: '4px',
              }}
            >
              {virtual.map((iface) => {
                const isUp = ['up', 'UP', 'active', 'Active'].includes(iface.state);
                return (
                  <div
                    key={iface.name}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '6px',
                      padding: '4px 8px',
                      borderRadius: '4px',
                      background: 'var(--bg-inset)',
                      fontSize: '11px',
                    }}
                  >
                    <StateDot up={isUp} />
                    <span
                      style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)' }}
                    >
                      {iface.name}
                    </span>
                  </div>
                );
              })}
            </div>
          </CollapsibleContent>
        </Collapsible>
      )}
    </div>
  );
}

function USBCard({ data }: { data: HardwareInfo }) {
  if (!data.usb || data.usb.length === 0) return null;
  return (
    <div ref={(el) => hoverHandlers(el)} style={{ ...CARD_STYLE, ...CARD_PAD_STYLE }}>
      <p style={CARD_TITLE_STYLE}>Peripherals (USB)</p>
      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '12px' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--border)' }}>
              {['Description', 'Vendor:Product'].map((h) => (
                <th
                  key={h}
                  style={{
                    textAlign: 'left',
                    padding: '6px 8px',
                    color: 'var(--text-muted)',
                    fontWeight: 600,
                    fontSize: '11px',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                  }}
                >
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.usb.map((dev) => (
              <tr
                key={`${dev.bus}-${dev.device_num}`}
                style={{
                  borderBottom: '1px solid var(--border-faint)',
                  color: 'var(--text-emphasis)',
                }}
              >
                <td style={{ padding: '6px 8px' }}>{dev.description}</td>
                <td
                  style={{ padding: '6px 8px', fontFamily: 'var(--font-mono)', fontSize: '11px' }}
                >
                  {dev.vendor_id}:{dev.product_id}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function BatteryCard({ data }: { data: HardwareInfo }) {
  if (!data.battery.present) return null;
  return (
    <div ref={(el) => hoverHandlers(el)} style={{ ...CARD_STYLE, ...CARD_PAD_STYLE }}>
      <p style={CARD_TITLE_STYLE}>Battery</p>
      <Row label="Status" value={data.battery.status || 'N/A'} />
      {data.battery.capacity_pct !== undefined && data.battery.capacity_pct > 0 && (
        <GaugeBar
          label="Charge"
          value={data.battery.capacity_pct}
          gradient="linear-gradient(90deg, var(--signal-healthy), var(--signal-healthy))"
        />
      )}
      {data.battery.health_pct !== undefined && data.battery.health_pct > 0 && (
        <Row label="Health" value={`${data.battery.health_pct}%`} />
      )}
      {data.battery.cycle_count !== undefined && data.battery.cycle_count > 0 && (
        <Row label="Cycles" value={data.battery.cycle_count} />
      )}
      {data.battery.technology && <Row label="Technology" value={data.battery.technology} />}
    </div>
  );
}

function HardwareInventorySection({ data }: { data: HardwareInfo }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <p style={{ ...CARD_TITLE_STYLE, marginBottom: 0, paddingLeft: '4px' }}>Hardware Inventory</p>

      {/* CPU + Memory side by side */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
        <CPUCard data={data} />
        <MemoryCard data={data} />
      </div>

      {/* Storage — full width per drive since partition tables need space */}
      {data.storage && data.storage.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          {data.storage.map((dev) => (
            <StorageCard key={dev.name || dev.serial} device={dev} />
          ))}
        </div>
      )}

      {/* GPU + Motherboard side by side */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
        <GPUCard data={data} />
        <MotherboardCard data={data} />
      </div>

      {/* Network */}
      <NetworkSection data={data} />

      {/* USB Peripherals */}
      <USBCard data={data} />

      {/* Battery — only if present */}
      <BatteryCard data={data} />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export function HardwarePage() {
  const hardware = useAgentHardware();
  const metrics = useMetrics();
  // useAgentStatus only needed if hardware needs hostname/os — keep for backward compat
  useAgentStatus();

  const isLoading = hardware.isLoading && metrics.isLoading;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Loading skeleton */}
      {isLoading && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <MetricsSkeleton />
          <Skeleton className="h-48 w-full" />
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
            <Skeleton className="h-48" />
            <Skeleton className="h-48" />
          </div>
        </div>
      )}

      {/* Hardware fetch error */}
      {hardware.isError && (
        <div style={{ ...CARD_STYLE }}>
          <ErrorState
            message="Failed to load hardware information."
            onRetry={() => hardware.refetch()}
          />
        </div>
      )}

      {/* LIVE MONITORING — show even if hardware fails */}
      {metrics.isLoading && !isLoading && <MetricsSkeleton />}
      {metrics.isError && <MetricsError refetch={() => metrics.refetch()} />}
      {metrics.data && <LiveMonitorSection metrics={metrics.data} />}

      {/* HARDWARE INVENTORY */}
      {hardware.data && <HardwareInventorySection data={hardware.data} />}
    </div>
  );
}
