import { useLocation, useNavigate, Link } from 'react-router';
import { Search, Bell, Sun, Moon, Download } from 'lucide-react';
import { useTheme } from '@patchiq/ui';
import { useQuery } from '@tanstack/react-query';
import { TenantSwitcher } from './TenantSwitcher';

const ROUTE_TITLES: Record<string, string> = {
  '/': 'Dashboard',
  '/endpoints': 'Endpoints',
  '/tags': 'Tags',
  '/patches': 'Patches',
  '/cves': 'CVEs',
  '/policies': 'Policies',
  '/deployments': 'Deployments',
  '/workflows': 'Workflows',
  '/compliance': 'Compliance',
  '/audit': 'Audit Log',
  '/notifications': 'Notifications',
  '/settings': 'Settings',
  '/admin/roles': 'Roles',
  '/admin/users/roles': 'User Roles',
};

// Patterns: [regex, parentRoute, parentLabel, apiPath for name resolution]
const DETAIL_PATTERNS: Array<[RegExp, string, string, string]> = [
  [/^\/endpoints\/([^/]+)$/, '/endpoints', 'Endpoints', '/api/v1/endpoints/{id}'],
  [/^\/patches\/([^/]+)$/, '/patches', 'Patches', '/api/v1/patches/{id}'],
  [/^\/cves\/([^/]+)$/, '/cves', 'CVEs', '/api/v1/cves/{id}'],
  [/^\/deployments\/([^/]+)$/, '/deployments', 'Deployments', '/api/v1/deployments/{id}'],
  [/^\/policies\/([^/]+)(?:\/edit)?$/, '/policies', 'Policies', '/api/v1/policies/{id}'],
  [/^\/workflows\/new$/, '/workflows', 'Workflows', ''],
  [/^\/compliance\/([^/]+)$/, '/compliance', 'Compliance', '/api/v1/compliance/frameworks/{id}'],
  [/^\/admin\/roles\/([^/]+)$/, '/admin/roles', 'Roles', '/api/v1/roles/{id}'],
];

interface BreadcrumbInfo {
  pageTitle: string;
  parentRoute: string | null;
  parentLabel: string | null;
  detailLabel: string | null;
  entityId: string | null;
  apiPath: string | null;
}

function useBreadcrumb(): BreadcrumbInfo {
  const { pathname } = useLocation();

  // Check detail patterns first
  for (const [pattern, parentRoute, parentLabel, apiPath] of DETAIL_PATTERNS) {
    const match = pathname.match(pattern);
    if (match) {
      return {
        pageTitle: parentLabel,
        parentRoute,
        parentLabel,
        detailLabel: match[1],
        entityId: match[1],
        apiPath,
      };
    }
  }

  // Settings sub-pages collapse to "Settings"
  if (pathname.startsWith('/settings/')) {
    return {
      pageTitle: 'Settings',
      parentRoute: null,
      parentLabel: null,
      detailLabel: null,
      entityId: null,
      apiPath: null,
    };
  }

  // Exact match
  if (ROUTE_TITLES[pathname]) {
    return {
      pageTitle: ROUTE_TITLES[pathname],
      parentRoute: null,
      parentLabel: null,
      detailLabel: null,
      entityId: null,
      apiPath: null,
    };
  }

  // Strip trailing segments for nested routes
  const parts = pathname.split('/').filter(Boolean);
  for (let i = parts.length; i > 0; i--) {
    const candidate = '/' + parts.slice(0, i).join('/');
    if (ROUTE_TITLES[candidate]) {
      return {
        pageTitle: ROUTE_TITLES[candidate],
        parentRoute: null,
        parentLabel: null,
        detailLabel: null,
        entityId: null,
        apiPath: null,
      };
    }
  }

  const fallback =
    parts.length === 0
      ? 'Dashboard'
      : parts[parts.length - 1].charAt(0).toUpperCase() + parts[parts.length - 1].slice(1);
  return {
    pageTitle: fallback,
    parentRoute: null,
    parentLabel: null,
    detailLabel: null,
    entityId: null,
    apiPath: null,
  };
}

/** Resolve a human-readable name for the entity shown in the breadcrumb. */
function useEntityName(entityId: string | null, apiPath: string | null): string | null {
  const { data } = useQuery({
    queryKey: ['topbar-entity', apiPath, entityId],
    queryFn: async () => {
      if (!entityId || !apiPath) return null;
      const url = apiPath.replace('{id}', entityId);
      const res = await fetch(url, { credentials: 'include' });
      if (!res.ok) return null;
      const data = await res.json();
      // Prefer human-readable identifiers; never expose raw UUIDs
      const resolved = (data?.cve_id ?? data?.name ?? data?.hostname ?? data?.title ?? null) as
        | string
        | null;
      // If resolved value looks like a UUID, discard it and let the page handle the label
      if (resolved && /^[0-9a-f]{8}-[0-9a-f]{4}-/i.test(resolved)) return null;
      return resolved;
    },
    enabled: !!entityId && !!apiPath,
    staleTime: 60_000,
    retry: false,
  });
  return data ?? null;
}

interface TopBarProps {
  onOpenCommandPalette?: () => void;
}

export const TopBar = ({ onOpenCommandPalette }: TopBarProps) => {
  const { resolvedMode, setMode } = useTheme();
  const { pageTitle, parentRoute, parentLabel, detailLabel, entityId, apiPath } = useBreadcrumb();
  const entityName = useEntityName(entityId, apiPath);
  const navigate = useNavigate();

  const toggleTheme = () => setMode(resolvedMode === 'dark' ? 'light' : 'dark');

  // Show resolved entity name, or a loading indicator while resolving
  const displayLabel = entityName ?? (detailLabel ? '…' : null);

  return (
    <header
      style={{
        height: 'var(--topbar-height, 48px)',
        background: 'var(--bg-topbar)',
        borderBottom: '1px solid var(--border)',
        display: 'flex',
        alignItems: 'center',
        gap: 16,
        padding: '0 24px',
        flexShrink: 0,
      }}
    >
      {/* Breadcrumb */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, flexShrink: 0 }}>
        {parentRoute && parentLabel && displayLabel ? (
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
                maxWidth: 300,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
              title={entityName ?? detailLabel ?? undefined}
            >
              {displayLabel}
            </span>
          </>
        ) : (
          <span style={{ color: 'var(--text-muted)' }}>{pageTitle}</span>
        )}
      </div>

      {/* Spacer (left) */}
      <div style={{ flex: 1 }} />

      {/* Search button (centered between breadcrumb and right actions) */}
      <button
        type="button"
        aria-label="Search"
        onClick={onOpenCommandPalette}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          width: 480,
          flexShrink: 0,
          padding: '6px 12px',
          border: '1px solid var(--btn-border)',
          borderRadius: 6,
          background: 'var(--bg-input, transparent)',
          color: 'var(--text-muted)',
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
          e.currentTarget.style.color = 'var(--text-muted)';
        }}
      >
        <Search style={{ width: 14, height: 14, strokeWidth: 1.5, flexShrink: 0 }} />
        <span style={{ flex: 1, textAlign: 'left' }}>Search endpoints, patches, CVEs…</span>
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            color: 'var(--text-faint)',
            border: '1px solid var(--border)',
            borderRadius: 3,
            padding: '1px 5px',
            flexShrink: 0,
          }}
        >
          {'\u2318'}K
        </span>
      </button>

      {/* Spacer (right) */}
      <div style={{ flex: 1 }} />

      {/* Right actions */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexShrink: 0 }}>
        {/* Tenant switcher (MSP operators only — hidden otherwise). */}
        <TenantSwitcher />

        {/* Register Endpoint */}
        <button
          type="button"
          aria-label="Register Endpoint"
          title="Register Endpoint"
          onClick={() => void navigate('/endpoints?register=true')}
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
          <Download style={{ width: 16, height: 16, strokeWidth: 1.5 }} />
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
          onClick={() => void navigate('/notifications')}
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
            position: 'relative',
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
