import { useState, useCallback } from 'react';
import type { LayoutItem } from 'react-grid-layout/legacy';
import { DEFAULT_LAYOUT, LAYOUT_STORAGE_KEY, LAYOUT_VERSION } from '@/data/dashboard-layout';

const VERSION_KEY = 'patchiq:dashboard-layout-version';

function loadLayout(): LayoutItem[] {
  try {
    // If the stored version doesn't match current, discard and use defaults.
    const storedVersion = localStorage.getItem(VERSION_KEY);
    if (storedVersion !== String(LAYOUT_VERSION)) {
      localStorage.removeItem(LAYOUT_STORAGE_KEY);
      localStorage.setItem(VERSION_KEY, String(LAYOUT_VERSION));
      return DEFAULT_LAYOUT;
    }
    const raw = localStorage.getItem(LAYOUT_STORAGE_KEY);
    if (!raw) return DEFAULT_LAYOUT;
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      console.warn('[useDashboardLayout] Stored layout is not an array, using default');
      return DEFAULT_LAYOUT;
    }
    return parsed as LayoutItem[];
  } catch {
    console.warn('[useDashboardLayout] Failed to parse layout from localStorage, using default');
    return DEFAULT_LAYOUT;
  }
}

function saveLayout(layout: LayoutItem[]): void {
  try {
    localStorage.setItem(LAYOUT_STORAGE_KEY, JSON.stringify(layout));
  } catch {
    console.warn('[useDashboardLayout] Failed to save layout to localStorage');
  }
}

function layoutsEqual(a: LayoutItem[], b: LayoutItem[]): boolean {
  const sort = (arr: LayoutItem[]) => [...arr].sort((x, y) => x.i.localeCompare(y.i));
  return JSON.stringify(sort(a)) === JSON.stringify(sort(b));
}

export interface UseDashboardLayout {
  layout: LayoutItem[];
  isEditMode: boolean;
  alertDismissed: boolean;
  toggleEditMode: () => void;
  onLayoutChange: (next: LayoutItem[]) => void;
  resetLayout: () => void;
  dismissAlert: () => void;
}

export function useDashboardLayout(): UseDashboardLayout {
  const [layout, setLayout] = useState<LayoutItem[]>(loadLayout);
  const [isEditMode, setIsEditMode] = useState(false);
  const [alertDismissed, setAlertDismissed] = useState(false);

  const toggleEditMode = useCallback(() => setIsEditMode((prev) => !prev), []);

  const onLayoutChange = useCallback((next: LayoutItem[]) => {
    // rgl fires onLayoutChange on mount — skip save if layout is unchanged
    // rgl never sees the alert item (filtered in DashboardGrid), so exclude it
    // from both sides before comparing to avoid a spurious save on mount
    setLayout((current) => {
      const currentWithoutAlert = current.filter((item) => item.i !== 'alert');
      if (layoutsEqual(currentWithoutAlert, next)) return current;
      saveLayout(next);
      return next;
    });
  }, []);

  const resetLayout = useCallback(() => {
    localStorage.removeItem(LAYOUT_STORAGE_KEY);
    localStorage.setItem(VERSION_KEY, String(LAYOUT_VERSION));
    setLayout(DEFAULT_LAYOUT);
  }, []);

  const dismissAlert = useCallback(() => setAlertDismissed(true), []);

  // Filter the alert item out of the layout when dismissed to prevent a blank gap
  const visibleLayout = alertDismissed ? layout.filter((item) => item.i !== 'alert') : layout;

  return {
    layout: visibleLayout,
    isEditMode,
    alertDismissed,
    toggleEditMode,
    onLayoutChange,
    resetLayout,
    dismissAlert,
  };
}
