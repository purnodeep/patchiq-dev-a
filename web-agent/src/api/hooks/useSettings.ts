import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';

export function useSettings() {
  return useQuery({
    queryKey: ['settings'],
    queryFn: async () => {
      const { data, error } = await (api as any).GET('/api/v1/settings');
      if (error) throw error;
      return data;
    },
  });
}

export interface SettingsUpdateBody {
  scan_interval?: string;
  log_level?: string;
  auto_deploy?: boolean;
  heartbeat_interval?: string;
  bandwidth_limit_kbps?: number;
  max_concurrent_installs?: number;
  proxy_url?: string;
  auto_reboot_window?: string;
  log_retention_days?: number;
  offline_mode?: boolean;
}

export function useUpdateSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: SettingsUpdateBody) => {
      const res = await fetch('/api/v1/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ message: 'Failed to update settings' }));
        throw new Error(err.message ?? 'Failed to update settings');
      }
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
    },
  });
}

export function useTriggerScan() {
  return useMutation({
    mutationFn: async () => {
      const res = await fetch('/api/v1/scan', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ message: 'Failed to trigger scan' }));
        throw new Error(err.message ?? 'Failed to trigger scan');
      }
      return res.json();
    },
  });
}
