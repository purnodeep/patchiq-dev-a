import { NavLink } from 'react-router';
import { LayoutGrid, Shield, Cpu, Package, Server, Clock, Terminal, Settings } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

interface RailItem {
  to: string;
  label: string;
  icon: LucideIcon;
  end?: boolean;
}

const topItems: RailItem[] = [
  { to: '/', label: 'Overview', icon: LayoutGrid, end: true },
  { to: '/pending', label: 'Patches', icon: Shield },
  { to: '/hardware', label: 'Hardware', icon: Cpu },
  { to: '/software', label: 'Software', icon: Package },
  { to: '/services', label: 'Services', icon: Server },
];

const bottomItems: RailItem[] = [
  { to: '/history', label: 'History', icon: Clock },
  { to: '/logs', label: 'Logs', icon: Terminal },
  { to: '/settings', label: 'Settings', icon: Settings },
];

const RailNavItem = ({ item }: { item: RailItem }) => {
  const Icon = item.icon;
  return (
    <NavLink
      to={item.to}
      end={item.end}
      style={({ isActive }) => ({
        position: 'relative',
        width: '36px',
        height: '36px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        borderRadius: '6px',
        cursor: 'pointer',
        color: isActive ? 'var(--accent)' : 'var(--text-faint)',
        textDecoration: 'none',
        transition: 'color 0.15s, background 0.15s',
        flexShrink: 0,
      })}
      className="rail-item"
      aria-label={item.label}
      title={item.label}
    >
      {({ isActive }) => (
        <>
          {isActive && (
            <span
              style={{
                position: 'absolute',
                left: '-6px',
                top: '50%',
                transform: 'translateY(-50%)',
                width: '2px',
                height: '18px',
                background: 'var(--accent)',
                borderRadius: '0 2px 2px 0',
              }}
            />
          )}
          <Icon size={16} />
          <span
            className="rail-tooltip"
            style={{
              position: 'absolute',
              left: 'calc(100% + 8px)',
              top: '50%',
              transform: 'translateY(-50%)',
              background: 'var(--bg-surface)',
              border: '1px solid var(--border)',
              borderRadius: '4px',
              padding: '4px 8px',
              fontSize: '11px',
              color: 'var(--text-muted)',
              whiteSpace: 'nowrap',
              pointerEvents: 'none',
              zIndex: 200,
            }}
          >
            {item.label}
          </span>
        </>
      )}
    </NavLink>
  );
};

export const IconRail = () => {
  return (
    <>
      <style>{`
        .rail-item:hover {
          color: var(--text-muted) !important;
          background: var(--bg-surface);
        }
        .rail-tooltip {
          opacity: 0;
          transition: opacity 0.1s;
        }
        .rail-item:hover .rail-tooltip {
          opacity: 1;
        }
        .rail-avatar:hover .rail-avatar-dropdown {
          opacity: 1;
          pointer-events: auto;
        }
        .rail-avatar-action:hover {
          background: var(--bg-inset);
          color: var(--text-primary) !important;
        }
      `}</style>
      <nav
        style={{
          position: 'fixed',
          top: '48px',
          left: 0,
          bottom: 0,
          width: '48px',
          background: 'var(--bg-sidebar, var(--bg-card))',
          borderRight: '1px solid var(--border)',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          padding: '8px 0',
          zIndex: 90,
        }}
        aria-label="Main navigation"
      >
        <div
          style={{
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            gap: '2px',
            alignItems: 'center',
          }}
        >
          {topItems.map((item) => (
            <RailNavItem key={item.to} item={item} />
          ))}
        </div>

        <div
          style={{
            width: '24px',
            height: '1px',
            background: 'var(--border)',
            margin: '6px 0',
          }}
        />

        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: '2px',
            alignItems: 'center',
            paddingBottom: '8px',
          }}
        >
          {bottomItems.map((item) => (
            <RailNavItem key={item.to} item={item} />
          ))}
        </div>

        {/* Divider */}
        <div
          style={{
            width: '24px',
            height: '1px',
            background: 'var(--border, #222222)',
            margin: '6px 0',
          }}
        />

        {/* User avatar */}
        <div
          className="rail-avatar"
          style={{
            position: 'relative',
            marginBottom: '8px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            cursor: 'default',
          }}
        >
          <div
            style={{
              width: '28px',
              height: '28px',
              borderRadius: '50%',
              background: 'var(--avatar-bg, var(--bg-inset, #1a1a1a))',
              border: '1px solid var(--avatar-border, var(--border, #333333))',
              color: 'var(--avatar-text, var(--text-muted, #a1a1a1))',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: '9px',
              fontWeight: 600,
              letterSpacing: '0.04em',
              userSelect: 'none',
              flexShrink: 0,
            }}
          >
            AG
          </div>
          {/* Hover dropdown */}
          <div
            className="rail-avatar-dropdown"
            style={{
              position: 'absolute',
              left: 'calc(100% + 8px)',
              bottom: 0,
              opacity: 0,
              pointerEvents: 'none',
              transition: 'opacity 0.1s',
              background: 'var(--bg-surface, #111111)',
              border: '1px solid var(--border, #222222)',
              borderRadius: '6px',
              padding: '6px',
              minWidth: '140px',
              zIndex: 200,
            }}
          >
            <div
              style={{
                padding: '4px 8px 6px',
                borderBottom: '1px solid var(--border-divider, #222222)',
                marginBottom: '4px',
              }}
            >
              <div
                style={{
                  fontSize: '11px',
                  fontWeight: 600,
                  color: 'var(--text-primary, #f5f5f5)',
                  whiteSpace: 'nowrap',
                }}
              >
                Agent
              </div>
              <div
                style={{
                  fontSize: '10px',
                  color: 'var(--text-muted, #a1a1a1)',
                  whiteSpace: 'nowrap',
                  marginTop: '1px',
                }}
              >
                Local device
              </div>
            </div>
          </div>
        </div>
      </nav>
    </>
  );
};
