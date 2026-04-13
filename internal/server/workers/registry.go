package workers

import (
	"log/slog"
	"reflect"

	"github.com/riverqueue/river"

	"github.com/skenzeriq/patchiq/internal/server/cve"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/discovery"
	"github.com/skenzeriq/patchiq/internal/server/notify"
	"github.com/skenzeriq/patchiq/internal/server/policy"
	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// addWorkerIfNotNil registers a worker with the River workers bundle.
// If the worker is nil, it logs a warning and skips registration.
func addWorkerIfNotNil[T river.JobArgs](workers *river.Workers, w river.Worker[T], kind string) {
	if w == nil || reflect.ValueOf(w).IsNil() {
		slog.Warn("river: skipping nil worker registration", "kind", kind)
		return
	}
	river.AddWorker(workers, w)
}

// RegisterWorkers returns a Workers bundle with all registered job workers.
func RegisterWorkers(
	discoveryWorker *discovery.DiscoveryWorker,
	nvdSyncWorker *cve.NVDSyncWorker,
	endpointMatchWorker *cve.EndpointMatchWorker,
	executorWorker *deployment.ExecutorWorker,
	timeoutWorker *deployment.TimeoutWorker,
	scanWorker *deployment.ScanWorker,
	notifySendWorker *notify.SendWorker,
	waveDispatcherWorker *deployment.WaveDispatcherWorker,
	scheduleCheckerWorker *deployment.ScheduleCheckerWorker,
	retentionWorker *AuditRetentionWorker,
	complianceEvalWorker *ComplianceEvalWorker,
	userSyncWorker *UserSyncWorker,
	catalogSyncWorker *CatalogSyncWorker,
	workflowExecWorker *workflow.WorkflowExecuteWorker,
	policySchedulerWorker *policy.PolicySchedulerWorker,
) *river.Workers {
	workers := river.NewWorkers()

	// Required workers — nil here is a programming error.
	river.AddWorker(workers, discoveryWorker)
	river.AddWorker(workers, nvdSyncWorker)
	river.AddWorker(workers, endpointMatchWorker)
	river.AddWorker(workers, executorWorker)
	river.AddWorker(workers, timeoutWorker)
	river.AddWorker(workers, scanWorker)
	river.AddWorker(workers, notifySendWorker)
	river.AddWorker(workers, waveDispatcherWorker)
	river.AddWorker(workers, scheduleCheckerWorker)
	river.AddWorker(workers, retentionWorker)
	river.AddWorker(workers, complianceEvalWorker)
	river.AddWorker(workers, catalogSyncWorker)
	river.AddWorker(workers, workflowExecWorker)

	// Optional workers — nil until their dependencies are implemented.
	addWorkerIfNotNil(workers, userSyncWorker, "user_sync")
	addWorkerIfNotNil(workers, policySchedulerWorker, "policy_scheduler")

	return workers
}
