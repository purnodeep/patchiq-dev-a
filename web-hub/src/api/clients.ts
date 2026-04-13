import type { Client, ClientListResponse } from '../types/client';
import { apiFetch } from './fetch';

export async function listClients(params: {
  limit?: number;
  offset?: number;
  status?: string;
}): Promise<ClientListResponse> {
  const searchParams = new URLSearchParams();
  if (params.limit) searchParams.set('limit', String(params.limit));
  if (params.offset) searchParams.set('offset', String(params.offset));
  if (params.status) searchParams.set('status', params.status);
  const res = await apiFetch(`/api/v1/clients?${searchParams}`);
  return res.json() as Promise<ClientListResponse>;
}

export async function getClient(id: string): Promise<Client> {
  const res = await apiFetch(`/api/v1/clients/${id}`);
  const data = (await res.json()) as { client: Client };
  if (!data.client) throw new Error('getClient: unexpected response shape');
  return data.client;
}

export async function approveClient(id: string): Promise<{ api_key: string }> {
  const res = await apiFetch(`/api/v1/clients/${id}/approve`, { method: 'POST' });
  const data = (await res.json()) as { api_key: string };
  if (!data.api_key) throw new Error('approveClient: unexpected response shape');
  return data;
}

export async function declineClient(id: string): Promise<{ status: string }> {
  const res = await apiFetch(`/api/v1/clients/${id}/decline`, { method: 'POST' });
  const data = (await res.json()) as { status: string };
  if (!data.status) throw new Error('declineClient: unexpected response shape');
  return data;
}

export async function suspendClient(id: string): Promise<{ status: string }> {
  const res = await apiFetch(`/api/v1/clients/${id}/suspend`, { method: 'POST' });
  const data = (await res.json()) as { status: string };
  if (!data.status) throw new Error('suspendClient: unexpected response shape');
  return data;
}

export async function updateClient(
  id: string,
  data: { sync_interval?: number; notes?: string },
): Promise<Client> {
  const res = await apiFetch(`/api/v1/clients/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
  const body = (await res.json()) as { client: Client };
  if (!body.client) throw new Error('updateClient: unexpected response shape');
  return body.client;
}

export async function deleteClient(id: string): Promise<void> {
  await apiFetch(`/api/v1/clients/${id}`, { method: 'DELETE' });
}

export async function getPendingClientCount(): Promise<{ count: number }> {
  const res = await apiFetch('/api/v1/clients/pending-count');
  return res.json() as Promise<{ count: number }>;
}

export interface SyncHistoryItem {
  id: string;
  client_id: string;
  synced_at: string;
  patches_synced: number;
  cves_synced: number;
  duration_ms: number;
  status: 'success' | 'partial' | 'failed';
  error?: string;
}

export interface EndpointTrendPoint {
  date: string;
  total: number;
  active: number;
  inactive: number;
}

export async function getClientSyncHistory(
  clientId: string,
  params: { limit?: number; offset?: number },
): Promise<{ items: SyncHistoryItem[]; total: number }> {
  const searchParams = new URLSearchParams();
  if (params.limit) searchParams.set('limit', String(params.limit));
  if (params.offset) searchParams.set('offset', String(params.offset));
  const res = await apiFetch(`/api/v1/clients/${clientId}/sync-history?${searchParams}`);
  return res.json() as Promise<{ items: SyncHistoryItem[]; total: number }>;
}

export async function getClientEndpointTrend(
  clientId: string,
  days: number,
): Promise<{ points: EndpointTrendPoint[] }> {
  const searchParams = new URLSearchParams();
  searchParams.set('days', String(days));
  const res = await apiFetch(`/api/v1/clients/${clientId}/endpoint-trend?${searchParams}`);
  return res.json() as Promise<{ points: EndpointTrendPoint[] }>;
}
