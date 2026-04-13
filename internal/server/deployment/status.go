package deployment

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// DeploymentStatus represents the state of a deployment.
type DeploymentStatus string

// Deployment statuses.
const (
	StatusCreated        DeploymentStatus = "created"
	StatusRunning        DeploymentStatus = "running"
	StatusCompleted      DeploymentStatus = "completed"
	StatusFailed         DeploymentStatus = "failed"
	StatusCancelled      DeploymentStatus = "cancelled"
	StatusScheduled      DeploymentStatus = "scheduled"
	StatusRollingBack    DeploymentStatus = "rolling_back"
	StatusRolledBack     DeploymentStatus = "rolled_back"
	StatusRollbackFailed DeploymentStatus = "rollback_failed"
)

// TargetStatus represents the state of a deployment target.
type TargetStatus string

// Deployment target statuses.
const (
	TargetPending   TargetStatus = "pending"
	TargetSent      TargetStatus = "sent"
	TargetExecuting TargetStatus = "executing"
	TargetSucceeded TargetStatus = "succeeded"
	TargetFailed    TargetStatus = "failed"
	TargetCancelled TargetStatus = "cancelled"
)

// CommandStatus represents the state of a command.
type CommandStatus string

// Command statuses.
const (
	CommandPending   CommandStatus = "pending"
	CommandDelivered CommandStatus = "delivered"
	CommandSucceeded CommandStatus = "succeeded"
	CommandFailed    CommandStatus = "failed"
	CommandCancelled CommandStatus = "cancelled"
)

// CommandType represents the type of command.
type CommandType string

// Command types.
const (
	CommandTypeInstallPatch  CommandType = "install_patch"
	CommandTypeRunScan       CommandType = "run_scan"
	CommandTypeUpdateConfig  CommandType = "update_config"
	CommandTypeReboot        CommandType = "reboot"
	CommandTypeRunScript     CommandType = "run_script"
	CommandTypeRollbackPatch CommandType = "rollback_patch"
)

// WaveStatus represents the state of a deployment wave.
type WaveStatus string

// Wave statuses.
const (
	WavePending   WaveStatus = "pending"
	WaveRunning   WaveStatus = "running"
	WaveCompleted WaveStatus = "completed"
	WaveFailed    WaveStatus = "failed"
	WaveCancelled WaveStatus = "cancelled"
)

// DefaultFailureThreshold is the failure rate (0.0–1.0) above which a deployment transitions to FAILED.
const DefaultFailureThreshold = 0.2

// thresholdQuerier is the minimal interface needed by checkDeploymentThreshold.
type thresholdQuerier interface {
	CompleteQuerier
	FailQuerier
}

// checkDeploymentThreshold checks if a deployment should transition to FAILED or COMPLETED
// based on its current counters. Returns pending domain events for the caller to emit
// post-commit. Shared between ResultHandler and TimeoutChecker.
func checkDeploymentThreshold(ctx context.Context, sm *StateMachine, q thresholdQuerier, d sqlcgen.Deployment, deployID, tenantID pgtype.UUID) ([]domain.DomainEvent, error) {
	if d.TotalTargets <= 0 || d.Status != string(StatusRunning) {
		return nil, nil
	}

	// Check failure threshold.
	failureRate := float64(d.FailedCount) / float64(d.TotalTargets)
	threshold := DefaultFailureThreshold
	ft, err := d.FailureThreshold.Float64Value()
	if err != nil {
		return nil, fmt.Errorf("check threshold: parse failure_threshold for deployment %v: %w", deployID, err)
	}
	if ft.Valid {
		threshold = ft.Float64
	}
	if failureRate > threshold {
		_, evts, err := sm.FailDeployment(ctx, q, deployID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("check threshold: fail deployment: %w", err)
		}
		return evts, nil
	}

	// Check completion.
	if d.CompletedCount >= d.TotalTargets {
		_, evts, err := sm.CompleteDeployment(ctx, q, deployID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("check threshold: complete deployment: %w", err)
		}
		return evts, nil
	}

	return nil, nil
}
