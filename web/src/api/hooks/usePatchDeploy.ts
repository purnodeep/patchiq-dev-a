import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';

export interface PatchDeployPayload {
  patchId: string;
  name: string;
  description?: string;
  config_type: 'install' | 'rollback';
  scope: string;
  target_endpoints: string;
  endpoint_ids?: string[];
  scheduled_at?: string;
}

export function usePatchDeploy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ patchId, ...body }: PatchDeployPayload) => {
      const { data, error } = await api.POST(
        '/api/v1/patches/{id}/deploy' as never,
        {
          params: { path: { id: patchId } },
          body,
        } as never,
      );
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['deployments'] });
      void queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });
}
