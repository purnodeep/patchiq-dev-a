import { useEffect } from 'react';
import { Command } from 'cmdk';
import { useNavigate } from 'react-router';
import {
  Monitor,
  Layers,
  Shield,
  Rocket,
  FileText,
  GitBranch,
  CheckCircle,
  PenLine,
  Settings,
  Users,
  Bell,
  LayoutDashboard,
  Search,
} from 'lucide-react';

const COMMANDS = [
  {
    group: 'Navigation',
    items: [
      { label: 'Dashboard', icon: <LayoutDashboard size={14} />, path: '/pm/dashboard' },
      { label: 'Endpoints', icon: <Monitor size={14} />, path: '/pm/endpoints' },
      { label: 'Patches', icon: <Layers size={14} />, path: '/pm/patches' },
      { label: 'CVEs', icon: <Shield size={14} />, path: '/pm/cves' },
      { label: 'Policies', icon: <FileText size={14} />, path: '/pm/policies' },
      { label: 'Deployments', icon: <Rocket size={14} />, path: '/pm/deployments' },
      { label: 'Workflows', icon: <GitBranch size={14} />, path: '/pm/workflows' },
      { label: 'Compliance', icon: <CheckCircle size={14} />, path: '/pm/compliance' },
      { label: 'Audit', icon: <PenLine size={14} />, path: '/pm/audit' },
      { label: 'Settings', icon: <Settings size={14} />, path: '/pm/settings' },
      { label: 'Roles', icon: <Users size={14} />, path: '/pm/roles' },
      { label: 'Notifications', icon: <Bell size={14} />, path: '/pm/notifications' },
    ],
  },
  {
    group: 'Endpoints',
    items: [
      { label: 'prod-web-01', icon: <Monitor size={14} />, path: '/pm/endpoints' },
      { label: 'db-primary-02', icon: <Monitor size={14} />, path: '/pm/endpoints' },
      { label: 'cache-node-03', icon: <Monitor size={14} />, path: '/pm/endpoints' },
    ],
  },
  {
    group: 'Patches',
    items: [
      {
        label: 'KB5034441 — Windows Security Update',
        icon: <Layers size={14} />,
        path: '/pm/patches',
      },
      { label: 'KB5034439 — .NET Framework', icon: <Layers size={14} />, path: '/pm/patches' },
    ],
  },
  {
    group: 'CVEs',
    items: [
      {
        label: 'CVE-2024-21412 — SmartScreen Bypass',
        icon: <Shield size={14} />,
        path: '/pm/cves',
      },
      { label: 'CVE-2024-21351 — Windows Defender', icon: <Shield size={14} />, path: '/pm/cves' },
    ],
  },
  {
    group: 'Actions',
    items: [
      { label: 'Create Deployment', icon: <Rocket size={14} />, path: '/pm/deployments' },
      { label: 'Run Compliance Scan', icon: <CheckCircle size={14} />, path: '/pm/compliance' },
      { label: 'New Policy', icon: <FileText size={14} />, path: '/pm/policies' },
    ],
  },
];

interface CommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const navigate = useNavigate();

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        onOpenChange(!open);
      }
      if (e.key === 'Escape' && open) {
        onOpenChange(false);
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [open, onOpenChange]);

  const runCommand = (path: string) => {
    onOpenChange(false);
    navigate(path);
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[20vh]">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-background/60 backdrop-blur-sm"
        onClick={() => onOpenChange(false)}
      />
      {/* Dialog */}
      <div className="glass relative w-full max-w-[520px] overflow-hidden">
        <Command className="flex flex-col">
          <div className="flex items-center gap-2 border-b border-separator px-4 py-3">
            <Search size={16} className="text-muted" />
            <Command.Input
              placeholder="Search endpoints, patches, CVEs..."
              className="flex-1 bg-transparent text-sm outline-none placeholder:text-subtle"
              autoFocus
            />
            <kbd className="rounded bg-foreground/5 px-1.5 py-0.5 text-[10px] font-medium text-muted">
              ESC
            </kbd>
          </div>
          <Command.List className="max-h-[300px] overflow-y-auto p-2">
            <Command.Empty className="px-4 py-8 text-center text-sm text-muted">
              No results found.
            </Command.Empty>
            {COMMANDS.map((group) => (
              <Command.Group
                key={group.group}
                heading={group.group}
                className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-[10px] [&_[cmdk-group-heading]]:font-semibold [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:tracking-wider [&_[cmdk-group-heading]]:text-muted"
              >
                {group.items.map((item) => (
                  <Command.Item
                    key={item.label}
                    onSelect={() => runCommand(item.path)}
                    className="flex cursor-pointer items-center gap-2.5 rounded-lg px-2.5 py-2 text-xs text-muted transition-colors data-[selected=true]:bg-foreground/5 data-[selected=true]:text-foreground"
                  >
                    {item.icon}
                    {item.label}
                  </Command.Item>
                ))}
              </Command.Group>
            ))}
          </Command.List>
        </Command>
      </div>
    </div>
  );
}
