/**
 * Derives display status from last_seen timestamp.
 * Overrides backend status to prevent misleading "online" when agent is unreachable.
 */
export function deriveStatus(backendStatus: string, lastSeen: string | null | undefined): string {
  if (!lastSeen) return backendStatus;
  const parsed = new Date(lastSeen).getTime();
  if (Number.isNaN(parsed)) return backendStatus;
  const diffMs = Date.now() - parsed;
  const diffMin = diffMs / 60_000;
  if (diffMin < 5) return 'online';
  if (diffMin < 30) return 'stale';
  return 'offline';
}
