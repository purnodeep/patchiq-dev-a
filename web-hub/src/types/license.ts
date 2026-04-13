export interface License {
  id: string;
  client_id: string | null;
  client_hostname: string | null;
  tier: 'community' | 'professional' | 'enterprise' | 'msp';
  max_endpoints: number;
  issued_at: string;
  expires_at: string;
  revoked_at: string | null;
  customer_name: string;
  customer_email: string | null;
  notes: string | null;
  license_key: string;
  client_endpoint_count: number | null;
  created_at: string;
  updated_at: string;
}

export interface LicenseListResponse {
  licenses: License[];
  total: number;
}

export interface CreateLicenseRequest {
  customer_name: string;
  customer_email?: string;
  tier: string;
  max_endpoints: number;
  expires_at: string;
  client_id?: string;
  notes?: string;
}
