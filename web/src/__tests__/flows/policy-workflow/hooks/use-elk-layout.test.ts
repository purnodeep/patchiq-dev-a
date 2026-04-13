import { describe, it, expect } from 'vitest';
import { computeLayout } from '../../../../flows/policy-workflow/hooks/use-elk-layout';
import type { Node, Edge } from '@xyflow/react';

describe('computeLayout', () => {
  it('positions nodes top-to-bottom', async () => {
    const nodes: Node[] = [
      { id: 'a', position: { x: 0, y: 0 }, data: {} },
      { id: 'b', position: { x: 0, y: 0 }, data: {} },
    ];
    const edges: Edge[] = [{ id: 'e1', source: 'a', target: 'b' }];

    const result = await computeLayout(nodes, edges);
    expect(result.nodes).toHaveLength(2);
    const nodeA = result.nodes.find((n) => n.id === 'a')!;
    const nodeB = result.nodes.find((n) => n.id === 'b')!;
    expect(nodeB.position.y).toBeGreaterThan(nodeA.position.y);
  });
});
