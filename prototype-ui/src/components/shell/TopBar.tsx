import { useState } from 'react';
import { useLocation } from 'react-router';
import { Search } from 'lucide-react';
import { ThemeToggle } from './ThemeToggle';
import { CommandPalette } from './CommandPalette';
import { NotificationPanel } from './NotificationPanel';

const BREADCRUMB_MAP: Record<string, [string, string]> = {
  '/pm/dashboard': ['Overview', 'Dashboard'],
  '/pm/endpoints': ['Inventory', 'Endpoints'],
  '/pm/patches': ['Inventory', 'Patches'],
  '/pm/cves': ['Inventory', 'CVEs'],
  '/pm/policies': ['Operations', 'Policies'],
  '/pm/deployments': ['Operations', 'Deployments'],
  '/pm/workflows': ['Operations', 'Workflows'],
  '/pm/compliance': ['Reports', 'Compliance'],
  '/pm/audit': ['Reports', 'Audit'],
  '/pm/settings': ['Configuration', 'Settings'],
  '/pm/roles': ['Configuration', 'Roles'],
  '/pm/notifications': ['Configuration', 'Notifications'],
};

export function TopBar() {
  const location = useLocation();
  const crumbs = BREADCRUMB_MAP[location.pathname] || ['', ''];
  const [cmdOpen, setCmdOpen] = useState(false);

  return (
    <>
      <div className="flex items-center gap-3 px-1">
        {/* Breadcrumbs */}
        <div className="flex items-center gap-1.5">
          <span
            className="text-[15px] font-bold"
            style={{
              background: 'linear-gradient(135deg, var(--color-primary), var(--color-cyan))',
              WebkitBackgroundClip: 'text',
              WebkitTextFillColor: 'transparent',
            }}
          >
            PatchIQ
          </span>
          <span className="text-xs text-subtle">/</span>
          <span className="text-xs text-muted">{crumbs[0]}</span>
          <span className="text-xs text-subtle">/</span>
          <span className="text-xs font-medium">{crumbs[1]}</span>
        </div>

        {/* Search trigger */}
        <button
          onClick={() => setCmdOpen(true)}
          className="glass-card ml-auto flex items-center gap-2 px-3 py-1.5 text-[11px] text-muted hover:text-foreground"
        >
          <Search size={13} />
          <span>Search...</span>
          <kbd className="rounded bg-foreground/5 px-1 py-0.5 text-[9px] font-medium">⌘K</kbd>
        </button>

        {/* Actions */}
        <div className="flex items-center gap-1">
          <ThemeToggle />
          <NotificationPanel />
        </div>
      </div>
      <CommandPalette open={cmdOpen} onOpenChange={setCmdOpen} />
    </>
  );
}
