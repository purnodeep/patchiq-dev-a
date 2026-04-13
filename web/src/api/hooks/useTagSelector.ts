import { useQuery } from '@tanstack/react-query';
import type { Selector } from '../../types/targeting';

export interface SelectorValidationResponse {
  valid: boolean;
  error?: string;
  matched_count: number;
}

/**
 * Live-preview hook for the selector builder. POSTs the current AST to
 * /tags/selectors/validate whenever it changes and returns { valid,
 * error, matched_count }. The backend rejects malformed ASTs with
 * valid=false (HTTP 200) so the UI shows inline feedback instead of a
 * noisy error toast.
 *
 * Debounce is the caller's responsibility — the builder holds the draft
 * AST in local state and only flips it into the query key after a short
 * pause so typing doesn't produce a burst of requests.
 */
export function useValidateSelector(selector: Selector | null | undefined) {
  return useQuery({
    queryKey: ['tag-selector', 'validate', selector],
    queryFn: async (): Promise<SelectorValidationResponse> => {
      if (!selector) return { valid: false, matched_count: 0, error: 'no selector' };
      const res = await fetch('/api/v1/tags/selectors/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ selector }),
      });
      if (!res.ok) {
        throw new Error(`Failed to validate selector: ${res.status}`);
      }
      return res.json() as Promise<SelectorValidationResponse>;
    },
    enabled: !!selector,
    staleTime: 5_000,
  });
}
