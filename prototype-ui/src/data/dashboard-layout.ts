import type { LayoutItem } from 'react-grid-layout/legacy';

export const LAYOUT_STORAGE_KEY = 'patchiq:dashboard-layout';
/** Bump this whenever DEFAULT_LAYOUT changes shape (new widgets, resized items). */
export const LAYOUT_VERSION = 5;

/**
 * Default 12-column grid layout. Row height = 80px, margin = 12px.
 * Stat cards: h=2 → 172px per card (80*2 + 12 = 172px).
 * Chart widgets: h=4 → 356px (80*4 + 36 = 356px).
 * AlertBanner is rendered above the rgl grid, retained here for reset compatibility.
 */
export const DEFAULT_LAYOUT: LayoutItem[] = [
  // AlertBanner — full width, always locked
  { i: 'alert', x: 0, y: 0, w: 12, h: 1, minW: 12, minH: 1, static: true },

  // Stat cards — row 1 (h=2 each)
  { i: 'stat-0', x: 0, y: 1, w: 3, h: 2, minW: 2, minH: 2 },
  { i: 'stat-1', x: 3, y: 1, w: 3, h: 2, minW: 2, minH: 2 },
  { i: 'stat-2', x: 6, y: 1, w: 3, h: 2, minW: 2, minH: 2 },
  { i: 'stat-3', x: 9, y: 1, w: 3, h: 2, minW: 2, minH: 2 },

  // Stat cards — row 2 (y=3, after row 1 which ends at y=3)
  { i: 'stat-4', x: 0, y: 3, w: 3, h: 2, minW: 2, minH: 2 },
  { i: 'stat-5', x: 3, y: 3, w: 3, h: 2, minW: 2, minH: 2 },
  { i: 'stat-6', x: 6, y: 3, w: 3, h: 2, minW: 2, minH: 2 },
  { i: 'stat-7', x: 9, y: 3, w: 3, h: 2, minW: 2, minH: 2 },

  // Hero widgets (y=5, after stat rows end at y=5)
  { i: 'blast-radius', x: 0, y: 5, w: 6, h: 4, minW: 4, minH: 3 },
  { i: 'risk-delta', x: 6, y: 5, w: 6, h: 4, minW: 4, minH: 3 },

  // Operational row
  { i: 'sla-waterfall', x: 0, y: 9, w: 4, h: 4, minW: 3, minH: 3 },
  { i: 'deployment-timeline', x: 4, y: 9, w: 4, h: 4, minW: 3, minH: 3 },
  { i: 'vuln-heatmap', x: 8, y: 9, w: 4, h: 4, minW: 3, minH: 3 },

  // Secondary row
  { i: 'compliance-rings', x: 0, y: 13, w: 4, h: 4, minW: 3, minH: 3 },
  { i: 'agent-rollout', x: 4, y: 13, w: 4, h: 4, minW: 3, minH: 3 },
  { i: 'sla-countdown', x: 8, y: 13, w: 4, h: 4, minW: 3, minH: 3 },

  // Bottom row
  { i: 'workflow-pipeline', x: 0, y: 17, w: 6, h: 4, minW: 4, minH: 3 },
  { i: 'patches-horizon', x: 6, y: 17, w: 6, h: 4, minW: 4, minH: 3 },

  // Analytics row
  { i: 'top-cves', x: 0, y: 21, w: 6, h: 5, minW: 4, minH: 3 },
  { i: 'mttp', x: 6, y: 21, w: 3, h: 5, minW: 3, minH: 3 },
  { i: 'patch-success-rate', x: 9, y: 21, w: 3, h: 5, minW: 3, minH: 3 },

  // Insights row
  { i: 'cve-age', x: 0, y: 26, w: 4, h: 4, minW: 3, minH: 3 },
  { i: 'patch-failure-reasons', x: 4, y: 26, w: 4, h: 4, minW: 3, minH: 3 },
  { i: 'endpoint-coverage', x: 8, y: 26, w: 4, h: 4, minW: 3, minH: 3 },

  // Calendar row
  { i: 'upcoming-sla', x: 0, y: 30, w: 12, h: 4, minW: 6, minH: 3 },
];
