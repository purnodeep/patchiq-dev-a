import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';
import type { components } from '../types';

// Type aliases from generated OpenAPI schema, re-exported for consumers.
export type FrameworkScoreSummary = components['schemas']['FrameworkScoreSummary'];
export type ComplianceSummary = components['schemas']['ComplianceSummary'];
export type FrameworkListItem = components['schemas']['FrameworkListItem'];
export type TenantFrameworkResponse = components['schemas']['TenantFrameworkConfig'];
export type ComplianceScore = components['schemas']['ComplianceScore'];
export type EndpointComplianceScore = components['schemas']['EndpointComplianceScore'];
export type ComplianceFrameworkDetail = components['schemas']['FrameworkDetailResponse'];
export type ComplianceEvaluation = components['schemas']['ComplianceEvaluation'];
export type OverallComplianceScore = components['schemas']['OverallComplianceScore'];
export type ControlResult = components['schemas']['ControlResult'];
export type CategoryBreakdown = components['schemas']['CategoryBreakdown'];
export type OverdueControl = components['schemas']['OverdueControl'];
export type NonCompliantEndpoint = components['schemas']['NonCompliantEndpoint'];
export type TrendPoint = components['schemas']['TrendPoint'];

// --- Query hooks ---

export function useOverallComplianceScore() {
  return useQuery({
    queryKey: ['compliance', 'score'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/compliance/score', {});
      if (error) throw error;
      return data;
    },
    refetchInterval: 60_000,
  });
}

export function useFrameworkControls(
  frameworkId: string,
  options?: { status?: string; search?: string },
) {
  return useQuery({
    queryKey: ['compliance', 'frameworks', frameworkId, 'controls', options],
    queryFn: async () => {
      const { data, error } = await api.GET(
        '/api/v1/compliance/frameworks/{frameworkId}/controls',
        {
          params: {
            path: { frameworkId },
            query: {
              status: options?.status as 'pass' | 'fail' | 'partial' | 'na' | undefined,
              search: options?.search,
            },
          },
        },
      );
      if (error) throw error;
      return data;
    },
    enabled: !!frameworkId,
  });
}

export function useOverdueControls() {
  return useQuery({
    queryKey: ['compliance', 'overdue'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/compliance/overdue', {});
      if (error) throw error;
      return data;
    },
    refetchInterval: 60_000,
  });
}

export function useComplianceTrend(frameworkId: string) {
  return useQuery({
    queryKey: ['compliance', 'frameworks', frameworkId, 'trend'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/compliance/frameworks/{frameworkId}/trend', {
        params: {
          path: { frameworkId },
        },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!frameworkId,
    retry: false,
  });
}

export function useComplianceSummary() {
  return useQuery({
    queryKey: ['compliance', 'summary'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/compliance/summary', {});
      if (error) throw error;
      return data;
    },
    refetchInterval: 60_000,
  });
}

export function useComplianceFrameworks() {
  return useQuery({
    queryKey: ['compliance', 'frameworks'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/compliance/frameworks', {});
      if (error) throw error;
      return data;
    },
  });
}

export function useComplianceFramework(frameworkId: string) {
  return useQuery({
    queryKey: ['compliance', 'frameworks', frameworkId],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/compliance/frameworks/{frameworkId}', {
        params: { path: { frameworkId } },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!frameworkId,
  });
}

export function useEndpointCompliance(endpointId: string) {
  return useQuery({
    queryKey: ['compliance', 'endpoints', endpointId],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/compliance/endpoints/{id}', {
        params: { path: { id: endpointId } },
      });
      if (error) throw error;
      return data;
    },
    enabled: !!endpointId,
  });
}

// --- Mutation hooks ---

export function useEnableFramework() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: { framework_id: string; scoring_method?: string }) => {
      const { data, error } = await api.POST('/api/v1/compliance/frameworks', {
        body: body as components['schemas']['EnableFrameworkRequest'],
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['compliance'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useUpdateFramework() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      ...body
    }: {
      id: string;
      scoring_method?: string;
      enabled?: boolean;
    }) => {
      const { data, error } = await api.PUT('/api/v1/compliance/frameworks/{id}', {
        params: { path: { id } },
        body: body as components['schemas']['UpdateFrameworkRequest'],
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['compliance'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useDisableFramework() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/api/v1/compliance/frameworks/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['compliance'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useTriggerEvaluation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST('/api/v1/compliance/evaluate', {});
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['compliance'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

export function useTriggerFrameworkEvaluation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (frameworkId: string) => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
      const { data, error } = await (api as any).POST(
        '/api/v1/compliance/frameworks/{frameworkId}/evaluate',
        { params: { path: { frameworkId } } },
      );
      if (error) throw error;
      return data as { status: string; frameworks_evaluated: number; total_evaluations: number };
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['compliance'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}

// --- Custom framework types (not yet in generated OpenAPI schema) ---

export interface ConditionValue {
  enabled: boolean;
  value?: number;
}

export interface CheckConfig {
  conditions?: Record<string, ConditionValue>;
  pass_threshold?: number;
  partial_threshold?: number;
}

export interface SLATierInput {
  label: string;
  days: number | null;
  cvss_min: number;
  cvss_max: number;
}

export interface CustomControlInput {
  control_id: string;
  name: string;
  description: string;
  category: string;
  check_type: string;
  remediation_hint: string;
  sla_tiers: SLATierInput[];
  check_config?: CheckConfig;
}

export interface CustomControlResponse {
  id: string;
  framework_id: string;
  control_id: string;
  name: string;
  description: string;
  category: string;
  check_type: string;
  remediation_hint: string;
  sla_tiers: SLATierInput[];
  check_config?: CheckConfig;
  created_at: string;
}

export interface CustomFrameworkResponse {
  id: string;
  name: string;
  version: string;
  description: string;
  scoring_method: string;
  control_count?: number;
  created_at: string;
  updated_at: string;
  controls?: CustomControlResponse[];
}

export interface CreateCustomFrameworkBody {
  name: string;
  version?: string;
  description?: string;
  scoring_method?: string;
  controls?: CustomControlInput[];
}

export interface UpdateCustomFrameworkBody {
  name: string;
  version?: string;
  description?: string;
  scoring_method?: string;
}

// --- Custom framework hooks ---

export function useCustomFrameworks() {
  return useQuery({
    queryKey: ['compliance', 'custom-frameworks'],
    queryFn: async () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
      const { data, error } = await (api as any).GET('/api/v1/compliance/custom-frameworks', {});
      if (error) throw error;
      return (data ?? []) as CustomFrameworkResponse[];
    },
  });
}

export function useCustomFramework(id: string) {
  return useQuery({
    queryKey: ['compliance', 'custom-frameworks', id],
    queryFn: async () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
      const { data, error } = await (api as any).GET('/api/v1/compliance/custom-frameworks/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
      return data as CustomFrameworkResponse;
    },
    enabled: !!id,
  });
}

export function useCreateCustomFramework() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: CreateCustomFrameworkBody) => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
      const { data, error } = await (api as any).POST('/api/v1/compliance/custom-frameworks', {
        body,
      });
      if (error) throw error;
      return data as CustomFrameworkResponse;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['compliance', 'custom-frameworks'] });
    },
  });
}

export function useUpdateCustomFramework() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, ...body }: UpdateCustomFrameworkBody & { id: string }) => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
      const { data, error } = await (api as any).PUT('/api/v1/compliance/custom-frameworks/{id}', {
        params: { path: { id } },
        body,
      });
      if (error) throw error;
      return data as CustomFrameworkResponse;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['compliance', 'custom-frameworks'] });
    },
  });
}

export function useDeleteCustomFramework() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
      const { error } = await (api as any).DELETE('/api/v1/compliance/custom-frameworks/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['compliance', 'custom-frameworks'] });
    },
  });
}

export function useCustomFrameworkControls(frameworkId: string) {
  return useQuery({
    queryKey: ['compliance', 'custom-frameworks', frameworkId, 'controls'],
    queryFn: async () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
      const { data, error } = await (api as any).GET('/api/v1/compliance/custom-frameworks/{id}', {
        params: { path: { id: frameworkId } },
      });
      if (error) throw error;
      return ((data as CustomFrameworkResponse)?.controls ?? []) as CustomControlResponse[];
    },
    enabled: !!frameworkId,
  });
}

export function useUpdateCustomControls() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, controls }: { id: string; controls: CustomControlInput[] }) => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- TODO(#332): remove cast after OpenAPI spec regeneration
      const { data, error } = await (api as any).PUT(
        '/api/v1/compliance/custom-frameworks/{id}/controls',
        {
          params: { path: { id } },
          body: controls,
        },
      );
      if (error) throw error;
      return (data ?? []) as CustomControlResponse[];
    },
    onSuccess: (_data, { id }) => {
      void queryClient.invalidateQueries({ queryKey: ['compliance', 'custom-frameworks', id] });
      void queryClient.invalidateQueries({ queryKey: ['compliance', 'custom-frameworks'] });
    },
  });
}
