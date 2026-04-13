import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router';
import { Bell, X, AlertTriangle, Info, CheckCircle, Clock } from 'lucide-react';

interface Notification {
  id: string;
  type: 'critical' | 'warning' | 'info' | 'success';
  title: string;
  body: string;
  time: string;
  read: boolean;
  path: string;
}

const INITIAL_NOTIFICATIONS: Notification[] = [
  {
    id: 'n1',
    type: 'critical',
    title: 'SLA Breach Imminent',
    body: 'KB5034441 due in 4h — 34 endpoints unpatched',
    time: '2m ago',
    read: false,
    path: '/pm/patches',
  },
  {
    id: 'n2',
    type: 'critical',
    title: 'CVE-2024-21762 Active',
    body: 'FortiOS RCE affects 73 endpoints. Patch available.',
    time: '18m ago',
    read: false,
    path: '/pm/cves',
  },
  {
    id: 'n3',
    type: 'warning',
    title: 'Deployment Failed',
    body: 'KB5036893 — SP-APP-02 pre-check failed: disk space',
    time: '1h ago',
    read: false,
    path: '/pm/deployments',
  },
  {
    id: 'n4',
    type: 'info',
    title: 'Workflow Awaiting Approval',
    body: 'Standard Monthly Rollout needs sec-team sign-off',
    time: '2h ago',
    read: true,
    path: '/pm/workflows',
  },
  {
    id: 'n5',
    type: 'success',
    title: 'Deployment Complete',
    body: 'KB5034441 — Production Web Servers (34 endpoints)',
    time: '6h ago',
    read: true,
    path: '/pm/deployments',
  },
];

const TYPE_META = {
  critical: {
    icon: AlertTriangle,
    color: 'text-danger',
    border: 'border-l-danger',
    bg: 'bg-danger/5',
  },
  warning: {
    icon: Clock,
    color: 'text-warning',
    border: 'border-l-warning',
    bg: 'bg-warning/5',
  },
  info: {
    icon: Info,
    color: 'text-primary',
    border: 'border-l-primary',
    bg: 'bg-primary/5',
  },
  success: {
    icon: CheckCircle,
    color: 'text-success',
    border: 'border-l-success',
    bg: 'bg-success/5',
  },
};

export function NotificationPanel() {
  const [open, setOpen] = useState(false);
  const [notifications, setNotifications] = useState<Notification[]>(INITIAL_NOTIFICATIONS);
  const panelRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  const unreadCount = notifications.filter((n) => !n.read).length;

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [open]);

  const markAllRead = () => setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));

  const dismiss = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setNotifications((prev) => prev.filter((n) => n.id !== id));
  };

  const handleClick = (n: Notification) => {
    setNotifications((prev) => prev.map((x) => (x.id === n.id ? { ...x, read: true } : x)));
    setOpen(false);
    navigate(n.path);
  };

  return (
    <div ref={panelRef} className="relative">
      {/* Bell button */}
      <button
        onClick={() => setOpen((o) => !o)}
        className="relative flex h-8 w-8 items-center justify-center rounded-lg text-muted transition-colors hover:bg-foreground/5 hover:text-foreground"
        aria-label="Notifications"
      >
        <Bell size={16} />
        {unreadCount > 0 && (
          <span className="absolute right-1 top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-danger px-1 text-[9px] font-bold text-white">
            {unreadCount}
          </span>
        )}
      </button>

      {/* Dropdown */}
      {open && (
        <div className="glass absolute right-0 top-10 z-50 w-[340px] overflow-hidden shadow-xl">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-separator px-4 py-3">
            <span className="text-xs font-semibold">Notifications</span>
            {unreadCount > 0 && (
              <button onClick={markAllRead} className="text-[10px] text-primary hover:underline">
                Mark all read
              </button>
            )}
          </div>

          {/* List */}
          <div className="max-h-[380px] overflow-y-auto">
            {notifications.length === 0 ? (
              <div className="px-4 py-8 text-center text-xs text-muted">No notifications</div>
            ) : (
              notifications.map((n) => {
                const meta = TYPE_META[n.type];
                const Icon = meta.icon;
                return (
                  <button
                    key={n.id}
                    onClick={() => handleClick(n)}
                    className={`group relative w-full border-l-2 px-4 py-3 text-left transition-colors hover:bg-foreground/5 ${meta.border} ${!n.read ? meta.bg : ''}`}
                  >
                    <div className="flex items-start gap-3">
                      <Icon size={13} className={`mt-0.5 shrink-0 ${meta.color}`} />
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <span
                            className={`text-[11px] font-semibold leading-none ${!n.read ? 'text-foreground' : 'text-muted'}`}
                          >
                            {n.title}
                          </span>
                          {!n.read && (
                            <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-primary" />
                          )}
                        </div>
                        <p className="mt-1 text-[10px] leading-relaxed text-muted">{n.body}</p>
                        <span className="mt-1 block text-[9px] text-subtle">{n.time}</span>
                      </div>
                    </div>
                    {/* Dismiss */}
                    <button
                      onClick={(e) => dismiss(n.id, e)}
                      className="absolute right-3 top-3 hidden rounded p-0.5 text-subtle hover:text-muted group-hover:flex"
                      aria-label="Dismiss"
                    >
                      <X size={11} />
                    </button>
                  </button>
                );
              })
            )}
          </div>

          {/* Footer */}
          {notifications.length > 0 && (
            <div className="border-t border-separator px-4 py-2">
              <button
                onClick={() => {
                  setOpen(false);
                  navigate('/pm/notifications');
                }}
                className="text-[10px] text-primary hover:underline"
              >
                View all in Notifications →
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
