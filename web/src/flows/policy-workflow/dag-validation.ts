import type { NodeConfig, DecisionConfig, FilterConfig } from './types';

export interface ValidationError {
  nodeId?: string;
  message: string;
}

export interface ValidationWarning {
  nodeId: string;
  message: string;
}

export function validateWorkflowDAG(
  nodes: Array<{ id: string; nodeType: string }>,
  edges: Array<{ source: string; target: string }>,
): ValidationError[] {
  const errors: ValidationError[] = [];

  if (nodes.length === 0) {
    errors.push({ message: 'Workflow must have at least one node' });
    return errors;
  }

  if (!nodes.some((n) => n.nodeType === 'trigger')) {
    errors.push({ message: 'Workflow must have a trigger node' });
  }

  // Find disconnected nodes when there are multiple nodes
  if (nodes.length > 1) {
    const connected = new Set<string>();
    for (const edge of edges) {
      connected.add(edge.source);
      connected.add(edge.target);
    }
    for (const node of nodes) {
      if (!connected.has(node.id)) {
        errors.push({
          nodeId: node.id,
          message: `Node is not connected to any other node`,
        });
      }
    }
  }

  return errors;
}

/** Check individual node configs for incomplete configuration (non-blocking warnings) */
export function validateNodeConfigs(
  nodes: Array<{ id: string; nodeType: string; config: NodeConfig }>,
): ValidationWarning[] {
  const warnings: ValidationWarning[] = [];

  for (const node of nodes) {
    const cfg = node.config;
    if (!cfg || Object.keys(cfg).length === 0) {
      if (node.nodeType !== 'trigger' && node.nodeType !== 'complete') {
        warnings.push({ nodeId: node.id, message: 'Node not configured' });
      }
      continue;
    }

    switch (node.nodeType) {
      case 'decision': {
        const dc = cfg as DecisionConfig;
        if (!dc.field) warnings.push({ nodeId: node.id, message: 'Decision field not set' });
        if (!dc.value) warnings.push({ nodeId: node.id, message: 'Decision value not set' });
        break;
      }
      case 'filter': {
        const fc = cfg as FilterConfig;
        const hasAnyCriteria =
          (fc.os_types && fc.os_types.length > 0) ||
          (fc.tags && fc.tags.length > 0) ||
          fc.min_severity ||
          fc.package_regex;
        if (!hasAnyCriteria) {
          warnings.push({ nodeId: node.id, message: 'No filter criteria set' });
        }
        break;
      }
    }
  }

  return warnings;
}
