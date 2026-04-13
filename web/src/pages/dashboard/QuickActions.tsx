import { useNavigate } from 'react-router';
import { Card, CardContent, CardHeader, CardTitle } from '@patchiq/ui';
import { Rocket, ScanSearch, ShieldAlert, Settings, FileText, Users } from 'lucide-react';

const actions = [
  {
    label: 'New Deployment',
    icon: Rocket,
    path: '/deployments/new',
    color: 'text-blue-500 bg-blue-500/10',
  },
  {
    label: 'Scan All',
    icon: ScanSearch,
    path: '/endpoints',
    color: 'text-emerald-500 bg-emerald-500/10',
  },
  {
    label: 'Review Critical',
    icon: ShieldAlert,
    path: '/patches?severity=critical',
    color: 'text-red-500 bg-red-500/10',
  },
  {
    label: 'Compliance Report',
    icon: FileText,
    path: '/compliance',
    color: 'text-purple-500 bg-purple-500/10',
  },
  {
    label: 'Manage Roles',
    icon: Users,
    path: '/settings/roles',
    color: 'text-amber-500 bg-amber-500/10',
  },
  { label: 'Settings', icon: Settings, path: '/settings', color: 'text-gray-500 bg-gray-500/10' },
];

export function QuickActions() {
  const navigate = useNavigate();

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Quick Actions</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-3 gap-2">
          {actions.map(({ label, icon: Icon, path, color }) => (
            <button
              key={label}
              onClick={() => navigate(path)}
              className="flex flex-col items-center gap-1.5 rounded-lg border border-border p-3 hover:bg-muted/50 transition-colors"
            >
              <div className={`h-8 w-8 rounded-lg flex items-center justify-center ${color}`}>
                <Icon className="h-4 w-4" />
              </div>
              <span className="text-[10px] font-medium">{label}</span>
            </button>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
