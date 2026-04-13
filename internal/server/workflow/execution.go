package workflow

import "context"

// ExecutionStatus represents the lifecycle state of a workflow execution.
type ExecutionStatus string

const (
	ExecPending   ExecutionStatus = "pending"
	ExecRunning   ExecutionStatus = "running"
	ExecCompleted ExecutionStatus = "completed"
	ExecFailed    ExecutionStatus = "failed"
	ExecPaused    ExecutionStatus = "paused"
	ExecCancelled ExecutionStatus = "cancelled"
)

// NodeExecStatus represents the lifecycle state of a single node execution.
type NodeExecStatus string

const (
	NodeExecPending   NodeExecStatus = "pending"
	NodeExecRunning   NodeExecStatus = "running"
	NodeExecCompleted NodeExecStatus = "completed"
	NodeExecFailed    NodeExecStatus = "failed"
	NodeExecSkipped   NodeExecStatus = "skipped"
)

// ExecutionContext carries all context a node handler needs to execute.
type ExecutionContext struct {
	TenantID    string
	ExecutionID string
	VersionID   string
	WorkflowID  string
	Node        Node
	Context     map[string]any // Shared mutable execution context
}

// NodeResult is what a node handler returns after execution.
type NodeResult struct {
	Status NodeExecStatus
	Output map[string]any
	Pause  bool   // If true, executor saves state and exits
	Error  string // Non-empty on failure
}

// NodeHandler is the interface that all node type handlers implement.
type NodeHandler interface {
	Execute(ctx context.Context, exec *ExecutionContext) (*NodeResult, error)
}
