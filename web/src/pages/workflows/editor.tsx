import { useState, useEffect, useMemo, useRef, useCallback } from 'react';
import { useParams, useNavigate, Link, useBlocker } from 'react-router';
import { toast } from 'sonner';
import {
  ReactFlowProvider,
  useNodesState,
  useEdgesState,
  useReactFlow,
  type Node,
  type Edge,
  type OnNodesChange,
  type OnEdgesChange,
} from '@xyflow/react';
import {
  LayoutGrid,
  Save,
  AlertCircle,
  CheckCircle2,
  Clock,
  CircleCheckBig,
  Plus,
  Undo2,
  Redo2,
  Loader2,
  HelpCircle,
} from 'lucide-react';
import { Palette } from '../../flows/policy-workflow/palette';
import { WorkflowCanvas } from '../../flows/policy-workflow/canvas';
import { ConfigPanel } from '../../flows/policy-workflow/panels/config-panel';
import { KeyboardHelp } from '../../flows/policy-workflow/keyboard-help';
import {
  useWorkflow,
  useCreateWorkflow,
  useUpdateWorkflow,
  usePublishWorkflow,
  useWorkflowTemplates,
} from '../../flows/policy-workflow/hooks/use-workflows';
import { useCan } from '../../app/auth/AuthContext';
import { computeLayout } from '../../flows/policy-workflow/hooks/use-elk-layout';
import { uuid } from '../../lib/uuid';
import { useUndoRedo } from '../../flows/policy-workflow/hooks/use-undo-redo';
import type {
  NodeConfig,
  DecisionConfig,
  WorkflowNodeRequest,
  WorkflowEdgeRequest,
} from '../../flows/policy-workflow/types';
import type { WorkflowNodeData } from '../../flows/policy-workflow/nodes/workflow-node';
import {
  validateWorkflowDAG,
  validateNodeConfigs,
} from '../../flows/policy-workflow/dag-validation';

/** Helper to read WorkflowNodeData from a generic ReactFlow Node */
// eslint-disable-next-line @typescript-eslint/no-explicit-any -- ReactFlow Node.data is Record<string, unknown>, we know it's WorkflowNodeData
function nodeData(node: { data: any }): WorkflowNodeData {
  return node.data as WorkflowNodeData;
}

/** Map an API workflow node to a ReactFlow Node */
function toFlowNode(n: {
  id: string;
  node_type: string;
  label: string;
  position_x: number;
  position_y: number;
  config: NodeConfig;
}): Node {
  return {
    id: n.id,
    type: 'workflowNode',
    position: { x: n.position_x, y: n.position_y },
    data: { nodeType: n.node_type, label: n.label, config: n.config } as unknown as Record<
      string,
      unknown
    >,
  };
}

/** Map an API workflow edge to a ReactFlow Edge */
function toFlowEdge(
  e: { id?: string; source_node_id: string; target_node_id: string; label: string },
  index?: number,
): Edge {
  const hasLabel = e.label && e.label.length > 0;
  return {
    id: e.id ?? `edge_${index}`,
    type: 'deletable',
    source: e.source_node_id,
    target: e.target_node_id,
    label: e.label,
    ...(hasLabel
      ? {
          labelStyle: { fill: 'var(--text-secondary)', fontSize: 10, fontWeight: 500 },
          labelShowBg: true,
          labelBgStyle: { fill: 'var(--bg-card)', stroke: 'var(--border)', strokeWidth: 1 },
          labelBgPadding: [4, 6] as [number, number],
          labelBgBorderRadius: 4,
        }
      : {}),
    style: { stroke: 'var(--border-strong)', strokeWidth: 2 },
  };
}

function EditorInner() {
  const can = useCan();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEditMode = !!id;
  const { fitView } = useReactFlow();
  const nameInputRef = useRef<HTMLInputElement>(null);
  const [nameError, setNameError] = useState(false);

  const { data: workflow, isLoading, isError, refetch } = useWorkflow(id ?? '');
  const { data: templates, isError: templatesError } = useWorkflowTemplates();
  const createWorkflow = useCreateWorkflow();
  const updateWorkflow = useUpdateWorkflow(id ?? '');
  const publishWorkflow = usePublishWorkflow(id ?? '');

  const defaultNodes = useMemo<Node[]>(() => {
    if (isEditMode) return [];
    return [
      {
        id: uuid(),
        type: 'workflowNode',
        position: { x: 250, y: 100 },
        data: { nodeType: 'trigger', label: 'Trigger', config: {} },
      },
    ];
  }, []);

  const [nodes, setNodes, onNodesChangeRaw] = useNodesState(defaultNodes);
  const [edges, setEdges, onEdgesChangeRaw] = useEdgesState([] as Edge[]);
  const onNodesChange = onNodesChangeRaw as unknown as OnNodesChange;
  const onEdgesChange = onEdgesChangeRaw as unknown as OnEdgesChange;
  const { takeSnapshot, undo, redo, canUndo, canRedo } = useUndoRedo(setNodes, setEdges);
  const [name, setName] = useState(workflow?.name ?? '');
  const [description, setDescription] = useState(workflow?.description ?? '');
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [paletteCollapsed, setPaletteCollapsed] = useState(false);
  const [panelOpen, setPanelOpen] = useState(false);
  const [showKeyboardHelp, setShowKeyboardHelp] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [lastSaved, setLastSaved] = useState<Date | null>(null);
  const [isDirty, setIsDirty] = useState(false);
  const [isLayouting, setIsLayouting] = useState(false);
  const initialLoadRef = useRef(true);
  const validationTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  // Snapshot for undo before canvas mutations
  const onBeforeChange = useCallback(() => {
    takeSnapshot(nodes, edges);
  }, [takeSnapshot, nodes, edges]);

  // Undo/redo keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)
        return;
      const isMod = e.metaKey || e.ctrlKey;
      if (e.key === '?' && !isMod) {
        setShowKeyboardHelp(true);
        return;
      }
      if (isMod && e.key === 'z' && !e.shiftKey) {
        e.preventDefault();
        undo();
        return;
      }
      if (isMod && ((e.key === 'z' && e.shiftKey) || e.key === 'y')) {
        e.preventDefault();
        redo();
        return;
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [undo, redo]);

  useEffect(() => {
    if (!workflow) return;
    if (workflow.nodes) setNodes(workflow.nodes.map(toFlowNode));
    if (workflow.edges) setEdges(workflow.edges.map((e, i) => toFlowEdge(e, i)));
    setName(workflow.name);
    setDescription(workflow.description ?? '');
    // Let ReactFlow render the nodes before fitting the view
    if (workflow.nodes && workflow.nodes.length > 0) {
      setTimeout(() => fitView({ padding: 0.2, duration: 300 }), 100);
    }
  }, [workflow, setNodes, setEdges, fitView]);

  useEffect(() => {
    document.title = name ? `Edit ${name} — PatchIQ` : 'New Workflow — PatchIQ';
    return () => {
      document.title = 'PatchIQ';
    };
  }, [name]);

  useEffect(() => {
    if (initialLoadRef.current) {
      initialLoadRef.current = false;
      return;
    }
    if (isEditMode) setIsDirty(true);
  }, [nodes.length, edges.length, name, description]);

  // Debounced inline validation warnings
  useEffect(() => {
    clearTimeout(validationTimerRef.current);
    validationTimerRef.current = setTimeout(() => {
      const warnings = validateNodeConfigs(
        nodes.map((n) => ({
          id: n.id,
          nodeType: nodeData(n).nodeType,
          config: nodeData(n).config,
        })),
      );
      const warningMap = new Map(warnings.map((w) => [w.nodeId, w.message]));
      setNodes((nds) =>
        nds.map((n) => {
          const warning = warningMap.get(n.id);
          const current = nodeData(n).validationWarning;
          if (warning === current) return n;
          return { ...n, data: { ...n.data, validationWarning: warning } };
        }),
      );
    }, 500);
    return () => clearTimeout(validationTimerRef.current);
  }, [nodes, setNodes]);

  useEffect(() => {
    if (!isDirty) return;
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
    };
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [isDirty]);

  const blocker = useBlocker(
    useCallback(
      ({
        currentLocation,
        nextLocation,
      }: {
        currentLocation: { pathname: string };
        nextLocation: { pathname: string };
      }) => isDirty && currentLocation.pathname !== nextLocation.pathname,
      [isDirty],
    ),
  );

  useEffect(() => {
    if (blocker.state === 'blocked') {
      const leave = window.confirm('You have unsaved changes. Leave this page?');
      if (leave) {
        blocker.proceed();
      } else {
        blocker.reset();
      }
    }
  }, [blocker]);

  if (isEditMode && isLoading) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: 'calc(100vh - 64px)',
          color: 'var(--text-muted)',
          fontSize: 13,
        }}
      >
        Loading workflow…
      </div>
    );
  }

  if (isEditMode && isError) {
    return (
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          height: 'calc(100vh - 64px)',
          gap: 12,
        }}
      >
        <p style={{ fontSize: 13, color: 'var(--signal-critical)' }}>Failed to load workflow.</p>
        <button
          onClick={() => refetch()}
          style={{
            padding: '5px 12px',
            borderRadius: 6,
            border: '1px solid var(--border)',
            background: 'none',
            color: 'var(--text-secondary)',
            fontSize: 12,
            cursor: 'pointer',
          }}
        >
          Retry
        </button>
      </div>
    );
  }

  const onNodeClick = (_: React.MouseEvent, node: Node) => {
    setSelectedNode(node);
    setPanelOpen(true);
  };

  const onConfigSave = (config: NodeConfig) => {
    if (!selectedNode) return;
    takeSnapshot(nodes, edges);
    setNodes((nds) =>
      nds.map((n) => (n.id === selectedNode.id ? { ...n, data: { ...n.data, config } } : n)),
    );
    // Update edge labels when saving a decision node
    const nd = nodeData(selectedNode);
    if (nd.nodeType === 'decision') {
      const dc = config as DecisionConfig;
      const outgoing = edges.filter((e) => e.source === selectedNode.id);
      if (outgoing.length > 0) {
        const labelStyle = { fill: 'var(--text-secondary)', fontSize: 10, fontWeight: 500 };
        const labelBgStyle = { fill: 'var(--bg-card)', stroke: 'var(--border)', strokeWidth: 1 };
        setEdges((eds) =>
          eds.map((e) => {
            if (e.source !== selectedNode.id) return e;
            const idx = outgoing.findIndex((o) => o.id === e.id);
            const label = idx === 0 ? dc.true_label || 'Yes' : dc.false_label || 'No';
            return {
              ...e,
              label,
              labelStyle,
              labelShowBg: true,
              labelBgStyle,
              labelBgPadding: [4, 6] as [number, number],
              labelBgBorderRadius: 4,
            };
          }),
        );
      }
    }
    setPanelOpen(false);
  };

  const handleSave = async () => {
    setSaveError(null);
    setNameError(false);

    if (!name.trim()) {
      toast.error('Workflow name is required');
      setNameError(true);
      nameInputRef.current?.focus();
      return;
    }

    const validationErrors = validateWorkflowDAG(
      nodes.map((n) => ({ id: n.id, nodeType: nodeData(n).nodeType })),
      edges.map((e) => ({ source: e.source, target: e.target })),
    );

    if (validationErrors.length > 0) {
      setNodes((nds) =>
        nds.map((n) => {
          const nodeError = validationErrors.find((e) => e.nodeId === n.id);
          return { ...n, data: { ...n.data, validationError: nodeError?.message } };
        }),
      );
      const globalErrors = validationErrors.filter((e) => !e.nodeId);
      setSaveError(
        globalErrors.map((e) => e.message).join('. ') || 'Workflow has validation errors',
      );
      return;
    }

    setNodes((nds) => nds.map((n) => ({ ...n, data: { ...n.data, validationError: undefined } })));

    try {
      const nodeRequests: WorkflowNodeRequest[] = nodes.map((n) => ({
        id: n.id,
        node_type: nodeData(n).nodeType,
        label: nodeData(n).label,
        position_x: n.position.x,
        position_y: n.position.y,
        config: nodeData(n).config,
      }));

      const edgeRequests: WorkflowEdgeRequest[] = edges.map((e) => ({
        source_node_id: e.source,
        target_node_id: e.target,
        label: typeof e.label === 'string' ? e.label : '',
      }));

      const body = {
        name: name.trim(),
        description,
        nodes: nodeRequests,
        edges: edgeRequests,
      };

      if (isEditMode) {
        await updateWorkflow.mutateAsync(body);
        setLastSaved(new Date());
        setIsDirty(false);
      } else {
        const result = await createWorkflow.mutateAsync(body);
        setIsDirty(false);
        navigate(`/workflows/${result.id}/edit`);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to save workflow';
      setSaveError(message);
    }
  };

  const handleAutoLayout = async () => {
    setSaveError(null);
    takeSnapshot(nodes, edges);
    setIsLayouting(true);
    try {
      const result = await computeLayout(nodes as Node[], edges as Edge[]);
      setNodes(result.nodes);
    } catch (err) {
      const detail = err instanceof Error ? err.message : 'Unknown error';
      setSaveError(`Auto layout failed: ${detail}`);
    } finally {
      setIsLayouting(false);
    }
  };

  const handleLoadTemplate = (templateId: string) => {
    const template = templates?.find((t) => t.id === templateId);
    if (!template) return;
    if (
      nodes.length > 0 &&
      !window.confirm('Load template? This will replace your current workflow.')
    )
      return;
    takeSnapshot(nodes, edges);
    setNodes(template.nodes.map(toFlowNode));
    setEdges(template.edges.map((e, i) => toFlowEdge(e, i)));
    setName(template.name);
  };

  const isSaving = createWorkflow.isPending || updateWorkflow.isPending;
  const nodeCount = nodes.length;
  const edgeCount = edges.length;
  const hasErrors = nodes.some((n) => nodeData(n).validationError);

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: 'calc(100vh - 64px)',
        background: 'var(--bg-page)',
      }}
    >
      <nav
        aria-label="Breadcrumb"
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          padding: '6px 16px',
          fontSize: 12,
          borderBottom: '1px solid var(--border)',
          background: 'var(--bg-card)',
        }}
      >
        <Link to="/workflows" style={{ color: 'var(--text-muted)', textDecoration: 'none' }}>
          Workflows
        </Link>
        <span style={{ color: 'var(--text-faint)' }}>/</span>
        <span style={{ color: 'var(--text-secondary)', fontWeight: 500 }}>
          {name || 'New Workflow'}
        </span>
      </nav>

      {/* Toolbar */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          padding: '8px 16px',
          borderBottom: '1px solid var(--border)',
          background: 'var(--bg-card)',
          height: 52,
          flexShrink: 0,
        }}
      >
        {/* Name input */}
        <input
          ref={nameInputRef}
          id="workflow-name"
          name="workflow-name"
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            setNameError(false);
          }}
          placeholder="Workflow name"
          onFocus={(e) => {
            if (!nameError) {
              e.currentTarget.style.borderColor = 'var(--accent)';
              e.currentTarget.style.boxShadow = '0 0 0 1px var(--accent)';
            }
          }}
          onBlur={(e) => {
            if (!nameError) {
              e.currentTarget.style.borderColor = 'var(--border)';
              e.currentTarget.style.boxShadow = 'none';
            }
          }}
          style={{
            width: 240,
            padding: '5px 10px',
            borderRadius: 6,
            border: nameError ? '1px solid var(--signal-critical)' : '1px solid var(--border)',
            background: 'var(--bg-inset)',
            color: 'var(--text-emphasis)',
            fontSize: 13,
            fontWeight: 500,
            outline: 'none',
            minWidth: 0,
          }}
        />

        {/* Template selector */}
        {!templatesError && templates && templates.length > 0 && (
          <select
            onChange={(e) => {
              if (e.target.value) handleLoadTemplate(e.target.value);
            }}
            defaultValue=""
            style={{
              padding: '5px 10px',
              borderRadius: 6,
              border: '1px solid var(--border)',
              background: 'var(--bg-inset)',
              color: 'var(--text-secondary)',
              fontSize: 12,
              outline: 'none',
              cursor: 'pointer',
            }}
          >
            <option value="" disabled>
              Load template
            </option>
            {templates.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name}
                {t.description ? ` — ${t.description}` : ''}
              </option>
            ))}
          </select>
        )}
        {templatesError && (
          <span style={{ fontSize: 11, color: 'var(--signal-critical)' }}>
            Templates unavailable
          </span>
        )}

        {/* Right side */}
        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 6 }}>
          {/* Save status */}
          {saveError ? (
            <span
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 4,
                fontSize: 12,
                color: 'var(--signal-critical)',
              }}
            >
              <AlertCircle style={{ width: 12, height: 12 }} />
              {saveError}
            </span>
          ) : lastSaved ? (
            <span
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 4,
                fontSize: 11,
                color: 'var(--text-muted)',
                fontFamily: 'var(--font-mono)',
              }}
            >
              <CheckCircle2 style={{ width: 11, height: 11, color: 'var(--signal-healthy)' }} />
              Saved
            </span>
          ) : (
            <span
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 5,
                fontSize: 11,
                color: 'var(--signal-warning)',
                fontFamily: 'var(--font-mono)',
                fontWeight: 600,
              }}
            >
              <span
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  background: 'var(--signal-warning)',
                  flexShrink: 0,
                }}
              />
              Unsaved
            </span>
          )}

          {/* Help */}
          <button
            onClick={() => setShowKeyboardHelp(true)}
            title="Keyboard shortcuts (?)"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: 30,
              height: 30,
              borderRadius: 6,
              border: '1px solid var(--border)',
              background: 'none',
              color: 'var(--text-muted)',
              cursor: 'pointer',
            }}
          >
            <HelpCircle style={{ width: 13, height: 13 }} />
          </button>

          {/* Undo/Redo */}
          <button
            onClick={undo}
            disabled={!canUndo()}
            title="Undo (Ctrl+Z)"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: 30,
              height: 30,
              borderRadius: 6,
              border: '1px solid var(--border)',
              background: 'none',
              color: canUndo() ? 'var(--text-secondary)' : 'var(--text-faint)',
              cursor: canUndo() ? 'pointer' : 'default',
              opacity: canUndo() ? 1 : 0.5,
            }}
          >
            <Undo2 style={{ width: 13, height: 13 }} />
          </button>
          <button
            onClick={redo}
            disabled={!canRedo()}
            title="Redo (Ctrl+Shift+Z)"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: 30,
              height: 30,
              borderRadius: 6,
              border: '1px solid var(--border)',
              background: 'none',
              color: canRedo() ? 'var(--text-secondary)' : 'var(--text-faint)',
              cursor: canRedo() ? 'pointer' : 'default',
              opacity: canRedo() ? 1 : 0.5,
            }}
          >
            <Redo2 style={{ width: 13, height: 13 }} />
          </button>

          <button
            onClick={handleAutoLayout}
            disabled={isLayouting}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              padding: '5px 10px',
              borderRadius: 6,
              border: '1px solid var(--border)',
              background: 'none',
              color: 'var(--text-secondary)',
              fontSize: 12,
              cursor: isLayouting ? 'not-allowed' : 'pointer',
              opacity: isLayouting ? 0.7 : 1,
            }}
          >
            {isLayouting ? (
              <Loader2 style={{ width: 13, height: 13, animation: 'spin 1s linear infinite' }} />
            ) : (
              <LayoutGrid style={{ width: 13, height: 13 }} />
            )}
            {isLayouting ? 'Laying out\u2026' : 'Auto Layout'}
          </button>

          <button
            onClick={handleSave}
            disabled={isSaving || !can('workflows', 'execute')}
            title={!can('workflows', 'execute') ? "You don't have permission" : undefined}
            onMouseEnter={(e) => {
              if (!isSaving) e.currentTarget.style.filter = 'brightness(1.2)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.filter = 'none';
            }}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              padding: '5px 12px',
              borderRadius: 7,
              border: 'none',
              background: 'var(--accent)',
              color: 'var(--text-on-color, #fff)',
              fontSize: 12,
              fontWeight: 600,
              cursor: isSaving ? 'not-allowed' : 'pointer',
              opacity: isSaving ? 0.7 : 1,
              transition: 'filter 0.15s',
            }}
          >
            <Save style={{ width: 13, height: 13 }} />
            {isSaving ? 'Saving…' : 'Save'}
          </button>

          {isEditMode && workflow?.version?.status === 'draft' && (
            <button
              onClick={() => {
                if (nodes.length < 2) {
                  toast.error('Add at least 2 nodes before publishing');
                  return;
                }
                publishWorkflow.mutate();
              }}
              disabled={publishWorkflow.isPending || !can('workflows', 'execute')}
              title={!can('workflows', 'execute') ? "You don't have permission" : undefined}
              onMouseEnter={(e) => {
                if (!publishWorkflow.isPending) e.currentTarget.style.filter = 'brightness(1.2)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.filter = 'none';
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '5px 12px',
                borderRadius: 7,
                border: 'none',
                background: 'var(--accent)',
                color: 'var(--text-on-color, #fff)',
                fontSize: 12,
                fontWeight: 600,
                cursor: publishWorkflow.isPending ? 'not-allowed' : 'pointer',
                opacity: publishWorkflow.isPending ? 0.7 : 1,
                transition: 'filter 0.15s',
              }}
            >
              <CircleCheckBig style={{ width: 13, height: 13 }} />
              {publishWorkflow.isPending ? 'Publishing…' : 'Publish'}
            </button>
          )}
        </div>
      </div>

      {/* Main area */}
      <div style={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
        <Palette collapsed={paletteCollapsed} onToggle={() => setPaletteCollapsed((c) => !c)} />
        <div style={{ flex: 1, position: 'relative' }}>
          <WorkflowCanvas
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={onNodeClick}
            onBeforeChange={onBeforeChange}
          />

          {/* First-use onboarding hint */}
          {!isEditMode && nodes.length <= 1 && (
            <div
              style={{
                position: 'absolute',
                top: '50%',
                left: '50%',
                transform: 'translate(-50%, -50%)',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                gap: 10,
                pointerEvents: 'none',
                zIndex: 1,
                padding: '32px 48px',
                border: '2px dashed var(--border)',
                borderRadius: 12,
                background: 'color-mix(in srgb, var(--bg-card) 60%, transparent)',
              }}
            >
              <Plus
                style={{ width: 36, height: 36, color: 'var(--text-secondary)', opacity: 0.7 }}
              />
              <p
                style={{ fontSize: 16, color: 'var(--text-secondary)', margin: 0, fontWeight: 500 }}
              >
                Drag a Trigger node from the palette to get started
              </p>
            </div>
          )}

          {/* Status bar */}
          <div
            style={{
              position: 'absolute',
              bottom: 12,
              left: '50%',
              transform: 'translateX(-50%)',
              display: 'flex',
              alignItems: 'center',
              gap: 10,
              padding: '5px 14px',
              borderRadius: 20,
              border: '1px solid var(--border)',
              background: 'var(--bg-card)',
              fontSize: 11,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-muted)',
              pointerEvents: 'none',
              boxShadow: 'var(--shadow-sm)',
            }}
          >
            <span>
              {nodeCount} node{nodeCount !== 1 ? 's' : ''}
            </span>
            <span style={{ color: 'var(--border)' }}>·</span>
            <span>
              {edgeCount} connection{edgeCount !== 1 ? 's' : ''}
            </span>
            <span style={{ color: 'var(--border)' }}>·</span>
            {hasErrors ? (
              <span
                style={{
                  color: 'var(--signal-critical)',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 3,
                }}
              >
                <AlertCircle style={{ width: 10, height: 10 }} />
                Errors
              </span>
            ) : (
              <span
                style={{
                  color: 'var(--signal-healthy)',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 3,
                }}
              >
                <CheckCircle2 style={{ width: 10, height: 10 }} />
                Valid
              </span>
            )}
            {lastSaved && (
              <>
                <span style={{ color: 'var(--border)' }}>·</span>
                <span style={{ display: 'flex', alignItems: 'center', gap: 3 }}>
                  <Clock style={{ width: 10, height: 10 }} />
                  Saved
                </span>
              </>
            )}
          </div>
        </div>
      </div>

      {selectedNode && (
        <ConfigPanel
          nodeType={nodeData(selectedNode).nodeType}
          nodeLabel={nodeData(selectedNode).label}
          config={nodeData(selectedNode).config}
          open={panelOpen}
          onClose={() => setPanelOpen(false)}
          onSave={onConfigSave}
        />
      )}
      <KeyboardHelp open={showKeyboardHelp} onClose={() => setShowKeyboardHelp(false)} />
    </div>
  );
}

export function WorkflowEditorPage() {
  return (
    <ReactFlowProvider>
      <EditorInner />
    </ReactFlowProvider>
  );
}
