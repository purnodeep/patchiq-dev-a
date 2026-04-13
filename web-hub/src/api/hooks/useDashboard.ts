import { useQuery } from '@tanstack/react-query';
import {
  getDashboardStats,
  getLicenseBreakdown,
  getCatalogGrowth,
  getClientSummary,
  getRecentActivity,
} from '../dashboard';

export function useDashboardStats() {
  return useQuery({
    queryKey: ['dashboard', 'stats'],
    queryFn: getDashboardStats,
    refetchInterval: 30_000,
  });
}

export function useLicenseBreakdown() {
  return useQuery({
    queryKey: ['dashboard', 'license-breakdown'],
    queryFn: getLicenseBreakdown,
    refetchInterval: 60_000,
  });
}

export function useCatalogGrowth(days = 90) {
  return useQuery({
    queryKey: ['dashboard', 'catalog-growth', days],
    queryFn: () => getCatalogGrowth(days),
    refetchInterval: 300_000,
  });
}

export function useClientSummary() {
  return useQuery({
    queryKey: ['dashboard', 'clients'],
    queryFn: getClientSummary,
    refetchInterval: 30_000,
  });
}

export function useRecentActivity() {
  return useQuery({
    queryKey: ['dashboard', 'activity'],
    queryFn: getRecentActivity,
    refetchInterval: 30_000,
  });
}
