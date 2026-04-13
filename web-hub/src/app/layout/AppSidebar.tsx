import { NavLink, useLocation } from 'react-router';
import {
  LayoutDashboard,
  Package,
  Rss,
  Key,
  Monitor,
  Settings,
  LogOut,
} from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '@patchiq/ui';
import { useAuth } from '../auth/AuthContext';
import { useLogout } from '../../api/hooks/useAuth';
import { usePendingClientCount } from '../../api/hooks/useClients';
import { SettingsSidebar } from '../../pages/settings/SettingsSidebar';

const navGroups = [
  {
    label: 'Overview',
    items: [{ label: 'Dashboard', icon: LayoutDashboard, to: '/' }],
  },
  {
    label: 'Fleet',
    items: [{ label: 'Clients', icon: Monitor, to: '/clients', badge: 'pending' as const }],
  },
  {
    label: 'Catalog',
    items: [
      { label: 'Catalog', icon: Package, to: '/catalog' },
      { label: 'Feeds', icon: Rss, to: '/feeds' },
    ],
  },
  {
    label: 'Licensing',
    items: [{ label: 'Licenses', icon: Key, to: '/licenses' }],
  },
  {
    label: 'Configuration',
    items: [{ label: 'Settings', icon: Settings, to: '/settings' }],
  },
];

export const AppSidebar = () => {
  const location = useLocation();
  const isSettings = location.pathname.startsWith('/settings');
  const { data: pendingCount } = usePendingClientCount();
  const count = pendingCount?.count ?? 0;
  const { user } = useAuth();
  const logout = useLogout();

  const initials = (user.name ?? user.email ?? 'U')
    .split(' ')
    .map((w: string) => w[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);

  const displayName = user.name ?? user.email ?? 'User';

  if (isSettings) {
    return <SettingsSidebar />;
  }

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
          padding: '0 20px',
          height: 'var(--topbar-height, 48px)',
          borderBottom: '1px solid var(--border-divider)',
        }}
      >
        <img
          src="/infraon-logo.png"
          alt="Infraon"
          style={{
            height: 32,
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
              {group.items.map((item) => (
                <li key={item.label} style={{ position: 'relative' }}>
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
                          borderLeft: isActive ? '2px solid var(--brand)' : '2px solid transparent',
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
                        {'badge' in item && item.badge === 'pending' && count > 0 && (
                          <span
                            aria-label={`${count} pending clients`}
                            style={{
                              marginLeft: 'auto',
                              background: 'var(--signal-critical)',
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
                            {count}
                          </span>
                        )}
                      </div>
                    )}
                  </NavLink>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </nav>

      {/* Footer */}
      <div
        style={{
          padding: '14px 20px',
          borderTop: '1px solid var(--border-divider)',
        }}
      >
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                width: '100%',
                padding: 0,
                borderRadius: 6,
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
                }}
              >
                {initials}
              </div>
              <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{displayName}</span>
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent side="top" align="start" className="w-48">
            <div className="px-2 py-1.5">
              <p className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>
                {user.name}
              </p>
              <p className="text-xs" style={{ color: 'var(--text-muted)' }}>
                {user.email}
              </p>
            </div>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={() => logout.mutate()}
              className="text-destructive focus:text-destructive"
            >
              <LogOut className="mr-2 h-4 w-4" />
              Sign out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </aside>
  );
};
