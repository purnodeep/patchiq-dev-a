import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listLicenses,
  getLicense,
  createLicense,
  revokeLicense,
  assignLicense,
  renewLicense,
  getLicenseUsageHistory,
  getLicenseAuditTrail,
} from '../licenses';
import type { CreateLicenseRequest } from '../../types/license';
export type { LicenseUsagePoint, LicenseAuditEvent } from '../licenses';

export function useLicenses(params?: {
  limit?: number;
  offset?: number;
  tier?: string;
  status?: string;
}) {
  return useQuery({
    queryKey: ['licenses', params],
    queryFn: () => listLicenses(params ?? {}),
  });
}

export function useLicense(id: string) {
  return useQuery({
    queryKey: ['licenses', id],
    queryFn: () => getLicense(id),
    enabled: !!id,
  });
}

export function useCreateLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateLicenseRequest) => createLicense(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
    },
  });
}

export function useRevokeLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: revokeLicense,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
      void queryClient.invalidateQueries({ queryKey: ['clients'] });
    },
  });
}

export function useAssignLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, clientId }: { id: string; clientId: string }) => assignLicense(id, clientId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
      void queryClient.invalidateQueries({ queryKey: ['clients'] });
    },
  });
}

export function useRenewLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: renewLicense,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
    },
  });
}

export function useLicenseUsageHistory(licenseId: string | undefined, days: number) {
  return useQuery({
    queryKey: ['licenses', licenseId, 'usage-history', days],
    queryFn: () => getLicenseUsageHistory(licenseId!, days),
    enabled: !!licenseId,
  });
}

export function useLicenseAuditTrail(licenseId: string | undefined, limit: number) {
  return useQuery({
    queryKey: ['licenses', licenseId, 'audit-trail', limit],
    queryFn: () => getLicenseAuditTrail(licenseId!, limit),
    enabled: !!licenseId,
  });
}
