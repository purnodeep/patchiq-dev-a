import { describe, it, expect } from 'vitest';
import { NODE_TYPE_REGISTRY, ALL_NODE_TYPES } from '../../../flows/policy-workflow/node-types';

describe('NODE_TYPE_REGISTRY', () => {
  it('has exactly 14 node types', () => {
    expect(ALL_NODE_TYPES).toHaveLength(14);
  });

  it('each entry has icon, label, and color', () => {
    for (const entry of ALL_NODE_TYPES) {
      const reg = NODE_TYPE_REGISTRY[entry];
      expect(reg).toBeDefined();
      expect(reg.label).toBeTruthy();
      expect(reg.color).toBeTruthy();
      expect(reg.icon).toBeDefined();
    }
  });

  it('includes all expected types', () => {
    const expected = [
      'trigger',
      'filter',
      'approval',
      'deployment_wave',
      'gate',
      'script',
      'notification',
      'rollback',
      'decision',
      'complete',
      'reboot',
      'scan',
      'tag_gate',
      'compliance_check',
    ];
    expect(ALL_NODE_TYPES).toEqual(expect.arrayContaining(expected));
  });
});
