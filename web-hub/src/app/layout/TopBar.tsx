import { useLocation, Link } from 'react-router';
import { Search, Bell, Sun, Moon } from 'lucide-react';
import { useTheme } from '@patchiq/ui';

const ROUTE_TITLES: Record<string, string> = {
  '/': 'Dashboard',
  '/clients': 'Clients',
  '/catalog': 'Catalog',
  '/feeds': 'Feeds',
  '/licenses': 'Licenses',
  '/deployments': 'Deployments',
  '/settings': 'Settings',
};

const DETAIL_PATTERNS: Array<[RegExp, string, string]> = [
  [/^\/clients\/([^/]+)$/, '/clients', 'Clients'],
  [/^\/catalog\/([^/]+)$/, '/catalog', 'Catalog'],
  [/^\/licenses\/([^/]+)$/, '/licenses', 'Licenses'],
];

interface BreadcrumbInfo {
  pageTitle: string;
  parentRoute: string | null;
  parentLabel: string | null;
  detailLabel: string | null;
}

function useBreadcrumb(): BreadcrumbInfo {
  const { pathname } = useLocation();

  for (const [pattern, parentRoute, parentLabel] of DETAIL_PATTERNS) {
    const match = pathname.match(pattern);
    if (match) {
      return { pageTitle: parentLabel, parentRoute, parentLabel, detailLabel: match[1] };
    }
  }

  if (pathname.startsWith('/settings/')) {
    return { pageTitle: 'Settings', parentRoute: null, parentLabel: null, detailLabel: null };
  }

  if (ROUTE_TITLES[pathname]) {
    return {
      pageTitle: ROUTE_TITLES[pathname],
      parentRoute: null,
      parentLabel: null,
      detailLabel: null,
    };
  }

  const parts = pathname.split('/').filter(Boolean);
  const fallback =
    parts.length === 0
      ? 'Dashboard'
      : parts[parts.length - 1].charAt(0).toUpperCase() + parts[parts.length - 1].slice(1);
  return { pageTitle: fallback, parentRoute: null, parentLabel: null, detailLabel: null };
}

export const TopBar = () => {
  const { resolvedMode, setMode } = useTheme();
  const { pageTitle, parentRoute, parentLabel, detailLabel } = useBreadcrumb();

  const toggleTheme = () => setMode(resolvedMode === 'dark' ? 'light' : 'dark');

  return (
    <header
      style={{
        height: 'var(--topbar-height, 48px)',
        background: 'var(--bg-topbar)',
        borderBottom: '1px solid var(--border)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '0 24px',
        flexShrink: 0,
      }}
    >
      {/* Breadcrumb */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13 }}>
        {parentRoute && parentLabel && detailLabel ? (
          <>
            <Link
              to={parentRoute}
              style={{ color: 'var(--text-muted)', textDecoration: 'none' }}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLAnchorElement).style.color = 'var(--text-secondary)';
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLAnchorElement).style.color = 'var(--text-muted)';
              }}
            >
              {parentLabel}
            </Link>
            <span style={{ color: 'var(--text-faint)' }}>/</span>
            <span
              style={{
                color: 'var(--text-primary)',
                maxWidth: 200,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {detailLabel}
            </span>
          </>
        ) : (
          <span style={{ color: 'var(--text-emphasis)', fontWeight: 600, fontSize: 16 }}>
            {pageTitle}
          </span>
        )}
      </div>

      {/* Right actions */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        {/* Search button */}
        <button
          type="button"
          aria-label="Search"
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 7,
            padding: '5px 10px',
            border: '1px solid var(--btn-border)',
            borderRadius: 6,
            background: 'transparent',
            color: 'var(--btn-text)',
            fontSize: 12,
            fontFamily: 'var(--font-sans)',
            cursor: 'pointer',
            transition: 'border-color 0.1s, color 0.1s',
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.borderColor = 'var(--btn-border-hover)';
            e.currentTarget.style.color = 'var(--btn-text-hover)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.borderColor = 'var(--btn-border)';
            e.currentTarget.style.color = 'var(--btn-text)';
          }}
        >
          <Search style={{ width: 13, height: 13, strokeWidth: 1.5 }} />
          <span>Search</span>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 10,
              color: 'var(--text-faint)',
            }}
          >
            {'\u2318'}K
          </span>
        </button>

        {/* Theme toggle */}
        <button
          type="button"
          onClick={toggleTheme}
          aria-label="Toggle theme"
          style={{
            width: 32,
            height: 32,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            borderRadius: 6,
            background: 'transparent',
            border: 'none',
            cursor: 'pointer',
            color: 'var(--text-muted)',
            transition: 'color 0.1s',
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.color = 'var(--text-secondary)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.color = 'var(--text-muted)';
          }}
        >
          {resolvedMode === 'dark' ? (
            <Sun style={{ width: 16, height: 16, strokeWidth: 1.5 }} />
          ) : (
            <Moon style={{ width: 16, height: 16, strokeWidth: 1.5 }} />
          )}
        </button>

        {/* Notification bell */}
        <button
          type="button"
          aria-label="Notifications"
          style={{
            width: 32,
            height: 32,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            borderRadius: 6,
            background: 'transparent',
            border: 'none',
            cursor: 'pointer',
            color: 'var(--text-muted)',
            transition: 'color 0.1s',
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.color = 'var(--text-secondary)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.color = 'var(--text-muted)';
          }}
        >
          <Bell style={{ width: 16, height: 16, strokeWidth: 1.5 }} />
        </button>
      </div>
    </header>
  );
};
