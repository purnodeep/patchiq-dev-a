import { lazy } from 'react';
import {
  Monitor,
  BarChart3,
  Rocket,
  TrendingUp,
  Activity,
  Bug,
  Crosshair,
  LineChart,
  Server,
  Zap,
  Terminal,
} from 'lucide-react';
import type { WidgetId, WidgetRegistryEntry, WidgetConfigSchema } from './types';

const StatCardsRow1 = lazy(() =>
  import('./StatCardsRow1').then((m) => ({ default: m.StatCardsRow1 })),
);
const StatCardsRow2 = lazy(() =>
  import('./StatCardsRow2').then((m) => ({ default: m.StatCardsRow2 })),
);
const RiskLandscape = lazy(() =>
  import('./RiskLandscape').then((m) => ({ default: m.RiskLandscape })),
);
const DeploymentPipeline = lazy(() =>
  import('./DeploymentPipeline').then((m) => ({ default: m.DeploymentPipeline })),
);
const PatchVelocity = lazy(() =>
  import('./PatchVelocity').then((m) => ({ default: m.PatchVelocity })),
);
const ActivityFeed = lazy(() =>
  import('./ActivityFeed').then((m) => ({ default: m.ActivityFeed })),
);
const TopVulnerabilities = lazy(() =>
  import('./TopVulnerabilities').then((m) => ({ default: m.TopVulnerabilities })),
);
const BlastRadiusWidget = lazy(() =>
  import('./BlastRadiusWidget').then((m) => ({ default: m.BlastRadiusWidget })),
);
const RiskProjectionWidget = lazy(() =>
  import('./RiskProjectionWidget').then((m) => ({ default: m.RiskProjectionWidget })),
);
const OSHeatmapWidget = lazy(() =>
  import('./OSHeatmapWidget').then((m) => ({ default: m.OSHeatmapWidget })),
);
const QuickActions = lazy(() =>
  import('./QuickActions').then((m) => ({ default: m.QuickActions })),
);
const CommandPalette = lazy(() =>
  import('./CommandPalette').then((m) => ({ default: m.CommandPalette })),
);

const entries: WidgetRegistryEntry[] = [
  {
    id: 'stat-cards-row-1',
    label: 'Primary KPIs',
    description: 'Endpoints online, critical patches, compliance rate, active deployments',
    category: 'kpi',
    icon: Monitor,
    component: StatCardsRow1,
    defaults: {
      lg: { x: 0, y: 0, w: 12, h: 2, minW: 6, maxW: 12, minH: 2, maxH: 3 },
      md: { x: 0, y: 0, w: 8, h: 2, minW: 6, maxW: 8, minH: 2, maxH: 3 },
      sm: { x: 0, y: 0, w: 4, h: 3, minW: 4, maxW: 4, minH: 2, maxH: 5 },
    },
  },
  {
    id: 'stat-cards-row-2',
    label: 'Operational Metrics',
    description: 'Failed deployments, workflows running, hub sync status',
    category: 'kpi',
    icon: BarChart3,
    component: StatCardsRow2,
    defaults: {
      lg: { x: 0, y: 2, w: 12, h: 2, minW: 6, maxW: 12, minH: 2, maxH: 4 },
      md: { x: 0, y: 2, w: 8, h: 2, minW: 6, maxW: 8, minH: 2, maxH: 4 },
      sm: { x: 0, y: 3, w: 4, h: 3, minW: 4, maxW: 4, minH: 2, maxH: 5 },
    },
  },
  {
    id: 'risk-landscape',
    label: 'Risk Landscape',
    description: 'Canvas dot-matrix visualization of endpoint health across your fleet',
    category: 'security',
    icon: Crosshair,
    component: RiskLandscape,
    configSchema: {
      viewMode: {
        type: 'select',
        label: 'View Mode',
        category: 'display',
        options: [
          { label: 'By Risk', value: 'risk' },
          { label: 'By Status', value: 'status' },
        ],
        default: 'risk',
      },
      limit: {
        type: 'number',
        label: 'Max Endpoints',
        category: 'data',
        default: 200,
        min: 100,
        max: 500,
      },
    } satisfies WidgetConfigSchema,
    defaults: {
      lg: { x: 0, y: 4, w: 7, h: 4, minW: 5, maxW: 12, minH: 3, maxH: 6 },
      md: { x: 0, y: 4, w: 5, h: 4, minW: 4, maxW: 8, minH: 3, maxH: 6 },
      sm: { x: 0, y: 6, w: 4, h: 5, minW: 4, maxW: 4, minH: 3, maxH: 6 },
    },
  },
  {
    id: 'risk-projection',
    label: 'Risk Projection',
    description: 'Projected risk score trend over time',
    category: 'security',
    icon: LineChart,
    component: RiskProjectionWidget,
    defaults: {
      lg: { x: 7, y: 4, w: 5, h: 4, minW: 4, maxW: 8, minH: 3, maxH: 6 },
      md: { x: 5, y: 4, w: 3, h: 4, minW: 3, maxW: 5, minH: 3, maxH: 6 },
      sm: { x: 0, y: 11, w: 4, h: 5, minW: 4, maxW: 4, minH: 4, maxH: 7 },
    },
  },
  {
    id: 'deployment-pipeline',
    label: 'Deployment Pipeline',
    description: 'Active deployments with progress bars',
    category: 'operations',
    icon: Rocket,
    component: DeploymentPipeline,
    defaults: {
      lg: { x: 0, y: 8, w: 4, h: 5, minW: 3, maxW: 8, minH: 3, maxH: 7 },
      md: { x: 0, y: 8, w: 4, h: 5, minW: 3, maxW: 8, minH: 3, maxH: 7 },
      sm: { x: 0, y: 16, w: 4, h: 6, minW: 4, maxW: 4, minH: 3, maxH: 7 },
    },
  },
  {
    id: 'patch-velocity',
    label: 'Patch Velocity',
    description: '13-week patch application trend line',
    category: 'operations',
    icon: TrendingUp,
    component: PatchVelocity,
    configSchema: {
      timeRange: {
        type: 'select',
        label: 'Time Range',
        category: 'data',
        options: [
          { label: '30 days', value: '30d' },
          { label: '60 days', value: '60d' },
          { label: '90 days', value: '90d' },
        ],
        default: '90d',
      },
      showLabels: {
        type: 'toggle',
        label: 'Show Labels',
        category: 'display',
        default: true,
      },
    } satisfies WidgetConfigSchema,
    defaults: {
      lg: { x: 4, y: 8, w: 4, h: 5, minW: 3, maxW: 8, minH: 3, maxH: 7 },
      md: { x: 4, y: 8, w: 4, h: 5, minW: 3, maxW: 8, minH: 3, maxH: 7 },
      sm: { x: 0, y: 22, w: 4, h: 5, minW: 4, maxW: 4, minH: 3, maxH: 7 },
    },
  },
  {
    id: 'activity-feed',
    label: 'Activity Feed',
    description: 'Timeline of recent events across the platform',
    category: 'activity',
    icon: Activity,
    component: ActivityFeed,
    configSchema: {
      limit: {
        type: 'select',
        label: 'Items to Show',
        category: 'data',
        options: [
          { label: '10 items', value: '10' },
          { label: '20 items', value: '20' },
          { label: '50 items', value: '50' },
        ],
        default: '20',
      },
      refreshInterval: {
        type: 'select',
        label: 'Refresh Interval',
        category: 'behavior',
        options: [
          { label: '30 seconds', value: '30' },
          { label: '1 minute', value: '60' },
          { label: '5 minutes', value: '300' },
        ],
        default: '60',
      },
    } satisfies WidgetConfigSchema,
    defaults: {
      lg: { x: 8, y: 8, w: 4, h: 5, minW: 3, maxW: 8, minH: 3, maxH: 7 },
      md: { x: 0, y: 13, w: 4, h: 5, minW: 3, maxW: 8, minH: 3, maxH: 7 },
      sm: { x: 0, y: 27, w: 4, h: 6, minW: 4, maxW: 4, minH: 3, maxH: 7 },
    },
  },
  {
    id: 'top-vulnerabilities',
    label: 'Top Vulnerabilities',
    description: 'Highest-severity CVEs with CVSS scores',
    category: 'security',
    icon: Bug,
    component: TopVulnerabilities,
    configSchema: {
      limit: {
        type: 'select',
        label: 'Items to Show',
        category: 'data',
        options: [
          { label: '5 items', value: '5' },
          { label: '10 items', value: '10' },
          { label: '15 items', value: '15' },
        ],
        default: '10',
      },
      severityFilter: {
        type: 'select',
        label: 'Severity Filter',
        category: 'data',
        options: [
          { label: 'Critical only', value: 'critical' },
          { label: 'High and above', value: 'high' },
          { label: 'All severities', value: 'all' },
        ],
        default: 'all',
      },
    } satisfies WidgetConfigSchema,
    defaults: {
      lg: { x: 0, y: 13, w: 6, h: 5, minW: 4, maxW: 12, minH: 3, maxH: 7 },
      md: { x: 4, y: 13, w: 4, h: 5, minW: 3, maxW: 8, minH: 3, maxH: 7 },
      sm: { x: 0, y: 33, w: 4, h: 6, minW: 4, maxW: 4, minH: 3, maxH: 7 },
    },
  },
  {
    id: 'os-heatmap',
    label: 'OS Heatmap',
    description: 'Operating system distribution across endpoints',
    category: 'operations',
    icon: Server,
    component: OSHeatmapWidget,
    defaults: {
      lg: { x: 6, y: 13, w: 6, h: 5, minW: 4, maxW: 12, minH: 3, maxH: 6 },
      md: { x: 0, y: 18, w: 4, h: 5, minW: 4, maxW: 8, minH: 3, maxH: 6 },
      sm: { x: 0, y: 39, w: 4, h: 5, minW: 4, maxW: 4, minH: 3, maxH: 6 },
    },
  },
  {
    id: 'blast-radius',
    label: 'Blast Radius',
    description: 'Impact analysis of unpatched vulnerabilities',
    category: 'security',
    icon: Crosshair,
    component: BlastRadiusWidget,
    defaults: {
      lg: { x: 0, y: 18, w: 6, h: 6, minW: 4, maxW: 12, minH: 5, maxH: 12 },
      md: { x: 4, y: 18, w: 4, h: 6, minW: 4, maxW: 8, minH: 5, maxH: 10 },
      sm: { x: 0, y: 44, w: 4, h: 7, minW: 4, maxW: 4, minH: 5, maxH: 10 },
    },
  },
  {
    id: 'quick-actions',
    label: 'Quick Actions',
    description: 'Shortcut buttons for common tasks',
    category: 'activity',
    icon: Zap,
    component: QuickActions,
    defaults: {
      lg: { x: 6, y: 18, w: 3, h: 5, minW: 2, maxW: 6, minH: 2, maxH: 6 },
      md: { x: 0, y: 24, w: 4, h: 5, minW: 3, maxW: 6, minH: 2, maxH: 6 },
      sm: { x: 0, y: 51, w: 4, h: 5, minW: 4, maxW: 4, minH: 2, maxH: 6 },
    },
  },
  {
    id: 'command-palette',
    label: 'Command Palette',
    description: 'Keyboard-driven command interface',
    category: 'activity',
    icon: Terminal,
    component: CommandPalette,
    defaults: {
      lg: { x: 9, y: 18, w: 3, h: 3, minW: 2, maxW: 6, minH: 2, maxH: 4 },
      md: { x: 4, y: 24, w: 4, h: 3, minW: 3, maxW: 6, minH: 2, maxH: 4 },
      sm: { x: 0, y: 56, w: 4, h: 3, minW: 4, maxW: 4, minH: 2, maxH: 4 },
    },
  },
];

export const WIDGET_REGISTRY = new Map<WidgetId, WidgetRegistryEntry>(
  entries.map((e) => [e.id, e]),
);

export const DEFAULT_WIDGET_IDS: WidgetId[] = [
  'stat-cards-row-1',
  'stat-cards-row-2',
  'risk-landscape',
  'risk-projection',
  'deployment-pipeline',
  'patch-velocity',
  'activity-feed',
  'top-vulnerabilities',
  'os-heatmap',
  'blast-radius',
  'quick-actions',
];

export const WIDGET_CATEGORIES: { key: WidgetRegistryEntry['category']; label: string }[] = [
  { key: 'kpi', label: 'Key Performance Indicators' },
  { key: 'security', label: 'Security & Compliance' },
  { key: 'operations', label: 'Operations' },
  { key: 'activity', label: 'Activity & Tools' },
];
