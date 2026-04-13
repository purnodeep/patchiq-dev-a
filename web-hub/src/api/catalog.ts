import type {
  CatalogEntry,
  CatalogDetailResponse,
  CatalogListResponse,
  CatalogStats,
  CreateCatalogRequest,
} from '../types/catalog';
import { apiFetch } from './fetch';

export async function listCatalog(params: {
  limit?: number;
  offset?: number;
  os_family?: string;
  severity?: string;
  search?: string;
  feed_source_id?: string;
  date_range?: string;
  entry_type?: string;
}): Promise<CatalogListResponse> {
  const searchParams = new URLSearchParams();
  if (params.limit !== undefined) searchParams.set('limit', String(params.limit));
  if (params.offset !== undefined) searchParams.set('offset', String(params.offset));
  if (params.os_family) searchParams.set('os_family', params.os_family);
  if (params.severity) searchParams.set('severity', params.severity);
  if (params.search) searchParams.set('search', params.search);
  if (params.feed_source_id) searchParams.set('feed_source_id', params.feed_source_id);
  if (params.date_range) searchParams.set('date_range', params.date_range);
  if (params.entry_type) searchParams.set('entry_type', params.entry_type);
  const qs = searchParams.toString();
  const res = await apiFetch(`/api/v1/catalog${qs ? `?${qs}` : ''}`);
  return res.json() as Promise<CatalogListResponse>;
}

export async function getCatalogEntry(id: string): Promise<CatalogDetailResponse> {
  const res = await apiFetch(`/api/v1/catalog/${id}`);
  return res.json() as Promise<CatalogDetailResponse>;
}

export async function getCatalogStats(): Promise<CatalogStats> {
  const res = await apiFetch('/api/v1/catalog/stats');
  return res.json() as Promise<CatalogStats>;
}

export async function createCatalogEntry(data: CreateCatalogRequest): Promise<CatalogEntry> {
  const res = await apiFetch('/api/v1/catalog', {
    method: 'POST',
    body: JSON.stringify(data),
  });
  return res.json() as Promise<CatalogEntry>;
}

export async function updateCatalogEntry(
  id: string,
  data: CreateCatalogRequest,
): Promise<CatalogEntry> {
  const res = await apiFetch(`/api/v1/catalog/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
  return res.json() as Promise<CatalogEntry>;
}

export async function deleteCatalogEntry(id: string): Promise<void> {
  await apiFetch(`/api/v1/catalog/${id}`, { method: 'DELETE' });
}
