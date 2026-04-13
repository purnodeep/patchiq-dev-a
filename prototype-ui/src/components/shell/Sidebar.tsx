import { useState, useRef, useEffect } from 'react';
import { NavLink } from 'react-router';
import { cn } from '@/lib/utils';
import {
  LayoutDashboard,
  Monitor,
  Layers,
  Shield,
  FileText,
  Rocket,
  GitBranch,
  CheckCircle,
  PenLine,
  Settings,
  Users,
  Bell,
  ChevronDown,
  PanelLeftClose,
  PanelLeft,
  User,
  Building2,
  LogOut,
  ChevronUp,
} from 'lucide-react';

interface NavItem {
  label: string;
  path: string;
  icon: React.ReactNode;
  badge?: { count: number; color: string };
}

interface NavGroup {
  label: string;
  items: NavItem[];
}

const PINNED: NavItem = {
  label: 'Dashboard',
  path: '/pm/dashboard',
  icon: <LayoutDashboard size={15} />,
};

const NAV_GROUPS: NavGroup[] = [
  {
    label: 'Inventory',
    items: [
      {
        label: 'Endpoints',
        path: '/pm/endpoints',
        icon: <Monitor size={15} />,
        badge: { count: 247, color: 'primary' },
      },
      {
        label: 'Patches',
        path: '/pm/patches',
        icon: <Layers size={15} />,
        badge: { count: 12, color: 'danger' },
      },
      { label: 'CVEs', path: '/pm/cves', icon: <Shield size={15} /> },
    ],
  },
  {
    label: 'Operations',
    items: [
      { label: 'Policies', path: '/pm/policies', icon: <FileText size={15} /> },
      {
        label: 'Deployments',
        path: '/pm/deployments',
        icon: <Rocket size={15} />,
        badge: { count: 3, color: 'warning' },
      },
      { label: 'Workflows', path: '/pm/workflows', icon: <GitBranch size={15} /> },
    ],
  },
  {
    label: 'Reports',
    items: [
      { label: 'Compliance', path: '/pm/compliance', icon: <CheckCircle size={15} /> },
      { label: 'Audit', path: '/pm/audit', icon: <PenLine size={15} /> },
    ],
  },
  {
    label: 'Configuration',
    items: [
      { label: 'Settings', path: '/pm/settings', icon: <Settings size={15} /> },
      { label: 'Roles', path: '/pm/roles', icon: <Users size={15} /> },
      { label: 'Notifications', path: '/pm/notifications', icon: <Bell size={15} /> },
    ],
  },
];

const BADGE_COLORS: Record<string, string> = {
  primary: 'bg-primary/10 text-primary',
  danger: 'bg-danger/10 text-danger',
  warning: 'bg-warning/10 text-warning',
};

const BADGE_DOT: Record<string, string> = {
  primary: 'bg-primary',
  danger: 'bg-danger',
  warning: 'bg-warning',
};

function NavItemLink({ item, collapsed }: { item: NavItem; collapsed: boolean }) {
  return (
    <NavLink
      to={item.path}
      className={({ isActive }) =>
        cn(
          'group/nav relative flex items-center gap-2.5 rounded-lg text-[11.5px] transition-all duration-150',
          collapsed ? 'justify-center px-0 py-[9px]' : 'px-2.5 py-[7px]',
          isActive
            ? 'bg-primary/[.12] font-semibold shadow-[0_0_0_1px_rgba(59,130,246,0.22),0_2px_14px_rgba(59,130,246,0.13)]'
            : 'text-muted hover:bg-foreground/[.06] hover:text-foreground',
        )
      }
    >
      {({ isActive }) => (
        <>
          {/* Left active bar with glow */}
          {isActive && (
            <div
              className="absolute left-0 top-1.5 bottom-1.5 w-[3px] rounded-r-full"
              style={{
                background: 'linear-gradient(180deg, var(--color-primary), var(--color-cyan))',
                boxShadow: '0 0 8px rgba(59, 130, 246, 0.7)',
              }}
            />
          )}

          {/* Icon */}
          <span className={cn('shrink-0 transition-colors', isActive ? 'text-primary' : '')}>
            {item.icon}
          </span>

          {/* Label + badge (expanded) */}
          {!collapsed && (
            <>
              <span
                className="flex-1 truncate"
                style={
                  isActive
                    ? {
                        background:
                          'linear-gradient(90deg, var(--color-primary), var(--color-cyan))',
                        WebkitBackgroundClip: 'text',
                        WebkitTextFillColor: 'transparent',
                      }
                    : undefined
                }
              >
                {item.label}
              </span>
              {item.badge && (
                <span
                  className={cn(
                    'rounded-full px-1.5 py-px text-[9px] font-semibold',
                    BADGE_COLORS[item.badge.color],
                  )}
                >
                  {item.badge.count}
                </span>
              )}
            </>
          )}

          {/* Badge dot (collapsed) */}
          {collapsed && item.badge && (
            <span
              className={cn(
                'absolute right-2.5 top-2 h-1.5 w-1.5 rounded-full ring-1 ring-background',
                BADGE_DOT[item.badge.color],
              )}
            />
          )}

          {/* Tooltip (collapsed) */}
          {collapsed && (
            <div className="pointer-events-none absolute left-full z-50 ml-3 flex items-center gap-1.5 whitespace-nowrap rounded-lg border border-separator bg-[var(--color-glass-card)] px-2.5 py-1.5 text-[11px] font-medium text-foreground opacity-0 shadow-lg backdrop-blur-sm transition-opacity duration-100 group-hover/nav:opacity-100">
              {item.label}
              {item.badge && (
                <span
                  className={cn(
                    'rounded-full px-1.5 py-px text-[9px] font-semibold',
                    BADGE_COLORS[item.badge.color],
                  )}
                >
                  {item.badge.count}
                </span>
              )}
            </div>
          )}
        </>
      )}
    </NavLink>
  );
}

export function Sidebar() {
  const [collapsed, setCollapsed] = useState(false);
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>(
    Object.fromEntries(NAV_GROUPS.map((g) => [g.label, true])),
  );

  const toggleGroup = (label: string) => {
    setOpenGroups((prev) => ({ ...prev, [label]: !prev[label] }));
  };

  return (
    <nav
      className={cn(
        'glass flex shrink-0 flex-col transition-all duration-200',
        collapsed ? 'w-[64px]' : 'w-[240px]',
      )}
    >
      {/* Logo + collapse toggle */}
      <div
        className={cn(
          'flex items-center gap-2.5 px-3 py-[14px]',
          collapsed && 'flex-col gap-2 px-0',
        )}
      >
        <div
          className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-xs font-bold text-white"
          style={{ background: 'linear-gradient(135deg, var(--color-primary), var(--color-cyan))' }}
        >
          P
        </div>
        {!collapsed ? (
          <>
            <div className="min-w-0 flex-1">
              <div className="text-[13px] font-bold leading-tight">PatchIQ</div>
              <div className="text-[9px] text-muted">Patch Manager</div>
            </div>
            <button
              onClick={() => setCollapsed(true)}
              title="Collapse sidebar"
              className="flex h-7 w-7 items-center justify-center rounded-lg text-muted transition-colors hover:bg-foreground/5 hover:text-foreground"
            >
              <PanelLeftClose size={14} />
            </button>
          </>
        ) : (
          <button
            onClick={() => setCollapsed(false)}
            title="Expand sidebar"
            className="flex h-6 w-6 items-center justify-center rounded-lg text-muted transition-colors hover:bg-foreground/5 hover:text-foreground"
          >
            <PanelLeft size={13} />
          </button>
        )}
      </div>

      {/* Nav */}
      <div className="flex-1 overflow-y-auto px-2 pb-2">
        {/* Pinned: Dashboard */}
        <div className="mb-2 mt-1">
          <NavItemLink item={PINNED} collapsed={collapsed} />
        </div>

        {/* Groups */}
        {NAV_GROUPS.map((group, gi) => (
          <div
            key={group.label}
            className={cn('mb-1', collapsed && gi > 0 && 'mt-1 border-t border-separator pt-1')}
          >
            {!collapsed && (
              <button
                onClick={() => toggleGroup(group.label)}
                className="flex w-full items-center justify-between px-2 pb-1 pt-3 text-[10px] font-semibold uppercase tracking-wider text-muted hover:text-foreground"
              >
                <span>{group.label}</span>
                <ChevronDown
                  size={10}
                  className={cn('transition-transform', !openGroups[group.label] && '-rotate-90')}
                />
              </button>
            )}
            {(collapsed || openGroups[group.label]) && (
              <div className="flex flex-col gap-0.5">
                {group.items.map((item) => (
                  <NavItemLink key={item.path} item={item} collapsed={collapsed} />
                ))}
              </div>
            )}
          </div>
        ))}
      </div>

      {/* User footer */}
      <UserFooter collapsed={collapsed} />
    </nav>
  );
}

function UserFooter({ collapsed }: { collapsed: boolean }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  const avatar = (
    <div
      className="relative flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-[10px] font-semibold text-white"
      style={{ background: 'linear-gradient(135deg, var(--color-primary), var(--color-purple))' }}
    >
      A{/* Online status dot */}
      <span className="absolute bottom-0 right-0 h-2 w-2 rounded-full border-[1.5px] border-background bg-success" />
    </div>
  );

  const menu = open && (
    <div className="absolute bottom-full left-0 right-0 z-50 mb-1 mx-2 overflow-hidden rounded-xl border border-separator bg-[var(--color-glass-card)] shadow-xl backdrop-blur-sm">
      {/* User identity header */}
      <div className="border-b border-separator px-3 py-2.5">
        <div className="flex items-center gap-2">
          {avatar}
          <div className="min-w-0">
            <div className="text-[11.5px] font-semibold leading-tight">Messi Dev</div>
            <div className="text-[9.5px] text-muted">messi@acmecorp.com</div>
          </div>
          <span className="ml-auto rounded-full bg-primary/10 px-1.5 py-px text-[9px] font-semibold text-primary">
            Admin
          </span>
        </div>
      </div>

      {/* Tenant */}
      <div className="border-b border-separator px-3 py-2">
        <div className="flex items-center gap-1.5 text-[10.5px] text-muted">
          <Building2 size={11} />
          <span className="flex-1 truncate font-medium text-foreground">Acme Corp</span>
          <button className="rounded px-1 py-px text-[9px] text-primary hover:bg-primary/10">
            Switch
          </button>
        </div>
      </div>

      {/* Actions */}
      <div className="p-1">
        {[
          { icon: <User size={12} />, label: 'Profile' },
          { icon: <Settings size={12} />, label: 'Settings', hint: '⌘,' },
        ].map(({ icon, label, hint }) => (
          <button
            key={label}
            className="flex w-full items-center gap-2 rounded-lg px-2.5 py-1.5 text-[11px] text-foreground hover:bg-foreground/[.06]"
          >
            <span className="text-muted">{icon}</span>
            {label}
            {hint && <span className="ml-auto text-[9px] text-muted">{hint}</span>}
          </button>
        ))}
        <div className="my-1 border-t border-separator" />
        <button className="flex w-full items-center gap-2 rounded-lg px-2.5 py-1.5 text-[11px] text-danger hover:bg-danger/[.08]">
          <LogOut size={12} />
          Sign out
        </button>
      </div>
    </div>
  );

  return (
    <div ref={ref} className="relative border-t border-separator">
      {menu}
      {!collapsed ? (
        <button
          onClick={() => setOpen((v) => !v)}
          className="flex w-full items-center gap-2.5 px-3 py-3 transition-colors hover:bg-foreground/[.04]"
        >
          {avatar}
          <div className="min-w-0 flex-1 text-left">
            <div className="text-[11.5px] font-medium leading-tight">Messi Dev</div>
            <div className="flex items-center gap-1.5">
              <span className="text-[9px] text-muted">Acme Corp</span>
              <span className="rounded-full bg-primary/10 px-1 py-px text-[8.5px] font-semibold text-primary">
                Admin
              </span>
            </div>
          </div>
          <ChevronUp
            size={12}
            className={cn('shrink-0 text-muted transition-transform', open ? 'rotate-180' : '')}
          />
        </button>
      ) : (
        <div className="flex justify-center py-3">
          <button onClick={() => setOpen((v) => !v)} className="group/avatar relative">
            {avatar}
            {!open && (
              <div className="pointer-events-none absolute left-full z-50 ml-3 whitespace-nowrap rounded-lg border border-separator bg-[var(--color-glass-card)] px-2.5 py-1.5 text-[11px] font-medium text-foreground opacity-0 shadow-lg backdrop-blur-sm transition-opacity duration-100 group-hover/avatar:opacity-100">
                Messi Dev · Acme Corp
              </div>
            )}
          </button>
        </div>
      )}
    </div>
  );
}
