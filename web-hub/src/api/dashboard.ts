import type {
  DashboardStats,
  LicenseBreakdownItem,
  CatalogGrowthItem,
  ClientSummaryItem,
  ActivityItem,
} from '../types/dashboard';

const TENANT_ID = '00000000-0000-0000-0000-000000000001'; // default tenant for M1

async function apiFetch(url: string, options?: RequestInit): Promise<Response> {
  const res = await fetch(url, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': TENANT_ID,
      ...options?.headers,
    },
  });
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || `API error: ${res.status}`);
  }
  return res;
}

export async function getDashboardStats(): Promise<DashboardStats> {
  const res = await apiFetch('/api/v1/dashboard/stats');
  return res.json() as Promise<DashboardStats>;
}

export async function getLicenseBreakdown(): Promise<LicenseBreakdownItem[]> {
  const res = await apiFetch('/api/v1/dashboard/license-breakdown');
  return res.json() as Promise<LicenseBreakdownItem[]>;
}

export async function getCatalogGrowth(days = 90): Promise<CatalogGrowthItem[]> {
  const res = await apiFetch(`/api/v1/dashboard/catalog-growth?days=${days}`);
  return res.json() as Promise<CatalogGrowthItem[]>;
}

export async function getClientSummary(): Promise<ClientSummaryItem[]> {
  const res = await apiFetch('/api/v1/dashboard/clients');
  return res.json() as Promise<ClientSummaryItem[]>;
}

export async function getRecentActivity(): Promise<ActivityItem[]> {
  const res = await apiFetch('/api/v1/dashboard/activity');
  return res.json() as Promise<ActivityItem[]>;
}
