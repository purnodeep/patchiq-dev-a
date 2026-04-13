import type { Feed, FeedHistoryResponse, UpdateFeedRequest } from '../types/feed';

const TENANT_ID = '00000000-0000-0000-0000-000000000001'; // default tenant for M1

async function apiFetch(url: string, options?: RequestInit): Promise<Response> {
  const res = await fetch(url, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': TENANT_ID,
      ...options?.headers,
    },
  });
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || `API error: ${res.status}`);
  }
  return res;
}

export async function listFeeds(): Promise<Feed[]> {
  const res = await apiFetch('/api/v1/feeds');
  return res.json() as Promise<Feed[]>;
}

export async function getFeed(id: string): Promise<Feed> {
  const res = await apiFetch(`/api/v1/feeds/${id}`);
  return res.json() as Promise<Feed>;
}

export async function getFeedHistory(
  id: string,
  params?: { limit?: number; offset?: number },
): Promise<FeedHistoryResponse> {
  const searchParams = new URLSearchParams();
  if (params?.limit !== undefined) searchParams.set('limit', String(params.limit));
  if (params?.offset !== undefined) searchParams.set('offset', String(params.offset));
  const qs = searchParams.toString();
  const res = await apiFetch(`/api/v1/feeds/${id}/history${qs ? `?${qs}` : ''}`);
  return res.json() as Promise<FeedHistoryResponse>;
}

export async function updateFeed(id: string, data: UpdateFeedRequest): Promise<Feed> {
  const res = await apiFetch(`/api/v1/feeds/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
  return res.json() as Promise<Feed>;
}

export async function triggerFeedSync(id: string): Promise<void> {
  await apiFetch(`/api/v1/feeds/${id}/sync`, { method: 'POST' });
}
