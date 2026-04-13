import { describe, it, expect } from 'vitest';
import { computeStatus } from '../lib/licenseUtils';

const future = (days: number) => new Date(Date.now() + days * 86400000).toISOString();
const past = (days: number) => new Date(Date.now() - days * 86400000).toISOString();

describe('computeStatus', () => {
  it('returns revoked when revoked_at is set', () => {
    expect(computeStatus({ revoked_at: past(1), expires_at: future(100) })).toBe('revoked');
  });

  it('returns expired when expires_at is in the past', () => {
    expect(computeStatus({ revoked_at: null, expires_at: past(1) })).toBe('expired');
  });

  it('returns expiring when 29 days remain', () => {
    expect(computeStatus({ revoked_at: null, expires_at: future(29) })).toBe('expiring');
  });

  it('returns expiring when exactly 30 days remain', () => {
    expect(computeStatus({ revoked_at: null, expires_at: future(30) })).toBe('expiring');
  });

  it('returns active when 31 days remain', () => {
    expect(computeStatus({ revoked_at: null, expires_at: future(31) })).toBe('active');
  });

  it('returns active when expires far in future', () => {
    expect(computeStatus({ revoked_at: null, expires_at: future(365) })).toBe('active');
  });
});
