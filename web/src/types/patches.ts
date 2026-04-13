import type { Severity } from './shared';

export interface PatchCVE {
  id: string;
  cve_id: string;
  cvss_v3_score: string | null;
  severity: Severity;
  published_at: string | null;
  exploit_available: boolean;
  cisa_kev: boolean;
  cvss_v3_vector: string | null;
  description: string | null;
  attack_vector: string | null;
}

export interface PatchAffectedEndpoint {
  id: string;
  hostname: string;
  os_family: string;
  agent_version: string | null;
  status: string;
  patch_status: 'deployed' | 'pending' | 'failed';
  last_deployed_at: string | null;
}

export interface DeploymentHistoryItem {
  id: string;
  status: 'success' | 'failed' | 'running' | 'pending' | 'partial';
  triggered_by: string;
  started_at: string | null;
  completed_at: string | null;
  total_targets: number;
  success_count: number;
  failed_count: number;
}

export interface PatchListItem {
  id: string;
  tenant_id?: string;
  name: string;
  version: string;
  severity: Severity;
  os_family: string;
  status: 'available' | 'superseded' | 'recalled';
  created_at: string;
  released_at?: string;
  updated_at?: string;
  os_distribution: string | null;
  package_url?: string | null;
  checksum_sha256?: string | null;
  source_repo?: string | null;
  description?: string | null;
  cve_count: number;
  highest_cvss_score: number;
  affected_endpoint_count: number;
  remediation_pct: number;
  endpoints_deployed_count: number;
}

export interface PatchListResponse {
  data: PatchListItem[];
  next_cursor: string | null;
  total_count: number;
}

export interface PatchDetail {
  id: string;
  tenant_id?: string;
  name: string;
  version: string;
  severity: Severity;
  os_family: string;
  os_distribution: string | null;
  status: 'available' | 'superseded' | 'recalled';
  package_url: string | null;
  checksum_sha256: string | null;
  source_repo: string | null;
  description: string | null;
  created_at?: string;
  updated_at?: string;
  released_at: string;
  file_size: number | null;
  highest_cvss_score: number;
  avg_install_time_ms: number | null;
  cves: PatchCVE[];
  remediation: {
    endpoints_affected: number;
    endpoints_patched: number;
    endpoints_pending: number;
    endpoints_failed: number;
  };
  affected_endpoints: {
    total: number;
    /** @deprecated use total */
    count?: number;
    items: PatchAffectedEndpoint[];
  };
  deployment_history: DeploymentHistoryItem[];
}
