import { NavLink } from 'react-router';
import { useAgentStatus } from '../../api/hooks/useStatus';

export const TabNav = () => {
  const { data } = useAgentStatus();
  const pendingCount = (data as any)?.pending_patch_count ?? 0;

  const tabs = [
    { to: '/', label: 'Overview', showDot: true },
    { to: '/pending', label: 'Patches', badge: pendingCount > 0 ? pendingCount : undefined },
    { to: '/hardware', label: 'Hardware' },
    { to: '/software', label: 'Software' },
    { to: '/services', label: 'Services' },
    { to: '/history', label: 'History' },
    { to: '/logs', label: 'Logs' },
    { to: '/settings', label: 'Settings' },
  ] as const;

  return (
    <nav
      style={{
        position: 'fixed',
        top: '64px',
        left: 0,
        right: 0,
        zIndex: 99,
        height: '48px',
        background: 'var(--bg-card)',
        borderBottom: '1px solid var(--border)',
        display: 'flex',
        alignItems: 'stretch',
        padding: '0 24px',
      }}
    >
      {tabs.map((tab) => (
        <NavLink
          key={tab.to}
          to={tab.to}
          end={tab.to === '/'}
          style={({ isActive }) => ({
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            padding: '0 18px',
            fontSize: '13px',
            fontWeight: 500,
            textDecoration: 'none',
            borderBottom: isActive ? '2px solid var(--accent)' : '2px solid transparent',
            color: isActive ? 'var(--accent)' : 'var(--text-muted)',
            transition: 'color 0.15s, background 0.15s',
            cursor: 'pointer',
          })}
        >
          {({ isActive }) => (
            <>
              {'showDot' in tab && tab.showDot && isActive && (
                <div
                  style={{
                    width: '7px',
                    height: '7px',
                    borderRadius: '50%',
                    background: 'var(--accent)',
                    flexShrink: 0,
                  }}
                />
              )}
              {tab.label}
              {'badge' in tab && tab.badge !== undefined && (
                <span
                  style={{
                    background: 'color-mix(in srgb, var(--signal-critical) 12%, transparent)',
                    color: 'var(--signal-critical)',
                    border: '1px solid color-mix(in srgb, var(--signal-critical) 20%, transparent)',
                    borderRadius: '10px',
                    padding: '1px 6px',
                    fontSize: '10px',
                    lineHeight: '1.4',
                  }}
                >
                  {tab.badge}
                </span>
              )}
            </>
          )}
        </NavLink>
      ))}
    </nav>
  );
};
