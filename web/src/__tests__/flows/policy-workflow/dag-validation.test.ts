import { describe, it, expect } from 'vitest';
import { validateWorkflowDAG } from '../../../flows/policy-workflow/dag-validation';

describe('validateWorkflowDAG', () => {
  it('returns empty for valid DAG', () => {
    const nodes = [
      { id: 'n1', nodeType: 'trigger' },
      { id: 'n2', nodeType: 'complete' },
    ];
    const edges = [{ source: 'n1', target: 'n2' }];
    expect(validateWorkflowDAG(nodes, edges)).toEqual([]);
  });

  it('reports disconnected nodes', () => {
    const nodes = [
      { id: 'n1', nodeType: 'trigger' },
      { id: 'n2', nodeType: 'filter' },
      { id: 'n3', nodeType: 'complete' },
    ];
    const edges = [{ source: 'n1', target: 'n3' }];
    const errors = validateWorkflowDAG(nodes, edges);
    expect(errors.find((e) => e.nodeId === 'n2')).toBeDefined();
  });

  it('reports missing trigger node', () => {
    const nodes = [{ id: 'n1', nodeType: 'filter' }];
    const errors = validateWorkflowDAG(nodes, []);
    expect(errors.find((e) => e.message.includes('trigger'))).toBeDefined();
  });

  it('reports empty workflow', () => {
    const errors = validateWorkflowDAG([], []);
    expect(errors.length).toBeGreaterThan(0);
  });
});
