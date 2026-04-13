import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { ChevronDown, Check } from 'lucide-react';

import { useAuth } from '../auth/AuthContext';
import { getActiveTenantId, setActiveTenantId, subscribeActiveTenant } from '../../api/activeTenantStore';

// TenantSwitcher lets an MSP operator with multiple accessible tenants pick
// the active tenant for their session. On change:
//   1. Updates the activeTenantStore (which flips X-Tenant-ID on new requests).
//   2. Clears the TanStack Query cache so stale data from the previous
//      tenant does not leak into the new tenant's views.
//
// Hidden when accessible_tenants.length <= 1 (normal direct customer).
export function TenantSwitcher() {
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [activeId, setActiveId] = useState<string | null>(getActiveTenantId());
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    return subscribeActiveTenant(() => setActiveId(getActiveTenantId()));
  }, []);

  // Click-outside + Escape handling. The previous implementation used
  // onBlur on a non-focusable <div>, which fired unpredictably as focus
  // moved between child <button> elements and often left the dropdown
  // open after an outside click. Wiring document-level listeners only
  // while open === true is the correct primitive here.
  useEffect(() => {
    if (!open) {
      return;
    }
    const handleMouseDown = (event: MouseEvent) => {
      const target = event.target as Node | null;
      if (containerRef.current && target && !containerRef.current.contains(target)) {
        setOpen(false);
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handleMouseDown);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handleMouseDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [open]);

  const tenants = user.accessible_tenants ?? [];
  if (tenants.length <= 1) {
    return null;
  }

  const current = tenants.find((t) => t.id === activeId) ?? tenants[0];

  const handleSelect = (tenantId: string) => {
    if (tenantId === activeId) {
      setOpen(false);
      return;
    }
    setActiveTenantId(tenantId);
    // Clear every cached query — the next render refetches under the new
    // tenant context. This is the simplest correct behavior; a more
    // targeted invalidation would need a tenant-aware query key strategy.
    queryClient.clear();
    setOpen(false);
  };

  return (
    <div ref={containerRef} className="relative inline-block text-left">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="inline-flex items-center gap-2 rounded-md border border-border bg-background px-3 py-1.5 text-sm font-medium hover:bg-muted"
      >
        <span className="text-muted-foreground text-xs">Tenant:</span>
        <span className="max-w-[160px] truncate">{current.name}</span>
        <ChevronDown className="h-4 w-4" />
      </button>
      {open && (
        <div
          tabIndex={-1}
          role="menu"
          className="absolute right-0 z-50 mt-1 w-64 rounded-md border border-border bg-popover shadow-lg"
        >
          <ul className="max-h-80 overflow-y-auto py-1">
            {tenants.map((t) => (
              <li key={t.id}>
                <button
                  type="button"
                  onClick={() => handleSelect(t.id)}
                  className="flex w-full items-center justify-between px-3 py-2 text-left text-sm hover:bg-muted"
                >
                  <div className="flex flex-col">
                    <span className="font-medium">{t.name}</span>
                    <span className="text-xs text-muted-foreground">{t.slug}</span>
                  </div>
                  {t.id === current.id && <Check className="h-4 w-4" />}
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
