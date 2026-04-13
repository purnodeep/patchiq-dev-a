import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router';
import {
  Search,
  Monitor,
  Package,
  ShieldAlert,
  Navigation,
  Settings,
  Zap,
  Loader2,
  LayoutDashboard,
  Tag,
  Shield,
  Rocket,
  FileText,
  Bell,
  ScrollText,
  Download,
  Users,
  UserCog,
} from 'lucide-react';

interface CommandPaletteProps {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

interface PaletteItem {
  id: string;
  label: string;
  path: string;
  category: string;
  icon: React.ReactNode;
}

const NAV_PAGES: PaletteItem[] = [
  {
    id: 'nav-dashboard',
    label: 'Dashboard',
    path: '/',
    category: 'Navigation',
    icon: <LayoutDashboard className="h-4 w-4" />,
  },
  {
    id: 'nav-endpoints',
    label: 'Endpoints',
    path: '/endpoints',
    category: 'Navigation',
    icon: <Monitor className="h-4 w-4" />,
  },
  {
    id: 'nav-tags',
    label: 'Tags',
    path: '/tags',
    category: 'Navigation',
    icon: <Tag className="h-4 w-4" />,
  },
  {
    id: 'nav-patches',
    label: 'Patches',
    path: '/patches',
    category: 'Navigation',
    icon: <Package className="h-4 w-4" />,
  },
  {
    id: 'nav-cves',
    label: 'CVEs',
    path: '/cves',
    category: 'Navigation',
    icon: <ShieldAlert className="h-4 w-4" />,
  },
  {
    id: 'nav-policies',
    label: 'Policies',
    path: '/policies',
    category: 'Navigation',
    icon: <Shield className="h-4 w-4" />,
  },
  {
    id: 'nav-deployments',
    label: 'Deployments',
    path: '/deployments',
    category: 'Navigation',
    icon: <Rocket className="h-4 w-4" />,
  },
  {
    id: 'nav-workflows',
    label: 'Workflows',
    path: '/workflows',
    category: 'Navigation',
    icon: <Navigation className="h-4 w-4" />,
  },
  {
    id: 'nav-compliance',
    label: 'Compliance',
    path: '/compliance',
    category: 'Navigation',
    icon: <Shield className="h-4 w-4" />,
  },
  {
    id: 'nav-alerts',
    label: 'Alerts',
    path: '/alerts',
    category: 'Navigation',
    icon: <Bell className="h-4 w-4" />,
  },
  {
    id: 'nav-audit',
    label: 'Audit Log',
    path: '/audit',
    category: 'Navigation',
    icon: <ScrollText className="h-4 w-4" />,
  },
  {
    id: 'nav-notifications',
    label: 'Notifications',
    path: '/notifications',
    category: 'Navigation',
    icon: <Bell className="h-4 w-4" />,
  },
  {
    id: 'nav-settings',
    label: 'Settings',
    path: '/settings',
    category: 'Navigation',
    icon: <Settings className="h-4 w-4" />,
  },
  {
    id: 'nav-roles',
    label: 'Roles',
    path: '/settings/roles',
    category: 'Navigation',
    icon: <Users className="h-4 w-4" />,
  },
  {
    id: 'nav-user-roles',
    label: 'User Roles',
    path: '/settings/user-roles',
    category: 'Navigation',
    icon: <UserCog className="h-4 w-4" />,
  },
  {
    id: 'nav-agent-downloads',
    label: 'Agent Downloads',
    path: '/agent-downloads',
    category: 'Navigation',
    icon: <Download className="h-4 w-4" />,
  },
];

const SETTINGS_PAGES: PaletteItem[] = [
  {
    id: 'set-general',
    label: 'General Settings',
    path: '/settings/general',
    category: 'Settings',
    icon: <Settings className="h-4 w-4" />,
  },
  {
    id: 'set-identity',
    label: 'Identity & IAM',
    path: '/settings/identity',
    category: 'Settings',
    icon: <Settings className="h-4 w-4" />,
  },
  {
    id: 'set-patch-sources',
    label: 'Patch Sources',
    path: '/settings/patch-sources',
    category: 'Settings',
    icon: <Settings className="h-4 w-4" />,
  },
  {
    id: 'set-agent-fleet',
    label: 'Agent Fleet',
    path: '/settings/agent-fleet',
    category: 'Settings',
    icon: <Settings className="h-4 w-4" />,
  },
  {
    id: 'set-notifications',
    label: 'Notification Settings',
    path: '/settings/notifications',
    category: 'Settings',
    icon: <Settings className="h-4 w-4" />,
  },
  {
    id: 'set-account',
    label: 'Account',
    path: '/settings/account',
    category: 'Settings',
    icon: <Settings className="h-4 w-4" />,
  },
  {
    id: 'set-license',
    label: 'License',
    path: '/settings/license',
    category: 'Settings',
    icon: <Settings className="h-4 w-4" />,
  },
  {
    id: 'set-appearance',
    label: 'Appearance',
    path: '/settings/appearance',
    category: 'Settings',
    icon: <Settings className="h-4 w-4" />,
  },
  {
    id: 'set-about',
    label: 'About',
    path: '/settings/about',
    category: 'Settings',
    icon: <Settings className="h-4 w-4" />,
  },
];

const ACTIONS: PaletteItem[] = [
  {
    id: 'act-register',
    label: 'Register Endpoint',
    path: '/endpoints',
    category: 'Actions',
    icon: <Zap className="h-4 w-4" />,
  },
  {
    id: 'act-deploy',
    label: 'Create Deployment',
    path: '/deployments/new',
    category: 'Actions',
    icon: <Zap className="h-4 w-4" />,
  },
  {
    id: 'act-critical',
    label: 'Review Critical Patches',
    path: '/patches?severity=critical',
    category: 'Actions',
    icon: <Zap className="h-4 w-4" />,
  },
  {
    id: 'act-compliance',
    label: 'View Compliance Dashboard',
    path: '/compliance',
    category: 'Actions',
    icon: <Zap className="h-4 w-4" />,
  },
];

const ALL_STATIC = [...NAV_PAGES, ...SETTINGS_PAGES, ...ACTIONS];

interface ApiResults {
  endpoints: Array<{ id: string; hostname: string }>;
  patches: Array<{ id: string; name: string }>;
  cves: Array<{ id: string; cve_id: string }>;
}

export function CommandPalette({
  open: openProp,
  onOpenChange: onOpenChangeProp,
}: CommandPaletteProps = {}) {
  const [internalOpen, setInternalOpen] = useState(openProp ?? false);
  const open = openProp ?? internalOpen;
  const onOpenChange = onOpenChangeProp ?? setInternalOpen;
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [apiResults, setApiResults] = useState<ApiResults>({
    endpoints: [],
    patches: [],
    cves: [],
  });
  const [apiLoading, setApiLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    if (open) {
      setQuery('');
      setSelectedIndex(0);
      setApiResults({ endpoints: [], patches: [], cves: [] });
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [open]);

  // Debounced API search
  useEffect(() => {
    if (!open || query.length < 2) {
      setApiResults({ endpoints: [], patches: [], cves: [] });
      setApiLoading(false);
      return;
    }

    setApiLoading(true);
    const timer = setTimeout(() => {
      const encoded = encodeURIComponent(query);
      const opts: RequestInit = { credentials: 'include' };

      Promise.all([
        fetch(`/api/v1/endpoints?search=${encoded}&limit=5`, opts)
          .then((r) => r.json())
          .catch(() => ({ data: [] })),
        fetch(`/api/v1/patches?search=${encoded}&limit=5`, opts)
          .then((r) => r.json())
          .catch(() => ({ data: [] })),
        fetch(`/api/v1/cves?search=${encoded}&limit=5`, opts)
          .then((r) => r.json())
          .catch(() => ({ data: [] })),
      ]).then(([endpointsRes, patchesRes, cvesRes]) => {
        setApiResults({
          endpoints: (endpointsRes.data ?? []).slice(0, 5),
          patches: (patchesRes.data ?? []).slice(0, 5),
          cves: (cvesRes.data ?? []).slice(0, 5),
        });
        setApiLoading(false);
      });
    }, 300);

    return () => clearTimeout(timer);
  }, [query, open]);

  // Build flat item list
  const buildItems = useCallback((): PaletteItem[] => {
    if (!query) {
      // Show suggested: nav + actions (skip settings for brevity)
      return [...NAV_PAGES, ...ACTIONS];
    }

    const lq = query.toLowerCase();
    const filteredStatic = ALL_STATIC.filter((item) => item.label.toLowerCase().includes(lq));

    const dynamicItems: PaletteItem[] = [
      ...apiResults.endpoints.map((e) => ({
        id: `ep-${e.id}`,
        label: e.hostname,
        path: `/endpoints/${e.id}`,
        category: 'Endpoints',
        icon: <Monitor className="h-4 w-4" />,
      })),
      ...apiResults.patches.map((p) => ({
        id: `pa-${p.id}`,
        label: p.name,
        path: `/patches/${p.id}`,
        category: 'Patches',
        icon: <Package className="h-4 w-4" />,
      })),
      ...apiResults.cves.map((c) => ({
        id: `cv-${c.id}`,
        label: c.cve_id,
        path: `/cves/${c.id}`,
        category: 'CVEs',
        icon: <ShieldAlert className="h-4 w-4" />,
      })),
    ];

    return [...filteredStatic, ...dynamicItems];
  }, [query, apiResults]);

  const items = buildItems();

  // Reset selected index when items change
  useEffect(() => {
    setSelectedIndex(0);
  }, [query, apiResults]);

  const handleSelect = useCallback(
    (path: string) => {
      onOpenChange(false);
      navigate(path);
    },
    [onOpenChange, navigate],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Escape') {
        onOpenChange(false);
      } else if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedIndex((prev) => (prev < items.length - 1 ? prev + 1 : 0));
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedIndex((prev) => (prev > 0 ? prev - 1 : items.length - 1));
      } else if (e.key === 'Enter' && items.length > 0) {
        e.preventDefault();
        const item = items[selectedIndex];
        if (item) handleSelect(item.path);
      }
    },
    [items, selectedIndex, handleSelect, onOpenChange],
  );

  // Scroll selected item into view
  useEffect(() => {
    if (!listRef.current) return;
    const el = listRef.current.querySelector(`[data-index="${selectedIndex}"]`);
    if (el) el.scrollIntoView({ block: 'nearest' });
  }, [selectedIndex]);

  if (!open) return null;

  // Group items by category preserving order
  const grouped: Array<{ category: string; items: Array<PaletteItem & { flatIndex: number }> }> =
    [];
  let flatIndex = 0;
  for (const item of items) {
    let group = grouped.find((g) => g.category === item.category);
    if (!group) {
      group = { category: item.category, items: [] };
      grouped.push(group);
    }
    group.items.push({ ...item, flatIndex });
    flatIndex++;
  }

  const categoryIcon = (cat: string) => {
    switch (cat) {
      case 'Navigation':
        return <Navigation className="h-3 w-3" />;
      case 'Settings':
        return <Settings className="h-3 w-3" />;
      case 'Actions':
        return <Zap className="h-3 w-3" />;
      case 'Endpoints':
        return <Monitor className="h-3 w-3" />;
      case 'Patches':
        return <Package className="h-3 w-3" />;
      case 'CVEs':
        return <ShieldAlert className="h-3 w-3" />;
      default:
        return <FileText className="h-3 w-3" />;
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex justify-center pt-20"
      onClick={() => onOpenChange(false)}
    >
      <div
        className="w-[580px] max-h-[440px] bg-card border border-border rounded-xl shadow-2xl overflow-hidden flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-3 px-4 py-3 border-b border-border">
          <Search className="h-4 w-4 text-muted-foreground" />
          <input
            ref={inputRef}
            type="text"
            placeholder="Search everything..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
          {apiLoading && <Loader2 className="h-4 w-4 text-muted-foreground animate-spin" />}
          <kbd className="text-[10px] text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
            ESC
          </kbd>
        </div>

        <div ref={listRef} className="flex-1 overflow-y-auto p-2">
          {grouped.map((group) => (
            <div key={group.category}>
              <div className="flex items-center gap-1.5 text-[10px] font-bold uppercase text-muted-foreground tracking-wider px-3 py-1">
                {categoryIcon(group.category)}
                <span>{group.category}</span>
              </div>
              {group.items.map((item) => (
                <button
                  key={item.id}
                  data-index={item.flatIndex}
                  onClick={() => handleSelect(item.path)}
                  className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-muted/50 text-left transition-colors ${
                    item.flatIndex === selectedIndex ? 'bg-muted/50' : ''
                  }`}
                >
                  <span className="text-muted-foreground">{item.icon}</span>
                  <span className="text-xs font-medium">{item.label}</span>
                </button>
              ))}
            </div>
          ))}

          {items.length === 0 && !apiLoading && (
            <p className="text-sm text-muted-foreground text-center py-8">No results found</p>
          )}

          {items.length === 0 && apiLoading && (
            <div className="flex items-center justify-center py-8 gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>Searching...</span>
            </div>
          )}
        </div>

        <div className="flex items-center gap-4 px-4 py-2 border-t border-border text-[10px] text-muted-foreground">
          <span>↑↓ navigate</span>
          <span>↵ select</span>
          <span>ESC close</span>
        </div>
      </div>
    </div>
  );
}
