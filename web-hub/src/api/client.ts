import createClient from 'openapi-fetch';
import type { paths } from './types';

const TENANT_ID = '00000000-0000-0000-0000-000000000001';

export const api = createClient<paths>({
  baseUrl: window.location.origin,
  credentials: 'include',
  headers: {
    'X-Tenant-ID': TENANT_ID,
  },
});
