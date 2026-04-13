import type { Severity } from './shared';

export interface CVEListItem {
  id: string;
  tenant_id: string;
  cve_id: string;
  severity: Severity;
  description: string | null;
  published_at: string | null;
  created_at: string;
  updated_at: string;
  cvss_v3_score: number | null;
  cvss_v3_vector: string | null;
  cisa_kev_due_date: string | null;
  exploit_available: boolean;
  nvd_last_modified: string;
  affected_endpoint_count: number;
  patch_available: boolean;
  patch_count: number;
  attack_vector: string | null;
}

export interface CVEListResponse {
  data: CVEListItem[];
  next_cursor: string | null;
  total_count: number;
}

export interface ExternalReference {
  url: string;
  source: string;
}

export interface AffectedEndpoint {
  id: string;
  hostname: string;
  os_family: string;
  os_version: string;
  ip_address: string | null;
  status: 'affected' | 'patched' | 'mitigated' | 'ignored';
  agent_version: string | null;
  last_seen: string | null;
  group_names: string | null;
}

export interface CVEPatch {
  id: string;
  name: string;
  version: string;
  severity: Severity;
  os_family: string;
  released_at: string;
  endpoints_covered: number;
  endpoints_patched: number;
}

export interface RelatedCVE {
  id: string;
  cve_id: string;
  severity: Severity;
  cvss_v3_score: number | null;
}

export interface CVEDetail extends Omit<
  CVEListItem,
  'affected_endpoint_count' | 'patch_available'
> {
  cwe_id: string | null;
  source: string;
  external_references: ExternalReference[];
  affected_endpoints: {
    count: number;
    items: AffectedEndpoint[];
    has_more: boolean;
  };
  patches: CVEPatch[];
  related_cves: RelatedCVE[];
}

export interface CVESummary {
  total: number;
  by_severity: Record<string, number>;
  kev_count: number;
  exploit_count: number;
}
