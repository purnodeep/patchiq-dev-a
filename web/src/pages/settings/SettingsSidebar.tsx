import { NavLink, useNavigate } from 'react-router';
import {
  ArrowLeft,
  Settings,
  LogIn,
  Bell,
  CheckCircle,
  Sun,
  Database,
  Server,
  User,
  Info,
  Tag,
  Shield,
  UserCog,
  Lock,
} from 'lucide-react';
import { useCan } from '../../app/auth/AuthContext';
import { useIAMSettings } from '../../api/hooks/useIAMSettings';

const configItems = [
  {
    to: '/settings/general',
    label: 'General',
    icon: Settings,
    resource: 'settings',
    action: 'read',
  },
  {
    to: '/settings/identity',
    label: 'Identity & Access',
    icon: LogIn,
    resource: 'settings',
    action: 'read',
  },
  {
    to: '/settings/patch-sources',
    label: 'Patch Sources',
    icon: Database,
    resource: 'settings',
    action: 'read',
  },
  {
    to: '/settings/agent-fleet',
    label: 'Agent Fleet',
    icon: Server,
    resource: 'settings',
    action: 'read',
  },
  {
    to: '/settings/notifications',
    label: 'Notifications',
    icon: Bell,
    resource: 'settings',
    action: 'read',
  },
];

const accountItems = [
  { to: '/settings/account', label: 'My Account', icon: User },
  { to: '/settings/license', label: 'License', icon: CheckCircle },
  { to: '/settings/appearance', label: 'Appearance', icon: Sun },
];

const adminItems = [
  { to: '/settings/tags', label: 'Tags', icon: Tag, resource: 'endpoints', action: 'read' },
  { to: '/settings/roles', label: 'Roles', icon: Shield, resource: 'roles', action: 'read' },
  {
    to: '/settings/user-roles',
    label: 'User Roles',
    icon: UserCog,
    resource: 'roles',
    action: 'read',
  },
];

const systemItems = [{ to: '/settings/about', label: 'About', icon: Info }];

function NavItem({
  to,
  label,
  icon: Icon,
  statusDot,
  resource,
  action,
}: {
  to: string;
  label: string;
  icon: React.ComponentType<{ size?: number; style?: React.CSSProperties }>;
  statusDot?: boolean;
  resource?: string;
  action?: string;
}) {
  const can = useCan();
  const restricted = resource && action && !can(resource, action);

  if (restricted) {
    return (
      <div
        title={`You don't have access to ${label}`}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          padding: '7px 20px',
          fontSize: '13px',
          fontFamily: 'var(--font-sans)',
          position: 'relative',
          borderLeft: '2px solid transparent',
          background: 'transparent',
          color: 'var(--nav-item-color)',
          cursor: 'not-allowed',
          opacity: 0.4,
        }}
      >
        <Icon
          size={16}
          style={{
            flexShrink: 0,
          }}
        />
        <span style={{ flex: 1 }}>{label}</span>
        <Lock
          size={10}
          style={{
            flexShrink: 0,
            marginLeft: 'auto',
          }}
        />
      </div>
    );
  }

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
      {statusDot && (
        <span
          style={{
            width: '6px',
            height: '6px',
            borderRadius: '50%',
            background: 'var(--signal-healthy)',
            flexShrink: 0,
          }}
        />
      )}
    </NavLink>
  );
}

export function SettingsSidebar() {
  const navigate = useNavigate();
  const { data: iamSettings } = useIAMSettings();
  const iamConnected = iamSettings?.connection_status === 'connected';

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
          Patch Manager
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
            fontSize: '10px',
            fontWeight: 600,
            fontFamily: 'var(--font-sans)',
            color: 'var(--text-faint)',
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
            padding: '0 20px 6px',
          }}
        >
          Configuration
        </div>
        <nav style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
          {configItems.map((item) => (
            <NavItem
              key={item.to}
              to={item.to}
              label={item.label}
              icon={item.icon}
              resource={item.resource}
              action={item.action}
              statusDot={item.to === '/settings/identity' && iamConnected}
            />
          ))}
        </nav>
      </div>

      {/* Account group */}
      <div style={{ padding: '8px 0 4px' }}>
        <div
          style={{
            fontSize: '10px',
            fontWeight: 600,
            fontFamily: 'var(--font-sans)',
            color: 'var(--text-faint)',
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
            padding: '0 20px 6px',
          }}
        >
          Account
        </div>
        <nav style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
          {accountItems.map((item) => (
            <NavItem key={item.to} to={item.to} label={item.label} icon={item.icon} />
          ))}
        </nav>
      </div>

      {/* Administration group */}
      <div style={{ padding: '8px 0 4px' }}>
        <div
          style={{
            fontSize: '10px',
            fontWeight: 600,
            fontFamily: 'var(--font-sans)',
            color: 'var(--text-faint)',
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
            padding: '0 20px 6px',
          }}
        >
          Administration
        </div>
        <nav style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
          {adminItems.map((item) => (
            <NavItem
              key={item.to}
              to={item.to}
              label={item.label}
              icon={item.icon}
              resource={item.resource}
              action={item.action}
            />
          ))}
        </nav>
      </div>

      {/* System group */}
      <div style={{ padding: '8px 0 4px' }}>
        <div
          style={{
            fontSize: '10px',
            fontWeight: 600,
            fontFamily: 'var(--font-sans)',
            color: 'var(--text-faint)',
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
            padding: '0 20px 6px',
          }}
        >
          System
        </div>
        <nav style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
          {systemItems.map((item) => (
            <NavItem key={item.to} to={item.to} label={item.label} icon={item.icon} />
          ))}
        </nav>
      </div>
    </aside>
  );
}
