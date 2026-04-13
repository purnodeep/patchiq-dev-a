export interface DashboardStats {
  total_catalog_entries: number;
  active_feeds: number;
  connected_clients: number;
  pending_clients: number;
  active_licenses: number;
}

export interface LicenseBreakdownItem {
  tier: string;
  status: 'active' | 'expiring' | 'expired' | 'revoked';
  count: number;
  total_endpoints: number;
}

export interface CatalogGrowthItem {
  day: string;
  entries_added: number;
}

export interface ClientSummaryItem {
  id: string;
  hostname: string;
  status: string;
  endpoint_count: number;
  last_sync_at: string | null;
  version: string | null;
  os: string | null;
}

export interface ActivityItem {
  id: string;
  type: string;
  actor_id: string;
  actor_type: string;
  resource: string;
  resource_id: string;
  action: string;
  payload: Record<string, unknown>;
  metadata: Record<string, unknown>;
  timestamp: string;
}
