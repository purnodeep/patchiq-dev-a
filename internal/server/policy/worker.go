package policy

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// PolicySchedulerJobArgs are the arguments for the policy scheduler periodic River job.
type PolicySchedulerJobArgs struct{}

// Kind implements river.JobArgs.
func (PolicySchedulerJobArgs) Kind() string { return "policy_scheduler" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (PolicySchedulerJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "default"}
}

// PolicySchedulerWorker wraps PolicyScheduler as a River worker.
type PolicySchedulerWorker struct {
	river.WorkerDefaults[PolicySchedulerJobArgs]
	scheduler *PolicyScheduler
}

// NewPolicySchedulerWorker creates a PolicySchedulerWorker.
func NewPolicySchedulerWorker(scheduler *PolicyScheduler) *PolicySchedulerWorker {
	if scheduler == nil {
		panic("policy: NewPolicySchedulerWorker called with nil PolicyScheduler")
	}
	return &PolicySchedulerWorker{scheduler: scheduler}
}

// Work runs the policy scheduler.
func (w *PolicySchedulerWorker) Work(ctx context.Context, _ *river.Job[PolicySchedulerJobArgs]) error {
	slog.InfoContext(ctx, "policy scheduler job: starting")
	if err := w.scheduler.Run(ctx, time.Now()); err != nil {
		slog.ErrorContext(ctx, "policy scheduler job: failed", "error", err)
		return err
	}
	slog.InfoContext(ctx, "policy scheduler job: completed")
	return nil
}

// DomainEventEmitter implements SchedulerEventEmitter using the domain event bus.
type DomainEventEmitter struct {
	eventBus domain.EventBus
}

// NewDomainEventEmitter creates a DomainEventEmitter.
func NewDomainEventEmitter(eventBus domain.EventBus) *DomainEventEmitter {
	if eventBus == nil {
		panic("policy: NewDomainEventEmitter called with nil EventBus")
	}
	return &DomainEventEmitter{eventBus: eventBus}
}

// EmitPolicyAutoDeployed emits a policy.auto_deployed domain event.
func (e *DomainEventEmitter) EmitPolicyAutoDeployed(ctx context.Context, tenantID, policyID, deploymentID string, newPatchCount int) {
	evt := domain.DomainEvent{
		ID:         domain.NewEventID(),
		Type:       events.PolicyAutoDeployed,
		TenantID:   tenantID,
		ActorID:    "system",
		ActorType:  domain.ActorSystem,
		Resource:   "policy",
		ResourceID: policyID,
		Action:     events.PolicyAutoDeployed,
		Payload: map[string]any{
			"policy_id":       policyID,
			"deployment_id":   deploymentID,
			"new_patch_count": newPatchCount,
		},
		Timestamp: time.Now(),
	}
	if err := e.eventBus.Emit(ctx, evt); err != nil {
		slog.ErrorContext(ctx, "policy scheduler: emit auto_deployed event failed",
			"policy_id", policyID, "error", err)
	}
}
