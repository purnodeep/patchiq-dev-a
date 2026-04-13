package workflow

import (
	"context"
	"fmt"
	"maps"
)

// ExecutionResult is the outcome of running the DAG executor.
type ExecutionResult struct {
	Status         ExecutionStatus
	NodeResults    map[string]*NodeResult
	PausedAtNodeID string
	Error          string
	Context        map[string]any
}

// InMemoryExecutor walks the DAG and executes nodes via handlers.
type InMemoryExecutor struct {
	handlers map[NodeType]NodeHandler
}

// NewInMemoryExecutor creates an executor with the given node handlers.
func NewInMemoryExecutor(handlers map[NodeType]NodeHandler) *InMemoryExecutor {
	return &InMemoryExecutor{handlers: handlers}
}

// Run executes the workflow DAG from the trigger node.
func (e *InMemoryExecutor) Run(ctx context.Context, nodes []Node, edges []Edge, execCtx map[string]any) (*ExecutionResult, error) {
	return e.RunFrom(ctx, nodes, edges, execCtx, "", nil)
}

// RunFrom executes the workflow DAG, optionally resuming from a specific node.
func (e *InMemoryExecutor) RunFrom(ctx context.Context, nodes []Node, edges []Edge, execCtx map[string]any, resumeAfterNodeID string, completedNodes map[string]bool) (*ExecutionResult, error) {
	graph, err := BuildGraph(nodes, edges)
	if err != nil {
		return nil, fmt.Errorf("executor: %w", err)
	}

	order := graph.TopologicalOrder()
	if execCtx == nil {
		execCtx = make(map[string]any)
	}

	result := &ExecutionResult{
		Status:      ExecRunning,
		NodeResults: make(map[string]*NodeResult),
		Context:     execCtx,
	}

	skippedNodes := make(map[string]bool)
	if completedNodes == nil {
		completedNodes = make(map[string]bool)
	}

	resuming := resumeAfterNodeID != ""
	pastResumePoint := !resuming

	for _, nodeID := range order {
		if !pastResumePoint {
			if nodeID == resumeAfterNodeID {
				pastResumePoint = true
			}
			continue
		}

		if completedNodes[nodeID] || skippedNodes[nodeID] {
			continue
		}

		node := graph.Nodes[nodeID]

		handler, ok := e.handlers[node.NodeType]
		if !ok {
			return nil, fmt.Errorf("executor: no handler for node type %q", node.NodeType)
		}

		nodeExecCtx := &ExecutionContext{
			Node:    node,
			Context: execCtx,
		}

		nodeResult, execErr := handler.Execute(ctx, nodeExecCtx)
		if execErr != nil {
			result.Status = ExecFailed
			result.Error = fmt.Sprintf("node %q (%s): %v", nodeID, node.NodeType, execErr)
			result.NodeResults[nodeID] = &NodeResult{
				Status: NodeExecFailed,
				Error:  execErr.Error(),
			}
			return result, nil
		}

		result.NodeResults[nodeID] = nodeResult

		maps.Copy(execCtx, nodeResult.Output)

		if nodeResult.Pause {
			result.Status = ExecPaused
			result.PausedAtNodeID = nodeID
			return result, nil
		}

		if nodeResult.Status == NodeExecFailed {
			result.Status = ExecFailed
			result.Error = fmt.Sprintf("node %q (%s): %s", nodeID, node.NodeType, nodeResult.Error)
			return result, nil
		}

		if node.NodeType == NodeDecision {
			branch, _ := nodeResult.Output["branch"].(string)
			yesID, noID := graph.DecisionBranches(nodeID)

			var skipRoot, keepRoot string
			if branch == "yes" {
				skipRoot = noID
				keepRoot = yesID
			} else {
				skipRoot = yesID
				keepRoot = noID
			}

			if skipRoot != "" && keepRoot != "" {
				for _, skippedID := range graph.SkippedDescendants(nodeID, skipRoot, keepRoot) {
					skippedNodes[skippedID] = true
					result.NodeResults[skippedID] = &NodeResult{Status: NodeExecSkipped}
				}
			}
		}
	}

	result.Status = ExecCompleted
	result.Context = execCtx
	return result, nil
}
