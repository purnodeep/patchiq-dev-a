import createClient, { type Middleware } from 'openapi-fetch';
import { toast } from 'sonner';
import type { paths } from './types';
import { getActiveTenantId } from './activeTenantStore';

// Injects the X-Tenant-ID header from the active-tenant store on every
// request. When the MSP tenant switcher updates the active tenant, all
// subsequent API calls target the newly selected tenant.
const activeTenantHeader: Middleware = {
  async onRequest({ request }) {
    const id = getActiveTenantId();
    if (id) {
      request.headers.set('X-Tenant-ID', id);
    }
    return request;
  },
};

const authRedirect: Middleware = {
  async onResponse({ response }) {
    if (response.status === 401 && !window.location.pathname.startsWith('/login')) {
      window.location.href = '/login';
    }
    return response;
  },
};

const forbiddenInterceptor: Middleware = {
  async onResponse({ response }) {
    if (response.status === 403) {
      toast.error('Access denied', {
        description: "You don't have permission to perform this action.",
      });
    }
    return response;
  },
};

const serverErrorInterceptor: Middleware = {
  async onResponse({ response }) {
    if (response.status >= 500) {
      throw new Error(`Server error: ${response.status} ${response.statusText} on ${response.url}`);
    }
    return response;
  },
};

export const api = createClient<paths>({
  baseUrl: window.location.origin,
  credentials: 'include',
});

api.use(activeTenantHeader);
api.use(authRedirect);
api.use(forbiddenInterceptor);
api.use(serverErrorInterceptor);
