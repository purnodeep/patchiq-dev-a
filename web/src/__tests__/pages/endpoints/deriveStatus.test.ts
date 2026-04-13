import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { deriveStatus } from '../../../pages/endpoints/deriveStatus';

const FIXED_NOW = new Date('2024-01-01T12:00:00.000Z').getTime();

describe('deriveStatus', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(FIXED_NOW);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns backendStatus when lastSeen is null', () => {
    expect(deriveStatus('unknown', null)).toBe('unknown');
  });

  it('returns backendStatus when lastSeen is undefined', () => {
    expect(deriveStatus('pending', undefined)).toBe('pending');
  });

  it('returns online when lastSeen is less than 5 minutes ago', () => {
    const lastSeen = new Date(FIXED_NOW - 4 * 60_000).toISOString();
    expect(deriveStatus('offline', lastSeen)).toBe('online');
  });

  it('returns stale when lastSeen is between 5 and 30 minutes ago', () => {
    const lastSeen = new Date(FIXED_NOW - 15 * 60_000).toISOString();
    expect(deriveStatus('online', lastSeen)).toBe('stale');
  });

  it('returns offline when lastSeen is more than 30 minutes ago', () => {
    const lastSeen = new Date(FIXED_NOW - 45 * 60_000).toISOString();
    expect(deriveStatus('online', lastSeen)).toBe('offline');
  });

  it('returns stale at exactly 5 minutes (not online)', () => {
    const lastSeen = new Date(FIXED_NOW - 5 * 60_000).toISOString();
    expect(deriveStatus('online', lastSeen)).toBe('stale');
  });

  it('returns offline at exactly 30 minutes (not stale)', () => {
    const lastSeen = new Date(FIXED_NOW - 30 * 60_000).toISOString();
    expect(deriveStatus('online', lastSeen)).toBe('offline');
  });
});
