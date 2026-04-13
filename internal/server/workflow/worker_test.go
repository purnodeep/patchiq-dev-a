package workflow

import (
	"context"
	"testing"

	"github.com/riverqueue/river"
)

func TestWorkflowExecuteJobArgs_Kind(t *testing.T) {
	args := WorkflowExecuteJobArgs{ExecutionID: "exec-1", TenantID: "tenant-1"}
	if args.Kind() != "workflow_execute" {
		t.Errorf("Kind() = %q, want %q", args.Kind(), "workflow_execute")
	}
}

func TestApprovalTimeoutJobArgs_Kind(t *testing.T) {
	args := ApprovalTimeoutJobArgs{ExecutionID: "exec-1", NodeID: "node-1", TenantID: "tenant-1"}
	if args.Kind() != "approval_timeout" {
		t.Errorf("Kind() = %q, want %q", args.Kind(), "approval_timeout")
	}
}

func TestGateTimeoutJobArgs_Kind(t *testing.T) {
	args := GateTimeoutJobArgs{ExecutionID: "exec-1", NodeID: "node-1", TenantID: "tenant-1"}
	if args.Kind() != "gate_timeout" {
		t.Errorf("Kind() = %q, want %q", args.Kind(), "gate_timeout")
	}
}

func TestWorkflowExecuteWorker_ImplementsInterface(t *testing.T) {
	// Verify the worker satisfies river.Worker interface at compile time.
	var _ river.Worker[WorkflowExecuteJobArgs] = &WorkflowExecuteWorker{}
}

func TestWorkflowExecuteWorker_Work_ReturnsError(t *testing.T) {
	w := &WorkflowExecuteWorker{}
	job := &river.Job[WorkflowExecuteJobArgs]{
		Args: WorkflowExecuteJobArgs{
			ExecutionID: "exec-1",
			TenantID:    "tenant-1",
		},
	}

	err := w.Work(context.Background(), job)
	if err == nil {
		t.Fatal("Work() should return error for stub implementation")
	}
}
