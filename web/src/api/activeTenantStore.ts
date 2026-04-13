// Module-level store for the currently active tenant ID. Used by the API
// client middleware to inject X-Tenant-ID on every request, and by the
// tenant switcher UI to persist the user's selection.
//
// This intentionally avoids adding a dependency on Zustand or similar — one
// value with subscribe/get/set is enough and we don't want state libraries
// creeping into every module.

type Listener = () => void;

const STORAGE_KEY = 'patchiq.activeTenantId';

let activeTenantId: string | null = null;
const listeners = new Set<Listener>();

function loadFromStorage(): string | null {
  try {
    return window.localStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
}

// Initialize from localStorage on module load so page refreshes preserve
// the MSP operator's last-selected tenant.
if (typeof window !== 'undefined') {
  activeTenantId = loadFromStorage();
}

export function getActiveTenantId(): string | null {
  return activeTenantId;
}

export function setActiveTenantId(id: string | null): void {
  activeTenantId = id;
  try {
    if (id) {
      window.localStorage.setItem(STORAGE_KEY, id);
    } else {
      window.localStorage.removeItem(STORAGE_KEY);
    }
  } catch {
    // localStorage disabled; in-memory value remains authoritative.
  }
  listeners.forEach((l) => l());
}

export function subscribeActiveTenant(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}
