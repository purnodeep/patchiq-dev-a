import { useState } from 'react';
import { Link, useNavigate } from 'react-router';
import { ShieldCheck, ChevronRight, Plus, X, Info } from 'lucide-react';
import { useCan } from '../../../app/auth/AuthContext';
import type { EndpointDetail } from '../../../api/hooks/useEndpoints';
import { useEndpointCVEs, useEndpointPatches } from '../../../api/hooks/useEndpoints';
import {
  computeRiskScore,
  riskColor as riskColorFn,
  riskLabel as riskLabelFn,
} from '../../../lib/risk';
import { usePolicies } from '../../../api/hooks/usePolicies';
import { useTags, useAssignTag, useUnassignTag } from '../../../api/hooks/useTags';
import { timeAgo } from '../../../lib/time';
import { CreateTagDialog } from '../CreateTagDialog';
import type { HardwareInfo } from '../../../types/hardware';

interface OverviewTabProps {
  endpoint: EndpointDetail;
  onTabChange?: (tab: string) => void;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function severityColor(sev: string): string {
  switch (sev.toLowerCase()) {
    case 'critical':
      return 'var(--signal-critical)';
    case 'high':
      return 'var(--signal-warning)';
    case 'medium':
      return 'var(--signal-warning)';
    default:
      return 'var(--signal-healthy)';
  }
}

function metricColor(pct: number): string {
  if (pct >= 80) return 'var(--signal-critical)';
  if (pct >= 60) return 'var(--signal-warning)';
  return 'var(--signal-healthy)';
}

// ─── Sub-components ───────────────────────────────────────────────────────────

/** Dark card tile with optional header right-slot. */
function Tile({
  title,
  subtitle,
  rightLabel,
  rightAction,
  children,
  style,
}: {
  title: string;
  subtitle?: string;
  rightLabel?: string;
  rightAction?: React.ReactNode;
  children: React.ReactNode;
  style?: React.CSSProperties;
}) {
  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 10,
        overflow: 'hidden',
        ...style,
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px 16px 10px',
          borderBottom: '1px solid var(--border)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 8 }}>
          <span
            style={{
              fontSize: 11,
              fontWeight: 600,
              letterSpacing: '0.06em',
              textTransform: 'uppercase',
              color: 'var(--text-secondary)',
              fontFamily: 'var(--font-mono)',
            }}
          >
            {title}
          </span>
          {subtitle && (
            <span
              style={{ fontSize: 10, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
            >
              {subtitle}
            </span>
          )}
        </div>
        {rightAction}
        {rightLabel && (
          <span
            style={{ fontSize: 10, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
          >
            {rightLabel}
          </span>
        )}
      </div>
      <div style={{ padding: '14px 16px' }}>{children}</div>
    </div>
  );
}

/** SVG ring gauge, 64px. */
function RingGauge({ label, value, sub }: { label: string; value: number | null; sub: string }) {
  const size = 64;
  const stroke = 5;
  const r = (size - stroke) / 2;
  const circ = 2 * Math.PI * r;
  const pct = value != null ? Math.min(100, Math.max(0, value)) : 0;
  const dash = (pct / 100) * circ;
  const color = value == null ? 'var(--border)' : metricColor(pct);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 5 }}>
      <div style={{ position: 'relative', width: size, height: size }}>
        <svg width={size} height={size} style={{ transform: 'rotate(-90deg)' }}>
          <circle
            cx={size / 2}
            cy={size / 2}
            r={r}
            fill="none"
            stroke="var(--border)"
            strokeWidth={stroke}
          />
          {value != null && (
            <circle
              cx={size / 2}
              cy={size / 2}
              r={r}
              fill="none"
              stroke={color}
              strokeWidth={stroke}
              strokeDasharray={`${dash} ${circ - dash}`}
              strokeLinecap="round"
            />
          )}
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
              fontSize: 12,
              fontWeight: 700,
              fontFamily: 'var(--font-mono)',
              color: value != null ? color : 'var(--text-faint)',
            }}
          >
            {value != null ? `${Math.round(pct)}%` : '—'}
          </span>
        </div>
      </div>
      <span
        style={{
          fontSize: 10,
          color: 'var(--text-muted)',
          fontWeight: 600,
          fontFamily: 'var(--font-mono)',
          letterSpacing: '0.06em',
          textTransform: 'uppercase',
        }}
      >
        {label}
      </span>
      <span style={{ fontSize: 9, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}>
        {sub}
      </span>
    </div>
  );
}

/** 7-day patch sparkline. */
function PatchSparkline({ patches }: { patches: { created_at: string }[] }) {
  const W = 220;
  const H = 40;
  const now = Date.now();
  const buckets = Array.from({ length: 7 }, (_, i) => {
    const s = now - (6 - i) * 86_400_000;
    const e = s + 86_400_000;
    return patches.filter((p) => {
      const t = new Date(p.created_at).getTime();
      return t >= s && t < e;
    }).length;
  });
  const maxVal = Math.max(...buckets, 1);
  const pts = buckets.map((v, i) => ({
    x: (i / 6) * (W - 2) + 1,
    y: H - 4 - (v / maxVal) * (H - 8),
  }));
  const lineD = pts.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ');
  const areaD = `${lineD} L ${pts[6].x} ${H} L ${pts[0].x} ${H} Z`;

  return (
    <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height: H, display: 'block' }}>
      <defs>
        <linearGradient id="spkGrad" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--signal-healthy)" stopOpacity={0.3} />
          <stop offset="100%" stopColor="var(--signal-healthy)" stopOpacity={0} />
        </linearGradient>
      </defs>
      <path d={areaD} fill="url(#spkGrad)" />
      <path d={lineD} fill="none" stroke="var(--signal-healthy)" strokeWidth={1.5} />
    </svg>
  );
}

/** SVG donut gauge for risk, 88px. */
function DonutGauge({ score, color }: { score: number; color: string }) {
  const size = 88;
  const stroke = 7;
  const r = (size - stroke) / 2;
  const circ = 2 * Math.PI * r;
  const pct = (score / 10) * 100;
  const dash = (pct / 100) * circ;

  return (
    <div style={{ position: 'relative', width: size, height: size, flexShrink: 0 }}>
      <svg width={size} height={size} style={{ transform: 'rotate(-90deg)' }}>
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke="var(--bg-inset)"
          strokeWidth={stroke}
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke={color}
          strokeWidth={stroke}
          strokeDasharray={`${dash} ${circ - dash}`}
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
        }}
      >
        <span style={{ fontSize: 13, fontWeight: 700, fontFamily: 'var(--font-mono)', color }}>
          {score.toFixed(1)}
        </span>
      </div>
    </div>
  );
}

/** Seeded pseudo-random for deterministic star positions. */
function seededRandom(seed: number) {
  let s = seed;
  return () => {
    s = (s * 16807 + 0) % 2147483647;
    return s / 2147483647;
  };
}

/**
 * Force-directed-style SVG blast radius network graph.
 * Animated: star-field background, data-flow particles along connections,
 * idle node drift, entrance fly-out from center, breathing center halo.
 * Hover for details, click CVEs to navigate.
 */
function BlastRadiusGraph({
  hostname,
  cveNodes,
  patchCount,
}: {
  hostname: string;
  cveNodes: { id: string; label: string; severity: string; cvss?: number }[];
  patchCount: number;
}) {
  const navigate = useNavigate();
  const [hoveredNode, setHoveredNode] = useState<number | null>(null);
  const [showCves, setShowCves] = useState(true);
  const [showPatches, setShowPatches] = useState(true);

  const W = 520;
  const H = 320;
  const cx = W / 2;
  const cy = H / 2;

  type Node = {
    label: string;
    color: string;
    x: number;
    y: number;
    r: number;
    ring: number;
    tooltip: string;
    href?: string;
    nodeType: 'cve' | 'patch';
  };
  const nodes: Node[] = [];

  // CVE nodes — top half (−90° to +90°)
  const topCves = cveNodes.slice(0, 6);
  topCves.forEach((cve, i) => {
    const total = topCves.length;
    const startA = -Math.PI / 2;
    const endA = Math.PI / 2;
    const spread = endA - startA;
    const angle = startA + (spread * (i + 0.5)) / Math.max(total, 1);
    const ring = cve.severity === 'critical' ? 148 : cve.severity === 'high' ? 125 : 105;
    const r = cve.severity === 'critical' ? 22 : cve.severity === 'high' ? 19 : 16;
    nodes.push({
      label: cve.label.replace('CVE-', 'CVE-\n').slice(0, 14),
      color: severityColor(cve.severity),
      x: cx + ring * Math.cos(angle),
      y: cy + ring * Math.sin(angle),
      r,
      ring,
      tooltip: `${cve.label}${cve.cvss != null ? ` (CVSS ${cve.cvss})` : ''}`,
      href: `/cves/${cve.id}`,
      nodeType: 'cve',
    });
  });

  // Patch nodes — bottom-right (90° to 160°)
  const patchSlice = Math.min(patchCount, 4);
  for (let i = 0; i < patchSlice; i++) {
    const startA = Math.PI / 2 + 0.15;
    const endA = Math.PI * 0.88;
    const spread = endA - startA;
    const angle = startA + (spread * (i + 0.5)) / Math.max(patchSlice, 1);
    const ring = 115;
    nodes.push({
      label: `PATCH-${i + 1}`,
      color: 'var(--signal-warning)',
      x: cx + ring * Math.cos(angle),
      y: cy + ring * Math.sin(angle),
      r: 16,
      ring,
      tooltip: `Pending patch ${i + 1} of ${patchCount}`,
      nodeType: 'patch',
    });
  }

  // Severity-weighted blast score (max 10)
  const cveWeight = cveNodes.reduce((sum, c) => {
    if (c.severity === 'critical') return sum + 1.0;
    if (c.severity === 'high') return sum + 0.6;
    if (c.severity === 'medium') return sum + 0.3;
    return sum + 0.1;
  }, 0);
  const blastScore = Math.min(10, cveWeight + patchCount * 0.3).toFixed(1);

  const visibleNodes = nodes.filter((n) => {
    if (n.nodeType === 'cve') return showCves;
    if (n.nodeType === 'patch') return showPatches;
    return false;
  });

  // Score color: green < 3, amber 3-6, red > 6
  const scoreNum = parseFloat(blastScore);
  const scoreColor =
    scoreNum > 6
      ? 'var(--signal-critical)'
      : scoreNum > 3
        ? 'var(--signal-warning)'
        : 'var(--signal-healthy)';

  // Deterministic star field (20 stars)
  const rand = seededRandom(42);
  const stars = Array.from({ length: 20 }, () => ({
    x: rand() * W,
    y: rand() * H,
    r: 0.3 + rand() * 0.8,
    opacity: 0.15 + rand() * 0.35,
    dur: 3 + rand() * 5,
  }));

  return (
    <div style={{ position: 'relative', margin: '-14px -16px -14px -16px' }}>
      <svg
        viewBox={`0 0 ${W} ${H}`}
        style={{ width: '100%', height: 300, display: 'block', background: '#000' }}
      >
        <defs>
          <filter id="glowCenter" x="-60%" y="-60%" width="220%" height="220%">
            <feGaussianBlur stdDeviation="8" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
          <filter id="glowNode" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur stdDeviation="4" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
          <radialGradient id="centerBg" cx="40%" cy="35%" r="60%">
            <stop offset="0%" stopColor="var(--bg-card)" />
            <stop offset="100%" stopColor="var(--bg-page)" />
          </radialGradient>
          {/* Vignette mask — fades edges to background */}
          <radialGradient id="vignette" cx="50%" cy="50%" r="50%">
            <stop offset="60%" stopColor="white" />
            <stop offset="100%" stopColor="black" />
          </radialGradient>
          <mask id="vignetteMask">
            <rect width={W} height={H} fill="url(#vignette)" />
          </mask>
        </defs>

        {/* ── Star field background ── */}
        <g mask="url(#vignetteMask)">
          {stars.map((s, i) => (
            <circle key={`star-${i}`} cx={s.x} cy={s.y} r={s.r} fill="var(--text-faint)">
              <animate
                attributeName="opacity"
                values={`${s.opacity};${s.opacity * 0.3};${s.opacity}`}
                dur={`${s.dur}s`}
                repeatCount="indefinite"
              />
            </circle>
          ))}
        </g>

        {/* Severity zones — subtle filled rings */}
        <circle cx={cx} cy={cy} r={155} fill="var(--signal-critical)" fillOpacity={0.03} />
        <circle cx={cx} cy={cy} r={125} fill="var(--signal-warning)" fillOpacity={0.03} />
        <circle cx={cx} cy={cy} r={95} fill="var(--text-faint)" fillOpacity={0.03} />

        {/* ── Connection lines + data flow particles ── */}
        {visibleNodes.map((n, i) => {
          const midX = (cx + n.x) / 2 + (n.y - cy) * 0.1;
          const midY = (cy + n.y) / 2 - (n.x - cx) * 0.1;
          const pathD = `M ${cx} ${cy} Q ${midX} ${midY} ${n.x} ${n.y}`;
          const isHovered = hoveredNode === i;
          // Approximate path length for dash animation
          const dx = n.x - cx;
          const dy = n.y - cy;
          const pathLen = Math.sqrt(dx * dx + dy * dy) * 1.15;
          const drawDelay = 0.1 + i * 0.08;
          return (
            <g key={`line-${i}`}>
              <path
                d={pathD}
                fill="none"
                stroke={n.color}
                strokeOpacity={isHovered ? 0.35 : 0.12}
                strokeWidth={isHovered ? 1.5 : 1}
                strokeDasharray={pathLen}
                strokeDashoffset={pathLen}
              >
                {/* Draw-in: line grows from center to node */}
                <animate
                  attributeName="stroke-dashoffset"
                  from={pathLen}
                  to={0}
                  dur="0.5s"
                  begin={`${drawDelay}s`}
                  fill="freeze"
                  calcMode="spline"
                  keySplines="0.25 0.1 0.25 1"
                />
              </path>
              {/* Data flow particle — starts after line draws in */}
              <circle r={1.5} fill={n.color} fillOpacity={0}>
                <animate
                  attributeName="fill-opacity"
                  from="0"
                  to="0.7"
                  dur="0.1s"
                  begin={`${drawDelay + 0.5}s`}
                  fill="freeze"
                />
                <animateMotion
                  dur={`${2.5 + i * 0.4}s`}
                  begin={`${drawDelay + 0.5}s`}
                  repeatCount="indefinite"
                  path={pathD}
                />
              </circle>
            </g>
          );
        })}

        {/* ── Satellite nodes with entrance + idle drift ── */}
        {visibleNodes.map((n, i) => {
          const isCrit = n.nodeType === 'cve' && n.color === 'var(--signal-critical)';
          const isHovered = hoveredNode === i;
          const displayR = isHovered ? n.r + 2 : n.r;
          const [line1, line2] = n.label.includes('\n') ? n.label.split('\n') : [n.label, ''];
          // Idle drift: gentle sinusoidal oscillation ±2px
          const driftDur = 4 + (i % 3) * 1.5;
          const driftX = 2;
          const driftY = 1.5;
          // Entrance: fly from center, staggered by index
          const entranceDelay = 0.1 + i * 0.08;
          return (
            <g
              key={`node-${i}`}
              filter={isCrit ? 'url(#glowNode)' : undefined}
              style={{ cursor: n.href ? 'pointer' : 'default' }}
              onMouseEnter={() => setHoveredNode(i)}
              onMouseLeave={() => setHoveredNode(null)}
              onClick={() => {
                if (n.href) navigate(n.href);
              }}
            >
              {/* Entrance animation: fly from center to position */}
              <animateTransform
                attributeName="transform"
                type="translate"
                from={`${cx - n.x} ${cy - n.y}`}
                to="0 0"
                dur="0.6s"
                begin={`${entranceDelay}s`}
                fill="freeze"
                calcMode="spline"
                keySplines="0.25 0.1 0.25 1"
              />
              {/* Idle drift after entrance settles */}
              <animateTransform
                attributeName="transform"
                type="translate"
                values={`0,0; ${driftX},${-driftY}; ${-driftX},${driftY}; 0,0`}
                dur={`${driftDur}s`}
                begin={`${entranceDelay + 0.6}s`}
                repeatCount="indefinite"
                additive="sum"
              />
              {/* Outer halo */}
              <circle
                cx={n.x}
                cy={n.y}
                r={displayR + 5}
                fill={n.color}
                fillOpacity={isHovered ? 0.2 : 0.1}
              />
              {/* Main circle */}
              <circle
                cx={n.x}
                cy={n.y}
                r={displayR}
                fill={n.color}
                fillOpacity={isHovered ? 1 : 0.88}
                stroke={n.color}
                strokeWidth={0.5}
                strokeOpacity={0.4}
              />
              {/* Label text */}
              {line2 ? (
                <>
                  <text
                    x={n.x}
                    y={n.y - 3}
                    textAnchor="middle"
                    dominantBaseline="central"
                    fill="var(--text-on-color, #fff)"
                    fontSize={6.5}
                    fontWeight="600"
                  >
                    {line1}
                  </text>
                  <text
                    x={n.x}
                    y={n.y + 7}
                    textAnchor="middle"
                    dominantBaseline="central"
                    fill="var(--text-on-color, #fff)"
                    fontSize={6.5}
                    fontWeight="600"
                  >
                    {line2}
                  </text>
                </>
              ) : (
                <text
                  x={n.x}
                  y={n.y}
                  textAnchor="middle"
                  dominantBaseline="central"
                  fill="var(--text-on-color, #fff)"
                  fontSize={n.r > 16 ? 7.5 : 7}
                  fontWeight="600"
                >
                  {n.label}
                </text>
              )}
              {/* Critical CVE pulse ring */}
              {isCrit && (
                <circle
                  cx={n.x}
                  cy={n.y}
                  r={n.r + 8}
                  fill="none"
                  stroke={n.color}
                  strokeWidth={1.5}
                  strokeOpacity={0.3}
                >
                  <animate
                    attributeName="r"
                    values={`${n.r + 5};${n.r + 14};${n.r + 5}`}
                    dur="2.5s"
                    repeatCount="indefinite"
                  />
                  <animate
                    attributeName="stroke-opacity"
                    values="0.3;0;0.3"
                    dur="2.5s"
                    repeatCount="indefinite"
                  />
                </circle>
              )}
            </g>
          );
        })}

        {/* ── Center node with breathing halo ── */}
        <circle cx={cx} cy={cy} r={50} fill="var(--accent)" fillOpacity={0.04}>
          <animate attributeName="r" values="48;52;48" dur="4s" repeatCount="indefinite" />
          <animate
            attributeName="fill-opacity"
            values="0.03;0.06;0.03"
            dur="4s"
            repeatCount="indefinite"
          />
        </circle>
        <circle
          cx={cx}
          cy={cy}
          r={42}
          fill="var(--accent)"
          fillOpacity={0.07}
          filter="url(#glowCenter)"
        >
          <animate
            attributeName="fill-opacity"
            values="0.06;0.1;0.06"
            dur="4s"
            repeatCount="indefinite"
          />
        </circle>

        {/* Center node circle */}
        <circle cx={cx} cy={cy} r={36} fill="url(#centerBg)" />
        <circle
          cx={cx}
          cy={cy}
          r={36}
          fill="none"
          stroke="var(--accent)"
          strokeWidth={2}
          strokeOpacity={0.7}
        />

        {/* Center hostname text */}
        {(() => {
          const parts = hostname.split('.');
          const lines: string[] = [];
          let cur = '';
          for (const p of parts) {
            if ((cur + p).length > 12) {
              lines.push(cur.replace(/\.$/, ''));
              cur = p + '.';
            } else {
              cur += p + '.';
            }
          }
          if (cur) lines.push(cur.replace(/\.$/, ''));
          const startY = cy - (lines.length - 1) * 7;
          return lines.map((l, li) => (
            <text
              key={li}
              x={cx}
              y={startY + li * 13}
              textAnchor="middle"
              dominantBaseline="central"
              fill="var(--text-on-color, #fff)"
              fontSize={8}
              fontWeight="600"
              letterSpacing="0.3"
            >
              {l}
            </text>
          ));
        })()}

        {/* ── Blast score pill — color-coded ── */}
        <rect
          x={W - 82}
          y={10}
          width={72}
          height={44}
          rx={10}
          fill="var(--bg-card)"
          fillOpacity={0.7}
          stroke={scoreColor}
          strokeWidth={0.5}
          strokeOpacity={0.5}
        />
        <text
          x={W - 46}
          y={32}
          textAnchor="middle"
          fill={scoreColor}
          fontSize={22}
          fontWeight="700"
          fontFamily="var(--font-mono)"
          opacity={0.95}
        >
          {blastScore}
        </text>
        <text
          x={W - 46}
          y={48}
          textAnchor="middle"
          fill="var(--text-faint)"
          fontSize={8}
          fontFamily="var(--font-mono)"
          letterSpacing="0.05em"
        >
          Blast Score
        </text>

        {/* ── Hover tooltip ── */}
        {hoveredNode !== null &&
          visibleNodes[hoveredNode] &&
          (() => {
            const n = visibleNodes[hoveredNode];
            const text = n.tooltip;
            const tw = Math.max(60, text.length * 5.5 + 16);
            const th = 22;
            const tx = Math.max(tw / 2 + 4, Math.min(W - tw / 2 - 4, n.x));
            const ty = Math.max(th, n.y - n.r - 16);
            return (
              <g>
                <rect
                  x={tx - tw / 2}
                  y={ty - th / 2}
                  width={tw}
                  height={th}
                  rx={4}
                  fill="var(--bg-card)"
                  stroke="var(--border)"
                  strokeWidth={1}
                />
                <text
                  x={tx}
                  y={ty}
                  textAnchor="middle"
                  dominantBaseline="central"
                  fill="var(--text-secondary)"
                  fontSize={9}
                  fontFamily="var(--font-mono)"
                >
                  {text}
                </text>
              </g>
            );
          })()}
      </svg>

      {/* Filter toggles */}
      <div style={{ display: 'flex', gap: 6, padding: '8px 16px', background: '#000' }}>
        {[
          { key: 'cves', label: 'CVEs', active: showCves, toggle: () => setShowCves(!showCves) },
          {
            key: 'patches',
            label: 'Patches',
            active: showPatches,
            toggle: () => setShowPatches(!showPatches),
          },
        ].map(({ key, label, active, toggle }) => (
          <button
            key={key}
            type="button"
            onClick={toggle}
            style={{
              padding: '3px 10px',
              borderRadius: 4,
              border: '1px solid',
              borderColor: active ? 'var(--accent)' : 'var(--border)',
              background: active
                ? 'color-mix(in srgb, var(--accent) 15%, transparent)'
                : 'transparent',
              color: active ? 'var(--accent)' : 'var(--text-muted)',
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              cursor: 'pointer',
            }}
          >
            {label}
          </button>
        ))}
      </div>

      {/* Legend */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 14,
          background: '#000',
          padding: '0 16px 12px',
        }}
      >
        {[
          { color: 'var(--signal-critical)', label: `CVE (${cveNodes.length})` },
          { color: 'var(--signal-warning)', label: `Pending Patch (${patchCount})` },
        ].map((item) => (
          <span key={item.label} style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
            <span
              style={{
                width: 8,
                height: 8,
                borderRadius: '50%',
                background: item.color,
                display: 'inline-block',
                flexShrink: 0,
              }}
            />
            <span
              style={{ fontSize: 10, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
            >
              {item.label}
            </span>
          </span>
        ))}
        <span
          title="Blast Score (0-10) measures exposure risk. Critical CVEs add 1.0, high add 0.6, medium add 0.3, low add 0.1. Pending patches add 0.3 each. Capped at 10."
          style={{ cursor: 'help', marginLeft: 4, display: 'inline-flex', alignItems: 'center' }}
        >
          <Info style={{ width: 11, height: 11, color: 'var(--text-muted)' }} />
        </span>
      </div>
    </div>
  );
}

/** Horizontal filmstrip lifecycle timeline. */
function LifecycleTimeline({ endpoint }: { endpoint: EndpointDetail }) {
  const events: { label: string; date: string | null | undefined; active?: boolean }[] = [
    { label: 'Enrolled', date: endpoint.enrolled_at ?? endpoint.created_at, active: true },
    { label: 'First Scan', date: endpoint.last_scan },
    {
      label: 'Last Heartbeat',
      date: endpoint.last_heartbeat,
      active: endpoint.status === 'online',
    },
    { label: 'Last Seen', date: endpoint.last_seen },
  ];

  return (
    <Tile title="Audit Timeline" subtitle="last 14 days">
      <div
        style={{
          position: 'relative',
          display: 'flex',
          alignItems: 'flex-start',
          justifyContent: 'space-between',
          paddingTop: 12,
          paddingBottom: 8,
        }}
      >
        {/* Connecting line */}
        <div
          style={{
            position: 'absolute',
            top: 15,
            left: '5%',
            right: '5%',
            height: 1,
            background: 'var(--border)',
            zIndex: 0,
          }}
        />

        {events.map((ev, i) => {
          const dotColor = ev.date ? 'var(--signal-healthy)' : 'var(--border)';
          return (
            <div
              key={i}
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                gap: 8,
                zIndex: 1,
                flex: 1,
                minWidth: 0,
              }}
            >
              {/* Dot */}
              <div
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  background: dotColor,
                  border: '2px solid var(--bg-card)',
                  boxShadow: ev.date ? `0 0 0 2px ${dotColor}44` : 'none',
                  flexShrink: 0,
                }}
              />
              {/* Label */}
              <span
                style={{
                  fontSize: 10,
                  fontWeight: 600,
                  fontFamily: 'var(--font-mono)',
                  letterSpacing: '0.06em',
                  textTransform: 'uppercase',
                  color: 'var(--text-secondary)',
                  textAlign: 'center',
                  whiteSpace: 'nowrap',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  maxWidth: '100%',
                }}
              >
                {ev.label}
              </span>
              {/* Date */}
              <span
                style={{
                  fontSize: 9,
                  color: 'var(--text-faint)',
                  fontFamily: 'var(--font-mono)',
                  textAlign: 'center',
                }}
              >
                {ev.date ? timeAgo(ev.date) : '—'}
              </span>
            </div>
          );
        })}
      </div>
    </Tile>
  );
}

// ─── Main Component ───────────────────────────────────────────────────────────

export function OverviewTab({ endpoint, onTabChange }: OverviewTabProps) {
  const { data: cvesData } = useEndpointCVEs(endpoint.id);
  const { data: patchesData } = useEndpointPatches(endpoint.id);

  const { data: policiesData } = usePolicies({ limit: 100 });
  const { data: allTagsData } = useTags({ limit: 50 });
  const assignTag = useAssignTag();
  const unassignTag = useUnassignTag();
  const can = useCan();
  const [showTagPicker, setShowTagPicker] = useState(false);
  const [showCreateTag, setShowCreateTag] = useState(false);

  const cves = cvesData?.data ?? [];
  const patches = patchesData?.data ?? [];

  const pendingPatches = patches.filter((p) => p.status === 'pending' || p.status === 'available');
  const installedCount = patches.filter((p) => p.status === 'installed').length;
  const failedCount = patches.filter((p) => p.status === 'failed').length;
  const pendingCount = pendingPatches.length;

  const criticalPending = pendingPatches.filter((p) => p.severity === 'critical').length;
  const highPending = pendingPatches.filter((p) => p.severity === 'high').length;
  const mediumPending = pendingPatches.filter((p) => p.severity === 'medium').length;

  const criticalCves = cves.filter((c) => c.cve_severity === 'critical').length;
  const highCves = cves.filter((c) => c.cve_severity === 'high').length;
  const mediumCves = cves.filter((c) => c.cve_severity === 'medium').length;

  const riskScore = computeRiskScore({ criticalCves, highCves, mediumCves });
  const riskColor = riskColorFn(riskScore);
  const riskLabel = riskLabelFn(riskScore);

  const BREAKDOWN_TAB_MAP: Record<string, string> = {
    'Unpatched CVEs': 'cves',
    'Critical Exposure': 'cves',
  };

  const riskBreakdown = [
    {
      label: 'Unpatched CVEs',
      score: Math.min(1, (criticalCves + highCves) / Math.max(cves.length, 1)),
      color: 'var(--signal-warning)',
    },
    {
      label: 'Critical Exposure',
      score: Math.min(1, criticalCves / 5),
      color: 'var(--signal-critical)',
    },
    { label: 'Config Drift', score: riskScore / 10, color: 'var(--text-muted)' },
    {
      label: 'Network Exposure',
      score: endpoint.network_interfaces
        ? Math.min(1, endpoint.network_interfaces.length / 8)
        : 0.2,
      color: 'var(--text-muted)',
    },
  ];

  // Resource percentages — prefer hardware_details JSONB, fall back to flat columns
  const hw = endpoint.hardware_details as HardwareInfo | null | undefined;

  const hwMemTotalGb = hw?.memory?.total_bytes
    ? hw.memory.total_bytes / (1024 * 1024 * 1024)
    : null;
  const hwMemAvailGb = hw?.memory?.available_bytes
    ? hw.memory.available_bytes / (1024 * 1024 * 1024)
    : null;
  const hwMemUsedGb =
    hwMemTotalGb != null && hwMemAvailGb != null ? hwMemTotalGb - hwMemAvailGb : null;
  const hwMemPct =
    hwMemTotalGb && hwMemUsedGb != null ? Math.round((hwMemUsedGb / hwMemTotalGb) * 100) : null;

  const hwDiskTotal = hw?.storage?.reduce((a, d) => a + (d.size_bytes ?? 0), 0) ?? 0;
  const hwDiskUsed =
    hw?.storage?.reduce(
      (a, d) =>
        a +
        (d.partitions?.reduce(
          (pa, p) => pa + (p.size_bytes ?? 0) * ((p.usage_pct ?? 0) / 100),
          0,
        ) ?? 0),
      0,
    ) ?? 0;
  const hwDiskPct = hwDiskTotal > 0 ? Math.round((hwDiskUsed / hwDiskTotal) * 100) : null;

  const cpuPct = Math.round(endpoint.cpu_usage_percent ?? 0);
  const memPct =
    endpoint.memory_total_mb && endpoint.memory_used_mb
      ? Math.round((endpoint.memory_used_mb / endpoint.memory_total_mb) * 100)
      : hwMemPct;
  const diskPct =
    endpoint.disk_total_gb && endpoint.disk_used_gb
      ? Math.round((endpoint.disk_used_gb / endpoint.disk_total_gb) * 100)
      : hwDiskPct;

  const hwCpuCores = hw?.cpu?.total_logical_cpus ?? null;
  const cpuCores = endpoint.cpu_cores ?? hwCpuCores;
  const cpuSub = cpuCores
    ? `${Math.round(((cpuPct ?? 0) / 100) * cpuCores * 10) / 10}/${cpuCores} cores`
    : '—';
  const memSub = endpoint.memory_total_mb
    ? `${((endpoint.memory_used_mb ?? 0) / 1024).toFixed(1)}/${Math.round(endpoint.memory_total_mb / 1024)} GB`
    : hwMemTotalGb
      ? `${(hwMemUsedGb ?? 0).toFixed(1)}/${hwMemTotalGb.toFixed(0)} GB`
      : '—';
  const diskSub = endpoint.disk_total_gb
    ? `${endpoint.disk_used_gb ?? 0}/${endpoint.disk_total_gb} GB`
    : hwDiskTotal > 0
      ? `${(hwDiskUsed / (1024 * 1024 * 1024)).toFixed(0)}/${(hwDiskTotal / (1024 * 1024 * 1024)).toFixed(0)} GB`
      : '—';

  const endpointTags = endpoint.tags ?? [];

  const allPolicies =
    (
      policiesData as unknown as
        | {
            data?: {
              id: string;
              name: string;
              mode: string;
              os_family?: string | null;
              enabled: boolean;
            }[];
          }
        | undefined
    )?.data ?? [];
  const matchedPolicies = allPolicies
    .filter((p) => !p.os_family || p.os_family === endpoint.os_family)
    .slice(0, 4);

  // CVE nodes for blast radius
  const cveNodes = cves.slice(0, 6).map((c) => ({
    id: c.id,
    label: c.cve_identifier,
    severity: c.cve_severity,
    cvss: (c as { cvss_score?: number }).cvss_score,
  }));

  const lastPatchedAt = patches.find((p) => p.status === 'installed')?.created_at ?? null;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16, paddingTop: 16 }}>
      {/* ── Row 1: Blast Radius (60%) + Risk Breakdown (40%) ── */}
      <div style={{ display: 'grid', gridTemplateColumns: '3fr 2fr', gap: 16 }}>
        {/* Blast Radius */}
        <Tile title="Blast Radius" subtitle="attack surface map">
          {cveNodes.length === 0 && pendingCount === 0 ? (
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                padding: '60px 0',
                gap: 12,
              }}
            >
              <div
                style={{
                  width: 48,
                  height: 48,
                  borderRadius: '50%',
                  background: 'color-mix(in srgb, var(--signal-healthy) 10%, transparent)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <ShieldCheck style={{ width: 24, height: 24, color: 'var(--signal-healthy)' }} />
              </div>
              <p
                style={{ fontSize: 13, fontWeight: 600, color: 'var(--signal-healthy)', margin: 0 }}
              >
                No attack surface detected
              </p>
              <p style={{ fontSize: 11, color: 'var(--text-muted)', margin: 0 }}>
                This endpoint is clean
              </p>
            </div>
          ) : (
            <BlastRadiusGraph
              hostname={endpoint.hostname}
              cveNodes={cveNodes}
              patchCount={pendingCount}
            />
          )}
        </Tile>

        {/* Risk Breakdown */}
        <Tile title="Risk Breakdown" subtitle="composite score">
          {/* Score + donut */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 20 }}>
            <div>
              <div
                style={{
                  fontSize: 48,
                  fontWeight: 700,
                  fontFamily: 'var(--font-mono)',
                  color: riskColor,
                  lineHeight: 1,
                }}
              >
                {riskScore.toFixed(1)}
              </div>
              <div style={{ fontSize: 12, color: riskColor, fontWeight: 600, marginTop: 3 }}>
                {riskLabel}
              </div>
            </div>
            <DonutGauge score={riskScore} color={riskColor} />
          </div>

          {/* Breakdown rows */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            {riskBreakdown.map((item) => {
              const barColor =
                item.score > 0.5
                  ? 'var(--signal-critical)'
                  : item.score > 0.3
                    ? 'var(--signal-warning)'
                    : 'var(--border)';
              const targetTab = BREAKDOWN_TAB_MAP[item.label];
              const isClickable = !!targetTab && !!onTabChange;
              return (
                <div
                  key={item.label}
                  style={{
                    cursor: isClickable ? 'pointer' : 'default',
                    borderRadius: 4,
                    padding: '4px 0',
                    transition: 'background 0.15s',
                  }}
                  onClick={isClickable ? () => onTabChange(targetTab) : undefined}
                  onMouseEnter={(e) => {
                    if (isClickable)
                      (e.currentTarget as HTMLElement).style.background = 'var(--bg-inset)';
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLElement).style.background = 'transparent';
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      marginBottom: 5,
                    }}
                  >
                    <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                      {item.label}
                    </span>
                    <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                      <span
                        style={{
                          fontSize: 12,
                          fontFamily: 'var(--font-mono)',
                          fontWeight: 700,
                          color: 'var(--text-emphasis)',
                        }}
                      >
                        {(item.score * 10).toFixed(1)}
                      </span>
                      {isClickable && (
                        <ChevronRight
                          style={{ width: 12, height: 12, color: 'var(--text-muted)' }}
                        />
                      )}
                    </span>
                  </div>
                  <div
                    style={{
                      height: 3,
                      borderRadius: 2,
                      background: 'color-mix(in srgb, var(--border) 50%, var(--bg-inset))',
                      overflow: 'hidden',
                    }}
                  >
                    <div
                      style={{
                        height: '100%',
                        width: `${item.score * 100}%`,
                        background: barColor,
                        borderRadius: 2,
                      }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </Tile>
      </div>

      {/* ── Row 2: System Resources + Patch Summary + Tags & Policies ── */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 16 }}>
        {/* System Resources */}
        <Tile title="System Resources">
          {/* Ring gauges row */}
          <div style={{ display: 'flex', justifyContent: 'space-around', marginBottom: 14 }}>
            <RingGauge label="CPU" value={cpuPct} sub={cpuSub} />
            <RingGauge label="Memory" value={memPct} sub={memSub} />
            <RingGauge label="Disk" value={diskPct} sub={diskSub} />
          </div>
          {/* Footer line */}
          <div
            style={{
              borderTop: '1px solid var(--border)',
              paddingTop: 10,
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-muted)',
              display: 'flex',
              gap: 12,
              flexWrap: 'wrap',
            }}
          >
            {endpoint.uptime_seconds != null && (
              <span>
                Uptime {Math.floor(endpoint.uptime_seconds / 86400)}d{' '}
                {Math.floor((endpoint.uptime_seconds % 86400) / 3600)}h
              </span>
            )}
            {endpoint.cpu_cores != null && (
              <span>Load {(((cpuPct ?? 0) / 100) * (endpoint.cpu_cores ?? 1)).toFixed(2)}</span>
            )}
          </div>
        </Tile>

        {/* Patch Summary */}
        <Tile title="Patch Summary">
          {/* Pending count */}
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 4 }}>
            <span
              style={{
                fontSize: 32,
                fontWeight: 700,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-emphasis)',
                lineHeight: 1,
              }}
            >
              {pendingCount}
            </span>
            <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>Pending</span>
          </div>
          {/* Severity text */}
          {pendingCount > 0 && (
            <div style={{ fontSize: 12, marginBottom: 10 }}>
              {criticalPending > 0 && (
                <span style={{ color: 'var(--signal-critical)', marginRight: 6 }}>
                  {criticalPending} critical
                </span>
              )}
              {highPending > 0 && (
                <span style={{ color: 'var(--signal-warning)', marginRight: 6 }}>
                  · {highPending} high
                </span>
              )}
              {mediumPending > 0 && (
                <span style={{ color: 'var(--signal-warning)' }}>· {mediumPending} medium</span>
              )}
            </div>
          )}
          {/* Installed */}
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 6, marginBottom: 4 }}>
            <span
              style={{
                fontSize: 20,
                fontWeight: 700,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-emphasis)',
              }}
            >
              {installedCount.toLocaleString()}
            </span>
            <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Installed</span>
          </div>
          {/* Failed */}
          <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 10 }}>
            <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)' }}>
              {failedCount}
            </span>{' '}
            Failed
          </div>
          {/* Sparkline */}
          <div>
            <span
              style={{
                fontSize: 9,
                color: 'var(--text-faint)',
                fontFamily: 'var(--font-mono)',
                letterSpacing: '0.06em',
                display: 'block',
                marginBottom: 4,
              }}
            >
              PATCHES DEPLOYED — LAST 7 DAYS
            </span>
            <PatchSparkline patches={patches} />
          </div>
          {lastPatchedAt && (
            <div
              style={{
                fontSize: 10,
                color: 'var(--text-faint)',
                fontFamily: 'var(--font-mono)',
                marginTop: 6,
              }}
            >
              Last patched {timeAgo(lastPatchedAt)}
            </div>
          )}
        </Tile>

        {/* Tags & Policies */}
        <Tile
          title="Tags"
          rightAction={
            <div style={{ position: 'relative' }}>
              <button
                type="button"
                disabled={!can('endpoints', 'create')}
                title={!can('endpoints', 'create') ? "You don't have permission" : undefined}
                onClick={() => setShowTagPicker((v) => !v)}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 4,
                  padding: '2px 8px',
                  borderRadius: 5,
                  border: '1px solid var(--border)',
                  background: 'var(--bg-inset)',
                  fontSize: 10,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-muted)',
                  cursor: !can('endpoints', 'create') ? 'not-allowed' : 'pointer',
                  opacity: !can('endpoints', 'create') ? 0.5 : 1,
                }}
              >
                <Plus style={{ width: 10, height: 10 }} />
                Add Tag
              </button>
              {showTagPicker && (
                <div
                  style={{
                    position: 'absolute',
                    top: '100%',
                    right: 0,
                    marginTop: 4,
                    width: 200,
                    maxHeight: 200,
                    overflowY: 'auto',
                    background: 'var(--bg-card)',
                    border: '1px solid var(--border)',
                    borderRadius: 8,
                    padding: 4,
                    zIndex: 50,
                    boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
                  }}
                >
                  {(() => {
                    const assignedIds = new Set(endpointTags.map((t) => t.id));
                    const available = (allTagsData ?? []).filter((t) => !assignedIds.has(t.id));
                    if (available.length === 0) {
                      return (
                        <div
                          style={{
                            padding: '8px',
                            fontSize: 11,
                            color: 'var(--text-faint)',
                            textAlign: 'center',
                          }}
                        >
                          No tags available
                        </div>
                      );
                    }
                    return available.map((tag) => (
                      <button
                        key={tag.id}
                        type="button"
                        onClick={() => {
                          assignTag.mutate(
                            { tagId: tag.id, endpointIds: [endpoint.id] },
                            { onSuccess: () => setShowTagPicker(false) },
                          );
                        }}
                        style={{
                          display: 'block',
                          width: '100%',
                          textAlign: 'left',
                          padding: '5px 8px',
                          borderRadius: 4,
                          border: 'none',
                          background: 'transparent',
                          fontSize: 11,
                          fontFamily: 'var(--font-mono)',
                          color: 'var(--text-secondary)',
                          cursor: 'pointer',
                        }}
                        onMouseEnter={(e) => {
                          (e.currentTarget as HTMLButtonElement).style.background =
                            'var(--bg-inset)';
                        }}
                        onMouseLeave={(e) => {
                          (e.currentTarget as HTMLButtonElement).style.background = 'transparent';
                        }}
                      >
                        {tag.key}:{tag.value}
                      </button>
                    ));
                  })()}
                  <button
                    type="button"
                    onClick={() => {
                      setShowTagPicker(false);
                      setShowCreateTag(true);
                    }}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 4,
                      width: '100%',
                      textAlign: 'left',
                      padding: '6px 8px',
                      borderRadius: 4,
                      border: 'none',
                      borderTop: '1px solid var(--border)',
                      background: 'transparent',
                      fontSize: 11,
                      fontFamily: 'var(--font-mono)',
                      color: 'var(--accent)',
                      cursor: 'pointer',
                      marginTop: 2,
                    }}
                    onMouseEnter={(e) => {
                      (e.currentTarget as HTMLButtonElement).style.background = 'var(--bg-inset)';
                    }}
                    onMouseLeave={(e) => {
                      (e.currentTarget as HTMLButtonElement).style.background = 'transparent';
                    }}
                  >
                    <Plus style={{ width: 10, height: 10 }} />
                    Create New Tag
                  </button>
                </div>
              )}
            </div>
          }
        >
          {/* Tags */}
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
            {endpointTags.length === 0 ? (
              <span style={{ fontSize: 11, color: 'var(--text-faint)' }}>No tags assigned</span>
            ) : (
              endpointTags.map((tag) => (
                <span
                  key={tag.id}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: 4,
                    padding: '3px 8px',
                    borderRadius: 5,
                    border: '1px solid var(--border)',
                    background: 'var(--bg-inset)',
                    fontSize: 11,
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-secondary)',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {tag.key}:{tag.value}
                  <button
                    type="button"
                    disabled={!can('endpoints', 'create')}
                    title={!can('endpoints', 'create') ? "You don't have permission" : 'Remove tag'}
                    onClick={() =>
                      unassignTag.mutate({ tagId: tag.id, endpointIds: [endpoint.id] })
                    }
                    style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      padding: 0,
                      border: 'none',
                      background: 'transparent',
                      cursor: !can('endpoints', 'create') ? 'not-allowed' : 'pointer',
                      opacity: !can('endpoints', 'create') ? 0.5 : 1,
                      color: 'var(--text-faint)',
                      lineHeight: 1,
                    }}
                  >
                    <X style={{ width: 10, height: 10 }} />
                  </button>
                </span>
              ))
            )}
          </div>

          {/* Divider between Tags and Matched Policies */}
          <div style={{ borderTop: '1px solid var(--border)', margin: '12px 0' }} />

          {/* Matched policies */}
          {matchedPolicies.length > 0 ? (
            <>
              <div
                style={{
                  fontSize: 10,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-muted)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  marginBottom: 8,
                }}
              >
                Matched Policies
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                {matchedPolicies.map((policy) => (
                  <Link
                    key={policy.id}
                    to={`/policies/${policy.id}`}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      padding: '5px 8px',
                      borderRadius: 5,
                      border: '1px solid var(--border)',
                      background: 'var(--bg-inset)',
                      textDecoration: 'none',
                    }}
                  >
                    <span
                      style={{
                        fontSize: 11,
                        color: 'var(--text-secondary)',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {policy.name}
                    </span>
                    <ChevronRight
                      style={{ width: 12, height: 12, color: 'var(--text-faint)', flexShrink: 0 }}
                    />
                  </Link>
                ))}
              </div>
            </>
          ) : (
            <div
              style={{
                fontSize: 10,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-muted)',
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                marginBottom: 4,
              }}
            >
              Matched Policies
            </div>
          )}
        </Tile>
      </div>

      {/* ── Row 3: Lifecycle Timeline ── */}
      <LifecycleTimeline endpoint={endpoint} />

      <CreateTagDialog
        open={showCreateTag}
        onOpenChange={setShowCreateTag}
        preSelectedEndpointId={endpoint.id}
      />
    </div>
  );
}
