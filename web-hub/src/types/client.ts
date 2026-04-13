export interface Client {
  id: string;
  hostname: string;
  version: string | null;
  os: string | null;
  endpoint_count: number;
  contact_email: string | null;
  status: 'pending' | 'approved' | 'declined' | 'suspended';
  sync_interval: number;
  last_sync_at: string | null;
  notes: string | null;
  created_at: string;
  updated_at: string;
}

export interface ClientListResponse {
  clients: Client[];
  total: number;
}
