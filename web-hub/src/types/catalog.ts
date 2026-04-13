export interface CatalogEntry {
  id: string;
  name: string;
  vendor: string;
  os_family: string;
  version: string;
  severity: string;
  release_date: string | null;
  description: string | null;
  created_at: string;
  updated_at: string;
  feed_source_id: string | null;
  feed_source_name: string | null;
  source_url: string;
  installer_type: string;
  cve_count: number;
  synced_count: number;
  total_clients: number;
}

export interface CVEFeed {
  id: string;
  cve_id: string;
  severity: string;
  description: string | null;
  published_at: string | null;
  source: string;
  cvss_v3_score: number | null;
  exploit_known: boolean;
  in_kev: boolean;
}

export interface CatalogSync {
  id: string;
  catalog_id: string;
  client_id: string;
  client_name: string;
  endpoint_count: number;
  status: string;
  synced_at: string | null;
  created_at: string;
}

export interface CatalogListResponse {
  entries: CatalogEntry[];
  total: number;
  limit: number;
  offset: number;
  total_clients: number;
}

export interface CatalogDetailResponse extends CatalogEntry {
  cves: CVEFeed[];
  syncs: CatalogSync[];
  feed_source_display_name: string | null;
  binary_ref: string;
  checksum_sha256: string;
}

export interface CatalogStats {
  total_entries: number;
  new_this_week: number;
  cves_tracked: number;
  synced_entries: number;
  total_for_sync_pct: number;
  by_severity: {
    critical: number;
    high: number;
    medium: number;
    low: number;
  };
}

export interface CreateCatalogRequest {
  name: string;
  vendor: string;
  os_family: string;
  version: string;
  severity: string;
  release_date?: string;
  description?: string;
  cve_ids?: string[];
}
