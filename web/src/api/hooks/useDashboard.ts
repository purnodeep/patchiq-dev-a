import { useQuery } from '@tanstack/react-query';
import { api } from '../client';

export interface ActiveDeployment {
  id: string;
  name: string;
  status: string;
  progress_pct: number;
}

export interface RunningWorkflow {
  id: string;
  name: string;
  current_stage: string;
}

export interface DashboardSummary {
  total_endpoints: number;
  active_endpoints: number;
  endpoints_degraded: number;
  total_patches: number;
  critical_patches: number;
  patches_high: number;
  patches_medium: number;
  patches_low: number;
  total_cves: number;
  critical_cves: number;
  unpatched_cves: number;
  pending_deployments: number;
  compliance_rate: number;
  framework_count: number;
  active_deployments: ActiveDeployment[];
  overdue_sla_count: number;
  oldest_overdue_age?: string;
  oldest_overdue_patch?: string;
  failed_deployments_count: number;
  failed_trend_7d: number[];
  workflows_running_count: number;
  workflows_running: RunningWorkflow[];
  hub_sync_status: string;
  hub_last_sync_at: string | null;
  hub_url: string;
}

export interface ActivityItem {
  id: string;
  type: string;
  title: string;
  status: string;
  meta: string;
  detail?: {
    deployment_id?: string;
    endpoint_id?: string;
    patch_id?: string;
    progress_pct?: number;
    total?: number;
    completed?: number;
  };
  timestamp: string;
}

export interface BlastRadiusData {
  cve: { id: string; cve_id: string; cvss: number; affected_count: number } | null;
  groups: { name: string; os: string; host_count: number }[];
}

export interface EndpointRisk {
  hostname: string;
  cve_count: number;
  risk_score: number;
}

export function useDashboardSummary() {
  return useQuery({
    queryKey: ['dashboard', 'summary'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/dashboard/summary', {});
      if (error) throw error;
      return data as unknown as DashboardSummary;
    },
    refetchInterval: 30_000,
  });
}

export function useDashboardActivity() {
  return useQuery({
    queryKey: ['dashboard', 'activity'],
    queryFn: async () => {
      // TODO(PIQ-233): remove `as any` cast after regenerating OpenAPI types
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const { data, error } = await (api as any).GET('/api/v1/dashboard/activity', {});
      if (error) throw error;
      return (data as { items: ActivityItem[] }).items;
    },
    refetchInterval: 60_000,
  });
}

export function useBlastRadius(cveId?: string) {
  return useQuery({
    queryKey: ['dashboard', 'blast-radius', cveId],
    queryFn: async () => {
      // TODO(PIQ-233): remove `as any` cast after regenerating OpenAPI types
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const { data, error } = await (api as any).GET('/api/v1/dashboard/blast-radius', {
        params: {
          query: cveId ? { cve_id: cveId } : undefined,
        },
      });
      if (error) throw error;
      return data as BlastRadiusData;
    },
    // Always enabled — backend returns highest unpatched CVE when no cve_id given
  });
}

export interface PatchesTimelineEntry {
  date: string;
  critical: number;
  high: number;
  medium: number;
}

export function usePatchesTimeline() {
  return useQuery({
    queryKey: ['dashboard', 'patches-timeline'],
    queryFn: async (): Promise<PatchesTimelineEntry[]> => {
      const { data, error } = await api.GET('/api/v1/patches', {
        params: {
          query: { limit: 500 },
        },
      });
      if (error) throw error;

      const patches =
        (data as unknown as { data: Array<{ severity: string; created_at: string }> }).data ?? [];

      // Group patches by week over the last 90 days
      const now = new Date();
      const ninetyDaysAgo = new Date(now.getTime() - 90 * 24 * 60 * 60 * 1000);

      // Build a map: weekKey -> { critical, high, medium }
      const weekMap = new Map<
        string,
        { critical: number; high: number; medium: number; weekStart: Date }
      >();

      // Pre-populate the last 13 weeks (≈91 days)
      for (let w = 12; w >= 0; w--) {
        const weekStart = new Date(now);
        weekStart.setDate(now.getDate() - w * 7);
        weekStart.setHours(0, 0, 0, 0);
        const weekLabel = `W${String(13 - w).padStart(2, '0')}`;
        weekMap.set(weekLabel, { critical: 0, high: 0, medium: 0, weekStart });
      }

      // Count patches per week by severity
      patches.forEach((p) => {
        const created = new Date(p.created_at);
        if (created < ninetyDaysAgo) return;

        for (const [, entry] of weekMap.entries()) {
          const weekEnd = new Date(entry.weekStart.getTime() + 7 * 24 * 60 * 60 * 1000);
          if (created >= entry.weekStart && created < weekEnd) {
            if (p.severity === 'critical') entry.critical++;
            else if (p.severity === 'high') entry.high++;
            else if (p.severity === 'medium') entry.medium++;
            break;
          }
        }
      });

      return Array.from(weekMap.entries()).map(([date, entry]) => ({
        date,
        critical: entry.critical,
        high: entry.high,
        medium: entry.medium,
      }));
    },
    refetchInterval: 5 * 60_000,
  });
}

export interface SLADeadlineEntry {
  endpoint_id: string;
  hostname: string;
  severity: string;
  patch_name: string;
  remaining_seconds: number;
}

export function useSLADeadlines() {
  return useQuery({
    queryKey: ['dashboard', 'sla-deadlines'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/dashboard/sla-deadlines', {});
      if (error) throw error;
      // Backend returns `{items: SLADeadlineEntry[]}` per the new dashboard
      // enhancement shape; OpenAPI spec still reflects the old flat array
      // and will be regenerated separately (see dashboard enhancement plan).
      return (data as unknown as { items: SLADeadlineEntry[] }).items;
    },
    refetchInterval: 60_000,
  });
}

// --- Exposure Windows ---

export interface ExposureWindow {
  id: string;
  cve_id: string;
  severity: string;
  cvss: number;
  affected_count: number;
  first_seen: string;
  patched_at: string | null;
}

export function useExposureWindows() {
  return useQuery({
    queryKey: ['dashboard', 'exposure-windows'],
    queryFn: async () => {
      // TODO(PIQ-233): remove `as any` cast after regenerating OpenAPI types
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const { data, error } = await (api as any).GET('/api/v1/dashboard/exposure-windows', {});
      if (error) throw error;
      return (data as { items: ExposureWindow[] }).items;
    },
    refetchInterval: 5 * 60_000,
  });
}

// --- MTTR ---

export interface MTTREntry {
  week: string;
  severity: string;
  avg_hours: number;
}

export function useMTTR() {
  return useQuery({
    queryKey: ['dashboard', 'mttr'],
    queryFn: async () => {
      // TODO(PIQ-233): remove `as any` cast after regenerating OpenAPI types
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const { data, error } = await (api as any).GET('/api/v1/dashboard/mttr', {});
      if (error) throw error;
      return (data as { items: MTTREntry[] }).items;
    },
    refetchInterval: 5 * 60_000,
  });
}

// --- Attack Paths ---

export interface AttackPathNode {
  id: string;
  hostname: string;
  os: string;
  critical_count: number;
  high_count: number;
  is_online: boolean;
}

export interface AttackPathEdge {
  source_id: string;
  target_id: string;
  shared_cve_count: number;
}

export function useAttackPaths() {
  return useQuery({
    queryKey: ['dashboard', 'attack-paths'],
    queryFn: async () => {
      // TODO(PIQ-233): remove `as any` cast after regenerating OpenAPI types
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const { data, error } = await (api as any).GET('/api/v1/dashboard/attack-paths', {});
      if (error) throw error;
      return data as { nodes: AttackPathNode[]; edges: AttackPathEdge[] };
    },
    refetchInterval: 5 * 60_000,
  });
}

// --- Drift ---

export interface DriftEntry {
  id: string;
  hostname: string;
  os: string;
  unpatched_count: number;
  total_cve_count: number;
  drift_score: number;
  last_compliant_at: string | null;
}

export function useDrift() {
  return useQuery({
    queryKey: ['dashboard', 'drift'],
    queryFn: async () => {
      // TODO(PIQ-233): remove `as any` cast after regenerating OpenAPI types
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const { data, error } = await (api as any).GET('/api/v1/dashboard/drift', {});
      if (error) throw error;
      return (data as { items: DriftEntry[] }).items;
    },
    refetchInterval: 5 * 60_000,
  });
}

// --- SLA Forecast ---

export interface SLAForecastEntry {
  id: string;
  hostname: string;
  severity: string;
  sla_window_hours: number;
  remaining_seconds: number;
  oldest_open_since: string;
}

export function useSLAForecast() {
  return useQuery({
    queryKey: ['dashboard', 'sla-forecast'],
    queryFn: async () => {
      // TODO(PIQ-233): remove `as any` cast after regenerating OpenAPI types
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const { data, error } = await (api as any).GET('/api/v1/dashboard/sla-forecast', {});
      if (error) throw error;
      return (data as { items: SLAForecastEntry[] }).items;
    },
    refetchInterval: 60_000,
  });
}

export function useTopEndpointsByRisk() {
  return useQuery({
    queryKey: ['dashboard', 'endpoints-risk'],
    queryFn: async () => {
      // TODO(PIQ-233): remove `as any` cast after regenerating OpenAPI types
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const { data, error } = await (api as any).GET('/api/v1/dashboard/endpoints-risk', {});
      if (error) throw error;
      return data as EndpointRisk[];
    },
    refetchInterval: 60_000,
  });
}
