import type { License } from '../types/license';

export type ComputedStatus = 'active' | 'expiring' | 'expired' | 'revoked';

export function computeStatus(license: Pick<License, 'revoked_at' | 'expires_at'>): ComputedStatus {
  if (license.revoked_at) return 'revoked';
  const expiresAt = new Date(license.expires_at);
  if (expiresAt < new Date()) return 'expired';
  const daysUntilExpiry = Math.ceil((expiresAt.getTime() - Date.now()) / (1000 * 60 * 60 * 24));
  if (daysUntilExpiry <= 30) return 'expiring';
  return 'active';
}
