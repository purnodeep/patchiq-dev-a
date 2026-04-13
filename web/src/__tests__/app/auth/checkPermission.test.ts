import { checkPermission } from '../../../app/auth/AuthContext';

describe('checkPermission', () => {
  it('returns true for undefined permissions (backwards compat)', () => {
    expect(checkPermission(undefined, 'endpoints', 'read')).toBe(true);
  });

  it('returns false for empty permissions array', () => {
    expect(checkPermission([], 'endpoints', 'read')).toBe(false);
  });

  it('returns true for exact match', () => {
    const perms = [{ resource: 'endpoints', action: 'read', scope: '*' }];
    expect(checkPermission(perms, 'endpoints', 'read')).toBe(true);
  });

  it('returns false for non-match', () => {
    const perms = [{ resource: 'endpoints', action: 'read', scope: '*' }];
    expect(checkPermission(perms, 'policies', 'read')).toBe(false);
  });

  it('wildcard resource: *:read matches endpoints:read', () => {
    const perms = [{ resource: '*', action: 'read', scope: '*' }];
    expect(checkPermission(perms, 'endpoints', 'read')).toBe(true);
  });

  it('wildcard action: endpoints:* matches endpoints:delete', () => {
    const perms = [{ resource: 'endpoints', action: '*', scope: '*' }];
    expect(checkPermission(perms, 'endpoints', 'delete')).toBe(true);
  });

  it('full wildcard: *:* matches anything', () => {
    const perms = [{ resource: '*', action: '*', scope: '*' }];
    expect(checkPermission(perms, 'deployments', 'write')).toBe(true);
  });

  it('multiple permissions: returns true if ANY matches', () => {
    const perms = [
      { resource: 'policies', action: 'read', scope: '*' },
      { resource: 'endpoints', action: 'read', scope: '*' },
    ];
    expect(checkPermission(perms, 'endpoints', 'read')).toBe(true);
  });

  it('partial match fails: endpoints:read does not match endpoints:delete', () => {
    const perms = [{ resource: 'endpoints', action: 'read', scope: '*' }];
    expect(checkPermission(perms, 'endpoints', 'delete')).toBe(false);
  });

  it('resource mismatch with action match: policies:read does not match endpoints:read', () => {
    const perms = [{ resource: 'policies', action: 'read', scope: '*' }];
    expect(checkPermission(perms, 'endpoints', 'read')).toBe(false);
  });
});
