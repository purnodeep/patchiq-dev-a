import type { License, LicenseListResponse, CreateLicenseRequest } from '../types/license';
import { apiFetch } from './fetch';

export async function listLicenses(params: {
  limit?: number;
  offset?: number;
  tier?: string;
  status?: string;
}): Promise<LicenseListResponse> {
  const searchParams = new URLSearchParams();
  if (params.limit) searchParams.set('limit', String(params.limit));
  if (params.offset) searchParams.set('offset', String(params.offset));
  if (params.tier) searchParams.set('tier', params.tier);
  if (params.status) searchParams.set('status', params.status);
  const res = await apiFetch(`/api/v1/licenses?${searchParams}`);
  return res.json() as Promise<LicenseListResponse>;
}

export async function createLicense(data: CreateLicenseRequest): Promise<License> {
  const res = await apiFetch('/api/v1/licenses', {
    method: 'POST',
    body: JSON.stringify(data),
  });
  const body = (await res.json()) as { license: License };
  if (!body.license) throw new Error('createLicense: unexpected response shape');
  return body.license;
}

export async function getLicense(id: string): Promise<License> {
  const res = await apiFetch(`/api/v1/licenses/${id}`);
  const data = (await res.json()) as { license: License };
  if (!data.license) throw new Error('getLicense: unexpected response shape');
  return data.license;
}

export async function revokeLicense(id: string): Promise<License> {
  const res = await apiFetch(`/api/v1/licenses/${id}/revoke`, { method: 'POST' });
  const data = (await res.json()) as { license: License };
  if (!data.license) throw new Error('revokeLicense: unexpected response shape');
  return data.license;
}

export async function assignLicense(id: string, clientId: string): Promise<License> {
  const res = await apiFetch(`/api/v1/licenses/${id}/assign`, {
    method: 'POST',
    body: JSON.stringify({ client_id: clientId }),
  });
  const data = (await res.json()) as { license: License };
  if (!data.license) throw new Error('assignLicense: unexpected response shape');
  return data.license;
}

export interface RenewLicensePayload {
  id: string;
  tier?: string;
  max_endpoints?: number;
  expires_at: string;
}

export async function renewLicense({ id, ...body }: RenewLicensePayload): Promise<License> {
  const res = await apiFetch(`/api/v1/licenses/${id}/renew`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
  const data = (await res.json()) as { license: License };
  if (!data.license) throw new Error('renewLicense: unexpected response shape');
  return data.license;
}

export interface LicenseUsagePoint {
  date: string;
  endpoints_used: number;
  endpoints_limit: number;
  utilization_pct: number;
}

export interface LicenseAuditEvent {
  id: string;
  license_id: string;
  event_type: string;
  actor: string;
  occurred_at: string;
  details?: Record<string, unknown>;
}

export async function getLicenseUsageHistory(
  licenseId: string,
  days: number,
): Promise<{ points: LicenseUsagePoint[] }> {
  const searchParams = new URLSearchParams();
  searchParams.set('days', String(days));
  const res = await apiFetch(`/api/v1/licenses/${licenseId}/usage-history?${searchParams}`);
  return res.json() as Promise<{ points: LicenseUsagePoint[] }>;
}

export async function getLicenseAuditTrail(
  licenseId: string,
  limit: number,
): Promise<{ events: LicenseAuditEvent[]; total: number }> {
  const searchParams = new URLSearchParams();
  searchParams.set('limit', String(limit));
  const res = await apiFetch(`/api/v1/licenses/${licenseId}/audit-trail?${searchParams}`);
  return res.json() as Promise<{ events: LicenseAuditEvent[]; total: number }>;
}
