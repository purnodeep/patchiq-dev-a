package workers_test

import (
	"testing"

	"github.com/riverqueue/river"

	"github.com/skenzeriq/patchiq/internal/server/cve"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/discovery"
	"github.com/skenzeriq/patchiq/internal/server/notify"
	"github.com/skenzeriq/patchiq/internal/server/policy"
	"github.com/skenzeriq/patchiq/internal/server/workers"
	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// insertOptsGetter is satisfied by JobArgs that implement InsertOpts().
type insertOptsGetter interface {
	InsertOpts() river.InsertOpts
}

func TestCriticalQueueAssignment(t *testing.T) {
	criticalJobs := []struct {
		name string
		args river.JobArgs
	}{
		{"deployment_executor", deployment.ExecutorJobArgs{}},
		{"deployment_timeout", deployment.TimeoutJobArgs{}},
		{"wave_dispatcher", deployment.WaveDispatcherJobArgs{}},
		{"schedule_checker", deployment.ScheduleCheckerJobArgs{}},
		{"scan_scheduler", deployment.ScanJobArgs{}},
		{"approval_timeout", workflow.ApprovalTimeoutJobArgs{}},
		{"gate_timeout", workflow.GateTimeoutJobArgs{}},
	}

	for _, tc := range criticalJobs {
		t.Run(tc.name, func(t *testing.T) {
			getter, ok := tc.args.(insertOptsGetter)
			if !ok {
				t.Fatalf("%s does not implement InsertOpts()", tc.name)
			}
			opts := getter.InsertOpts()
			if opts.Queue != workers.QueueCritical {
				t.Errorf("expected queue %q, got %q", workers.QueueCritical, opts.Queue)
			}
		})
	}
}

func TestDefaultQueueAssignment(t *testing.T) {
	defaultJobs := []struct {
		name string
		args river.JobArgs
	}{
		{"notification_send", notify.SendJobArgs{}},
		{"compliance_eval", workers.ComplianceEvalJobArgs{}},
		{"nvd_sync", cve.NVDSyncJobArgs{}},
		{"endpoint_match", cve.EndpointMatchJobArgs{}},
		{"catalog_sync", workers.CatalogSyncJobArgs{}},
		{"workflow_execute", workflow.WorkflowExecuteJobArgs{}},
		{"policy_scheduler", policy.PolicySchedulerJobArgs{}},
		{"user_sync", workers.UserSyncJobArgs{}},
	}

	for _, tc := range defaultJobs {
		t.Run(tc.name, func(t *testing.T) {
			getter, ok := tc.args.(insertOptsGetter)
			if !ok {
				t.Fatalf("%s does not implement InsertOpts()", tc.name)
			}
			opts := getter.InsertOpts()
			if opts.Queue != workers.QueueDefault {
				t.Errorf("expected queue %q, got %q", workers.QueueDefault, opts.Queue)
			}
		})
	}
}

func TestBackgroundQueueAssignment(t *testing.T) {
	bgJobs := []struct {
		name string
		args river.JobArgs
	}{
		{"discovery", discovery.DiscoveryJobArgs{}},
		{"audit_retention", workers.AuditRetentionJobArgs{}},
	}

	for _, tc := range bgJobs {
		t.Run(tc.name, func(t *testing.T) {
			getter, ok := tc.args.(insertOptsGetter)
			if !ok {
				t.Fatalf("%s does not implement InsertOpts()", tc.name)
			}
			opts := getter.InsertOpts()
			if opts.Queue != workers.QueueBackground {
				t.Errorf("expected queue %q, got %q", workers.QueueBackground, opts.Queue)
			}
		})
	}
}
