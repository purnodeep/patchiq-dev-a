import { NavLink, useNavigate } from 'react-router';
import { ArrowLeft, Settings, LogIn, Rss, Terminal } from 'lucide-react';

const configItems = [
  { to: '/settings/general', label: 'General', icon: Settings },
  { to: '/settings/feeds', label: 'Feed Sources', icon: Rss },
  { to: '/settings/api', label: 'API & Webhooks', icon: Terminal },
];

const securityItems = [{ to: '/settings/iam', label: 'Identity & Access', icon: LogIn }];

function NavItem({
  to,
  label,
  icon: Icon,
}: {
  to: string;
  label: string;
  icon: React.ComponentType<{ size?: number; style?: React.CSSProperties }>;
}) {
  return (
    <NavLink
      to={to}
      style={({ isActive }) => ({
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        padding: '7px 20px',
        fontSize: '13px',
        fontFamily: 'var(--font-sans)',
        textDecoration: 'none',
        cursor: 'pointer',
        position: 'relative',
        borderLeft: isActive ? '2px solid var(--brand)' : '2px solid transparent',
        background: isActive ? 'var(--brand-subtle)' : 'transparent',
        color: isActive ? 'var(--text-emphasis)' : 'var(--nav-item-color)',
        fontWeight: isActive ? 600 : 400,
        transition: 'background 150ms ease, color 150ms ease',
      })}
      onMouseEnter={(e) => {
        const target = e.currentTarget;
        if (!target.classList.contains('active')) {
          target.style.background = 'var(--nav-item-hover)';
        }
      }}
      onMouseLeave={(e) => {
        const target = e.currentTarget;
        if (!target.classList.contains('active')) {
          target.style.background = 'transparent';
        }
      }}
    >
      <Icon
        size={16}
        style={{
          flexShrink: 0,
        }}
      />
      <span style={{ flex: 1 }}>{label}</span>
    </NavLink>
  );
}

export function SettingsSidebar() {
  const navigate = useNavigate();

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
      {/* Back link */}
      <button
        onClick={() => navigate('/')}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '6px',
          padding: '12px 16px',
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          fontSize: '13px',
          fontFamily: 'var(--font-sans)',
          color: 'var(--text-muted)',
          transition: 'color 150ms ease',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.color = 'var(--text-emphasis)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.color = 'var(--text-muted)';
        }}
      >
        <ArrowLeft size={14} />
        <span>Back</span>
      </button>

      {/* Header */}
      <div style={{ padding: '4px 16px 16px' }}>
        <div
          style={{
            fontSize: '15px',
            fontWeight: 600,
            fontFamily: 'var(--font-sans)',
            color: 'var(--text-emphasis)',
          }}
        >
          Settings
        </div>
        <div
          style={{
            fontSize: '11px',
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-faint)',
            marginTop: '2px',
          }}
        >
          Hub Manager
        </div>
      </div>

      <div
        style={{
          borderBottom: '1px solid var(--border-divider)',
          margin: '0 16px',
        }}
      />

      {/* Configuration group */}
      <div style={{ padding: '12px 0 4px' }}>
        <div
          style={{
            fontSize: 9,
            fontWeight: 600,
            fontFamily: 'var(--font-mono)',
            color: 'var(--nav-label)',
            textTransform: 'uppercase',
            letterSpacing: '0.08em',
            padding: '10px 20px 4px',
          }}
        >
          Configuration
        </div>
        <nav style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
          {configItems.map((item) => (
            <NavItem key={item.to} to={item.to} label={item.label} icon={item.icon} />
          ))}
        </nav>
      </div>

      {/* Security group */}
      <div style={{ padding: '8px 0 4px' }}>
        <div
          style={{
            fontSize: 9,
            fontWeight: 600,
            fontFamily: 'var(--font-mono)',
            color: 'var(--nav-label)',
            textTransform: 'uppercase',
            letterSpacing: '0.08em',
            padding: '10px 20px 4px',
          }}
        >
          Security
        </div>
        <nav style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
          {securityItems.map((item) => (
            <NavItem key={item.to} to={item.to} label={item.label} icon={item.icon} />
          ))}
        </nav>
      </div>
    </aside>
  );
}
