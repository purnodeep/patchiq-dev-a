import type { ComponentType } from 'react';
import type { LucideIcon } from 'lucide-react';

export type WidgetCategory = 'kpi' | 'security' | 'operations' | 'activity';

export type WidgetConfigFieldType = 'select' | 'multi-select' | 'toggle' | 'number';
export type WidgetConfigCategory = 'data' | 'display' | 'behavior';

export interface WidgetConfigField {
  type: WidgetConfigFieldType;
  label: string;
  category: WidgetConfigCategory;
  options?: { label: string; value: string }[];
  default: string | number | boolean | string[];
  min?: number;
  max?: number;
}

export type WidgetConfigSchema = Record<string, WidgetConfigField>;
export type WidgetConfig = Record<string, unknown>;

export type WidgetId =
  | 'stat-cards-row-1'
  | 'stat-cards-row-2'
  | 'risk-landscape'
  | 'deployment-pipeline'
  | 'patch-velocity'
  | 'activity-feed'
  | 'top-vulnerabilities'
  | 'blast-radius'
  | 'risk-projection'
  | 'os-heatmap'
  | 'quick-actions'
  | 'command-palette';

export type Breakpoint = 'lg' | 'md' | 'sm';

export interface WidgetSize {
  w: number;
  h: number;
  x: number;
  y: number;
  minW?: number;
  maxW?: number;
  minH?: number;
  maxH?: number;
}

export interface WidgetRegistryEntry {
  id: WidgetId;
  label: string;
  description: string;
  category: WidgetCategory;
  icon: LucideIcon;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- widgets have heterogeneous prop signatures; config is passed optionally
  component: ComponentType<any>;
  configSchema?: WidgetConfigSchema;
  defaults: Record<Breakpoint, WidgetSize>;
}
