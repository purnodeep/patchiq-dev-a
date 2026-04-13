import React, { useState, useMemo } from 'react';
import { RefreshCw, Search, ArrowUpDown, Package, ChevronDown, ChevronUp } from 'lucide-react';
import { Skeleton, Button } from '@patchiq/ui';
import { useAgentSoftware } from '../../api/hooks/useSoftware';

const CARD_STYLE: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 'var(--radius-xl)',
  padding: '20px',
};

const CARD_TITLE_STYLE: React.CSSProperties = {
  fontSize: '12px',
  textTransform: 'uppercase',
  color: 'var(--text-muted)',
  letterSpacing: '0.08em',
  fontWeight: 600,
};

const PILL_BASE: React.CSSProperties = {
  padding: '4px 14px',
  borderRadius: '20px',
  fontSize: '12px',
  fontWeight: 500,
  cursor: 'pointer',
  border: '1px solid var(--border)',
  background: 'transparent',
  color: 'var(--text-muted)',
  transition: 'all 0.15s',
};

const PILL_ACTIVE: React.CSSProperties = {
  borderColor: 'var(--accent)',
  background: 'var(--accent-subtle)',
  color: 'var(--accent)',
};

const MONOCHROME_BADGE = {
  color: 'var(--text-secondary)',
  bg: 'var(--bg-card-hover)',
  border: 'var(--border)',
};

const CATEGORY_COLORS: Record<string, { color: string; bg: string; border: string }> = {
  Application: MONOCHROME_BADGE,
  Library: MONOCHROME_BADGE,
  Development: MONOCHROME_BADGE,
  System: MONOCHROME_BADGE,
  Network: MONOCHROME_BADGE,
  'Language Runtime': MONOCHROME_BADGE,
  Kernel: MONOCHROME_BADGE,
  Font: MONOCHROME_BADGE,
  Security: MONOCHROME_BADGE,
  Other: MONOCHROME_BADGE,
};

function CategoryBadge({ category }: { category: string | undefined }) {
  const cat = category || 'Other';
  const c = CATEGORY_COLORS[cat] ?? CATEGORY_COLORS['Other'];
  return (
    <span
      style={{
        fontSize: '11px',
        padding: '2px 8px',
        borderRadius: '4px',
        border: `1px solid ${c.border}`,
        background: c.bg,
        color: c.color,
        fontWeight: 500,
        whiteSpace: 'nowrap',
      }}
    >
      {cat}
    </span>
  );
}

function SourceBadge({ source }: { source: string }) {
  const c = {
    color: 'var(--text-secondary)',
    bg: 'var(--bg-card-hover)',
    border: 'var(--border)',
  };
  return (
    <span
      style={{
        fontSize: '11px',
        padding: '2px 8px',
        borderRadius: '4px',
        border: `1px solid ${c.border}`,
        background: c.bg,
        color: c.color,
        fontFamily: 'var(--font-mono)',
      }}
    >
      {source}
    </span>
  );
}

type SortKey = 'name' | 'installed_size_kb' | 'install_date';
type SortDir = 'asc' | 'desc';

function formatSize(kb: number | undefined): string {
  if (!kb || kb === 0) return '\u2014';
  if (kb >= 1024 * 1024) return `${(kb / (1024 * 1024)).toFixed(1)} GB`;
  if (kb >= 1024) return `${(kb / 1024).toFixed(1)} MB`;
  return `${kb} KB`;
}

function formatRelativeDate(iso: string | undefined): string {
  if (!iso) return '\u2014';
  const date = new Date(iso);
  if (isNaN(date.getTime())) return iso;
  const now = Date.now();
  const diffMs = now - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return 'just now';
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHrs = Math.floor(diffMin / 60);
  if (diffHrs < 24) return `${diffHrs}h ago`;
  const diffDays = Math.floor(diffHrs / 24);
  if (diffDays < 30) return `${diffDays}d ago`;
  const diffMonths = Math.floor(diffDays / 30);
  if (diffMonths < 12) return `${diffMonths} month${diffMonths !== 1 ? 's' : ''} ago`;
  const diffYears = Math.floor(diffDays / 365);
  return `${diffYears} year${diffYears !== 1 ? 's' : ''} ago`;
}

const PAGE_SIZE = 50;

const SORT_OPTIONS: { key: SortKey; label: string }[] = [
  { key: 'name', label: 'Name' },
  { key: 'installed_size_kb', label: 'Size (largest)' },
  { key: 'install_date', label: 'Install Date (newest)' },
];

const MANAGED_CATEGORIES = ['Application', 'System', 'Kernel', 'Network'];

type ViewMode = 'managed' | 'all';

export function SoftwarePage() {
  const { data, isLoading, isError, refetch } = useAgentSoftware();
  const [viewMode, setViewMode] = useState<ViewMode>('managed');
  const [search, setSearch] = useState('');
  const [sourceFilter, setSourceFilter] = useState<string>('all');
  const [categoryFilter, setCategoryFilter] = useState<string>('all');
  const [sortKey, setSortKey] = useState<SortKey>('name');
  const [sortDir, setSortDir] = useState<SortDir>('asc');
  const [page, setPage] = useState(0);
  const [expandedRow, setExpandedRow] = useState<string | null>(null);

  // Base dataset filtered by view mode
  const viewData = useMemo(() => {
    if (!data) return [];
    if (viewMode === 'all') return data;
    return data.filter((p) => MANAGED_CATEGORIES.includes(p.category || 'Other'));
  }, [data, viewMode]);

  // Counts for the toggle labels
  const managedCount = useMemo(() => {
    if (!data) return 0;
    return data.filter((p) => MANAGED_CATEGORIES.includes(p.category || 'Other')).length;
  }, [data]);

  const allCount = data?.length ?? 0;

  const sources = useMemo(() => {
    if (!viewData.length) return [];
    const set = new Set(viewData.map((p) => p.source));
    return Array.from(set).sort();
  }, [viewData]);

  const categoryBreakdown = useMemo(() => {
    if (!viewData.length) return {};
    const counts: Record<string, number> = {};
    for (const p of viewData) {
      const cat = p.category || 'Other';
      counts[cat] = (counts[cat] || 0) + 1;
    }
    return counts;
  }, [viewData]);

  const categories = useMemo(() => {
    return Object.entries(categoryBreakdown)
      .sort((a, b) => b[1] - a[1])
      .map(([cat]) => cat);
  }, [categoryBreakdown]);

  const sourceBreakdown = useMemo(() => {
    if (!viewData.length) return {};
    const counts: Record<string, number> = {};
    for (const p of viewData) {
      counts[p.source] = (counts[p.source] || 0) + 1;
    }
    return counts;
  }, [viewData]);

  const filtered = useMemo(() => {
    if (!viewData.length) return [];
    return viewData
      .filter((p) => {
        if (search) {
          const q = search.toLowerCase();
          return (
            p.name.toLowerCase().includes(q) ||
            (p.description ?? '').toLowerCase().includes(q) ||
            (p.version ?? '').toLowerCase().includes(q)
          );
        }
        return true;
      })
      .filter((p) => sourceFilter === 'all' || p.source === sourceFilter)
      .filter((p) => categoryFilter === 'all' || (p.category || 'Other') === categoryFilter)
      .sort((a, b) => {
        if (sortKey === 'name') {
          const dir = sortDir === 'asc' ? 1 : -1;
          return a.name.localeCompare(b.name) * dir;
        }
        if (sortKey === 'installed_size_kb') {
          // Default to desc for size (largest first)
          const dir = sortDir === 'asc' ? 1 : -1;
          return ((a.installed_size_kb ?? 0) - (b.installed_size_kb ?? 0)) * dir;
        }
        if (sortKey === 'install_date') {
          // Default to desc for date (newest first)
          const dir = sortDir === 'asc' ? 1 : -1;
          const dateA = a.install_date ? new Date(a.install_date).getTime() : 0;
          const dateB = b.install_date ? new Date(b.install_date).getTime() : 0;
          return (dateA - dateB) * dir;
        }
        return 0;
      });
  }, [viewData, search, sourceFilter, categoryFilter, sortKey, sortDir]);

  const paged = useMemo(() => {
    const start = page * PAGE_SIZE;
    return filtered.slice(start, start + PAGE_SIZE);
  }, [filtered, page]);

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE);

  function handleSortChange(key: SortKey) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortKey(key);
      // Default directions: name=asc, size=desc, date=desc
      setSortDir(key === 'name' ? 'asc' : 'desc');
    }
    setPage(0);
  }

  function SortableHeader({ label, sortKeyVal }: { label: string; sortKeyVal: SortKey }) {
    const active = sortKey === sortKeyVal;
    return (
      <th
        onClick={() => handleSortChange(sortKeyVal)}
        style={{
          textAlign: 'left',
          padding: '8px 10px',
          color: active ? 'var(--text-primary)' : 'var(--text-muted)',
          fontWeight: 600,
          fontSize: '11px',
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
          whiteSpace: 'nowrap',
          cursor: 'pointer',
          userSelect: 'none',
        }}
      >
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
          {label}
          <ArrowUpDown
            style={{
              width: '12px',
              height: '12px',
              opacity: active ? 1 : 0.3,
              transform: active && sortDir === 'desc' ? 'scaleY(-1)' : undefined,
            }}
          />
        </span>
      </th>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {/* View Mode Toggle */}
      {!isLoading && !isError && data && data.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          <div
            style={{
              display: 'inline-flex',
              background: 'var(--bg-page)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius-lg)',
              padding: '4px',
              gap: '4px',
              alignSelf: 'flex-start',
            }}
          >
            <button
              onClick={() => {
                setViewMode('managed');
                setCategoryFilter('all');
                setSourceFilter('all');
                setPage(0);
              }}
              style={{
                padding: '8px 20px',
                borderRadius: '8px',
                fontSize: '13px',
                fontWeight: 600,
                border: 'none',
                cursor: 'pointer',
                transition: 'all 0.15s',
                background: viewMode === 'managed' ? 'var(--accent)' : 'transparent',
                color: viewMode === 'managed' ? 'var(--bg-canvas)' : 'var(--text-muted)',
              }}
            >
              Managed Software ({managedCount.toLocaleString()})
            </button>
            <button
              onClick={() => {
                setViewMode('all');
                setCategoryFilter('all');
                setSourceFilter('all');
                setPage(0);
              }}
              style={{
                padding: '8px 20px',
                borderRadius: '8px',
                fontSize: '13px',
                fontWeight: 600,
                border: 'none',
                cursor: 'pointer',
                transition: 'all 0.15s',
                background: viewMode === 'all' ? 'var(--accent)' : 'transparent',
                color: viewMode === 'all' ? 'var(--bg-canvas)' : 'var(--text-muted)',
              }}
            >
              All Packages ({allCount.toLocaleString()})
            </button>
          </div>
          <p style={{ fontSize: '13px', color: 'var(--text-muted)', margin: 0 }}>
            {viewMode === 'managed'
              ? 'Applications, system tools, and services installed on this endpoint'
              : 'Complete package inventory including libraries and dependencies'}
          </p>
        </div>
      )}

      {/* Summary Section */}
      {!isLoading && !isError && viewData.length > 0 && (
        <div style={CARD_STYLE}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '16px' }}>
            <Package style={{ width: '18px', height: '18px', color: 'var(--accent)' }} />
            <span style={{ fontSize: '20px', fontWeight: 700, color: 'var(--text-emphasis)' }}>
              {viewData.length.toLocaleString()}
            </span>
            <span style={{ fontSize: '14px', color: 'var(--text-secondary)' }}>
              {viewMode === 'managed' ? 'managed packages' : 'packages installed'}
            </span>
          </div>

          {/* Category breakdown bar */}
          <div style={{ marginBottom: '12px' }}>
            <p style={{ ...CARD_TITLE_STYLE, marginBottom: '10px' }}>By Category</p>
            <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
              {categories.map((cat) => {
                const count = categoryBreakdown[cat];
                const c = CATEGORY_COLORS[cat] ?? CATEGORY_COLORS['Other'];
                return (
                  <div
                    key={cat}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '6px',
                      padding: '6px 12px',
                      borderRadius: '8px',
                      border: `1px solid ${c.border}`,
                      background: c.bg,
                    }}
                  >
                    <span
                      style={{
                        width: '8px',
                        height: '8px',
                        borderRadius: '2px',
                        background: c.color,
                        flexShrink: 0,
                      }}
                    />
                    <span style={{ fontSize: '12px', color: c.color, fontWeight: 500 }}>{cat}</span>
                    <span
                      style={{
                        fontSize: '12px',
                        color: 'var(--text-secondary)',
                        fontFamily: 'var(--font-mono)',
                      }}
                    >
                      {count}
                    </span>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Source breakdown (secondary) */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Sources:</span>
            {Object.entries(sourceBreakdown)
              .sort((a, b) => b[1] - a[1])
              .map(([src, count]) => (
                <span key={src} style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                  {src}: {count}
                </span>
              ))}
          </div>
        </div>
      )}

      {/* Filter bar */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        {/* Row 1: Source filter + Search */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
          <span style={{ ...CARD_TITLE_STYLE, minWidth: '52px' }}>Source</span>
          <button
            onClick={() => {
              setSourceFilter('all');
              setPage(0);
            }}
            style={{ ...PILL_BASE, ...(sourceFilter === 'all' ? PILL_ACTIVE : {}) }}
          >
            All ({viewData.length})
          </button>
          {sources.map((src) => {
            const count = viewData.filter((p) => p.source === src).length;
            return (
              <button
                key={src}
                onClick={() => {
                  setSourceFilter(src);
                  setPage(0);
                }}
                style={{ ...PILL_BASE, ...(sourceFilter === src ? PILL_ACTIVE : {}) }}
              >
                {src} ({count})
              </button>
            );
          })}
          <div
            style={{
              marginLeft: 'auto',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: '6px',
              padding: '6px 10px',
            }}
          >
            <Search style={{ width: '14px', height: '14px', color: 'var(--text-muted)' }} />
            <input
              type="text"
              placeholder="Search packages..."
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(0);
              }}
              style={{
                background: 'transparent',
                border: 'none',
                outline: 'none',
                color: 'var(--text-emphasis)',
                fontSize: '13px',
                width: '180px',
              }}
            />
          </div>
        </div>

        {/* Row 2: Category filter + Sort */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
          <span style={{ ...CARD_TITLE_STYLE, minWidth: '52px' }}>Category</span>
          <button
            onClick={() => {
              setCategoryFilter('all');
              setPage(0);
            }}
            style={{ ...PILL_BASE, ...(categoryFilter === 'all' ? PILL_ACTIVE : {}) }}
          >
            All
          </button>
          {categories.map((cat) => (
            <button
              key={cat}
              onClick={() => {
                setCategoryFilter(cat);
                setPage(0);
              }}
              style={{ ...PILL_BASE, ...(categoryFilter === cat ? PILL_ACTIVE : {}) }}
            >
              {cat} ({categoryBreakdown[cat]})
            </button>
          ))}

          {/* Sort dropdown */}
          <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: '6px' }}>
            <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Sort:</span>
            {SORT_OPTIONS.map((opt) => (
              <button
                key={opt.key}
                onClick={() => handleSortChange(opt.key)}
                style={{
                  ...PILL_BASE,
                  padding: '3px 10px',
                  fontSize: '11px',
                  ...(sortKey === opt.key ? PILL_ACTIVE : {}),
                }}
              >
                {opt.label}
                {sortKey === opt.key && (
                  <span style={{ marginLeft: '4px' }}>
                    {sortDir === 'asc' ? '\u2191' : '\u2193'}
                  </span>
                )}
              </button>
            ))}
          </div>
        </div>
      </div>

      {isLoading && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <Skeleton className="h-24 w-full rounded-xl" />
          {[1, 2, 3, 4, 5].map((i) => (
            <Skeleton key={i} className="h-12 w-full rounded-xl" />
          ))}
        </div>
      )}

      {isError && (
        <div
          style={{
            ...CARD_STYLE,
            border: '1px solid var(--signal-critical)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <p style={{ fontSize: '13px', color: 'var(--signal-critical)' }}>
            Failed to load software packages.
          </p>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" /> Retry
          </Button>
        </div>
      )}

      {!isLoading && !isError && filtered.length === 0 && (
        <div
          style={{
            ...CARD_STYLE,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            padding: '48px 20px',
            color: 'var(--text-muted)',
          }}
        >
          <p>
            {data && data.length === 0
              ? 'No packages found'
              : 'No packages match the current filter'}
          </p>
        </div>
      )}

      {paged.length > 0 && (
        <div style={CARD_STYLE}>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)' }}>
                  <SortableHeader label="Name" sortKeyVal="name" />
                  <th
                    style={{
                      textAlign: 'left',
                      padding: '8px 10px',
                      color: 'var(--text-muted)',
                      fontWeight: 600,
                      fontSize: '11px',
                      textTransform: 'uppercase',
                      letterSpacing: '0.05em',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    Version
                  </th>
                  <th
                    style={{
                      textAlign: 'left',
                      padding: '8px 10px',
                      color: 'var(--text-muted)',
                      fontWeight: 600,
                      fontSize: '11px',
                      textTransform: 'uppercase',
                      letterSpacing: '0.05em',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    Category
                  </th>
                  <th
                    style={{
                      textAlign: 'left',
                      padding: '8px 10px',
                      color: 'var(--text-muted)',
                      fontWeight: 600,
                      fontSize: '11px',
                      textTransform: 'uppercase',
                      letterSpacing: '0.05em',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    Source
                  </th>
                  <SortableHeader label="Size" sortKeyVal="installed_size_kb" />
                  <SortableHeader label="Installed" sortKeyVal="install_date" />
                  <th
                    style={{
                      textAlign: 'left',
                      padding: '8px 10px',
                      color: 'var(--text-muted)',
                      fontWeight: 600,
                      fontSize: '11px',
                      textTransform: 'uppercase',
                      letterSpacing: '0.05em',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    License
                  </th>
                  <th style={{ width: '32px' }} />
                </tr>
              </thead>
              <tbody>
                {paged.map((pkg) => {
                  const rowKey = `${pkg.name}-${pkg.source}`;
                  const isExpanded = expandedRow === rowKey;
                  return (
                    <React.Fragment key={rowKey}>
                      <tr
                        onClick={() => setExpandedRow(isExpanded ? null : rowKey)}
                        style={{
                          borderBottom: isExpanded ? 'none' : '1px solid var(--border-faint)',
                          color: 'var(--text-emphasis)',
                          cursor: 'pointer',
                        }}
                      >
                        <td
                          style={{
                            padding: '8px 10px',
                            fontFamily: 'var(--font-mono)',
                            fontSize: '12px',
                            fontWeight: 500,
                            whiteSpace: 'nowrap',
                          }}
                        >
                          {pkg.name}
                        </td>
                        <td
                          style={{
                            padding: '8px 10px',
                            fontFamily: 'var(--font-mono)',
                            fontSize: '11px',
                            color: 'var(--text-secondary)',
                          }}
                        >
                          {pkg.version}
                        </td>
                        <td style={{ padding: '8px 10px' }}>
                          <CategoryBadge category={pkg.category} />
                        </td>
                        <td style={{ padding: '8px 10px' }}>
                          <SourceBadge source={pkg.source} />
                        </td>
                        <td
                          style={{
                            padding: '8px 10px',
                            fontFamily: 'var(--font-mono)',
                            fontSize: '11px',
                            color: 'var(--text-secondary)',
                            whiteSpace: 'nowrap',
                          }}
                        >
                          {formatSize(pkg.installed_size_kb)}
                        </td>
                        <td
                          style={{
                            padding: '8px 10px',
                            color: 'var(--text-secondary)',
                            whiteSpace: 'nowrap',
                          }}
                          title={pkg.install_date || undefined}
                        >
                          {formatRelativeDate(pkg.install_date)}
                        </td>
                        <td style={{ padding: '8px 10px', color: 'var(--text-secondary)' }}>
                          {pkg.license || '\u2014'}
                        </td>
                        <td style={{ padding: '8px 10px', color: 'var(--text-muted)' }}>
                          {isExpanded ? (
                            <ChevronUp style={{ width: '14px', height: '14px' }} />
                          ) : (
                            <ChevronDown style={{ width: '14px', height: '14px' }} />
                          )}
                        </td>
                      </tr>
                      {isExpanded && (
                        <tr
                          key={`${rowKey}-detail`}
                          style={{ borderBottom: '1px solid var(--border-faint)' }}
                        >
                          <td
                            colSpan={8}
                            style={{
                              padding: '8px 10px 12px 10px',
                              background: 'var(--bg-inset)',
                            }}
                          >
                            <div
                              style={{
                                display: 'grid',
                                gridTemplateColumns: '1fr 1fr 1fr',
                                gap: '8px 24px',
                                fontSize: '12px',
                              }}
                            >
                              {pkg.maintainer && (
                                <div>
                                  <span style={{ color: 'var(--text-muted)' }}>Maintainer: </span>
                                  <span style={{ color: 'var(--text-secondary)' }}>
                                    {pkg.maintainer}
                                  </span>
                                </div>
                              )}
                              {pkg.section && (
                                <div>
                                  <span style={{ color: 'var(--text-muted)' }}>Section: </span>
                                  <span style={{ color: 'var(--text-secondary)' }}>
                                    {pkg.section}
                                  </span>
                                </div>
                              )}
                              {pkg.architecture && (
                                <div>
                                  <span style={{ color: 'var(--text-muted)' }}>Architecture: </span>
                                  <span
                                    style={{
                                      color: 'var(--text-secondary)',
                                      fontFamily: 'var(--font-mono)',
                                    }}
                                  >
                                    {pkg.architecture}
                                  </span>
                                </div>
                              )}
                              {pkg.homepage && (
                                <div>
                                  <span style={{ color: 'var(--text-muted)' }}>Homepage: </span>
                                  <span
                                    style={{
                                      color: 'var(--accent)',
                                      fontSize: '11px',
                                      wordBreak: 'break-all',
                                    }}
                                  >
                                    {pkg.homepage}
                                  </span>
                                </div>
                              )}
                              {pkg.priority && (
                                <div>
                                  <span style={{ color: 'var(--text-muted)' }}>Priority: </span>
                                  <span style={{ color: 'var(--text-secondary)' }}>
                                    {pkg.priority}
                                  </span>
                                </div>
                              )}
                              {pkg.description && (
                                <div style={{ gridColumn: '1 / -1' }}>
                                  <span style={{ color: 'var(--text-muted)' }}>Description: </span>
                                  <span style={{ color: 'var(--text-secondary)' }}>
                                    {pkg.description}
                                  </span>
                                </div>
                              )}
                            </div>
                          </td>
                        </tr>
                      )}
                    </React.Fragment>
                  );
                })}
              </tbody>
            </table>
          </div>

          {/* Pagination footer */}
          <div
            style={{
              marginTop: '12px',
              padding: '8px 10px',
              borderTop: '1px solid var(--border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              fontSize: '12px',
              color: 'var(--text-muted)',
            }}
          >
            <span>
              Showing {page * PAGE_SIZE + 1}&ndash;
              {Math.min((page + 1) * PAGE_SIZE, filtered.length)} of {filtered.length} packages
            </span>
            {totalPages > 1 && (
              <div style={{ display: 'flex', gap: '8px' }}>
                <button
                  disabled={page === 0}
                  onClick={() => setPage((p) => p - 1)}
                  style={{
                    background: 'transparent',
                    border: '1px solid var(--border)',
                    borderRadius: '4px',
                    padding: '4px 10px',
                    color: page === 0 ? 'var(--text-muted)' : 'var(--text-primary)',
                    cursor: page === 0 ? 'not-allowed' : 'pointer',
                    fontSize: '12px',
                  }}
                >
                  Prev
                </button>
                <span
                  style={{ display: 'flex', alignItems: 'center', color: 'var(--text-secondary)' }}
                >
                  {page + 1} / {totalPages}
                </span>
                <button
                  disabled={page >= totalPages - 1}
                  onClick={() => setPage((p) => p + 1)}
                  style={{
                    background: 'transparent',
                    border: '1px solid var(--border)',
                    borderRadius: '4px',
                    padding: '4px 10px',
                    color: page >= totalPages - 1 ? 'var(--text-muted)' : 'var(--text-primary)',
                    cursor: page >= totalPages - 1 ? 'not-allowed' : 'pointer',
                    fontSize: '12px',
                  }}
                >
                  Next
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
