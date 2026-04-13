import { useRef, useEffect, useCallback, useMemo, useState } from 'react';
import { createPortal } from 'react-dom';
import { useNavigate } from 'react-router';
import { useEndpoints, type Endpoint } from '@/api/hooks/useEndpoints';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const DOT_RADIUS = 2;
const ENTRANCE_TARGET_MS = 500;
const PULSE_PERIOD = 300; // divisor for sine wave

// ---------------------------------------------------------------------------
// Dynamic dot/layout helpers (pure functions, outside component)
// ---------------------------------------------------------------------------

interface DotLayout {
  DOT: number;
  GAP: number;
  STEP: number;
  cols: number;
}

function computeDotLayout(
  containerWidth: number,
  containerHeight: number,
  totalDots: number,
): DotLayout {
  if (containerWidth <= 0 || containerHeight <= 0 || totalDots === 0) {
    return { DOT: 8, GAP: 4, STEP: 12, cols: Math.max(1, Math.floor(containerWidth / 12)) };
  }
  const containerArea = containerWidth * containerHeight;
  const dotArea = containerArea / totalDots;
  const cellSize = Math.sqrt(dotArea);
  const DOT = Math.max(6, Math.min(24, Math.round(cellSize * 0.7)));
  const GAP = Math.max(2, Math.round(DOT * 0.35));
  const STEP = DOT + GAP;
  const cols = Math.max(1, Math.floor(containerWidth / STEP));
  return { DOT, GAP, STEP, cols };
}

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  transition: 'border-color 150ms ease',
  display: 'flex',
  flexDirection: 'column',
  height: '100%',
};

const dropdownStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  color: 'var(--text-muted)',
  borderRadius: 4,
  fontSize: 11,
  padding: '2px 6px',
  fontFamily: 'var(--font-mono)',
  cursor: 'pointer',
  outline: 'none',
};

// ---------------------------------------------------------------------------
// Color helpers
// ---------------------------------------------------------------------------

function getCanvasColors() {
  const s = getComputedStyle(document.documentElement);
  return {
    colorHealthy: s.getPropertyValue('--signal-healthy').trim() || '#22c55e',
    colorCritical: s.getPropertyValue('--signal-critical').trim() || '#ef4444',
    colorWarning: s.getPropertyValue('--signal-warning').trim() || '#f59e0b',
    colorOffline:
      s.getPropertyValue('--text-faint').trim() ||
      s.getPropertyValue('--chart-axis-fill').trim() ||
      '#6b7280',
    colorPending: '#3b82f6',
  };
}

type ViewMode = 'status' | 'risk';

interface CategoryDef {
  key: string;
  label: string;
  color: string;
}

function getColor(
  ep: Endpoint,
  mode: ViewMode,
  colors: ReturnType<typeof getCanvasColors>,
): string {
  if (mode === 'status') {
    switch (ep.status) {
      case 'online':
        return colors.colorHealthy;
      case 'offline':
        return colors.colorOffline;
      case 'stale':
        return colors.colorWarning;
      case 'pending':
        return colors.colorPending;
      default:
        return colors.colorOffline;
    }
  }
  // risk mode
  if (ep.status === 'offline') return colors.colorOffline;
  if ((ep.critical_patch_count ?? 0) > 0) return colors.colorCritical;
  if ((ep.high_patch_count ?? 0) > 0) return colors.colorWarning;
  return colors.colorHealthy;
}

function getCategory(ep: Endpoint, mode: ViewMode): string {
  if (mode === 'status') return ep.status;
  if (ep.status === 'offline') return 'offline';
  if ((ep.critical_patch_count ?? 0) > 0) return 'critical';
  if ((ep.high_patch_count ?? 0) > 0) return 'high';
  return 'healthy';
}

function sortKey(ep: Endpoint, mode: ViewMode): number {
  if (mode === 'status') {
    const order: Record<string, number> = { online: 2, stale: 1, pending: 3, offline: 4 };
    return order[ep.status] ?? 5;
  }
  if (ep.status === 'offline') return 4;
  if ((ep.critical_patch_count ?? 0) > 0) return 0;
  if ((ep.high_patch_count ?? 0) > 0) return 1;
  if ((ep.pending_patches_count ?? 0) > 0) return 2;
  return 3;
}

// ---------------------------------------------------------------------------
// Relative time
// ---------------------------------------------------------------------------

function relativeTime(iso: string | null | undefined): string {
  if (!iso) return 'never';
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 0) return 'just now';
  const s = Math.floor(diff / 1000);
  if (s < 60) return 'just now';
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  return `${d}d ago`;
}

// ---------------------------------------------------------------------------
// Hovered dot overlay state
// ---------------------------------------------------------------------------

interface HoveredDot {
  dotX: number; // canvas logical X of dot top-left
  dotY: number; // canvas logical Y of dot top-left
  size: number; // DOT size in logical pixels
  color: string;
  ep: Endpoint;
}

// ---------------------------------------------------------------------------
// Tooltip portal component
// ---------------------------------------------------------------------------

function osColor(os: string): string {
  const lower = os.toLowerCase();
  if (lower.includes('linux')) return '#22c55e';
  if (lower.includes('windows')) return '#3b82f6';
  return '#9ca3af';
}

function osLabel(os: string): string {
  const lower = os.toLowerCase();
  if (lower.includes('linux')) return 'Linux';
  if (lower.includes('windows')) return 'Windows';
  if (lower.includes('darwin') || lower.includes('macos')) return 'macOS';
  return os || 'Unknown';
}

function statusBadge(status: string): { bg: string; text: string; label: string } {
  switch (status) {
    case 'online':
      return { bg: 'rgba(34,197,94,0.15)', text: '#22c55e', label: 'Online' };
    case 'offline':
      return { bg: 'rgba(107,114,128,0.15)', text: '#9ca3af', label: 'Offline' };
    case 'stale':
      return { bg: 'rgba(245,158,11,0.15)', text: '#f59e0b', label: 'Stale' };
    case 'pending':
      return { bg: 'rgba(59,130,246,0.15)', text: '#3b82f6', label: 'Pending' };
    default:
      return { bg: 'rgba(107,114,128,0.15)', text: '#9ca3af', label: status };
  }
}

function complianceColor(pct: number): string {
  if (pct >= 80) return '#22c55e';
  if (pct >= 50) return '#f59e0b';
  return '#ef4444';
}

interface TooltipContentProps {
  ep: Endpoint;
  x: number;
  y: number;
}

function TooltipContent({ ep, x, y }: TooltipContentProps) {
  const badge = statusBadge(ep.status);
  return (
    <div
      style={{
        position: 'fixed',
        left: x,
        top: y,
        background: 'var(--bg-elevated, var(--bg-card))',
        border: '1px solid var(--border)',
        borderRadius: 6,
        padding: '10px 12px',
        boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
        pointerEvents: 'none',
        zIndex: 9999,
        minWidth: 160,
      }}
    >
      <div
        style={{
          fontWeight: 600,
          fontSize: 12,
          color: 'var(--text)',
          marginBottom: 6,
          fontFamily: 'var(--font-mono)',
        }}
      >
        {ep.hostname}
      </div>
      {/* OS */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 5,
          fontSize: 11,
          color: 'var(--text-muted)',
          marginBottom: 4,
        }}
      >
        <div
          style={{
            width: 6,
            height: 6,
            borderRadius: '50%',
            background: osColor(ep.os_family),
            flexShrink: 0,
          }}
        />
        {osLabel(ep.os_family)}
      </div>
      {/* Status badge */}
      <div
        style={{
          display: 'inline-block',
          padding: '1px 6px',
          borderRadius: 9,
          fontSize: 10,
          fontWeight: 500,
          background: badge.bg,
          color: badge.text,
          marginBottom: 6,
        }}
      >
        {badge.label}
      </div>
      {/* Critical patches */}
      {(ep.critical_patch_count ?? 0) > 0 && (
        <div style={{ fontSize: 11, color: '#ef4444', marginBottom: 3 }}>
          {ep.critical_patch_count} critical patches
        </div>
      )}
      {/* Compliance */}
      {ep.compliance_pct != null && (
        <div
          style={{
            fontSize: 11,
            color: complianceColor(ep.compliance_pct),
            marginBottom: 3,
          }}
        >
          {Math.round(ep.compliance_pct ?? 0)}% compliant
        </div>
      )}
      {/* Last seen */}
      <div
        style={{
          fontSize: 10,
          color: 'var(--text-faint)',
          fontFamily: 'var(--font-mono)',
          marginTop: 2,
        }}
      >
        seen {relativeTime(ep.last_seen)}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function RiskLandscape() {
  const [viewMode, setViewMode] = useState<ViewMode>('risk');
  const [highlightCategory, setHighlightCategory] = useState<string | null>(null);
  const [hoveredDot, setHoveredDot] = useState<HoveredDot | null>(null);
  const [tooltipViewport, setTooltipViewport] = useState<{ x: number; y: number } | null>(null);

  const canvasRef = useRef<HTMLCanvasElement>(null);
  const canvasWrapperRef = useRef<HTMLDivElement>(null);
  const endpointMapRef = useRef<Endpoint[]>([]);
  const layoutRef = useRef<DotLayout>({ DOT: 8, GAP: 4, STEP: 12, cols: 1 });
  const animFrameRef = useRef(0);
  const entranceDoneRef = useRef(false);
  // Store critical dot positions for pulse
  const criticalDotsRef = useRef<{ idx: number; x: number; y: number; color: string }[]>([]);
  const colorsRef = useRef(getCanvasColors());

  useEffect(() => {
    const update = () => {
      colorsRef.current = getCanvasColors();
    };
    const observer = new MutationObserver(update);
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class', 'data-theme', 'style'],
    });
    return () => observer.disconnect();
  }, []);

  const { data: endpointData, isLoading } = useEndpoints({ limit: 500 });
  const navigate = useNavigate();

  // Sorted endpoints
  const sortedEndpoints = useMemo(() => {
    if (!endpointData?.data) return [];
    return [...endpointData.data].sort((a, b) => sortKey(a, viewMode) - sortKey(b, viewMode));
  }, [endpointData, viewMode]);

  // Category counts for legend
  const categories = useMemo((): CategoryDef[] => {
    const colors = colorsRef.current;
    const counts: Record<string, number> = {};
    for (const ep of sortedEndpoints) {
      const cat = getCategory(ep, viewMode);
      counts[cat] = (counts[cat] ?? 0) + 1;
    }
    if (viewMode === 'status') {
      return [
        { key: 'online', label: `${counts['online'] ?? 0} online`, color: colors.colorHealthy },
        { key: 'offline', label: `${counts['offline'] ?? 0} offline`, color: colors.colorOffline },
        { key: 'stale', label: `${counts['stale'] ?? 0} stale`, color: colors.colorWarning },
        { key: 'pending', label: `${counts['pending'] ?? 0} pending`, color: colors.colorPending },
      ];
    }
    return [
      {
        key: 'critical',
        label: `${counts['critical'] ?? 0} critical`,
        color: colors.colorCritical,
      },
      { key: 'high', label: `${counts['high'] ?? 0} high`, color: colors.colorWarning },
      { key: 'healthy', label: `${counts['healthy'] ?? 0} healthy`, color: colors.colorHealthy },
      { key: 'offline', label: `${counts['offline'] ?? 0} offline`, color: colors.colorOffline },
    ];
  }, [sortedEndpoints, viewMode]);

  // ---------------------------------------------------------------------------
  // Draw (static frame, no entrance animation)
  // ---------------------------------------------------------------------------

  const drawStatic = useCallback(
    (highlightCat: string | null) => {
      const canvas = canvasRef.current;
      const wrapper = canvasWrapperRef.current;
      if (!canvas || !wrapper || sortedEndpoints.length === 0) return;

      const w = wrapper.offsetWidth;
      const h = wrapper.offsetHeight;
      if (w <= 0 || h <= 0) return;

      const dpr = window.devicePixelRatio || 1;
      const layout = computeDotLayout(w, h, sortedEndpoints.length);
      layoutRef.current = layout;
      const { DOT, STEP, cols } = layout;

      canvas.width = w * dpr;
      canvas.height = h * dpr;
      canvas.style.width = '100%';
      canvas.style.height = '100%';

      const ctxRaw = canvas.getContext('2d');
      if (!ctxRaw) return;
      const ctx: CanvasRenderingContext2D = ctxRaw;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.clearRect(0, 0, w, h);

      const colors = colorsRef.current;
      endpointMapRef.current = sortedEndpoints;
      const criticals: { idx: number; x: number; y: number; color: string }[] = [];

      for (let i = 0; i < sortedEndpoints.length; i++) {
        const ep = sortedEndpoints[i];
        const col = i % cols;
        const row = Math.floor(i / cols);
        const x = col * STEP;
        const y = row * STEP;
        const color = getColor(ep, viewMode, colors);
        const cat = getCategory(ep, viewMode);
        const dimmed = highlightCat !== null && cat !== highlightCat;

        ctx.globalAlpha = dimmed ? 0.2 : 1;

        const isCritical = viewMode === 'risk' && cat === 'critical';
        if (isCritical && !dimmed) {
          ctx.shadowColor = color;
          ctx.shadowBlur = 6;
          criticals.push({ idx: i, x, y, color });
        }

        ctx.beginPath();
        ctx.roundRect(x, y, DOT, DOT, DOT_RADIUS);
        ctx.fillStyle = color;
        ctx.fill();

        if (isCritical && !dimmed) {
          ctx.shadowBlur = 0;
        }
      }
      ctx.globalAlpha = 1;
      criticalDotsRef.current = criticals;
    },
    [sortedEndpoints, viewMode],
  );

  // ---------------------------------------------------------------------------
  // Entrance animation
  // ---------------------------------------------------------------------------

  const pulseFrameRef = useRef(0);
  const pulseCleanupRef = useRef<(() => void) | null>(null);

  const startPulse = useCallback(() => {
    cancelAnimationFrame(pulseFrameRef.current);

    let running = true;

    function pulse() {
      if (!running) return;
      const canvas = canvasRef.current;
      if (!canvas || !entranceDoneRef.current) return;
      const criticals = criticalDotsRef.current;
      if (criticals.length === 0) return;

      const ctxRaw = canvas.getContext('2d');
      if (!ctxRaw) return;
      const ctx: CanvasRenderingContext2D = ctxRaw;
      const dpr = window.devicePixelRatio || 1;
      const time = performance.now();
      const opacity = 0.85 + 0.15 * Math.sin(time / PULSE_PERIOD);
      const { DOT } = layoutRef.current;

      ctx.save();
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

      for (const dot of criticals) {
        ctx.clearRect(dot.x - 8, dot.y - 8, DOT + 16, DOT + 16);
        ctx.globalAlpha = opacity;
        ctx.shadowColor = dot.color;
        ctx.shadowBlur = 6;
        ctx.beginPath();
        ctx.roundRect(dot.x, dot.y, DOT, DOT, DOT_RADIUS);
        ctx.fillStyle = dot.color;
        ctx.fill();
        ctx.shadowBlur = 0;
      }

      ctx.globalAlpha = 1;
      ctx.restore();

      if (running && !document.hidden) {
        pulseFrameRef.current = requestAnimationFrame(pulse);
      }
    }

    // Pause/resume on visibility change
    function onVisibility() {
      if (document.hidden) {
        cancelAnimationFrame(pulseFrameRef.current);
      } else if (running) {
        pulseFrameRef.current = requestAnimationFrame(pulse);
      }
    }
    document.addEventListener('visibilitychange', onVisibility);

    pulseFrameRef.current = requestAnimationFrame(pulse);

    // Return cleanup — store it on a ref so the parent effect can call it
    pulseCleanupRef.current = () => {
      running = false;
      cancelAnimationFrame(pulseFrameRef.current);
      document.removeEventListener('visibilitychange', onVisibility);
    };
  }, []);

  const runEntranceAnimation = useCallback(() => {
    const canvas = canvasRef.current;
    const wrapper = canvasWrapperRef.current;
    if (!canvas || !wrapper || sortedEndpoints.length === 0) return;

    const w = wrapper.offsetWidth;
    const h = wrapper.offsetHeight;
    if (w <= 0 || h <= 0) return;

    const dpr = window.devicePixelRatio || 1;
    const layout = computeDotLayout(w, h, sortedEndpoints.length);
    layoutRef.current = layout;
    const { DOT, STEP, cols } = layout;

    canvas.width = w * dpr;
    canvas.height = h * dpr;
    canvas.style.width = '100%';
    canvas.style.height = '100%';

    const ctxRaw = canvas.getContext('2d');
    if (!ctxRaw) return;
    const ctx: CanvasRenderingContext2D = ctxRaw;

    endpointMapRef.current = sortedEndpoints;
    const colors = colorsRef.current;
    const total = sortedEndpoints.length;
    const dotsPerFrame = Math.max(1, Math.ceil(total / (ENTRANCE_TARGET_MS / 16)));
    const fadeFrames = 3;

    const birthFrame: number[] = new Array(total).fill(-1);
    let currentFrame = 0;
    let revealedCount = 0;
    const criticals: { idx: number; x: number; y: number; color: string }[] = [];

    entranceDoneRef.current = false;

    function frame() {
      ctx.save();
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.clearRect(0, 0, w, h);

      // Reveal new dots
      const newReveal = Math.min(revealedCount + dotsPerFrame, total);
      for (let i = revealedCount; i < newReveal; i++) {
        birthFrame[i] = currentFrame;
      }
      revealedCount = newReveal;

      for (let i = 0; i < revealedCount; i++) {
        const ep = sortedEndpoints[i];
        const col = i % cols;
        const row = Math.floor(i / cols);
        const x = col * STEP;
        const y = row * STEP;
        const color = getColor(ep, viewMode, colors);

        const age = currentFrame - birthFrame[i];
        const alpha = Math.min(1, age / fadeFrames);
        ctx.globalAlpha = alpha;

        const cat = getCategory(ep, viewMode);
        const isCritical = viewMode === 'risk' && cat === 'critical';
        if (isCritical) {
          ctx.shadowColor = color;
          ctx.shadowBlur = 6;
        }

        ctx.beginPath();
        ctx.roundRect(x, y, DOT, DOT, DOT_RADIUS);
        ctx.fillStyle = color;
        ctx.fill();

        if (isCritical) {
          ctx.shadowBlur = 0;
          if (alpha >= 1) {
            criticals.push({ idx: i, x, y, color });
          }
        }
      }
      ctx.globalAlpha = 1;
      ctx.restore();

      currentFrame++;
      if (revealedCount < total || currentFrame - birthFrame[revealedCount - 1] < fadeFrames) {
        animFrameRef.current = requestAnimationFrame(frame);
      } else {
        entranceDoneRef.current = true;
        criticalDotsRef.current = criticals;
        startPulse();
      }
    }

    animFrameRef.current = requestAnimationFrame(frame);
  }, [sortedEndpoints, viewMode, startPulse]);

  // ---------------------------------------------------------------------------
  // Effects
  // ---------------------------------------------------------------------------

  const prevEndpointCountRef = useRef(0);
  useEffect(() => {
    cancelAnimationFrame(animFrameRef.current);
    pulseCleanupRef.current?.();

    if (sortedEndpoints.length === 0) return;

    if (prevEndpointCountRef.current === 0 && sortedEndpoints.length > 0) {
      prevEndpointCountRef.current = sortedEndpoints.length;
      runEntranceAnimation();
    } else {
      entranceDoneRef.current = true;
      drawStatic(highlightCategory);
      startPulse();
    }

    return () => {
      cancelAnimationFrame(animFrameRef.current);
      pulseCleanupRef.current?.();
    };
  }, [sortedEndpoints, viewMode]);

  // Redraw on highlight change (skip entrance)
  useEffect(() => {
    if (!entranceDoneRef.current) return;
    pulseCleanupRef.current?.();
    drawStatic(highlightCategory);
    startPulse();
  }, [highlightCategory, drawStatic, startPulse]);

  // ResizeObserver on the canvas wrapper
  useEffect(() => {
    const wrapper = canvasWrapperRef.current;
    if (!wrapper) return;

    let timer: ReturnType<typeof setTimeout>;
    const ro = new ResizeObserver(() => {
      clearTimeout(timer);
      timer = setTimeout(() => {
        if (!entranceDoneRef.current) return;
        pulseCleanupRef.current?.();
        drawStatic(highlightCategory);
        startPulse();
      }, 150);
    });
    ro.observe(wrapper);
    return () => {
      clearTimeout(timer);
      ro.disconnect();
    };
  }, [drawStatic, highlightCategory, startPulse]);

  // ---------------------------------------------------------------------------
  // Mouse handlers — no canvas hover drawing, use CSS overlay instead
  // ---------------------------------------------------------------------------

  const handleMouseMove = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      const canvas = canvasRef.current;
      if (!canvas) return;
      const rect = canvas.getBoundingClientRect();
      // Canvas logical size matches container size (DPR scaling is internal)
      const mx = e.clientX - rect.left;
      const my = e.clientY - rect.top;

      const { DOT, STEP, cols } = layoutRef.current;
      const col = Math.floor(mx / STEP);
      const row = Math.floor(my / STEP);

      // Check within dot bounds (not in gap)
      const dotX = col * STEP;
      const dotY = row * STEP;
      if (mx < dotX || mx > dotX + DOT || my < dotY || my > dotY + DOT) {
        setHoveredDot(null);
        setTooltipViewport(null);
        return;
      }

      const idx = row * cols + col;
      const ep = endpointMapRef.current[idx];
      if (!ep) {
        setHoveredDot(null);
        setTooltipViewport(null);
        return;
      }

      const colors = colorsRef.current;
      const color = getColor(ep, viewMode, colors);

      setHoveredDot({ dotX, dotY, size: DOT, color, ep });
      setTooltipViewport({ x: e.clientX, y: e.clientY });
    },
    [viewMode],
  );

  const handleMouseLeave = useCallback(() => {
    setHoveredDot(null);
    setTooltipViewport(null);
  }, []);

  const handleCanvasClick = useCallback(() => {
    if (hoveredDot) navigate(`/endpoints/${hoveredDot.ep.id}`);
  }, [navigate, hoveredDot]);

  // ---------------------------------------------------------------------------
  // Tooltip clamped position
  // ---------------------------------------------------------------------------

  const clampedTooltipX = tooltipViewport
    ? Math.min(tooltipViewport.x + 14, window.innerWidth - 200)
    : 0;
  const clampedTooltipY = tooltipViewport ? Math.max(tooltipViewport.y - 70, 8) : 0;

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  const totalCount = endpointData?.data?.length ?? 0;

  return (
    <div
      style={cardStyle}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--text-faint)';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)';
      }}
    >
      {/* padding div */}
      <div
        style={{
          padding: '16px 20px 14px',
          flex: 1,
          minHeight: 0,
          display: 'flex',
          flexDirection: 'column',
          position: 'relative',
        }}
      >
        {/* Header */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 14,
            flexShrink: 0,
          }}
        >
          <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
            Risk Landscape
          </span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <select
              value={viewMode}
              onChange={(e) => {
                setViewMode(e.target.value as ViewMode);
                setHighlightCategory(null);
              }}
              style={dropdownStyle}
            >
              <option value="risk">By Risk</option>
              <option value="status">By Status</option>
            </select>
            <span
              onClick={() => navigate('/endpoints')}
              style={{
                fontSize: 11,
                color: 'var(--text-faint)',
                fontFamily: 'var(--font-mono)',
                cursor: isLoading ? 'default' : 'pointer',
              }}
              onMouseEnter={(e) => {
                if (!isLoading)
                  (e.currentTarget as HTMLSpanElement).style.color = 'var(--text-muted)';
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLSpanElement).style.color = 'var(--text-faint)';
              }}
            >
              {isLoading ? '\u2014' : `${totalCount.toLocaleString()} endpoints`}
            </span>
          </div>
        </div>

        {/* Content */}
        {isLoading ? (
          <div
            style={{
              flex: 1,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--text-faint)',
              fontSize: 12,
            }}
          >
            Loading...
          </div>
        ) : (
          <>
            {/* Canvas wrapper — fills remaining flex space, position:relative for overlay */}
            <div
              ref={canvasWrapperRef}
              style={{
                flex: 1,
                minHeight: 0,
                position: 'relative',
              }}
            >
              <canvas
                ref={canvasRef}
                style={{
                  display: 'block',
                  width: '100%',
                  height: '100%',
                  cursor: 'pointer',
                }}
                onMouseMove={handleMouseMove}
                onMouseLeave={handleMouseLeave}
                onClick={handleCanvasClick}
              />

              {/* CSS hover overlay — avoids canvas redraw fighting pulse loop */}
              {hoveredDot && (
                <div
                  style={{
                    position: 'absolute',
                    left: hoveredDot.dotX - hoveredDot.size * 0.2,
                    top: hoveredDot.dotY - hoveredDot.size * 0.2,
                    width: hoveredDot.size * 1.4,
                    height: hoveredDot.size * 1.4,
                    borderRadius: 3,
                    background: hoveredDot.color,
                    border: '1.5px solid rgba(255,255,255,0.75)',
                    boxShadow: `0 0 10px ${hoveredDot.color}88`,
                    pointerEvents: 'none',
                    zIndex: 5,
                    transition: 'width 80ms ease, height 80ms ease, left 80ms ease, top 80ms ease',
                  }}
                />
              )}
            </div>

            {/* Legend */}
            <div
              style={{ display: 'flex', gap: 16, marginTop: 12, flexWrap: 'wrap', flexShrink: 0 }}
            >
              {categories.map((cat) => (
                <div
                  key={cat.key}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 5,
                    fontSize: 11,
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                    cursor: 'pointer',
                    opacity: highlightCategory !== null && highlightCategory !== cat.key ? 0.4 : 1,
                    transition: 'opacity 150ms ease',
                  }}
                  onMouseEnter={() => setHighlightCategory(cat.key)}
                  onMouseLeave={() => setHighlightCategory(null)}
                  onClick={() => {
                    navigate(`/endpoints?status=${cat.key}`);
                  }}
                >
                  <div
                    style={{
                      width: 7,
                      height: 7,
                      borderRadius: 2,
                      flexShrink: 0,
                      background: cat.color,
                    }}
                  />
                  {cat.label}
                </div>
              ))}
            </div>
          </>
        )}
      </div>

      {/* Tooltip portal — rendered into document.body with position:fixed */}
      {tooltipViewport &&
        hoveredDot &&
        createPortal(
          <TooltipContent ep={hoveredDot.ep} x={clampedTooltipX} y={clampedTooltipY} />,
          document.body,
        )}
    </div>
  );
}
