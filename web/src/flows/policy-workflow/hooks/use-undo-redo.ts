import { useCallback, useRef } from 'react';
import type { Node, Edge } from '@xyflow/react';

interface Snapshot {
  nodes: Node[];
  edges: Edge[];
}

const MAX_HISTORY = 50;

export function useUndoRedo(
  setNodes: (nodes: Node[] | ((prev: Node[]) => Node[])) => void,
  setEdges: (edges: Edge[] | ((prev: Edge[]) => Edge[])) => void,
) {
  const past = useRef<Snapshot[]>([]);
  const future = useRef<Snapshot[]>([]);
  const currentRef = useRef<Snapshot | null>(null);

  const takeSnapshot = useCallback((nodes: Node[], edges: Edge[]) => {
    if (currentRef.current) {
      past.current = [...past.current.slice(-(MAX_HISTORY - 1)), currentRef.current];
    }
    currentRef.current = { nodes: structuredClone(nodes), edges: structuredClone(edges) };
    future.current = [];
  }, []);

  const undo = useCallback(() => {
    const prev = past.current.pop();
    if (!prev || !currentRef.current) return;
    future.current = [...future.current, currentRef.current];
    currentRef.current = prev;
    setNodes(prev.nodes);
    setEdges(prev.edges);
  }, [setNodes, setEdges]);

  const redo = useCallback(() => {
    const next = future.current.pop();
    if (!next || !currentRef.current) return;
    past.current = [...past.current, currentRef.current];
    currentRef.current = next;
    setNodes(next.nodes);
    setEdges(next.edges);
  }, [setNodes, setEdges]);

  const canUndo = useCallback(() => past.current.length > 0, []);
  const canRedo = useCallback(() => future.current.length > 0, []);

  return { takeSnapshot, undo, redo, canUndo, canRedo };
}
