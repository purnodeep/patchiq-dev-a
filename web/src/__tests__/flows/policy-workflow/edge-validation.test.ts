import { describe, it, expect } from 'vitest';
import { isValidWorkflowConnection } from '../../../flows/policy-workflow/canvas';

describe('isValidWorkflowConnection', () => {
  it('rejects self-connections', () => {
    expect(isValidWorkflowConnection('n1', 'n1', [])).toBe(false);
  });

  it('rejects duplicate edges', () => {
    const edges = [{ id: 'e1', source: 'n1', target: 'n2' }];
    expect(isValidWorkflowConnection('n1', 'n2', edges)).toBe(false);
  });

  it('allows valid connections', () => {
    expect(isValidWorkflowConnection('n1', 'n2', [])).toBe(true);
  });

  it('rejects connections that create cycles', () => {
    const edges = [
      { id: 'e1', source: 'n1', target: 'n2' },
      { id: 'e2', source: 'n2', target: 'n3' },
    ];
    // n3 -> n1 would create a cycle: n1 -> n2 -> n3 -> n1
    expect(isValidWorkflowConnection('n3', 'n1', edges)).toBe(false);
  });

  it('allows non-cyclic connections in complex graphs', () => {
    const edges = [
      { id: 'e1', source: 'n1', target: 'n2' },
      { id: 'e2', source: 'n1', target: 'n3' },
    ];
    // n2 -> n3 is fine (no cycle)
    expect(isValidWorkflowConnection('n2', 'n3', edges)).toBe(true);
  });
});
