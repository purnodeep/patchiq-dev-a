import { useQuery } from '@tanstack/react-query';
import { getActiveTenantId } from '../activeTenantStore';

// Helper mirroring the X-Tenant-ID injection the openapi-fetch client applies.
// Used by the two hooks below that cannot go through api.GET because the
// OpenAPI spec doesn't yet describe /commands/{id} or /endpoints/{id}/active-scan.
function tenantHeaders(): Record<string, string> {
  const id = getActiveTenantId();
  return id ? { 'X-Tenant-ID': id } : {};
}

export interface Command {
  id: string;
  agent_id: string;
  type: string;
  status: 'pending' | 'delivered' | 'succeeded' | 'failed' | 'cancelled';
  created_at: string;
  delivered_at: string | null;
  completed_at: string | null;
  error_message: string | null;
}

const TERMINAL_STATUSES: ReadonlyArray<Command['status']> = ['succeeded', 'failed', 'cancelled'];

export function isTerminalCommandStatus(status: Command['status']): boolean {
  return TERMINAL_STATUSES.includes(status);
}

/**
 * Polls GET /api/v1/commands/{id} every 3 seconds until the command reaches a
 * terminal status (succeeded, failed, cancelled). Pass `null` to disable
 * polling.
 */
export function useCommand(commandId: string | null) {
  return useQuery({
    queryKey: ['commands', commandId],
    queryFn: async () => {
      const res = await fetch(`/api/v1/commands/${commandId}`, {
        credentials: 'include',
        headers: tenantHeaders(),
      });
      if (!res.ok) {
        throw new Error(`Failed to fetch command: ${res.status}`);
      }
      return res.json() as Promise<Command>;
    },
    enabled: commandId != null,
    refetchInterval: (query) => {
      const data = query.state.data as Command | undefined;
      if (!data) return 3000;
      if (isTerminalCommandStatus(data.status)) return false;
      return 3000;
    },
  });
}

interface ActiveScanResponse {
  command: Command | null;
}

/**
 * On mount, fetches the latest non-terminal run_scan command for the given
 * endpoint, if any. Returns the command's ID so the UI can resume polling
 * after the user navigates away and back. Runs once per mount — does not poll.
 */
export function useActiveScan(endpointId: string | undefined) {
  const query = useQuery({
    queryKey: ['endpoints', endpointId, 'active-scan'],
    queryFn: async () => {
      const res = await fetch(`/api/v1/endpoints/${endpointId}/active-scan`, {
        credentials: 'include',
        headers: tenantHeaders(),
      });
      if (!res.ok) {
        throw new Error(`Failed to fetch active scan: ${res.status}`);
      }
      return res.json() as Promise<ActiveScanResponse>;
    },
    enabled: endpointId != null,
    refetchInterval: false,
    staleTime: 0,
  });

  return {
    activeCommandId: query.data?.command?.id ?? null,
    isLoading: query.isLoading,
  };
}
