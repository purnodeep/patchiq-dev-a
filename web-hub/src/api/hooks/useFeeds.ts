import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listFeeds, getFeed, getFeedHistory, updateFeed, triggerFeedSync } from '../feeds';
import { toast } from 'sonner';
import type { UpdateFeedRequest } from '../../types/feed';

export function useFeeds() {
  return useQuery({
    queryKey: ['feeds'],
    queryFn: listFeeds,
  });
}

export function useFeed(id: string) {
  return useQuery({
    queryKey: ['feed', id],
    queryFn: () => getFeed(id),
    enabled: !!id,
  });
}

export function useFeedHistory(id: string, params?: { limit?: number; offset?: number }) {
  return useQuery({
    queryKey: ['feed-history', id, params],
    queryFn: () => getFeedHistory(id, params),
    enabled: !!id,
  });
}

export function useUpdateFeed() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateFeedRequest }) => updateFeed(id, data),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: ['feeds'] });
      void queryClient.invalidateQueries({ queryKey: ['feed', variables.id] });
      if (variables.data.enabled === true) {
        toast.success('Feed enabled');
      } else if (variables.data.enabled === false) {
        toast.success('Feed disabled');
      } else {
        toast.success('Feed configuration updated');
      }
    },
    onError: (err) => {
      toast.error('Update failed: ' + (err instanceof Error ? err.message : 'Unknown error'));
    },
  });
}

export function useTriggerFeedSync() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: triggerFeedSync,
    onSuccess: (_data, feedId) => {
      void queryClient.invalidateQueries({ queryKey: ['feeds'] });
      void queryClient.invalidateQueries({ queryKey: ['feed', feedId] });
      void queryClient.invalidateQueries({ queryKey: ['feed-history', feedId] });
      void queryClient.invalidateQueries({ queryKey: ['catalog'] });
      void queryClient.invalidateQueries({ queryKey: ['catalog-stats'] });
      toast.success('Feed sync triggered');
    },
    onError: (err) => {
      toast.error('Sync failed: ' + (err instanceof Error ? err.message : 'Unknown error'));
    },
  });
}
