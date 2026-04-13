import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';

// Types (inline — no generated types yet)
export interface ReportRecord {
  id: string;
  tenant_id: string;
  report_type: 'endpoints' | 'patches' | 'cves' | 'deployments' | 'compliance' | 'executive';
  format: 'pdf' | 'csv' | 'xlsx';
  status: 'pending' | 'generating' | 'completed' | 'failed';
  name: string;
  filters: Record<string, string>;
  file_path?: string;
  file_size_bytes?: number;
  checksum_sha256?: string;
  row_count?: number;
  error_message?: string;
  created_by: string;
  created_at: string;
  completed_at?: string;
  expires_at: string;
}

export interface ReportCounts {
  total: number;
  completed: number;
  generating: number;
  failed: number;
  today: number;
}

interface ReportsResponse {
  data: ReportRecord[];
  total_count: number;
  next_cursor?: string;
}

interface ReportParams {
  cursor?: string;
  limit?: number;
  status?: string;
  report_type?: string;
  format?: string;
  search?: string;
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: 'include',
    ...init,
  });
  if (res.status === 401 && !window.location.pathname.startsWith('/login')) {
    window.location.href = '/login';
  }
  if (res.status >= 500) {
    throw new Error(`Server error: ${res.status} ${res.statusText}`);
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    const msg = (body as { message?: string }).message || `Request failed: ${res.status}`;
    throw new Error(msg);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export function useReports(params?: ReportParams) {
  return useQuery({
    queryKey: ['reports', params],
    queryFn: async () => {
      const searchParams = new URLSearchParams();
      if (params?.cursor) searchParams.set('cursor', params.cursor);
      if (params?.limit) searchParams.set('limit', String(params.limit));
      if (params?.status) searchParams.set('status', params.status);
      if (params?.report_type) searchParams.set('report_type', params.report_type);
      if (params?.format) searchParams.set('format', params.format);
      if (params?.search) searchParams.set('search', params.search);
      const qs = searchParams.toString();
      return apiFetch<ReportsResponse>(`/api/v1/reports${qs ? `?${qs}` : ''}`);
    },
    refetchInterval: (query) => {
      const reports = query.state.data?.data;
      if (reports?.some((r) => r.status === 'generating' || r.status === 'pending')) {
        return 5_000;
      }
      return 30_000;
    },
  });
}

export function useReportCounts() {
  return useQuery({
    queryKey: ['reports', 'counts'],
    queryFn: () => apiFetch<ReportCounts>('/api/v1/reports/counts'),
    refetchInterval: 30_000,
  });
}

export function useGenerateReport() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: {
      report_type: string;
      format: string;
      filters?: Record<string, string>;
    }) => {
      return apiFetch<ReportRecord>('/api/v1/reports/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['reports'] });
    },
  });
}

export function useDeleteReport() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await apiFetch<void>(`/api/v1/reports/${id}`, { method: 'DELETE' });
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['reports'] });
    },
  });
}

export function useDownloadReport() {
  return useCallback(async (id: string, filename: string) => {
    const res = await fetch(`/api/v1/reports/${id}/download`, {
      credentials: 'include',
    });
    if (!res.ok) {
      throw new Error(`Download failed: ${res.status}`);
    }
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }, []);
}
