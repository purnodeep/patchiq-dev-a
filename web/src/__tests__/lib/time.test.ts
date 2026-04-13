import { describe, it, expect, vi, afterEach } from 'vitest';
import { timeAgo } from '../../lib/time';

const NOW = new Date('2025-06-15T12:00:00Z').getTime();

describe('timeAgo', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  function setup() {
    vi.useFakeTimers();
    vi.setSystemTime(NOW);
  }

  it.each([
    { input: null, expected: '—', label: 'null' },
    { input: undefined, expected: '—', label: 'undefined' },
    { input: '', expected: '—', label: 'empty string' },
    { input: 'not-a-date', expected: '—', label: 'invalid date string' },
  ])('returns "—" for $label', ({ input, expected }) => {
    setup();
    expect(timeAgo(input)).toBe(expected);
  });

  it.each([
    { offsetMs: 10_000, expected: 'just now', label: '< 1 min ago' },
    { offsetMs: 5 * 60_000, expected: '5 min ago', label: '5 minutes ago' },
    { offsetMs: 2 * 60 * 60_000, expected: '2h ago', label: '2 hours ago' },
    { offsetMs: 3 * 24 * 60 * 60_000, expected: '3d ago', label: '3 days ago' },
  ])('returns "$expected" for $label', ({ offsetMs, expected }) => {
    setup();
    const dateStr = new Date(NOW - offsetMs).toISOString();
    expect(timeAgo(dateStr)).toBe(expected);
  });
});
