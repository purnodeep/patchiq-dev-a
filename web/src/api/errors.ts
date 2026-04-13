import type { components } from './types';

export type ApiErrorEnvelope = components['schemas']['ErrorResponse'];

// extractApiError normalizes anything thrown by an openapi-fetch hook into
// a stable {code, message} shape. The fetch client surfaces server errors
// as the raw ErrorResponse body, but pages still need a single accessor
// rather than re-parsing it everywhere.
export function extractApiError(err: unknown): { code: string; message: string } {
  if (err && typeof err === 'object') {
    const maybe = err as Partial<ApiErrorEnvelope> & { message?: unknown };
    if (typeof maybe.code === 'string' && typeof maybe.message === 'string') {
      return { code: maybe.code, message: maybe.message };
    }
    if (typeof maybe.message === 'string' && maybe.message) {
      return { code: 'UNKNOWN', message: maybe.message };
    }
  }
  if (err instanceof Error && err.message) {
    return { code: 'UNKNOWN', message: err.message };
  }
  return { code: 'UNKNOWN', message: 'Request failed' };
}
