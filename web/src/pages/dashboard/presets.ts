import type { WidgetId } from './types';

export type PresetId = 'executive' | 'operations' | 'security' | 'custom';

export interface DashboardPreset {
  id: PresetId;
  label: string;
  description: string;
  widgets: WidgetId[];
}

export const DASHBOARD_PRESETS: DashboardPreset[] = [
  {
    id: 'executive',
    label: 'Executive',
    description: 'High-level KPIs, risk trends, and patch velocity',
    widgets: [
      'stat-cards-row-1',
      'stat-cards-row-2',
      'risk-landscape',
      'risk-projection',
      'patch-velocity',
    ],
  },
  {
    id: 'operations',
    label: 'Operations',
    description: 'Deployment pipelines, OS coverage, and activity',
    widgets: [
      'stat-cards-row-1',
      'stat-cards-row-2',
      'deployment-pipeline',
      'patch-velocity',
      'activity-feed',
      'os-heatmap',
      'quick-actions',
    ],
  },
  {
    id: 'security',
    label: 'Security',
    description: 'Vulnerability landscape, blast radius, and top CVEs',
    widgets: [
      'stat-cards-row-1',
      'risk-landscape',
      'risk-projection',
      'top-vulnerabilities',
      'blast-radius',
    ],
  },
];
