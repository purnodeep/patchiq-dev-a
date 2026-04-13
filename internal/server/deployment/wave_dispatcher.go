package deployment

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"google.golang.org/protobuf/proto"
)

// WaveDispatcherQuerier defines the queries needed by the wave dispatcher.
type WaveDispatcherQuerier interface {
	// Tenant discovery for running deployments
	ListTenantIDsWithRunningDeployments(ctx context.Context) ([]pgtype.UUID, error)

	// Tenant-scoped deployment listing
	ListRunningDeployments(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.Deployment, error)

	// Tenant-scoped operations
	GetCurrentWave(ctx context.Context, arg sqlcgen.GetCurrentWaveParams) (sqlcgen.DeploymentWave, error)
	ListPendingWaveTargets(ctx context.Context, arg sqlcgen.ListPendingWaveTargetsParams) ([]sqlcgen.DeploymentTarget, error)
	CountActiveTargets(ctx context.Context, arg sqlcgen.CountActiveTargetsParams) (int64, error)
	GetEndpointMaintenanceWindow(ctx context.Context, arg sqlcgen.GetEndpointMaintenanceWindowParams) ([]byte, error)
	SetWaveRunning(ctx context.Context, arg sqlcgen.SetWaveRunningParams) (sqlcgen.DeploymentWave, error)
	SetWaveCompleted(ctx context.Context, arg sqlcgen.SetWaveCompletedParams) (sqlcgen.DeploymentWave, error)
	SetWaveFailed(ctx context.Context, arg sqlcgen.SetWaveFailedParams) (sqlcgen.DeploymentWave, error)
	SetWaveEligibleAt(ctx context.Context, arg sqlcgen.SetWaveEligibleAtParams) error
	CreateCommand(ctx context.Context, arg sqlcgen.CreateCommandParams) (sqlcgen.Command, error)
	GetPatchByID(ctx context.Context, arg sqlcgen.GetPatchByIDParams) (sqlcgen.Patch, error)
	UpdateDeploymentTargetStatus(ctx context.Context, arg sqlcgen.UpdateDeploymentTargetStatusParams) (sqlcgen.DeploymentTarget, error)
	ListDeploymentWaves(ctx context.Context, arg sqlcgen.ListDeploymentWavesParams) ([]sqlcgen.DeploymentWave, error)

	// For rollback and completion
	RollbackQuerier
	CompleteQuerier
}

// WaveDispatcherTxFactory creates a tenant-scoped transaction and returns a querier bound to it,
// along with commit and rollback functions.
type WaveDispatcherTxFactory func(ctx context.Context, tenantID string) (WaveDispatcherQuerier, func() error, func() error, error)

// WaveDispatcher is a periodic job that processes running deployments with waves.
// It finds running deployments, transitions eligible waves, dispatches pending targets,
// evaluates wave completion, and triggers rollbacks when failure thresholds are exceeded.
type WaveDispatcher struct {
	q              WaveDispatcherQuerier
	txFactory      WaveDispatcherTxFactory
	sm             *StateMachine
	eventBus       domain.EventBus
	commandTimeout time.Duration
}

// NewWaveDispatcher creates a WaveDispatcher.
func NewWaveDispatcher(q WaveDispatcherQuerier, sm *StateMachine, eventBus domain.EventBus, commandTimeout time.Duration, opts ...func(*WaveDispatcher)) *WaveDispatcher {
	if q == nil {
		panic("deployment: NewWaveDispatcher called with nil querier")
	}
	if sm == nil {
		panic("deployment: NewWaveDispatcher called with nil stateMachine")
	}
	if eventBus == nil {
		panic("deployment: NewWaveDispatcher called with nil eventBus")
	}
	if commandTimeout <= 0 {
		panic("deployment: NewWaveDispatcher called with non-positive commandTimeout")
	}
	wd := &WaveDispatcher{q: q, sm: sm, eventBus: eventBus, commandTimeout: commandTimeout}
	for _, opt := range opts {
		opt(wd)
	}
	return wd
}

// WithWaveDispatcherTxFactory sets the transaction factory for tenant-scoped writes.
func WithWaveDispatcherTxFactory(f WaveDispatcherTxFactory) func(*WaveDispatcher) {
	return func(wd *WaveDispatcher) {
		wd.txFactory = f
	}
}

// Dispatch finds all running deployments and processes their wave state.
func (wd *WaveDispatcher) Dispatch(ctx context.Context) error {
	tenantIDs, err := wd.q.ListTenantIDsWithRunningDeployments(ctx)
	if err != nil {
		return fmt.Errorf("wave dispatcher: list tenant IDs with running deployments: %w", err)
	}

	var failCount int
	for _, tenantID := range tenantIDs {
		deployments, err := wd.q.ListRunningDeployments(ctx, tenantID)
		if err != nil {
			failCount++
			slog.ErrorContext(ctx, "wave dispatcher: list running deployments",
				"tenant_id", uuid.UUID(tenantID.Bytes).String(), "error", err)
			continue
		}

		for _, dep := range deployments {
			if err := wd.processDeployment(ctx, dep); err != nil {
				slog.ErrorContext(ctx, "wave dispatcher: process deployment",
					"deployment_id", uuid.UUID(dep.ID.Bytes).String(), "error", err)
				// Log and continue — don't let one deployment block others
			}
		}
	}
	if failCount == len(tenantIDs) && len(tenantIDs) > 0 {
		return fmt.Errorf("wave dispatcher: all %d tenants failed", failCount)
	}
	return nil
}

func (wd *WaveDispatcher) processDeployment(ctx context.Context, dep sqlcgen.Deployment) error {
	tenantID := dep.TenantID
	tenantIDStr := uuid.UUID(tenantID.Bytes).String()
	deployID := dep.ID

	// Get a querier for writes (tenant-scoped tx if configured).
	writeQ, commit, rollback, txErr := wd.beginWriteTx(ctx, tenantIDStr)
	if txErr != nil {
		return fmt.Errorf("begin tenant tx: %w", txErr)
	}
	defer func() {
		if rbErr := rollback(); rbErr != nil {
			slog.ErrorContext(ctx, "wave dispatcher: rollback failed", "error", rbErr)
		}
	}()

	// Get the current wave (first pending or running, ordered by wave_number).
	wave, err := writeQ.GetCurrentWave(ctx, sqlcgen.GetCurrentWaveParams{
		DeploymentID: deployID,
		TenantID:     tenantID,
	})
	if err != nil {
		// No current wave found — all waves are completed/failed/cancelled.
		// Mark deployment as completed.
		_, completeEvts, completeErr := wd.sm.CompleteDeployment(ctx, writeQ, deployID, tenantID)
		if completeErr != nil {
			return fmt.Errorf("complete deployment (no active waves): %w", completeErr)
		}
		if commitErr := commit(); commitErr != nil {
			return fmt.Errorf("commit complete deployment: %w", commitErr)
		}
		EmitBestEffort(ctx, wd.eventBus, completeEvts)
		return nil
	}

	var pendingEvents []domain.DomainEvent

	// If wave is pending and eligible_at has passed, transition to running.
	if wave.Status == string(WavePending) {
		if !wd.isWaveEligible(wave) {
			// Not yet eligible — skip this deployment for now.
			if commitErr := commit(); commitErr != nil {
				return fmt.Errorf("commit (wave not eligible): %w", commitErr)
			}
			return nil
		}

		wave, err = writeQ.SetWaveRunning(ctx, sqlcgen.SetWaveRunningParams{
			ID:       wave.ID,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("set wave running: %w", err)
		}

		waveIDStr := uuid.UUID(wave.ID.Bytes).String()
		pendingEvents = append(pendingEvents,
			domain.NewSystemEvent(events.DeploymentWaveStarted, tenantIDStr, "deployment_wave", waveIDStr, events.DeploymentWaveStarted, nil),
		)
	}

	// Wave is now running — dispatch pending targets.
	if wave.Status == string(WaveRunning) {
		dispatchEvts, err := wd.dispatchWaveTargets(ctx, writeQ, dep, wave)
		if err != nil {
			return fmt.Errorf("dispatch wave targets: %w", err)
		}
		pendingEvents = append(pendingEvents, dispatchEvts...)

		// Check if all wave targets are terminal (no more pending or active).
		completionEvts, err := wd.checkWaveCompletion(ctx, writeQ, dep, wave)
		if err != nil {
			return fmt.Errorf("check wave completion: %w", err)
		}
		pendingEvents = append(pendingEvents, completionEvts...)
	}

	if commitErr := commit(); commitErr != nil {
		return fmt.Errorf("commit wave processing: %w", commitErr)
	}

	EmitBestEffort(ctx, wd.eventBus, pendingEvents)
	return nil
}

// isWaveEligible checks whether a pending wave is eligible to start.
func (wd *WaveDispatcher) isWaveEligible(wave sqlcgen.DeploymentWave) bool {
	if !wave.EligibleAt.Valid {
		// No eligible_at set — first wave, always eligible.
		return true
	}
	return time.Now().After(wave.EligibleAt.Time)
}

// dispatchWaveTargets dispatches pending targets for a running wave,
// respecting maintenance windows and max_concurrent throttle.
func (wd *WaveDispatcher) dispatchWaveTargets(ctx context.Context, q WaveDispatcherQuerier, dep sqlcgen.Deployment, wave sqlcgen.DeploymentWave) ([]domain.DomainEvent, error) {
	tenantID := dep.TenantID
	tenantIDStr := uuid.UUID(tenantID.Bytes).String()

	targets, err := q.ListPendingWaveTargets(ctx, sqlcgen.ListPendingWaveTargetsParams{
		WaveID:   wave.ID,
		TenantID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("list pending wave targets: %w", err)
	}
	if len(targets) == 0 {
		return nil, nil
	}

	// Check throttle: count active targets vs max_concurrent.
	activeCount, err := q.CountActiveTargets(ctx, sqlcgen.CountActiveTargetsParams{
		DeploymentID: dep.ID,
		TenantID:     tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("count active targets: %w", err)
	}

	maxConcurrent := int64(0)
	if dep.MaxConcurrent.Valid {
		maxConcurrent = int64(dep.MaxConcurrent.Int32)
	}

	deadline := pgtype.Timestamptz{Time: time.Now().Add(wd.commandTimeout), Valid: true}
	var pendingEvents []domain.DomainEvent

	for _, target := range targets {
		// Enforce bandwidth throttle.
		if maxConcurrent > 0 && activeCount >= maxConcurrent {
			break
		}

		// Check maintenance window. On any error, treat as no window configured
		// (always allow dispatch) rather than permanently blocking the target.
		mwData, mwErr := q.GetEndpointMaintenanceWindow(ctx, sqlcgen.GetEndpointMaintenanceWindowParams{
			ID:       target.EndpointID,
			TenantID: tenantID,
		})
		if mwErr != nil {
			slog.WarnContext(ctx, "wave dispatcher: get maintenance window, proceeding without restriction",
				"endpoint_id", uuid.UUID(target.EndpointID.Bytes).String(), "error", mwErr)
		}

		mw, parseErr := ParseMaintenanceWindow(mwData)
		if parseErr != nil {
			slog.WarnContext(ctx, "wave dispatcher: parse maintenance window, proceeding without restriction",
				"endpoint_id", uuid.UUID(target.EndpointID.Bytes).String(), "error", parseErr)
			mw = nil
		}

		if !IsInMaintenanceWindow(mw, time.Now()) {
			continue
		}

		// Look up the patch to build the install payload.
		patch, patchErr := q.GetPatchByID(ctx, sqlcgen.GetPatchByIDParams{
			ID:       target.PatchID,
			TenantID: tenantID,
		})
		if patchErr != nil {
			slog.ErrorContext(ctx, "wave dispatcher: get patch for target",
				"target_id", uuid.UUID(target.ID.Bytes).String(),
				"patch_id", uuid.UUID(target.PatchID.Bytes).String(), "error", patchErr)
			continue
		}

		// Use the actual package name (e.g. "curl") for installation when available,
		// falling back to the advisory name (e.g. "RHSA-2024:0893") otherwise.
		pkgName := patch.Name
		if patch.PackageName != "" {
			pkgName = patch.PackageName
		}
		installPayload := &pb.InstallPatchPayload{
			Packages: []*pb.PatchTarget{{
				Name:    pkgName,
				Version: patch.Version,
				Source:  installerTypeOrFallback(patch.InstallerType, patch.OsFamily),
			}},
		}

		// Populate download_url when a binary_ref (stored as package_url) is available.
		// The server file server serves binaries at /repo/files/{os}/{filename}.
		if patch.PackageUrl.Valid && patch.PackageUrl.String != "" {
			filename := path.Base(patch.PackageUrl.String)
			installPayload.DownloadUrl = "/repo/files/" + patch.OsFamily + "/" + filename
		}
		// Propagate checksum so the agent can verify the downloaded binary.
		if patch.ChecksumSha256.Valid && patch.ChecksumSha256.String != "" {
			installPayload.ChecksumSha256 = patch.ChecksumSha256.String
		}
		if patch.SilentArgs != "" {
			installPayload.SilentArgs = patch.SilentArgs
		}

		payload, payloadErr := proto.Marshal(installPayload)
		if payloadErr != nil {
			slog.ErrorContext(ctx, "wave dispatcher: marshal install payload",
				"target_id", uuid.UUID(target.ID.Bytes).String(), "error", payloadErr)
			continue
		}

		// Create command for target.
		cmd, cmdErr := q.CreateCommand(ctx, sqlcgen.CreateCommandParams{
			TenantID:     tenantID,
			AgentID:      target.EndpointID,
			DeploymentID: pgtype.UUID{Bytes: dep.ID.Bytes, Valid: true},
			TargetID:     pgtype.UUID{Bytes: target.ID.Bytes, Valid: true},
			Type:         string(CommandTypeInstallPatch),
			Payload:      payload,
			Priority:     0,
			Status:       string(CommandPending),
			Deadline:     deadline,
		})
		if cmdErr != nil {
			slog.ErrorContext(ctx, "wave dispatcher: create command",
				"target_id", uuid.UUID(target.ID.Bytes).String(), "error", cmdErr)
			continue
		}

		// Mark target as sent.
		if _, updateErr := q.UpdateDeploymentTargetStatus(ctx, sqlcgen.UpdateDeploymentTargetStatusParams{
			ID:       target.ID,
			Status:   string(TargetSent),
			TenantID: tenantID,
		}); updateErr != nil {
			slog.ErrorContext(ctx, "wave dispatcher: update target status",
				"target_id", uuid.UUID(target.ID.Bytes).String(), "error", updateErr)
			continue
		}

		activeCount++

		cmdIDStr := uuid.UUID(cmd.ID.Bytes).String()
		targetIDStr := uuid.UUID(target.ID.Bytes).String()
		pendingEvents = append(pendingEvents,
			domain.NewSystemEvent(events.CommandDispatched, tenantIDStr, "command", cmdIDStr, events.CommandDispatched, nil),
			domain.NewSystemEvent(events.DeploymentTargetSent, tenantIDStr, "deployment_target", targetIDStr, "sent", nil),
		)
	}

	return pendingEvents, nil
}

// checkWaveCompletion evaluates whether a running wave is complete and handles
// success criteria evaluation, rollback triggering, and wave advancement.
func (wd *WaveDispatcher) checkWaveCompletion(ctx context.Context, q WaveDispatcherQuerier, dep sqlcgen.Deployment, wave sqlcgen.DeploymentWave) ([]domain.DomainEvent, error) {
	tenantID := dep.TenantID
	tenantIDStr := uuid.UUID(tenantID.Bytes).String()

	// Check if there are still active or pending targets.
	activeCount, err := q.CountActiveTargets(ctx, sqlcgen.CountActiveTargetsParams{
		DeploymentID: dep.ID,
		TenantID:     tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("count active targets for completion check: %w", err)
	}

	pendingTargets, err := q.ListPendingWaveTargets(ctx, sqlcgen.ListPendingWaveTargetsParams{
		WaveID:   wave.ID,
		TenantID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("list pending targets for completion check: %w", err)
	}

	// If there are still active or pending targets, wave is not complete.
	if activeCount > 0 || len(pendingTargets) > 0 {
		return nil, nil
	}

	// All targets are terminal. Evaluate success criteria.
	if wave.TargetCount <= 0 {
		return nil, nil
	}

	var pendingEvents []domain.DomainEvent
	waveIDStr := uuid.UUID(wave.ID.Bytes).String()

	failureRate := float64(wave.FailedCount) / float64(wave.TargetCount)
	errorRateMax := DefaultFailureThreshold
	erm, err := wave.ErrorRateMax.Float64Value()
	if err != nil {
		return nil, fmt.Errorf("parse error_rate_max for wave %s: %w", waveIDStr, err)
	}
	if erm.Valid {
		errorRateMax = erm.Float64
	}

	// Check if failure rate exceeds threshold — trigger rollback.
	if failureRate > errorRateMax {
		if _, failErr := q.SetWaveFailed(ctx, sqlcgen.SetWaveFailedParams{
			ID:       wave.ID,
			TenantID: tenantID,
		}); failErr != nil {
			return nil, fmt.Errorf("set wave failed: %w", failErr)
		}
		pendingEvents = append(pendingEvents,
			domain.NewSystemEvent(events.DeploymentWaveFailed, tenantIDStr, "deployment_wave", waveIDStr, events.DeploymentWaveFailed, nil),
		)

		// Trigger rollback.
		_, rollbackEvts, rollbackErr := wd.sm.RollbackDeployment(ctx, q, dep.ID, tenantID)
		if rollbackErr != nil {
			return nil, fmt.Errorf("rollback deployment: %w", rollbackErr)
		}
		pendingEvents = append(pendingEvents, rollbackEvts...)
		return pendingEvents, nil
	}

	// Check success criteria.
	successRate := float64(wave.SuccessCount) / float64(wave.TargetCount)
	successThreshold := 0.8
	st, err := wave.SuccessThreshold.Float64Value()
	if err != nil {
		return nil, fmt.Errorf("parse success_threshold for wave %s: %w", waveIDStr, err)
	}
	if st.Valid {
		successThreshold = st.Float64
	}

	if successRate >= successThreshold {
		// Wave succeeded — complete it.
		if _, completeErr := q.SetWaveCompleted(ctx, sqlcgen.SetWaveCompletedParams{
			ID:       wave.ID,
			TenantID: tenantID,
		}); completeErr != nil {
			return nil, fmt.Errorf("set wave completed: %w", completeErr)
		}
		pendingEvents = append(pendingEvents,
			domain.NewSystemEvent(events.DeploymentWaveCompleted, tenantIDStr, "deployment_wave", waveIDStr, events.DeploymentWaveCompleted, nil),
		)

		// Check if there's a next wave. List all waves and find the next pending one.
		advanceEvts, advanceErr := wd.advanceToNextWave(ctx, q, dep, wave)
		if advanceErr != nil {
			return nil, fmt.Errorf("advance to next wave: %w", advanceErr)
		}
		pendingEvents = append(pendingEvents, advanceEvts...)
	}

	return pendingEvents, nil
}

// advanceToNextWave finds the next pending wave and sets its eligible_at,
// or completes the deployment if there are no more waves.
func (wd *WaveDispatcher) advanceToNextWave(ctx context.Context, q WaveDispatcherQuerier, dep sqlcgen.Deployment, completedWave sqlcgen.DeploymentWave) ([]domain.DomainEvent, error) {
	tenantID := dep.TenantID

	waves, err := q.ListDeploymentWaves(ctx, sqlcgen.ListDeploymentWavesParams{
		DeploymentID: dep.ID,
		TenantID:     tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("list deployment waves: %w", err)
	}

	// Find the next pending wave after the completed one.
	var nextWave *sqlcgen.DeploymentWave
	for i := range waves {
		if waves[i].WaveNumber > completedWave.WaveNumber && waves[i].Status == string(WavePending) {
			nextWave = &waves[i]
			break
		}
	}

	if nextWave == nil {
		// No more waves — deployment is complete.
		_, completeEvts, completeErr := wd.sm.CompleteDeployment(ctx, q, dep.ID, tenantID)
		if completeErr != nil {
			return nil, fmt.Errorf("complete deployment (last wave): %w", completeErr)
		}
		return completeEvts, nil
	}

	// Set eligible_at on the next wave based on the completed wave's delay_after_minutes.
	eligibleAt := time.Now().Add(time.Duration(completedWave.DelayAfterMinutes) * time.Minute)
	if err := q.SetWaveEligibleAt(ctx, sqlcgen.SetWaveEligibleAtParams{
		ID:         nextWave.ID,
		EligibleAt: pgtype.Timestamptz{Time: eligibleAt, Valid: true},
		TenantID:   tenantID,
	}); err != nil {
		return nil, fmt.Errorf("set next wave eligible_at: %w", err)
	}

	return nil, nil
}

// beginWriteTx returns a querier for writes. If txFactory is set, it starts a tenant-scoped
// transaction. Otherwise, it returns the default querier with no-op commit/rollback.
func (wd *WaveDispatcher) beginWriteTx(ctx context.Context, tenantID string) (WaveDispatcherQuerier, func() error, func() error, error) {
	if wd.txFactory != nil {
		return wd.txFactory(ctx, tenantID)
	}
	// No txFactory — writes bypass RLS. This is only safe in tests.
	slog.WarnContext(ctx, "WaveDispatcher: no txFactory configured, writes bypass RLS tenant isolation")
	noop := func() error { return nil }
	return wd.q, noop, noop, nil
}

// installerTypeOrFallback returns the patch's installer_type if set,
// otherwise falls back to the legacy OS-family-based mapping.
func installerTypeOrFallback(installerType, osFamily string) string {
	if installerType != "" {
		return installerType
	}
	return osFamilyToSource(osFamily)
}

// osFamilyToSource maps an os_family value from the patch catalog to the
// package manager source string the agent uses to pick an installer.
// Returns an empty string for unknown families so the agent falls back to
// its own heuristics.
func osFamilyToSource(osFamily string) string {
	switch osFamily {
	case "linux-debian", "linux-ubuntu":
		return "apt"
	case "linux-rhel", "linux-centos", "linux-fedora":
		return "yum"
	case "macos", "darwin":
		return "homebrew"
	case "windows":
		return "msi"
	default:
		return ""
	}
}

// --- River Worker ---

// WaveDispatcherJobArgs is the payload for the wave dispatcher periodic River job.
type WaveDispatcherJobArgs struct{}

// Kind implements river.JobArgs.
func (WaveDispatcherJobArgs) Kind() string { return "wave_dispatcher" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (WaveDispatcherJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "critical"}
}

// WaveDispatcherWorker wraps WaveDispatcher as a River worker.
type WaveDispatcherWorker struct {
	river.WorkerDefaults[WaveDispatcherJobArgs]
	dispatcher *WaveDispatcher
}

// NewWaveDispatcherWorker creates a WaveDispatcherWorker.
func NewWaveDispatcherWorker(dispatcher *WaveDispatcher) *WaveDispatcherWorker {
	return &WaveDispatcherWorker{dispatcher: dispatcher}
}

// Work implements river.Worker.
func (w *WaveDispatcherWorker) Work(ctx context.Context, _ *river.Job[WaveDispatcherJobArgs]) error {
	return w.dispatcher.Dispatch(ctx)
}
