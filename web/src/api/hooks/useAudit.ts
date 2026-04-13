import { useQuery } from '@tanstack/react-query';
import { api } from '../client';

export interface AuditFilters {
  cursor?: string;
  limit?: number;
  actor_id?: string;
  actor_type?: 'user' | 'system' | 'ai_assistant';
  resource?: string;
  action?: string;
  type?: string;
  exclude_type?: string;
  from_date?: string;
  to_date?: string;
  search?: string;
  resource_id?: string;
}

export function useAuditLog(params?: AuditFilters) {
  return useQuery({
    queryKey: ['audit', params],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/audit', {
        params: { query: params },
      });
      if (error) throw error;
      return data;
    },
  });
}
