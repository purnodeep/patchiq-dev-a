import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';
import { getActiveTenantId } from '../activeTenantStore';
import type { Severity } from '../../types/shared';
import type { HardwareInfo, SoftwareSummary } from '../../types/hardware';

export interface NetworkInterface {
  id: string;
  name: string;
  ip_address?: string | null;
  mac_address?: string | null;
  status: string;
}

/** Extended endpoint detail returned by GET /endpoints/{id}. */
export interface EndpointDetail {
  id: string;
  tenant_id: string;
  hostname: string;
  os_family: string;
  os_version: string;
  agent_version: string | null;
  status: 'pending' | 'online' | 'offline' | 'stale';
  last_seen: string | null;
  created_at: string;
  updated_at: string;
  package_count: number;
  last_scan: string | null;
  vulnerable_cve_count: number;
  cve_count: number;
  pending_patches_count: number;
  critical_patch_count: number;
  high_patch_count: number;
  medium_patch_count: number;
  // hardware
  cpu_model?: string | null;
  cpu_cores?: number | null;
  cpu_usage_percent?: number | null;
  memory_total_mb?: number | null;
  memory_used_mb?: number | null;
  disk_total_gb?: number | null;
  disk_used_gb?: number | null;
  gpu_model?: string | null;
  uptime_seconds?: number | null;
  // agent
  ip_address?: string | null;
  arch?: string | null;
  kernel_version?: string | null;
  enrolled_at?: string | null;
  last_heartbeat?: string | null;
  cert_expiry?: string | null;
  // Deep hardware & software JSONB
  hardware_details?: HardwareInfo | null;
  software_summary?: SoftwareSummary | null;
  // tags
  tags?: { id: string; key: string; value: string }[];
  // network
  network_interfaces?: NetworkInterface[];
}

/** A package row from the endpoint packages API. */
export interface EndpointPackage {
  id: string;
  package_name: string;
  version: string;
  arch: string | null;
  source: string | null;
  release: string | null;
}

/** A patch summary relevant to an endpoint (tenant-scoped). */
export interface EndpointPatch {
  id: string;
  name: string;
  version: string;
  severity: Severity;
  os_family: string;
  status: string;
  cve_count: number;
  created_at: string;
}

/** A deployment target that involves a specific endpoint. */
export interface EndpointDeploymentTarget {
  id: string;
  deployment_id: string;
  endpoint_id: string;
  patch_id: string;
  status: string;
  started_at?: string | null;
  completed_at?: string | null;
  duration_seconds?: number | null;
  error_message?: string | null;
}

/** Endpoint row returned by GET /api/v1/endpoints. Includes enriched fields
 *  (cve_count, pending_patches_count, compliance_pct, group_name, hardware)
 *  computed by the server-side query. */
export interface Endpoint {
  id: string;
  tenant_id: string;
  hostname: string;
  os_family: string;
  os_version: string;
  agent_version?: string | null;
  status: 'pending' | 'online' | 'offline' | 'stale';
  last_seen?: string | null;
  created_at: string;
  updated_at: string;
  // Enriched fields
  cve_count?: number;
  pending_patches_count?: number;
  compliance_pct?: number | null;
  tags?: { id: string; key: string; value: string }[];
  ip_address?: string | null;
  arch?: string | null;
  kernel_version?: string | null;
  // Hardware snapshot (for expandable row)
  cpu_cores?: number | null;
  cpu_usage_percent?: number | null;
  memory_total_mb?: number | null;
  memory_used_mb?: number | null;
  disk_total_gb?: number | null;
  disk_used_gb?: number | null;
  // Severity-split CVE counts
  critical_cve_count?: number;
  high_cve_count?: number;
  medium_cve_count?: number;
  // Severity-split patch counts
  critical_patch_count?: number;
  high_patch_count?: number;
  medium_patch_count?: number;
}

export interface UseEndpointsParams {
  cursor?: string;
  limit?: number;
  status?: string;
  search?: string;
  tag_id?: string;
  os_family?: string;
  page_size?: number;
  cursor_created_at?: string;
  cursor_id?: string;
}

export function useEndpoints(params?: UseEndpointsParams) {
  return useQuery({
    queryKey: ['endpoints', params],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/endpoints', {
        params: { query: params },
      });
      if (error) throw error;
      // Cast needed: OpenAPI generated types don't include enriched CTE fields
      // (cve_count, patch counts, compliance_pct, group_name, hardware).
      // Regenerate types once the OpenAPI spec is updated for the enriched list query.
      return data as unknown as {
        data: Endpoint[];
        next_cursor: string | null;
        total_count: number;
      };
    },
    refetchInterval: 30_000,
  });
}

export function useEndpoint(id: string) {
  return useQuery({
    queryKey: ['endpoints', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/endpoints/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
      // Cast needed: OpenAPI types don't include hardware/network enrichment fields yet.
      return data as unknown as EndpointDetail;
    },
    enabled: !!id,
    refetchInterval: 30_000,
  });
}

/** Fetch patches pending for a specific endpoint. */
export function useEndpointPatches(endpointId: string | undefined) {
  return useQuery({
    queryKey: ['endpoints', endpointId, 'patches'],
    queryFn: async () => {
      const res = await fetch(`/api/v1/endpoints/${endpointId}/patches`, {
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`Failed to fetch endpoint patches: ${res.status}`);
      return res.json() as Promise<{ data: EndpointPatch[]; total_count: number }>;
    },
    enabled: !!endpointId,
    refetchInterval: 30_000,
  });
}

/** Fetch deployments for a specific endpoint. */
export function useEndpointDeployments(endpointId?: string) {
  return useQuery({
    queryKey: ['endpoints', endpointId, 'deployments'],
    queryFn: async () => {
      const res = await fetch(`/api/v1/endpoints/${endpointId}/deployments`, {
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`deployments failed: ${res.status}`);
      return res.json() as Promise<{
        data: EndpointDeploymentTarget[];
        total_count: number;
      }>;
    },
    enabled: !!endpointId,
  });
}

/** A CVE affecting an endpoint (from GET /endpoints/{id}/cves). */
export interface EndpointCVE {
  id: string;
  cve_id: string;
  cve_identifier: string;
  cve_severity: string;
  cvss_v3_score: number | null;
  risk_score: number | null;
  status: string;
  detected_at: string;
}

/** Fetch CVEs affecting a specific endpoint. */
export function useEndpointCVEs(endpointId: string) {
  return useQuery({
    queryKey: ['endpoints', endpointId, 'cves'],
    queryFn: async () => {
      const res = await fetch(`/api/v1/endpoints/${endpointId}/cves`, {
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`Failed to fetch endpoint CVEs: ${res.status}`);
      return res.json() as Promise<{ data: EndpointCVE[]; total_count: number }>;
    },
    enabled: !!endpointId,
  });
}

export function useScanCVEs() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (endpointId: string) => {
      const res = await fetch(`/api/v1/endpoints/${endpointId}/scan-cves`, {
        method: 'POST',
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`Failed to trigger CVE scan: ${res.status}`);
      return res.json() as Promise<{ status: string }>;
    },
    onSuccess: (_data, endpointId) => {
      void queryClient.invalidateQueries({ queryKey: ['endpoints', endpointId, 'cves'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
      void queryClient.invalidateQueries({ queryKey: ['cves'] });
    },
  });
}

/** Fetch packages installed on a specific endpoint. */
export function useEndpointPackages(endpointId: string) {
  return useQuery({
    queryKey: ['endpoints', endpointId, 'packages'],
    queryFn: async () => {
      const res = await fetch(`/api/v1/endpoints/${endpointId}/packages`, {
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`Failed to fetch endpoint packages: ${res.status}`);
      return res.json() as Promise<{ data: EndpointPackage[]; total_count: number }>;
    },
    enabled: !!endpointId,
    refetchInterval: 30_000,
  });
}

export function useTriggerScan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string): Promise<{ status: string; command_id: string }> => {
      // Raw fetch (not api.POST) because the OpenAPI spec for this endpoint
      // predates the command_id response field. Tenant header is injected
      // manually to mirror the middleware the openapi-fetch client applies.
      const headers: Record<string, string> = {};
      const tenantId = getActiveTenantId();
      if (tenantId) headers['X-Tenant-ID'] = tenantId;
      const res = await fetch(`/api/v1/endpoints/${id}/scan`, {
        method: 'POST',
        credentials: 'include',
        headers,
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || `Failed to trigger scan: ${res.status}`);
      }
      return res.json();
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

/** Registration token response from POST /api/v1/registrations. */
export interface Registration {
  id: string;
  tenant_id: string;
  endpoint_id: string | null;
  registration_token: string;
  status: string;
  registered_at: string | null;
  created_at: string;
}

/** Create a new registration token for agent enrollment. */
export function useCreateRegistration() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const res = await fetch('/api/v1/registrations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`Failed to create registration: ${res.status}`);
      return res.json() as Promise<Registration>;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['registrations'] });
    },
  });
}

export function useDecommissionEndpoint() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/api/v1/endpoints/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useDeployCritical() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      endpointId,
      patchIds,
      name,
    }: {
      endpointId: string;
      patchIds: string[];
      name?: string;
    }) => {
      const res = await fetch(`/api/v1/endpoints/${endpointId}/deploy-critical`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ patch_ids: patchIds, name }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error((body as { message?: string }).message ?? `Deploy failed: ${res.status}`);
      }
      return res.json() as Promise<{
        id: string;
        status: string;
        total_targets: number;
        name: string;
      }>;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['deployments'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}
