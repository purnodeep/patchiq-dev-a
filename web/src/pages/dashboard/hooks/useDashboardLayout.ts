import { useState, useCallback, useRef, useEffect } from 'react';
import type { LayoutItem, Layout, ResponsiveLayouts } from 'react-grid-layout';
import type { WidgetId, Breakpoint, WidgetConfig } from '../types';
import { WIDGET_REGISTRY, DEFAULT_WIDGET_IDS } from '../registry';
import type { PresetId } from '../presets';

const STORAGE_KEY = 'patchiq-dashboard-layout-v15';
const DEBOUNCE_MS = 400;

interface StoredState {
  activeWidgets: WidgetId[];
  layouts: ResponsiveLayouts;
  widgetConfigs?: Record<string, WidgetConfig>;
  presetId?: PresetId;
}

function buildDefaultLayouts(widgetIds: WidgetId[]): ResponsiveLayouts {
  const breakpoints: Breakpoint[] = ['lg', 'md', 'sm'];
  const layouts: Record<string, LayoutItem[]> = {};
  for (const bp of breakpoints) {
    const items: LayoutItem[] = [];
    for (const id of widgetIds) {
      const entry = WIDGET_REGISTRY.get(id);
      if (!entry) continue;
      const d = entry.defaults[bp];
      items.push({
        i: id,
        x: d.x,
        y: d.y,
        w: d.w,
        h: d.h,
        minW: d.minW,
        maxW: d.maxW,
        minH: d.minH,
        maxH: d.maxH,
      });
    }
    layouts[bp] = items;
  }
  return layouts;
}

function loadState(): StoredState | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as StoredState;
    const validIds = parsed.activeWidgets.filter((id) => WIDGET_REGISTRY.has(id));
    if (validIds.length === 0) return null;
    return { ...parsed, activeWidgets: validIds };
  } catch {
    return null;
  }
}

function saveState(state: StoredState): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // localStorage full
  }
}

function findNextY(layout: Layout): number {
  if (layout.length === 0) return 0;
  return layout.reduce((max, l) => Math.max(max, l.y + l.h), 0);
}

export interface UseDashboardLayoutReturn {
  layouts: ResponsiveLayouts;
  activeWidgets: WidgetId[];
  onLayoutChange: (layout: Layout, allLayouts: ResponsiveLayouts) => void;
  addWidget: (id: WidgetId) => void;
  addWidgetAt: (id: WidgetId, x: number, y: number) => void;
  removeWidget: (id: WidgetId) => void;
  resetLayout: () => void;
  isEditing: boolean;
  setIsEditing: (v: boolean) => void;
  widgetConfigs: Record<string, WidgetConfig>;
  updateWidgetConfig: (id: WidgetId, config: WidgetConfig) => void;
  getWidgetConfig: (id: WidgetId) => WidgetConfig;
  presetId: PresetId;
  applyPreset: (presetId: PresetId, widgets: WidgetId[]) => void;
}

export function useDashboardLayout(): UseDashboardLayoutReturn {
  const stored = useRef(loadState());

  const [activeWidgets, setActiveWidgets] = useState<WidgetId[]>(
    () => stored.current?.activeWidgets ?? [...DEFAULT_WIDGET_IDS],
  );

  const [layouts, setLayouts] = useState<ResponsiveLayouts>(
    () => stored.current?.layouts ?? buildDefaultLayouts(DEFAULT_WIDGET_IDS),
  );

  const [isEditing, setIsEditing] = useState(false);

  // Guard to skip onLayoutChange immediately after addWidget/addWidgetAt
  // to prevent react-grid-layout from overwriting the intended position
  const dropGuard = useRef(false);

  const [widgetConfigs, setWidgetConfigs] = useState<Record<string, WidgetConfig>>(
    () => stored.current?.widgetConfigs ?? {},
  );

  const [presetId, setPresetId] = useState<PresetId>(
    () => stored.current?.presetId ?? 'custom',
  );

  const saveTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const persist = useCallback(
    (widgets: WidgetId[], lays: ResponsiveLayouts, configs?: Record<string, WidgetConfig>, preset?: PresetId) => {
      clearTimeout(saveTimer.current);
      saveTimer.current = setTimeout(
        () =>
          saveState({
            activeWidgets: widgets,
            layouts: lays,
            widgetConfigs: configs,
            presetId: preset,
          }),
        DEBOUNCE_MS,
      );
    },
    [],
  );

  useEffect(() => () => clearTimeout(saveTimer.current), []);

  const onLayoutChange = useCallback(
    (_currentLayout: Layout, allLayouts: ResponsiveLayouts) => {
      if (dropGuard.current) {
        dropGuard.current = false;
        return;
      }
      const merged: Record<string, LayoutItem[]> = {};
      for (const [bp, items] of Object.entries(allLayouts)) {
        if (!items) continue;
        merged[bp] = items
          .map((l: LayoutItem) => {
            const entry = WIDGET_REGISTRY.get(l.i as WidgetId);
            if (!entry) return l;
            const d = entry.defaults[bp as Breakpoint];
            if (!d) return l;
            return { ...l, minW: d.minW, maxW: d.maxW, minH: d.minH, maxH: d.maxH };
          })
          .filter((l: LayoutItem) => activeWidgets.includes(l.i as WidgetId));
      }
      setLayouts(merged);
      persist(activeWidgets, merged, widgetConfigs, presetId);
    },
    [activeWidgets, persist, widgetConfigs, presetId],
  );

  const addWidget = useCallback(
    (id: WidgetId) => {
      if (activeWidgets.includes(id)) return;
      const entry = WIDGET_REGISTRY.get(id);
      if (!entry) return;

      dropGuard.current = true;
      const newWidgets = [...activeWidgets, id];
      const newLayouts: Record<string, LayoutItem[]> = {};
      const breakpoints: Breakpoint[] = ['lg', 'md', 'sm'];

      for (const bp of breakpoints) {
        const d = entry.defaults[bp];
        const existing = (layouts[bp] ?? []) as LayoutItem[];
        const nextY = findNextY(existing);
        newLayouts[bp] = [
          ...existing,
          {
            i: id,
            x: 0,
            y: nextY,
            w: d.w,
            h: d.h,
            minW: d.minW,
            maxW: d.maxW,
            minH: d.minH,
            maxH: d.maxH,
          },
        ];
      }

      setActiveWidgets(newWidgets);
      setLayouts(newLayouts);
      setPresetId('custom');
      persist(newWidgets, newLayouts, widgetConfigs, 'custom');
    },
    [activeWidgets, layouts, persist, widgetConfigs],
  );

  const addWidgetAt = useCallback(
    (id: WidgetId, x: number, y: number) => {
      if (activeWidgets.includes(id)) return;
      const entry = WIDGET_REGISTRY.get(id);
      if (!entry) return;

      dropGuard.current = true;
      const newWidgets = [...activeWidgets, id];
      const newLayouts: Record<string, LayoutItem[]> = {};
      const breakpoints: Breakpoint[] = ['lg', 'md', 'sm'];

      for (const bp of breakpoints) {
        const d = entry.defaults[bp];
        const existing = (layouts[bp] ?? []) as LayoutItem[];
        // Use the drop position for lg (current breakpoint), findNextY for others
        const posX = bp === 'lg' ? x : 0;
        const posY = bp === 'lg' ? y : findNextY(existing);
        newLayouts[bp] = [
          ...existing,
          {
            i: id,
            x: posX,
            y: posY,
            w: d.w,
            h: d.h,
            minW: d.minW,
            maxW: d.maxW,
            minH: d.minH,
            maxH: d.maxH,
          },
        ];
      }

      setActiveWidgets(newWidgets);
      setLayouts(newLayouts);
      setPresetId('custom');
      persist(newWidgets, newLayouts, widgetConfigs, 'custom');
    },
    [activeWidgets, layouts, persist, widgetConfigs],
  );

  const removeWidget = useCallback(
    (id: WidgetId) => {
      const newWidgets = activeWidgets.filter((w) => w !== id);
      const newLayouts: Record<string, LayoutItem[]> = {};
      for (const [bp, items] of Object.entries(layouts)) {
        if (!items) continue;
        newLayouts[bp] = (items as LayoutItem[]).filter((l) => l.i !== id);
      }
      setActiveWidgets(newWidgets);
      setLayouts(newLayouts);
      setPresetId('custom');
      persist(newWidgets, newLayouts, widgetConfigs, 'custom');
    },
    [activeWidgets, layouts, persist, widgetConfigs],
  );

  const resetLayout = useCallback(() => {
    const defaults = [...DEFAULT_WIDGET_IDS];
    const defaultLayouts = buildDefaultLayouts(defaults);
    setActiveWidgets(defaults);
    setLayouts(defaultLayouts);
    setWidgetConfigs({});
    setPresetId('custom');
    localStorage.removeItem(STORAGE_KEY);
    persist(defaults, defaultLayouts, {}, 'custom');
  }, [persist]);

  const updateWidgetConfig = useCallback(
    (id: WidgetId, config: WidgetConfig) => {
      const next = { ...widgetConfigs, [id]: config };
      setWidgetConfigs(next);
      persist(activeWidgets, layouts, next, presetId);
    },
    [widgetConfigs, activeWidgets, layouts, presetId, persist],
  );

  const getWidgetConfig = useCallback(
    (id: WidgetId): WidgetConfig => {
      const entry = WIDGET_REGISTRY.get(id);
      const defaults: WidgetConfig = {};
      if (entry?.configSchema) {
        for (const [key, field] of Object.entries(entry.configSchema)) {
          defaults[key] = field.default;
        }
      }
      return { ...defaults, ...(widgetConfigs[id] ?? {}) };
    },
    [widgetConfigs],
  );

  const applyPreset = useCallback(
    (id: PresetId, widgets: WidgetId[]) => {
      const newLayouts = buildDefaultLayouts(widgets);
      setActiveWidgets(widgets);
      setLayouts(newLayouts);
      setPresetId(id);
      persist(widgets, newLayouts, widgetConfigs, id);
    },
    [persist, widgetConfigs],
  );

  return {
    layouts,
    activeWidgets,
    onLayoutChange,
    addWidget,
    addWidgetAt,
    removeWidget,
    resetLayout,
    isEditing,
    setIsEditing,
    widgetConfigs,
    updateWidgetConfig,
    getWidgetConfig,
    presetId,
    applyPreset,
  };
}
