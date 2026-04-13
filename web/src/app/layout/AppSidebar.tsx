import { useEffect, useRef, useState } from 'react';
import { NavLink, useLocation } from 'react-router';
import {
  LayoutDashboard,
  Monitor,
  Package,
  ShieldAlert,
  ScrollText,
  Rocket,
  FileSearch,
  FileBarChart,
  Settings,
  BellRing,
  Download,
  LogOut,
  Lock,
} from 'lucide-react';
import { useAuth, useCan } from '../auth/AuthContext';
import { useAlertCount } from '../../api/hooks/useAlerts';
import { useLogout } from '../../api/hooks/useAuth';
import { SettingsSidebar } from '../../pages/settings/SettingsSidebar';

const navGroups = [
  {
    label: 'Overview',
    items: [
      { label: 'Dashboard', icon: LayoutDashboard, to: '/', resource: 'endpoints', action: 'read' },
    ],
  },
  {
    label: 'Assets',
    items: [
      {
        label: 'Endpoints',
        icon: Monitor,
        to: '/endpoints',
        resource: 'endpoints',
        action: 'read',
      },
    ],
  },
  {
    label: 'Security',
    items: [
      { label: 'Patches', icon: Package, to: '/patches', resource: 'patches', action: 'read' },
      { label: 'CVEs', icon: ShieldAlert, to: '/cves', resource: 'patches', action: 'read' },
      {
        label: 'Policies',
        icon: ScrollText,
        to: '/policies',
        resource: 'policies',
        action: 'read',
      },
    ],
  },
  {
    label: 'Operations',
    items: [
      {
        label: 'Deployments',
        icon: Rocket,
        to: '/deployments',
        resource: 'deployments',
        action: 'read',
      },
    ],
  },
  {
    label: 'Compliance',
    items: [
      { label: 'Alerts', icon: BellRing, to: '/alerts', resource: 'alerts', action: 'read' },
      { label: 'Audit', icon: FileSearch, to: '/audit', resource: 'audit', action: 'read' },
      { label: 'Reports', icon: FileBarChart, to: '/reports', resource: 'reports', action: 'read' },
    ],
  },
  {
    label: 'System',
    items: [
      { label: 'Settings', icon: Settings, to: '/settings', resource: 'settings', action: 'read' },
      {
        label: 'Agent Downloads',
        icon: Download,
        to: '/agent-downloads',
        resource: 'endpoints',
        action: 'read',
      },
    ],
  },
];

export const AppSidebar = () => {
  const location = useLocation();
  const isSettings = location.pathname.startsWith('/settings');
  const { user } = useAuth();
  const can = useCan();
  const { data: alertCountData } = useAlertCount(undefined, 30000);
  const { mutate: logout } = useLogout();
  const [menuOpen, setMenuOpen] = useState(false);
  const footerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!menuOpen) return;
    const onClick = (e: MouseEvent) => {
      if (footerRef.current && !footerRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, [menuOpen]);

  // Render settings sidebar when on /settings/* routes
  if (isSettings) {
    return <SettingsSidebar />;
  }
  const alertCount = (alertCountData?.critical_unread ?? 0) + (alertCountData?.warning_unread ?? 0);
  const hasCritical = (alertCountData?.critical_unread ?? 0) > 0;

  const displayName = user.name || user.email || user.user_id || 'User';
  const roleName = user.roles?.[0];
  const initials = displayName
    .split(' ')
    .map((w) => w[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);

  return (
    <aside
      style={{
        width: 'var(--sidebar-width, 220px)',
        flexShrink: 0,
        background: 'var(--bg-sidebar)',
        borderRight: '1px solid var(--border-sidebar)',
        display: 'flex',
        flexDirection: 'column',
        height: '100vh',
        overflowY: 'auto',
        overflowX: 'hidden',
      }}
    >
      {/* Logo */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          height: 'var(--topbar-height, 48px)',
          padding: '0 20px',
          borderBottom: '1px solid var(--border)',
          flexShrink: 0,
        }}
      >
        <img
          src="/infraon-logo.png"
          alt="Infraon"
          style={{
            height: 50,
            objectFit: 'contain',
          }}
        />
      </div>

      {/* Navigation */}
      <nav style={{ flex: 1, padding: '12px 0' }}>
        {navGroups.map((group) => (
          <div key={group.label} role="group" aria-label={group.label} style={{ marginBottom: 4 }}>
            <div
              aria-hidden="true"
              style={{
                padding: '10px 20px 4px',
                fontFamily: 'var(--font-mono)',
                fontSize: 9,
                fontWeight: 600,
                letterSpacing: '0.08em',
                color: 'var(--nav-label)',
                textTransform: 'uppercase' as const,
              }}
            >
              {group.label}
            </div>
            <ul style={{ listStyle: 'none', margin: 0, padding: 0 }}>
              {group.items.map((item) => {
                const allowed = can(item.resource, item.action);
                return (
                  <li key={item.label}>
                    {allowed ? (
                      <NavLink
                        to={item.to}
                        end={item.to === '/'}
                        style={{ textDecoration: 'none' }}
                        aria-label={item.label}
                      >
                        {({ isActive }) => (
                          <div
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 9,
                              padding: '7px 20px',
                              fontSize: 13,
                              color: isActive ? 'var(--text-primary)' : 'var(--nav-item-color)',
                              borderLeft: isActive
                                ? '2px solid var(--brand)'
                                : '2px solid transparent',
                              background: isActive ? 'var(--brand-subtle)' : 'transparent',
                              cursor: 'pointer',
                              transition: 'color 0.1s',
                            }}
                            onMouseEnter={(e) => {
                              if (!isActive) {
                                e.currentTarget.style.color = 'var(--nav-item-hover)';
                              }
                            }}
                            onMouseLeave={(e) => {
                              if (!isActive) {
                                e.currentTarget.style.color = 'var(--nav-item-color)';
                              }
                            }}
                          >
                            <item.icon
                              style={{
                                width: 15,
                                height: 15,
                                flexShrink: 0,
                                strokeWidth: 1.5,
                              }}
                            />
                            <span>{item.label}</span>
                            {item.label === 'Alerts' && alertCount > 0 && (
                              <>
                                <span
                                  aria-hidden="true"
                                  style={{
                                    marginLeft: 'auto',
                                    background: hasCritical
                                      ? 'var(--signal-critical)'
                                      : 'var(--signal-warning)',
                                    color: 'var(--text-on-color, #fff)',
                                    fontSize: 10,
                                    fontWeight: 600,
                                    fontFamily: 'var(--font-mono)',
                                    borderRadius: 'var(--radius-full)',
                                    minWidth: 18,
                                    padding: '0 5px',
                                    textAlign: 'center' as const,
                                    lineHeight: '16px',
                                  }}
                                >
                                  {alertCount}
                                </span>
                                <span
                                  style={{
                                    position: 'absolute',
                                    width: 1,
                                    height: 1,
                                    padding: 0,
                                    margin: -1,
                                    overflow: 'hidden',
                                    clip: 'rect(0, 0, 0, 0)',
                                    whiteSpace: 'nowrap',
                                    borderWidth: 0,
                                  }}
                                >
                                  {alertCount} unread alerts
                                </span>
                              </>
                            )}
                          </div>
                        )}
                      </NavLink>
                    ) : (
                      <div
                        title={`You don't have access to ${item.label}`}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 9,
                          padding: '7px 20px',
                          fontSize: 13,
                          color: 'var(--nav-item-color)',
                          borderLeft: '2px solid transparent',
                          background: 'transparent',
                          cursor: 'not-allowed',
                          opacity: 0.4,
                        }}
                      >
                        <item.icon
                          style={{
                            width: 15,
                            height: 15,
                            flexShrink: 0,
                            strokeWidth: 1.5,
                          }}
                        />
                        <span>{item.label}</span>
                        <Lock
                          style={{
                            width: 10,
                            height: 10,
                            flexShrink: 0,
                            marginLeft: 'auto',
                            strokeWidth: 1.5,
                          }}
                        />
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </nav>

      {/* Footer */}
      <div
        ref={footerRef}
        style={{
          position: 'relative',
          borderTop: '1px solid var(--border-divider)',
        }}
      >
        <button
          type="button"
          onClick={() => setMenuOpen((v) => !v)}
          aria-label="User menu"
          aria-expanded={menuOpen}
          style={{
            width: '100%',
            padding: '14px 20px',
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            background: menuOpen ? 'var(--brand-subtle)' : 'transparent',
            border: 'none',
            cursor: 'pointer',
            textAlign: 'left',
            transition: 'background 0.1s',
          }}
          onMouseEnter={(e) => {
            if (!menuOpen)
              e.currentTarget.style.background = 'var(--nav-item-hover, rgba(255,255,255,0.04))';
          }}
          onMouseLeave={(e) => {
            if (!menuOpen) e.currentTarget.style.background = 'transparent';
          }}
        >
          <div
            style={{
              width: 28,
              height: 28,
              borderRadius: '50%',
              background: 'var(--avatar-bg)',
              border: '1px solid var(--avatar-border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 10,
              fontWeight: 600,
              color: 'var(--avatar-text)',
              letterSpacing: '0.02em',
              flexShrink: 0,
              position: 'relative',
            }}
          >
            {initials}
            <div
              style={{
                position: 'absolute',
                bottom: -1,
                right: -1,
                width: 8,
                height: 8,
                borderRadius: '50%',
                background: 'var(--signal-healthy, #22c55e)',
                border: '2px solid var(--bg-sidebar)',
              }}
            />
          </div>
          <div style={{ minWidth: 0, flex: 1 }}>
            <div
              style={{
                fontSize: 12,
                color: 'var(--text-muted)',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {displayName}
            </div>
            {roleName && (
              <div
                style={{
                  fontSize: 10,
                  color: 'var(--text-faint)',
                  fontFamily: 'var(--font-mono)',
                  marginTop: 1,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {roleName}
              </div>
            )}
          </div>
        </button>

        {menuOpen && (
          <div
            style={{
              position: 'absolute',
              left: 12,
              right: 12,
              bottom: 'calc(100% + 4px)',
              background: 'var(--bg-elevated)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              boxShadow: 'var(--shadow-md, 0 4px 16px rgba(0,0,0,0.18))',
              zIndex: 50,
              overflow: 'hidden',
            }}
          >
            {/* User info */}
            <div style={{ padding: '10px 12px 6px', borderBottom: '1px solid var(--border)' }}>
              <div
                style={{
                  fontSize: 12,
                  fontWeight: 600,
                  color: 'var(--text-primary)',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {displayName}
              </div>
              {user.email && (
                <div
                  style={{
                    fontSize: 11,
                    color: 'var(--text-muted)',
                    marginTop: 2,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {user.email}
                </div>
              )}
              {roleName && (
                <div
                  style={{
                    display: 'inline-block',
                    marginTop: 4,
                    marginBottom: 2,
                    fontSize: 10,
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-secondary)',
                    background: 'var(--brand-subtle)',
                    padding: '2px 8px',
                    borderRadius: 'var(--radius-full)',
                  }}
                >
                  {roleName}
                </div>
              )}
            </div>

            {/* Account Settings */}
            <NavLink
              to="/settings/account"
              onClick={() => setMenuOpen(false)}
              style={{ textDecoration: 'none' }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '9px 12px',
                  fontSize: 12,
                  color: 'var(--text-secondary)',
                  cursor: 'pointer',
                  transition: 'background 0.1s',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--brand-subtle)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'transparent';
                }}
              >
                <Settings style={{ width: 14, height: 14, strokeWidth: 1.5 }} />
                Account Settings
              </div>
            </NavLink>

            {/* Divider */}
            <div style={{ height: 1, background: 'var(--border)', margin: '2px 0' }} />

            <button
              type="button"
              onClick={() => {
                setMenuOpen(false);
                logout();
              }}
              style={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '9px 12px',
                background: 'transparent',
                border: 'none',
                cursor: 'pointer',
                fontSize: 12,
                color: 'var(--text-secondary)',
                textAlign: 'left',
                transition: 'background 0.1s, color 0.1s',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.background =
                  'var(--signal-critical-subtle, rgba(239,68,68,0.12))';
                e.currentTarget.style.color = 'var(--signal-critical)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = 'transparent';
                e.currentTarget.style.color = 'var(--text-secondary)';
              }}
            >
              <LogOut style={{ width: 14, height: 14, strokeWidth: 1.5 }} />
              Sign out
            </button>
          </div>
        )}
      </div>
    </aside>
  );
};
