package cve

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// NVDSyncJobArgs are the arguments for an NVD CVE sync job.
type NVDSyncJobArgs struct {
	TenantID string `json:"tenant_id"`
}

// Kind returns the unique job kind identifier.
func (NVDSyncJobArgs) Kind() string { return "cve_nvd_sync" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (NVDSyncJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "default"}
}

// EndpointMatchJobArgs are the arguments for a CVE endpoint matching job.
type EndpointMatchJobArgs struct {
	TenantID   string `json:"tenant_id"`
	EndpointID string `json:"endpoint_id"`
}

// Kind returns the unique job kind identifier.
func (EndpointMatchJobArgs) Kind() string { return "cve_endpoint_match" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (EndpointMatchJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "default"}
}

// SyncService abstracts the NVD sync operation for the worker.
type SyncService interface {
	SyncNVD(ctx context.Context, tenantID string) error
}

// EndpointIDLister lists non-decommissioned endpoint IDs for a tenant.
type EndpointIDLister interface {
	ListEndpointIDs(ctx context.Context, tenantID string) ([]string, error)
}

// JobInserter enqueues River jobs.
type JobInserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// NVDSyncWorker processes NVD CVE sync jobs via River.
// After a successful sync, it enqueues endpoint match jobs for all active endpoints.
type NVDSyncWorker struct {
	river.WorkerDefaults[NVDSyncJobArgs]
	syncSvc     SyncService
	epLister    EndpointIDLister
	jobInserter JobInserter
}

// NewNVDSyncWorker creates an NVDSyncWorker with the given sync service.
func NewNVDSyncWorker(syncSvc SyncService) *NVDSyncWorker {
	return &NVDSyncWorker{syncSvc: syncSvc}
}

// WithPostSyncMatching configures the worker to enqueue endpoint match jobs after sync.
func (w *NVDSyncWorker) WithPostSyncMatching(epLister EndpointIDLister, inserter JobInserter) *NVDSyncWorker {
	w.epLister = epLister
	w.jobInserter = inserter
	return w
}

// Work executes the NVD sync for the tenant specified in the job args.
// After a successful sync, if configured, it enqueues CVE match jobs for all endpoints.
func (w *NVDSyncWorker) Work(ctx context.Context, job *river.Job[NVDSyncJobArgs]) error {
	if w.syncSvc == nil {
		return fmt.Errorf("cve nvd sync worker: sync service not configured")
	}
	ctx = tenant.WithTenantID(ctx, job.Args.TenantID)
	slog.InfoContext(ctx, "cve: starting NVD sync", "tenant_id", job.Args.TenantID)
	if err := w.syncSvc.SyncNVD(ctx, job.Args.TenantID); err != nil {
		return fmt.Errorf("cve nvd sync worker: %w", err)
	}
	slog.InfoContext(ctx, "cve: NVD sync complete", "tenant_id", job.Args.TenantID)

	// Enqueue endpoint match jobs for all active endpoints after sync.
	if w.epLister != nil && w.jobInserter != nil {
		epIDs, err := w.epLister.ListEndpointIDs(ctx, job.Args.TenantID)
		if err != nil {
			slog.ErrorContext(ctx, "cve: failed to list endpoints for post-sync matching", "error", err)
			return nil // sync succeeded, don't fail the job
		}
		enqueued := 0
		for _, epID := range epIDs {
			if _, err := w.jobInserter.Insert(ctx, EndpointMatchJobArgs{
				TenantID:   job.Args.TenantID,
				EndpointID: epID,
			}, nil); err != nil {
				slog.ErrorContext(ctx, "cve: failed to enqueue post-sync match job",
					"endpoint_id", epID, "error", err)
				continue
			}
			enqueued++
		}
		slog.InfoContext(ctx, "cve: enqueued post-sync endpoint match jobs",
			"tenant_id", job.Args.TenantID, "endpoint_count", enqueued)
	}

	return nil
}

// EndpointOsFamilyGetter retrieves the os_family for a given endpoint ID.
type EndpointOsFamilyGetter interface {
	GetEndpointOsFamily(ctx context.Context, endpointID string) (string, error)
}

// EndpointMatchWorker processes CVE endpoint matching jobs via River.
type EndpointMatchWorker struct {
	river.WorkerDefaults[EndpointMatchJobArgs]
	matcher        *Matcher
	osFamilyGetter EndpointOsFamilyGetter
}

// NewEndpointMatchWorker creates an EndpointMatchWorker with the given matcher.
func NewEndpointMatchWorker(matcher *Matcher) *EndpointMatchWorker {
	return &EndpointMatchWorker{matcher: matcher}
}

// WithOsFamilyGetter configures the worker to look up endpoint os_family before matching.
func (w *EndpointMatchWorker) WithOsFamilyGetter(g EndpointOsFamilyGetter) *EndpointMatchWorker {
	w.osFamilyGetter = g
	return w
}

// Work matches CVEs against the endpoint specified in the job args.
func (w *EndpointMatchWorker) Work(ctx context.Context, job *river.Job[EndpointMatchJobArgs]) error {
	if w.matcher == nil {
		return fmt.Errorf("cve endpoint match worker: matcher not configured")
	}
	ctx = tenant.WithTenantID(ctx, job.Args.TenantID)
	slog.InfoContext(ctx, "cve: matching endpoint", "tenant_id", job.Args.TenantID, "endpoint_id", job.Args.EndpointID)

	var osFamily string
	if w.osFamilyGetter != nil {
		var err error
		osFamily, err = w.osFamilyGetter.GetEndpointOsFamily(ctx, job.Args.EndpointID)
		if err != nil {
			slog.WarnContext(ctx, "cve: failed to get endpoint os_family, skipping os-family matching",
				"endpoint_id", job.Args.EndpointID, "error", err)
		}
	}

	result, err := w.matcher.MatchEndpoint(ctx, job.Args.TenantID, job.Args.EndpointID, osFamily, time.Now())
	if err != nil {
		return fmt.Errorf("cve endpoint match worker: %w", err)
	}
	slog.InfoContext(ctx, "cve: endpoint matching complete", "endpoint_id", job.Args.EndpointID, "affected", result.Affected, "patched", result.Patched)
	return nil
}
