import { useState, useRef, useEffect, useMemo } from 'react';
import { Copy, Download, Trash2, Pause, Play } from 'lucide-react';
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
  TooltipProvider,
  Button,
  Skeleton,
} from '@patchiq/ui';
import { useLogs } from '../../api/hooks/useLogs';
import type { components } from '../../api/types';

type LogEntry = components['schemas']['LogEntry'];

const LEVELS = ['ALL', 'DEBUG', 'INFO', 'WARN', 'ERROR'] as const;
type LevelFilter = (typeof LEVELS)[number];

const ALL_SOURCES_VALUE = 'All Sources';

const levelGutterColor: Record<string, string> = {
  error: 'var(--signal-critical)',
  warn: 'var(--signal-warning)',
  info: 'var(--text-muted)',
  debug: 'var(--border)',
};

const levelBadgeStyle: Record<string, { bg: string; color: string; border: string }> = {
  error: {
    bg: 'color-mix(in srgb, var(--signal-critical) 15%, transparent)',
    color: 'var(--signal-critical)',
    border: 'color-mix(in srgb, var(--signal-critical) 30%, transparent)',
  },
  warn: {
    bg: 'color-mix(in srgb, var(--signal-warning) 15%, transparent)',
    color: 'var(--signal-warning)',
    border: 'color-mix(in srgb, var(--signal-warning) 30%, transparent)',
  },
  info: {
    bg: 'color-mix(in srgb, var(--text-muted) 15%, transparent)',
    color: 'var(--text-muted)',
    border: 'color-mix(in srgb, var(--text-muted) 30%, transparent)',
  },
  debug: {
    bg: 'transparent',
    color: 'var(--text-faint)',
    border: 'var(--border)',
  },
};

const defaultSourceBadgeColors = { color: 'var(--text-secondary)', border: 'var(--border)' };

function formatTimestamp(iso: string): { time: string; tooltip: string } {
  const date = new Date(iso);
  const now = new Date();
  const isToday = date.toDateString() === now.toDateString();
  const yesterday = new Date(now);
  yesterday.setDate(yesterday.getDate() - 1);
  const isYesterday = date.toDateString() === yesterday.toDateString();

  const timeStr = date.toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
  const fullDate = date.toLocaleString();

  const prefix = isToday
    ? 'today'
    : isYesterday
      ? 'yesterday'
      : date.toLocaleDateString([], { month: 'short', day: 'numeric' });

  return {
    time: `${prefix} ${timeStr}`,
    tooltip: fullDate,
  };
}

function highlight(text: string, search: string): React.ReactNode {
  if (!search) return text;
  const idx = text.toLowerCase().indexOf(search.toLowerCase());
  if (idx === -1) return text;
  return (
    <>
      {text.slice(0, idx)}
      <mark
        style={{
          background: 'color-mix(in srgb, var(--signal-warning) 30%, transparent)',
          color: 'var(--signal-warning)',
          borderRadius: '2px',
          padding: '0 2px',
        }}
      >
        {text.slice(idx, idx + search.length)}
      </mark>
      {highlight(text.slice(idx + search.length), search)}
    </>
  );
}

function LogLine({ entry, search }: { entry: LogEntry; search: string }) {
  const [hovered, setHovered] = useState(false);
  const faded = search !== '' && !entry.message.toLowerCase().includes(search.toLowerCase());
  const gutterColor = levelGutterColor[entry.level ?? 'debug'] ?? 'var(--text-muted)';
  const badge = levelBadgeStyle[entry.level ?? 'debug'] ?? levelBadgeStyle.debug;
  const srcColors = defaultSourceBadgeColors;
  const ts = formatTimestamp(entry.timestamp);

  function handleCopy() {
    navigator.clipboard
      .writeText(
        `${entry.timestamp} [${entry.level?.toUpperCase()}] ${entry.source ? `(${entry.source}) ` : ''}${entry.message}`,
      )
      .catch(() => {});
  }

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'stretch',
        fontFamily: 'var(--font-mono)',
        fontSize: '12px',
        opacity: faded ? 0.25 : 1,
        transition: 'opacity 0.1s',
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <div style={{ width: '4px', flexShrink: 0, background: gutterColor }} />
      <div
        style={{
          display: 'flex',
          flex: 1,
          alignItems: 'flex-start',
          gap: '8px',
          padding: '4px 12px',
          background: hovered ? 'var(--accent-subtle)' : 'transparent',
        }}
      >
        <TooltipProvider delayDuration={200}>
          <Tooltip>
            <TooltipTrigger asChild>
              <span
                style={{
                  color: 'var(--text-faint)',
                  whiteSpace: 'nowrap',
                  flexShrink: 0,
                  fontSize: '10px',
                  cursor: 'default',
                }}
              >
                {ts.time}
              </span>
            </TooltipTrigger>
            <TooltipContent side="top">
              <span style={{ fontSize: '11px' }}>{ts.tooltip}</span>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>

        <span
          style={{
            fontSize: '10px',
            padding: '1px 5px',
            borderRadius: '3px',
            border: `1px solid ${badge.border}`,
            background: badge.bg,
            color: badge.color,
            fontWeight: 700,
            textTransform: 'uppercase',
            flexShrink: 0,
            lineHeight: '14px',
          }}
        >
          {entry.level}
        </span>

        {entry.source && (
          <span
            style={{
              fontSize: '10px',
              padding: '1px 5px',
              borderRadius: '3px',
              border: `1px solid ${srcColors.border}`,
              color: srcColors.color,
              flexShrink: 0,
              lineHeight: '14px',
            }}
          >
            {entry.source}
          </span>
        )}

        <span style={{ color: 'var(--text-primary)', flex: 1, wordBreak: 'break-all' }}>
          {highlight(entry.message, search)}
        </span>

        {hovered && (
          <button
            onClick={handleCopy}
            style={{
              flexShrink: 0,
              color: 'var(--text-faint)',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              padding: '0',
            }}
          >
            <Copy style={{ width: '12px', height: '12px' }} />
          </button>
        )}
      </div>
    </div>
  );
}

const SELECT_STYLE: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: '6px',
  color: 'var(--text-emphasis)',
  padding: '6px 10px',
  fontSize: '13px',
  cursor: 'pointer',
};

const INPUT_STYLE: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: '6px',
  color: 'var(--text-emphasis)',
  padding: '6px 10px',
  fontSize: '13px',
  outline: 'none',
  width: '180px',
};

const BTN_STYLE: React.CSSProperties = {
  background: 'transparent',
  border: '1px solid var(--border)',
  borderRadius: '6px',
  color: 'var(--text-secondary)',
  padding: '6px 12px',
  fontSize: '12px',
  cursor: 'pointer',
  display: 'flex',
  alignItems: 'center',
  gap: '6px',
};

function LevelPill({
  level,
  active,
  onClick,
}: {
  level: string;
  active: boolean;
  onClick: () => void;
}) {
  const colors =
    level === 'ALL'
      ? { bg: 'var(--accent-subtle)', color: 'var(--text-emphasis)', border: 'var(--border)' }
      : (levelBadgeStyle[level.toLowerCase()] ?? {
          bg: 'transparent',
          color: 'var(--text-secondary)',
          border: 'var(--border)',
        });
  return (
    <button
      onClick={onClick}
      style={{
        padding: '4px 12px',
        borderRadius: '16px',
        fontSize: '12px',
        fontWeight: 500,
        cursor: 'pointer',
        border: `1px solid ${active ? colors.border : 'var(--border)'}`,
        background: active ? (level === 'ALL' ? 'var(--accent-subtle)' : colors.bg) : 'transparent',
        color: active ? colors.color : 'var(--text-muted)',
        transition: 'all 0.15s',
      }}
    >
      {level.charAt(0) + level.slice(1).toLowerCase()}
    </button>
  );
}

export const LogsPage = () => {
  const [levelFilter, setLevelFilter] = useState<LevelFilter>('ALL');
  const [sourceFilter, setSourceFilter] = useState<string>(ALL_SOURCES_VALUE);
  const [search, setSearch] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [cleared, setCleared] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);

  const apiLevel =
    levelFilter === 'ALL'
      ? undefined
      : (levelFilter.toLowerCase() as 'debug' | 'info' | 'warn' | 'error');

  const { data, isLoading, isError } = useLogs({
    limit: 200,
    level: apiLevel,
    refetchInterval: autoRefresh ? 5000 : undefined,
  });

  const allData = data?.data ?? [];

  const uniqueSources = useMemo(() => {
    if (!allData.length) return [ALL_SOURCES_VALUE];
    const sources = new Set(
      allData.map((entry) => entry.source).filter((s): s is string => Boolean(s)),
    );
    return [ALL_SOURCES_VALUE, ...Array.from(sources).sort()];
  }, [allData]);

  const all = cleared ? [] : allData;

  // Apply source filter client-side
  const afterSource =
    sourceFilter === ALL_SOURCES_VALUE ? all : all.filter((e) => e.source === sourceFilter);

  const entries = search
    ? afterSource.filter((e) => e.message.toLowerCase().includes(search.toLowerCase()))
    : afterSource;

  useEffect(() => {
    if (autoRefresh) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [entries.length, autoRefresh]);

  function handleExport() {
    const text = all
      .map(
        (e) =>
          `${e.timestamp} [${e.level?.toUpperCase()}] ${e.source ? `(${e.source}) ` : ''}${e.message}`,
      )
      .join('\n');
    const blob = new Blob([text], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'agent-logs.txt';
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: '12px',
        height: 'calc(100vh - 140px)',
      }}
    >
      {/* Subtitle */}
      <p style={{ fontSize: '13px', color: 'var(--text-muted)', margin: 0 }}>
        Real-time agent operation logs
      </p>

      {/* Control bar */}
      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'center',
          gap: '8px',
          flexShrink: 0,
        }}
      >
        {/* Level pill buttons */}
        <div style={{ display: 'flex', gap: '4px' }}>
          {LEVELS.map((l) => (
            <LevelPill
              key={l}
              level={l}
              active={levelFilter === l}
              onClick={() => setLevelFilter(l)}
            />
          ))}
        </div>

        <div style={{ width: '1px', height: '24px', background: 'var(--border)' }} />

        {/* Source filter */}
        <select
          value={sourceFilter}
          onChange={(e) => setSourceFilter(e.target.value)}
          style={SELECT_STYLE}
        >
          {uniqueSources.map((s) => (
            <option key={s} value={s ?? ''}>
              {s}
            </option>
          ))}
        </select>

        {/* Search */}
        <input
          style={INPUT_STYLE}
          placeholder="Search logs..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />

        <div style={{ marginLeft: 'auto', display: 'flex', gap: '8px', alignItems: 'center' }}>
          {/* Auto-refresh toggle */}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setAutoRefresh((r) => !r)}
            style={{
              gap: '6px',
              borderColor: autoRefresh ? 'var(--accent)' : 'var(--border)',
              color: autoRefresh ? 'var(--accent)' : 'var(--text-muted)',
            }}
          >
            {autoRefresh ? (
              <>
                <Pause style={{ width: '14px', height: '14px' }} />
                <span>Pause</span>
                <span
                  style={{
                    width: '6px',
                    height: '6px',
                    borderRadius: '50%',
                    background: 'var(--signal-healthy)',
                    display: 'inline-block',
                  }}
                  className="animate-pulse"
                />
              </>
            ) : (
              <>
                <Play style={{ width: '14px', height: '14px' }} />
                <span>Resume</span>
              </>
            )}
          </Button>

          <button style={BTN_STYLE} onClick={() => setCleared(true)}>
            <Trash2 style={{ width: '14px', height: '14px' }} />
            Clear
          </button>

          <button style={BTN_STYLE} onClick={handleExport}>
            <Download style={{ width: '14px', height: '14px' }} />
            Export
          </button>
        </div>
      </div>

      {/* Log viewer */}
      <div
        data-testid="log-viewer"
        style={{
          flex: 1,
          overflowY: 'auto',
          borderRadius: '8px',
          background: 'var(--bg-canvas)',
          border: '1px solid var(--border-faint)',
          minHeight: 0,
        }}
      >
        {isLoading ? (
          <div style={{ padding: '16px', display: 'flex', flexDirection: 'column', gap: 8 }}>
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <Skeleton key={i} className="h-5 w-full rounded" />
            ))}
          </div>
        ) : isError ? (
          <div
            style={{
              display: 'flex',
              height: '200px',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <span style={{ fontSize: 13, color: 'var(--signal-critical)' }}>
              Failed to load logs.
            </span>
          </div>
        ) : entries.length === 0 ? (
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              height: '200px',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '8px',
              color: 'var(--text-faint)',
              fontSize: '13px',
            }}
          >
            <p style={{ margin: 0, fontWeight: 500 }}>No agent logs yet</p>
            <p style={{ margin: 0, fontSize: '12px', color: 'var(--text-faint)' }}>
              Logs will appear as the agent performs operations.
            </p>
          </div>
        ) : (
          <div style={{ paddingTop: '8px', paddingBottom: '8px' }}>
            {entries.map((entry) => (
              <LogLine key={entry.id} entry={entry} search={search} />
            ))}
            <div ref={bottomRef} />
          </div>
        )}
      </div>
    </div>
  );
};
