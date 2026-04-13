import { describe, it, expect } from 'vitest';
import {
  getEventCategory,
  getCategoryColor,
  getCategoryBadgeClassName,
  getActorInitials,
  groupEventsByDate,
} from '../../../lib/audit-utils';
import type { components } from '../../../api/types';

type AuditEvent = components['schemas']['AuditEvent'];

describe('getEventCategory', () => {
  it('maps endpoint. prefix to Endpoint', () => {
    expect(getEventCategory('endpoint.enrolled')).toBe('Endpoint');
    expect(getEventCategory('endpoint.decommissioned')).toBe('Endpoint');
  });
  it('maps heartbeat. prefix to Endpoint', () => {
    expect(getEventCategory('heartbeat.received')).toBe('Endpoint');
  });
  it('maps inventory. prefix to Endpoint', () => {
    expect(getEventCategory('inventory.scan.completed')).toBe('Endpoint');
  });
  it('maps agent. prefix to Endpoint', () => {
    expect(getEventCategory('agent.offline')).toBe('Endpoint');
  });
  it('maps deployment. prefix to Deployment', () => {
    expect(getEventCategory('deployment.created')).toBe('Deployment');
    expect(getEventCategory('deployment.completed')).toBe('Deployment');
    expect(getEventCategory('deployment.rollback_initiated')).toBe('Deployment');
  });
  it('maps policy. prefix to Policy', () => {
    expect(getEventCategory('policy.created')).toBe('Policy');
    expect(getEventCategory('policy.updated')).toBe('Policy');
  });
  it('maps compliance. prefix to Compliance', () => {
    expect(getEventCategory('compliance.evaluation.completed')).toBe('Compliance');
  });
  it('maps auth. prefix to Auth', () => {
    expect(getEventCategory('auth.login')).toBe('Auth');
    expect(getEventCategory('auth.logout')).toBe('Auth');
  });
  it('maps cve. and patch. prefixes to Patch', () => {
    expect(getEventCategory('cve.kev_added')).toBe('Patch');
    expect(getEventCategory('patch.discovered')).toBe('Patch');
  });
  it('maps unknown prefixes to System', () => {
    expect(getEventCategory('workflow.created')).toBe('System');
    expect(getEventCategory('role.updated')).toBe('System');
    expect(getEventCategory('group.created')).toBe('System');
    expect(getEventCategory('license.activated')).toBe('System');
    expect(getEventCategory('catalog.synced')).toBe('System');
    expect(getEventCategory('notification.sent')).toBe('System');
    expect(getEventCategory('schedule.created')).toBe('System');
  });
});

describe('getCategoryColor', () => {
  it('returns a color string (hex or CSS variable) for all categories', () => {
    const categories = [
      'Endpoint',
      'Deployment',
      'Policy',
      'Compliance',
      'Auth',
      'Patch',
      'System',
    ] as const;
    for (const cat of categories) {
      const color = getCategoryColor(cat);
      // Must be a hex color or a CSS variable reference
      expect(color).toMatch(/^(#[0-9a-f]{6}|var\(--[a-z-]+\))$/i);
    }
  });
});

describe('getCategoryBadgeClassName', () => {
  it('returns tailwind class strings for all categories', () => {
    const categories = [
      'Endpoint',
      'Deployment',
      'Policy',
      'Compliance',
      'Auth',
      'Patch',
      'System',
    ] as const;
    for (const cat of categories) {
      const cls = getCategoryBadgeClassName(cat);
      expect(typeof cls).toBe('string');
      expect(cls.length).toBeGreaterThan(0);
    }
  });
});

describe('getActorInitials', () => {
  it('returns SY for system', () => {
    expect(getActorInitials('system')).toBe('SY');
  });
  it('returns first two chars of email local part uppercased', () => {
    expect(getActorInitials('admin@acme.com')).toBe('AD');
    expect(getActorInitials('sandy@acme.com')).toBe('SA');
    expect(getActorInitials('rishab@acme.com')).toBe('RI');
    expect(getActorInitials('danish@acme.com')).toBe('DA');
  });
  it('returns ?? for UUID-like strings', () => {
    expect(getActorInitials('550e8400-e29b-41d4-a716-446655440000')).toBe('??');
  });
  it('returns ?? for empty string', () => {
    expect(getActorInitials('')).toBe('??');
  });
});

describe('groupEventsByDate', () => {
  it('groups events by calendar date', () => {
    const events: AuditEvent[] = [
      { id: '1', timestamp: '2026-03-10T08:00:00Z' } as AuditEvent,
      { id: '2', timestamp: '2026-03-10T15:00:00Z' } as AuditEvent,
      { id: '3', timestamp: '2026-03-09T10:00:00Z' } as AuditEvent,
    ];
    const groups = groupEventsByDate(events);
    expect(groups.size).toBe(2);
    const keys = Array.from(groups.keys());
    // Both March 10 events should be in same group
    const march10Events = groups.get(keys.find((k) => k.includes('March 10'))!);
    expect(march10Events?.length).toBe(2);
    const march9Events = groups.get(keys.find((k) => k.includes('March 9'))!);
    expect(march9Events?.length).toBe(1);
  });

  it('skips events without timestamps', () => {
    const events: AuditEvent[] = [
      { id: '1' } as AuditEvent,
      { id: '2', timestamp: '2026-03-10T08:00:00Z' } as AuditEvent,
    ];
    const groups = groupEventsByDate(events);
    expect(groups.size).toBe(1);
  });
});
