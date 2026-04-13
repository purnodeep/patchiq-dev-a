import React, { useState, useMemo } from 'react';
import { Skeleton } from '@patchiq/ui';
import { RefreshCw } from 'lucide-react';
import { useEndpointPackages, useEndpoint } from '../../../api/hooks/useEndpoints';
import { timeAgo } from '../../../lib/time';

interface SoftwareTabProps {
  endpointId: string;
  packageCount?: number;
}

const PAGE_SIZE = 50;
type SortKey = 'name' | 'version' | 'arch';
type SortDir = 'asc' | 'desc';

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
    fontWeight: 500,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    color: 'var(--text-muted)',
    padding: '12px 16px 8px',
    borderBottom: '1px solid var(--border)',
    background: 'var(--bg-inset)',
  },
  th: {
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 500,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    color: 'var(--text-muted)',
    padding: '9px 12px',
    background: 'var(--bg-inset)',
    borderBottom: '1px solid var(--border)',
    textAlign: 'left' as const,
    whiteSpace: 'nowrap' as const,
  },
  td: {
    padding: '10px 12px',
    borderBottom: '1px solid var(--border)',
    color: 'var(--text-primary)',
    fontSize: 13,
  },
};

// Source label color — monochrome per design system (Rule #1: Color = Signal only)
const SOURCE_COLOR: Record<string, string> = {};

const SYSTEM_SOURCES = new Set([
  'apt',
  'yum',
  'dnf',
  'apk',
  'dpkg',
  'rpm',
  'pacman',
  'system',
  'wua',
  'hotfix',
  'softwareupdate',
]);

function classifyPackage(source: string | null | undefined): 'system' | 'third-party' {
  if (!source) return 'third-party';
  return SYSTEM_SOURCES.has(source.toLowerCase()) ? 'system' : 'third-party';
}

export function SoftwareTab({ endpointId, packageCount }: SoftwareTabProps) {
  const { data, isLoading, isError, error, refetch } = useEndpointPackages(endpointId);
  const { data: endpoint } = useEndpoint(endpointId);
  if (isError && error) console.error('[SoftwareTab] Failed to load packages:', error);

  const [search, setSearch] = useState('');
  const [sourceFilter, setSourceFilter] = useState('all');
  const [archFilter, setArchFilter] = useState('all');
  const [typeFilter, setTypeFilter] = useState<'all' | 'system' | 'third-party'>('all');
  const [sortKey, setSortKey] = useState<SortKey>('name');
  const [sortDir, setSortDir] = useState<SortDir>('asc');
  const [page, setPage] = useState(0);
  const [expandedRow, setExpandedRow] = useState<string | null>(null);

  const packages = data?.data ?? [];

  const sources = useMemo(
    () => Array.from(new Set(packages.map((p) => p.source).filter(Boolean) as string[])).sort(),
    [packages],
  );

  const arches = useMemo(
    () =>
      Array.from(new Set(packages.map((p) => p.arch).filter(Boolean) as string[]))
        .filter((a) => a !== 'all')
        .sort(),
    [packages],
  );

  const sourceBreakdown = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const p of packages) {
      if (p.source) counts[p.source] = (counts[p.source] || 0) + 1;
    }
    return counts;
  }, [packages]);

  const archBreakdown = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const p of packages) {
      if (p.arch) counts[p.arch] = (counts[p.arch] || 0) + 1;
    }
    return counts;
  }, [packages]);

  const filtered = useMemo(() => {
    return packages
      .filter((p) => {
        if (!search) return true;
        const q = search.toLowerCase();
        return (
          p.package_name.toLowerCase().includes(q) || (p.version ?? '').toLowerCase().includes(q)
        );
      })
      .filter((p) => sourceFilter === 'all' || p.source === sourceFilter)
      .filter((p) => archFilter === 'all' || p.arch === archFilter)
      .filter((p) => typeFilter === 'all' || classifyPackage(p.source) === typeFilter)
      .sort((a, b) => {
        const dir = sortDir === 'asc' ? 1 : -1;
        if (sortKey === 'name') return a.package_name.localeCompare(b.package_name) * dir;
        if (sortKey === 'version') return (a.version ?? '').localeCompare(b.version ?? '') * dir;
        if (sortKey === 'arch') return (a.arch ?? '').localeCompare(b.arch ?? '') * dir;
        return 0;
      });
  }, [packages, search, sourceFilter, archFilter, typeFilter, sortKey, sortDir]);

  const paged = useMemo(
    () => filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE),
    [filtered, page],
  );
  const totalPages = Math.ceil(filtered.length / PAGE_SIZE);

  function handleSort(key: SortKey) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortKey(key);
      setSortDir('asc');
    }
    setPage(0);
  }

  if (isLoading) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Skeleton className="h-20 w-full rounded-lg" />
        {Array.from({ length: 8 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full rounded" />
        ))}
      </div>
    );
  }

  if (isError) {
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
          Failed to load packages
          {error instanceof Error && error.message ? `: ${error.message}` : ''}
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

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Hero stat strip */}
      <div
        style={{
          ...S.card,
          padding: '16px 20px',
          display: 'flex',
          alignItems: 'center',
          gap: 32,
          flexWrap: 'wrap' as const,
        }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 26,
              fontWeight: 700,
              color: 'var(--text-emphasis)',
              lineHeight: 1,
            }}
          >
            {(packageCount ?? packages.length).toLocaleString()}
          </span>
          <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>packages installed</span>
        </div>

        {/* Source breakdown */}
        {sources.length > 0 &&
          Object.entries(sourceBreakdown)
            .sort((a, b) => b[1] - a[1])
            .map(([src, count]) => {
              const color = SOURCE_COLOR[src.toLowerCase()] ?? 'var(--text-muted)';
              const active = sourceFilter === src;
              return (
                <React.Fragment key={src}>
                  <div
                    style={{ width: 1, height: 28, background: 'var(--border)', flexShrink: 0 }}
                  />
                  <button
                    onClick={() => {
                      setSourceFilter(active ? 'all' : src);
                      setPage(0);
                    }}
                    style={{
                      background: 'none',
                      border: 'none',
                      cursor: 'pointer',
                      padding: 0,
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'flex-start',
                      gap: 2,
                      opacity: active ? 1 : 0.65,
                    }}
                  >
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 22,
                        fontWeight: 700,
                        color,
                        lineHeight: 1,
                      }}
                    >
                      {count}
                    </span>
                    <span
                      style={{
                        fontSize: 12,
                        color: 'var(--text-muted)',
                      }}
                    >
                      {src}
                    </span>
                  </button>
                </React.Fragment>
              );
            })}

        {endpoint?.last_scan && (
          <div
            style={{
              marginLeft: 'auto',
              fontSize: 11,
              color: 'var(--text-muted)',
              fontFamily: 'var(--font-mono)',
            }}
          >
            Last scan {timeAgo(endpoint.last_scan)}
          </div>
        )}
      </div>

      {/* Filter + search bar */}
      {packages.length > 0 && (
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' as const, alignItems: 'center' }}>
          {/* Type filter — System / Third-party */}
          <div
            style={{
              display: 'flex',
              gap: 2,
              background: 'var(--bg-inset)',
              borderRadius: 6,
              padding: 2,
            }}
          >
            {(['all', 'system', 'third-party'] as const).map((t) => (
              <button
                key={t}
                type="button"
                onClick={() => {
                  setTypeFilter(t);
                  setPage(0);
                }}
                style={{
                  padding: '4px 10px',
                  borderRadius: 4,
                  border: 'none',
                  fontSize: 11,
                  fontFamily: 'var(--font-mono)',
                  fontWeight: 500,
                  cursor: 'pointer',
                  background: typeFilter === t ? 'var(--bg-card)' : 'transparent',
                  color: typeFilter === t ? 'var(--text-emphasis)' : 'var(--text-muted)',
                  boxShadow: typeFilter === t ? 'var(--shadow-sm)' : 'none',
                }}
              >
                {t === 'all' ? 'All' : t === 'system' ? 'System' : 'Third-party'}
              </button>
            ))}
          </div>

          {/* Arch filter */}
          {arches.length > 1 && (
            <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
              <span
                style={{
                  fontSize: 10,
                  color: 'var(--text-muted)',
                  fontFamily: 'var(--font-mono)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                }}
              >
                Arch
              </span>
              {['all', ...arches].map((arch) => (
                <button
                  key={`arch-filter-${arch}`}
                  onClick={() => {
                    setArchFilter(arch);
                    setPage(0);
                  }}
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 11,
                    padding: '3px 8px',
                    borderRadius: 4,
                    border: '1px solid var(--border)',
                    cursor: 'pointer',
                    background: archFilter === arch ? 'var(--accent)' : 'var(--bg-card)',
                    color:
                      archFilter === arch
                        ? 'var(--btn-accent-text, #000)'
                        : 'var(--text-secondary)',
                    transition: 'all 0.1s',
                  }}
                >
                  {arch === 'all' ? 'All' : `${arch} (${archBreakdown[arch]})`}
                </button>
              ))}
            </div>
          )}

          {/* Sort buttons */}
          <div style={{ display: 'flex', gap: 4, alignItems: 'center', marginLeft: 8 }}>
            <span
              style={{
                fontSize: 10,
                color: 'var(--text-muted)',
                fontFamily: 'var(--font-mono)',
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
              }}
            >
              Sort
            </span>
            {(['name', 'version', 'arch'] as SortKey[]).map((k) => (
              <button
                key={k}
                onClick={() => handleSort(k)}
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 11,
                  padding: '3px 8px',
                  borderRadius: 4,
                  border: '1px solid var(--border)',
                  cursor: 'pointer',
                  background: sortKey === k ? 'var(--bg-inset)' : 'var(--bg-card)',
                  color: sortKey === k ? 'var(--accent)' : 'var(--text-secondary)',
                }}
              >
                {k === 'name' ? 'Name' : k === 'version' ? 'Version' : 'Arch'}
                {sortKey === k && (
                  <span style={{ marginLeft: 4 }}>{sortDir === 'asc' ? '↑' : '↓'}</span>
                )}
              </button>
            ))}
          </div>

          {/* Search */}
          <div
            style={{
              marginLeft: 'auto',
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              background: 'var(--bg-inset)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '5px 10px',
            }}
          >
            <svg
              width="12"
              height="12"
              viewBox="0 0 24 24"
              fill="none"
              stroke="var(--text-muted)"
              strokeWidth="2"
            >
              <circle cx="11" cy="11" r="8" />
              <path d="m21 21-4.35-4.35" />
            </svg>
            <input
              placeholder="Search packages..."
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(0);
              }}
              style={{
                background: 'none',
                border: 'none',
                outline: 'none',
                fontSize: 12,
                color: 'var(--text-primary)',
                width: 160,
                fontFamily: 'var(--font-mono)',
              }}
            />
          </div>
        </div>
      )}

      {/* Empty state */}
      {!isLoading && !isError && filtered.length === 0 && (
        <div
          style={{
            ...S.card,
            padding: 48,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 8,
          }}
        >
          <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>
            {packages.length === 0
              ? 'No packages found. Run a scan to populate software inventory.'
              : 'No packages match the current filter.'}
          </span>
        </div>
      )}

      {/* Table */}
      {paged.length > 0 && (
        <div style={S.card}>
          <div
            style={{
              ...S.cardTitle,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}
          >
            <span>Installed Packages</span>
            <span style={{ color: 'var(--text-faint)' }}>
              {filtered.length.toLocaleString()} packages
            </span>
          </div>
          <div style={{ overflowX: 'auto', maxHeight: 600, overflowY: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead style={{ position: 'sticky', top: 0, zIndex: 1 }}>
                <tr>
                  <th style={S.th}>Package Name</th>
                  <th style={S.th}>Version</th>
                  <th style={S.th}>Arch</th>
                  <th style={S.th}>Source</th>
                  <th style={S.th}>Release</th>
                  <th style={{ ...S.th, width: 28 }} />
                </tr>
              </thead>
              <tbody>
                {paged.map((pkg, index) => {
                  const rowKey = `${pkg.package_name}-${pkg.version ?? ''}-${index}`;
                  const isExpanded = expandedRow === rowKey;
                  const srcColor =
                    SOURCE_COLOR[(pkg.source ?? '').toLowerCase()] ?? 'var(--text-muted)';
                  return (
                    <React.Fragment key={rowKey}>
                      <tr
                        onClick={() => setExpandedRow(isExpanded ? null : rowKey)}
                        style={{
                          cursor: 'pointer',
                          background: isExpanded ? 'var(--bg-card-hover)' : undefined,
                        }}
                        onMouseEnter={(e) => {
                          if (!isExpanded)
                            (e.currentTarget as HTMLTableRowElement).style.background =
                              'var(--bg-card-hover)';
                        }}
                        onMouseLeave={(e) => {
                          if (!isExpanded)
                            (e.currentTarget as HTMLTableRowElement).style.background = '';
                        }}
                      >
                        <td
                          style={{
                            ...S.td,
                            fontFamily: 'var(--font-mono)',
                            fontSize: 12,
                            fontWeight: 500,
                          }}
                        >
                          {pkg.package_name}
                        </td>
                        <td
                          style={{
                            ...S.td,
                            fontFamily: 'var(--font-mono)',
                            fontSize: 11,
                            color: 'var(--text-secondary)',
                          }}
                        >
                          {pkg.version ?? '—'}
                        </td>
                        <td style={{ ...S.td, fontSize: 11, color: 'var(--text-muted)' }}>
                          {pkg.arch ?? '—'}
                        </td>
                        <td style={S.td}>
                          {pkg.source ? (
                            <span
                              style={{
                                fontFamily: 'var(--font-mono)',
                                fontSize: 11,
                                color: srcColor,
                              }}
                            >
                              {pkg.source}
                            </span>
                          ) : (
                            <span style={{ color: 'var(--text-muted)' }}>—</span>
                          )}
                        </td>
                        <td
                          style={{
                            ...S.td,
                            fontSize: 11,
                            color: 'var(--text-muted)',
                            fontFamily: 'var(--font-mono)',
                          }}
                        >
                          {pkg.release ?? '—'}
                        </td>
                        <td
                          style={{
                            ...S.td,
                            textAlign: 'center',
                            color: 'var(--text-muted)',
                            fontSize: 10,
                          }}
                        >
                          {isExpanded ? '▲' : '▼'}
                        </td>
                      </tr>
                      {isExpanded && (
                        <tr>
                          <td
                            colSpan={6}
                            style={{
                              background: 'var(--bg-inset)',
                              padding: '12px 16px',
                              borderBottom: '1px solid var(--border)',
                            }}
                          >
                            <div
                              style={{
                                display: 'grid',
                                gridTemplateColumns: 'repeat(3, 1fr)',
                                gap: 12,
                              }}
                            >
                              {[
                                { label: 'Package', value: pkg.package_name },
                                { label: 'Version', value: pkg.version ?? '—' },
                                pkg.arch ? { label: 'Architecture', value: pkg.arch } : null,
                                pkg.source ? { label: 'Source', value: pkg.source } : null,
                                pkg.release ? { label: 'Release', value: pkg.release } : null,
                              ]
                                .filter(Boolean)
                                .map((item) => (
                                  <div key={item!.label}>
                                    <div
                                      style={{
                                        fontSize: 10,
                                        color: 'var(--text-muted)',
                                        marginBottom: 2,
                                      }}
                                    >
                                      {item!.label}
                                    </div>
                                    <div
                                      style={{
                                        fontFamily: 'var(--font-mono)',
                                        fontSize: 12,
                                        color: 'var(--text-primary)',
                                      }}
                                    >
                                      {item!.value}
                                    </div>
                                  </div>
                                ))}
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

          {/* Pagination */}
          <div
            style={{
              padding: '10px 16px',
              borderTop: '1px solid var(--border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}
          >
            <span
              style={{ fontSize: 11, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
            >
              {filtered.length === 0 ? 0 : page * PAGE_SIZE + 1}–
              {Math.min((page + 1) * PAGE_SIZE, filtered.length)} of {filtered.length}
            </span>
            {totalPages > 1 && (
              <div style={{ display: 'flex', gap: 6 }}>
                <button
                  disabled={page === 0}
                  onClick={() => setPage((p) => p - 1)}
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 11,
                    padding: '3px 10px',
                    borderRadius: 4,
                    border: '1px solid var(--border)',
                    background: 'var(--bg-card)',
                    color: page === 0 ? 'var(--text-faint)' : 'var(--text-secondary)',
                    cursor: page === 0 ? 'not-allowed' : 'pointer',
                  }}
                >
                  ← Prev
                </button>
                <span
                  style={{
                    fontSize: 11,
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                    lineHeight: '26px',
                  }}
                >
                  {page + 1} / {totalPages}
                </span>
                <button
                  disabled={page >= totalPages - 1}
                  onClick={() => setPage((p) => p + 1)}
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 11,
                    padding: '3px 10px',
                    borderRadius: 4,
                    border: '1px solid var(--border)',
                    background: 'var(--bg-card)',
                    color: page >= totalPages - 1 ? 'var(--text-faint)' : 'var(--text-secondary)',
                    cursor: page >= totalPages - 1 ? 'not-allowed' : 'pointer',
                  }}
                >
                  Next →
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
