package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// DeploymentQuerier defines the sqlc queries needed by DeploymentHandler.
type DeploymentQuerier interface {
	deployment.EvalQuerier
	deployment.CancelQuerier
	CreateDeployment(ctx context.Context, arg sqlcgen.CreateDeploymentParams) (sqlcgen.Deployment, error)
	CreateDeploymentWave(ctx context.Context, arg sqlcgen.CreateDeploymentWaveParams) (sqlcgen.DeploymentWave, error)
	CreateDeploymentTarget(ctx context.Context, arg sqlcgen.CreateDeploymentTargetParams) (sqlcgen.DeploymentTarget, error)
	SetDeploymentTotalTargets(ctx context.Context, arg sqlcgen.SetDeploymentTotalTargetsParams) (sqlcgen.Deployment, error)
	GetDeploymentByID(ctx context.Context, arg sqlcgen.GetDeploymentByIDParams) (sqlcgen.Deployment, error)
	ListDeploymentsByTenantFiltered(ctx context.Context, arg sqlcgen.ListDeploymentsByTenantFilteredParams) ([]sqlcgen.Deployment, error)
	CountDeploymentsByTenantFiltered(ctx context.Context, arg sqlcgen.CountDeploymentsByTenantFilteredParams) (int64, error)
	ListDeploymentTargets(ctx context.Context, arg sqlcgen.ListDeploymentTargetsParams) ([]sqlcgen.DeploymentTarget, error)
	CreateDeploymentWithWaveConfig(ctx context.Context, arg sqlcgen.CreateDeploymentWithWaveConfigParams) (sqlcgen.Deployment, error)
	CreateDeploymentWithOrchestration(ctx context.Context, arg sqlcgen.CreateDeploymentWithOrchestrationParams) (sqlcgen.Deployment, error)
	CreateDeploymentWaveWithConfig(ctx context.Context, arg sqlcgen.CreateDeploymentWaveWithConfigParams) (sqlcgen.DeploymentWave, error)
	CreateDeploymentTargetWithWave(ctx context.Context, arg sqlcgen.CreateDeploymentTargetWithWaveParams) (sqlcgen.DeploymentTarget, error)
	SetDeploymentWaveTargetCount(ctx context.Context, arg sqlcgen.SetDeploymentWaveTargetCountParams) (sqlcgen.DeploymentWave, error)
	ListDeploymentWaves(ctx context.Context, arg sqlcgen.ListDeploymentWavesParams) ([]sqlcgen.DeploymentWave, error)
	ListDeploymentTargetsWithHostname(ctx context.Context, arg sqlcgen.ListDeploymentTargetsWithHostnameParams) ([]sqlcgen.ListDeploymentTargetsWithHostnameRow, error)
	ListDeploymentTargetsByWave(ctx context.Context, arg sqlcgen.ListDeploymentTargetsByWaveParams) ([]sqlcgen.ListDeploymentTargetsByWaveRow, error)
	RetryFailedTargets(ctx context.Context, arg sqlcgen.RetryFailedTargetsParams) (int64, error)
	CountDeploymentsByStatus(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.CountDeploymentsByStatusRow, error)
	ListDeploymentPatchSummary(ctx context.Context, arg sqlcgen.ListDeploymentPatchSummaryParams) ([]sqlcgen.ListDeploymentPatchSummaryRow, error)
	SetDeploymentRetrying(ctx context.Context, arg sqlcgen.SetDeploymentRetryingParams) (sqlcgen.Deployment, error)
}

// CancelTxFactory starts a tenant-scoped transaction and returns:
//   - a CancelQuerier bound to the transaction
//   - a commit function
//   - a rollback function
//   - any error from starting the transaction
type CancelTxFactory func(ctx context.Context, tenantID string) (deployment.CancelQuerier, func() error, func() error, error)

// DeploymentHandler serves deployment REST API endpoints.
type DeploymentHandler struct {
	q               DeploymentQuerier
	pool            TxBeginner
	riverClient     *river.Client[pgx.Tx]
	eventBus        domain.EventBus
	evaluator       *deployment.Evaluator
	sm              *deployment.StateMachine
	cancelTxFactory CancelTxFactory
}

// NewDeploymentHandler creates a DeploymentHandler.
func NewDeploymentHandler(q DeploymentQuerier, pool TxBeginner, riverClient *river.Client[pgx.Tx], eventBus domain.EventBus, evaluator *deployment.Evaluator, sm *deployment.StateMachine) *DeploymentHandler {
	if q == nil {
		panic("deployments: NewDeploymentHandler called with nil querier")
	}
	if pool == nil {
		panic("deployments: NewDeploymentHandler called with nil pool")
	}
	if riverClient == nil {
		panic("deployments: NewDeploymentHandler called with nil riverClient")
	}
	if eventBus == nil {
		panic("deployments: NewDeploymentHandler called with nil eventBus")
	}
	if evaluator == nil {
		panic("deployments: NewDeploymentHandler called with nil evaluator")
	}
	if sm == nil {
		panic("deployments: NewDeploymentHandler called with nil stateMachine")
	}
	return &DeploymentHandler{q: q, pool: pool, riverClient: riverClient, eventBus: eventBus, evaluator: evaluator, sm: sm}
}

// deploymentResponse is the clean API response type for a deployment.
// It does NOT expose TenantID or FailureThreshold.
type deploymentResponse struct {
	ID             string                  `json:"id"`
	Name           string                  `json:"name,omitempty"`
	PolicyID       string                  `json:"policy_id"`
	PolicyName     string                  `json:"policy_name,omitempty"`
	Status         string                  `json:"status"`
	TargetCount    int32                   `json:"target_count"`
	CompletedCount int32                   `json:"completed_count"`
	SuccessCount   int32                   `json:"success_count"`
	FailedCount    int32                   `json:"failed_count"`
	CreatedBy      *string                 `json:"created_by,omitempty"`
	StartedAt      *time.Time              `json:"started_at,omitempty"`
	CompletedAt    *time.Time              `json:"completed_at,omitempty"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
	WaveConfig     []deployment.WaveConfig `json:"wave_config,omitempty"`
	MaxConcurrent  *int32                  `json:"max_concurrent,omitempty"`
	ScheduledAt    *time.Time              `json:"scheduled_at,omitempty"`
}

// toDeploymentResponse converts a sqlcgen.Deployment to a clean API response.
func toDeploymentResponse(d sqlcgen.Deployment) deploymentResponse {
	resp := deploymentResponse{
		ID:             uuidToString(d.ID),
		Name:           d.Name.String,
		PolicyID:       uuidToString(d.PolicyID),
		Status:         d.Status,
		TargetCount:    d.TotalTargets,
		CompletedCount: d.CompletedCount,
		SuccessCount:   d.SuccessCount,
		FailedCount:    d.FailedCount,
		CreatedAt:      d.CreatedAt.Time,
		UpdatedAt:      d.UpdatedAt.Time,
	}
	if d.StartedAt.Valid {
		t := d.StartedAt.Time
		resp.StartedAt = &t
	}
	if d.CompletedAt.Valid {
		t := d.CompletedAt.Time
		resp.CompletedAt = &t
	}
	if d.CreatedBy.Valid {
		s := uuidToString(d.CreatedBy)
		resp.CreatedBy = &s
	}
	if len(d.WaveConfig) > 0 {
		var wc []deployment.WaveConfig
		if err := json.Unmarshal(d.WaveConfig, &wc); err != nil {
			slog.Warn("unmarshal wave_config", "deployment_id", uuidToString(d.ID), "error", err)
		} else {
			resp.WaveConfig = wc
		}
	}
	if d.MaxConcurrent.Valid {
		v := d.MaxConcurrent.Int32
		resp.MaxConcurrent = &v
	}
	if d.ScheduledAt.Valid {
		t := d.ScheduledAt.Time
		resp.ScheduledAt = &t
	}
	return resp
}

// deploymentTargetResponse is the clean API response type for a deployment target.
type deploymentTargetResponse struct {
	ID           string     `json:"id"`
	DeploymentID string     `json:"deployment_id"`
	EndpointID   string     `json:"endpoint_id"`
	Hostname     string     `json:"hostname"`
	PatchID      string     `json:"patch_id"`
	Status       string     `json:"status"`
	ExitCode     *int32     `json:"exit_code,omitempty"`
	Output       *string    `json:"output,omitempty"`
	Error        *string    `json:"error,omitempty"`
	WaveID       *string    `json:"wave_id,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// toDeploymentTargetWithHostnameResponse converts a ListDeploymentTargetsWithHostnameRow to a clean API response.
func toDeploymentTargetWithHostnameResponse(dt sqlcgen.ListDeploymentTargetsWithHostnameRow) deploymentTargetResponse {
	r := deploymentTargetResponse{
		ID:           uuidToString(dt.ID),
		DeploymentID: uuidToString(dt.DeploymentID),
		EndpointID:   uuidToString(dt.EndpointID),
		Hostname:     dt.Hostname,
		PatchID:      uuidToString(dt.PatchID),
		Status:       dt.Status,
		CreatedAt:    dt.CreatedAt.Time,
	}
	if dt.ExitCode.Valid {
		v := dt.ExitCode.Int32
		r.ExitCode = &v
	}
	if dt.Stdout.Valid && dt.Stdout.String != "" {
		r.Output = &dt.Stdout.String
	}
	// Combine stderr and error_message into Error field
	errMsg := ""
	if dt.Stderr.Valid && dt.Stderr.String != "" {
		errMsg = dt.Stderr.String
	} else if dt.ErrorMessage.Valid && dt.ErrorMessage.String != "" {
		errMsg = dt.ErrorMessage.String
	}
	if errMsg != "" {
		r.Error = &errMsg
	}
	if dt.WaveID.Valid {
		s := uuidToString(dt.WaveID)
		r.WaveID = &s
	}
	if dt.StartedAt.Valid {
		t := dt.StartedAt.Time
		r.StartedAt = &t
	}
	if dt.CompletedAt.Valid {
		t := dt.CompletedAt.Time
		r.CompletedAt = &t
	}
	return r
}

// WaveConfig mirrors deployment.WaveConfig for API request parsing.
type WaveConfig struct {
	Percentage       int     `json:"percentage"`
	SuccessThreshold float64 `json:"success_threshold"`
	ErrorRateMax     float64 `json:"error_rate_max"`
	DelayMinutes     int     `json:"delay_minutes"`
}

// createDeploymentRequest is the JSON body for POST /deployments.
// policy_id is optional; for catalog/adhoc source types it may be omitted.
type createDeploymentRequest struct {
	PolicyID         string       `json:"policy_id"`
	SourceType       string       `json:"source_type,omitempty"`
	EndpointIDs      []string     `json:"endpoint_ids,omitempty"`
	PatchIDs         []string     `json:"patch_ids,omitempty"`
	Name             string       `json:"name,omitempty"`
	Description      string       `json:"description,omitempty"`
	TargetExpression any          `json:"target_expression,omitempty"`
	WaveConfig       []WaveConfig `json:"wave_config,omitempty"`
	MaxConcurrent    *int32       `json:"max_concurrent,omitempty"`
	ScheduledAt      *time.Time   `json:"scheduled_at,omitempty"`
	RollbackConfig   any          `json:"rollback_config,omitempty"`
	RebootConfig     any          `json:"reboot_config,omitempty"`
	WorkflowTemplate string       `json:"workflow_template_id,omitempty"`
}

// createDeploymentResponse is the JSON response for POST /deployments.
type createDeploymentResponse struct {
	deploymentResponse
	TargetCount int `json:"target_count"`
}

// deploymentListResponse is the JSON response for GET /deployments.
type deploymentListResponse struct {
	Data         []deploymentResponse `json:"data"`
	TotalCount   int64                `json:"total_count"`
	NextCursor   *string              `json:"next_cursor,omitempty"`
	StatusCounts map[string]int       `json:"status_counts"`
}

// deploymentDetailResponse is the JSON response for GET /deployments/{id}.
type deploymentDetailResponse struct {
	deploymentResponse
	Targets []deploymentTargetResponse `json:"targets"`
}

// Create handles POST /api/v1/deployments.
func (h *DeploymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var body createDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}

	// Resolve source type: default to "policy" if policy_id given, otherwise "adhoc".
	sourceType := body.SourceType
	if sourceType == "" {
		if body.PolicyID != "" {
			sourceType = "policy"
		} else {
			sourceType = "adhoc"
		}
	}

	// policy_id is required only for policy-source deployments.
	if sourceType == "policy" && body.PolicyID == "" {
		WriteError(w, http.StatusBadRequest, "MISSING_POLICY_ID", "policy_id is required for policy-source deployments")
		return
	}

	var policyID pgtype.UUID
	if body.PolicyID != "" {
		var parseErr error
		policyID, parseErr = scanUUID(body.PolicyID)
		if parseErr != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_POLICY_ID", "policy_id is not a valid UUID")
			return
		}
	}

	// Parse wave config: use request config or defaults.
	var waveConfigs []deployment.WaveConfig
	if len(body.WaveConfig) > 0 {
		totalPct := 0
		for _, wc := range body.WaveConfig {
			totalPct += wc.Percentage
		}
		if totalPct != 100 {
			WriteError(w, http.StatusBadRequest, "INVALID_WAVE_CONFIG", "wave percentages must sum to 100")
			return
		}
		waveConfigs = make([]deployment.WaveConfig, len(body.WaveConfig))
		for i, wc := range body.WaveConfig {
			waveConfigs[i] = deployment.WaveConfig{
				Percentage:       wc.Percentage,
				SuccessThreshold: wc.SuccessThreshold,
				ErrorRateMax:     wc.ErrorRateMax,
				DelayMinutes:     wc.DelayMinutes,
			}
		}
	} else {
		waveConfigs, err = deployment.ParseWaveConfig(nil)
		if err != nil {
			slog.ErrorContext(ctx, "parse default wave config", "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to parse default wave config")
			return
		}
	}

	// For policy-source deployments: evaluate the policy to get targets.
	// For adhoc/catalog: build targets from the explicit endpoint_ids × patch_ids cross-product.
	var adhocTargets []deployment.Target
	var evalResult *deployment.EvalResult
	switch sourceType {
	case "policy":
		result, evalErr := h.evaluator.Evaluate(ctx, h.q, policyID, tid)
		if evalErr != nil {
			slog.ErrorContext(ctx, "evaluate policy for deployment", "policy_id", body.PolicyID, "tenant_id", tenantID, "error", evalErr)
			switch {
			case errors.Is(evalErr, deployment.ErrPolicyDisabled):
				WriteError(w, http.StatusUnprocessableEntity, "POLICY_DISABLED", "policy is disabled")
			case errors.Is(evalErr, deployment.ErrNoEndpoints):
				WriteError(w, http.StatusUnprocessableEntity, "NO_ENDPOINTS", "no endpoints matched policy tag selector")
			case errors.Is(evalErr, deployment.ErrNoPatchesMatched):
				WriteError(w, http.StatusUnprocessableEntity, "NO_PATCHES", "no patches matched policy filters")
			default:
				WriteError(w, http.StatusInternalServerError, "EVALUATION_FAILED", "policy evaluation failed")
			}
			return
		}
		evalResult = result
	case "adhoc", "catalog":
		if len(body.EndpointIDs) == 0 {
			WriteError(w, http.StatusBadRequest, "MISSING_ENDPOINT_IDS", "endpoint_ids is required for adhoc/catalog deployments")
			return
		}
		if len(body.PatchIDs) == 0 {
			WriteError(w, http.StatusBadRequest, "MISSING_PATCH_IDS", "patch_ids is required for adhoc/catalog deployments")
			return
		}
		endpointUUIDs := make([]pgtype.UUID, len(body.EndpointIDs))
		for i, eid := range body.EndpointIDs {
			u, parseErr := scanUUID(eid)
			if parseErr != nil {
				WriteError(w, http.StatusBadRequest, "INVALID_ENDPOINT_ID", fmt.Sprintf("endpoint_ids[%d] is not a valid UUID: %s", i, eid))
				return
			}
			endpointUUIDs[i] = u
		}
		patchUUIDs := make([]pgtype.UUID, len(body.PatchIDs))
		for i, pid := range body.PatchIDs {
			u, parseErr := scanUUID(pid)
			if parseErr != nil {
				WriteError(w, http.StatusBadRequest, "INVALID_PATCH_ID", fmt.Sprintf("patch_ids[%d] is not a valid UUID: %s", i, pid))
				return
			}
			patchUUIDs[i] = u
		}
		// Build cross-product of endpoints × patches.
		adhocTargets = make([]deployment.Target, 0, len(endpointUUIDs)*len(patchUUIDs))
		for _, ep := range endpointUUIDs {
			for _, p := range patchUUIDs {
				adhocTargets = append(adhocTargets, deployment.Target{EndpointID: ep, PatchID: p})
			}
		}
	}

	// Begin transaction to ensure atomicity of deployment creation.
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin deployment tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to begin transaction")
		return
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.WarnContext(ctx, "rollback deployment tx", "error", err)
		}
	}()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "set tenant context in tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	txQ := sqlcgen.New(tx)

	// Marshal wave config to JSON for storage.
	waveConfigJSON, err := json.Marshal(waveConfigs)
	if err != nil {
		slog.ErrorContext(ctx, "marshal wave config", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to marshal wave config")
		return
	}

	// Determine deployment status: scheduled or created.
	status := string(deployment.StatusCreated)
	if body.ScheduledAt != nil {
		status = string(deployment.StatusScheduled)
	}

	// Build optional fields.
	var maxConcurrent pgtype.Int4
	if body.MaxConcurrent != nil {
		maxConcurrent = pgtype.Int4{Int32: *body.MaxConcurrent, Valid: true}
	}
	var scheduledAt pgtype.Timestamptz
	if body.ScheduledAt != nil {
		scheduledAt = pgtype.Timestamptz{Time: *body.ScheduledAt, Valid: true}
	}

	// Marshal optional JSON fields (target_expression, rollback_config, reboot_config).
	var targetExprJSON, rollbackJSON, rebootJSON []byte
	if body.TargetExpression != nil {
		targetExprJSON, err = json.Marshal(body.TargetExpression)
		if err != nil {
			slog.ErrorContext(ctx, "marshal target_expression", "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to marshal target_expression")
			return
		}
	}
	if body.RollbackConfig != nil {
		rollbackJSON, err = json.Marshal(body.RollbackConfig)
		if err != nil {
			slog.ErrorContext(ctx, "marshal rollback_config", "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to marshal rollback_config")
			return
		}
	}
	if body.RebootConfig != nil {
		rebootJSON, err = json.Marshal(body.RebootConfig)
		if err != nil {
			slog.ErrorContext(ctx, "marshal reboot_config", "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to marshal reboot_config")
			return
		}
	}

	// Resolve optional workflow template ID.
	var workflowTemplateID pgtype.UUID
	if body.WorkflowTemplate != "" {
		workflowTemplateID, err = scanUUID(body.WorkflowTemplate)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_WORKFLOW_TEMPLATE_ID", "workflow_template_id is not a valid UUID")
			return
		}
	}

	// Create the deployment record using the orchestration query which supports all fields.
	dep, err := txQ.CreateDeploymentWithOrchestration(ctx, sqlcgen.CreateDeploymentWithOrchestrationParams{
		TenantID:           tid,
		PolicyID:           policyID,
		Status:             status,
		WaveConfig:         waveConfigJSON,
		MaxConcurrent:      maxConcurrent,
		ScheduledAt:        scheduledAt,
		SourceType:         sourceType,
		TargetExpression:   targetExprJSON,
		RollbackConfig:     rollbackJSON,
		RebootConfig:       rebootJSON,
		WorkflowTemplateID: workflowTemplateID,
		Name:               pgtype.Text{String: body.Name, Valid: body.Name != ""},
	})
	if err != nil {
		slog.ErrorContext(ctx, "create deployment", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create deployment")
		return
	}

	// Create waves from config.
	createdWaves := make([]sqlcgen.DeploymentWave, len(waveConfigs))
	for i, wc := range waveConfigs {
		wave, waveErr := txQ.CreateDeploymentWaveWithConfig(ctx, sqlcgen.CreateDeploymentWaveWithConfigParams{
			TenantID:          tid,
			DeploymentID:      dep.ID,
			WaveNumber:        int32(i + 1),
			Status:            string(deployment.WavePending),
			Percentage:        int32(wc.Percentage),
			SuccessThreshold:  pgtype.Numeric{Int: big.NewInt(int64(wc.SuccessThreshold * 100)), Exp: -2, Valid: true},
			ErrorRateMax:      pgtype.Numeric{Int: big.NewInt(int64(wc.ErrorRateMax * 100)), Exp: -2, Valid: true},
			DelayAfterMinutes: int32(wc.DelayMinutes),
		})
		if waveErr != nil {
			slog.ErrorContext(ctx, "create deployment wave", "deployment_id", uuidToString(dep.ID), "wave_number", i+1, "tenant_id", tenantID, "error", waveErr)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create deployment wave")
			return
		}
		createdWaves[i] = wave
	}

	// Collect the resolved targets — either from policy evaluation or adhoc cross-product.
	var resolvedTargets []deployment.Target
	if evalResult != nil {
		resolvedTargets = evalResult.Targets
	} else if len(adhocTargets) > 0 {
		resolvedTargets = adhocTargets
	}

	// Assign targets to waves.
	targetCount := 0
	if len(resolvedTargets) > 0 {
		targetCount = len(resolvedTargets)
		waveAssignments := deployment.AssignTargetsToWaves(waveConfigs, targetCount)

		// Shuffle targets randomly before assigning to waves.
		shuffled := make([]deployment.Target, len(resolvedTargets))
		copy(shuffled, resolvedTargets)
		rand.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		targetIdx := 0
		for waveIdx, count := range waveAssignments {
			for j := 0; j < count && targetIdx < len(shuffled); j++ {
				t := shuffled[targetIdx]
				_, targetErr := txQ.CreateDeploymentTargetWithWave(ctx, sqlcgen.CreateDeploymentTargetWithWaveParams{
					TenantID:     tid,
					DeploymentID: dep.ID,
					EndpointID:   t.EndpointID,
					PatchID:      t.PatchID,
					Status:       string(deployment.TargetPending),
					WaveID:       createdWaves[waveIdx].ID,
				})
				if targetErr != nil {
					slog.ErrorContext(ctx, "create deployment target", "deployment_id", uuidToString(dep.ID), "tenant_id", tenantID, "error", targetErr)
					WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create deployment target")
					return
				}
				targetIdx++
			}

			// Set wave target count.
			_, setErr := txQ.SetDeploymentWaveTargetCount(ctx, sqlcgen.SetDeploymentWaveTargetCountParams{
				ID:          createdWaves[waveIdx].ID,
				TargetCount: int32(count),
				TenantID:    tid,
			})
			if setErr != nil {
				slog.ErrorContext(ctx, "set wave target count", "deployment_id", uuidToString(dep.ID), "wave_number", waveIdx+1, "tenant_id", tenantID, "error", setErr)
				WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set wave target count")
				return
			}
		}
	}

	// Set total_targets count on the deployment.
	dep, err = txQ.SetDeploymentTotalTargets(ctx, sqlcgen.SetDeploymentTotalTargetsParams{
		ID:           dep.ID,
		TotalTargets: int32(targetCount),
		TenantID:     tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "set deployment total_targets", "deployment_id", uuidToString(dep.ID), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set deployment target count")
		return
	}

	// Enqueue executor River job inside the transaction so it commits atomically.
	var insertOpts *river.InsertOpts
	if body.ScheduledAt != nil {
		insertOpts = &river.InsertOpts{ScheduledAt: *body.ScheduledAt}
	}
	_, err = h.riverClient.InsertTx(ctx, tx, deployment.ExecutorJobArgs{
		DeploymentID: uuidToString(dep.ID),
		TenantID:     tenantID,
	}, insertOpts)
	if err != nil {
		slog.ErrorContext(ctx, "enqueue executor job", "deployment_id", uuidToString(dep.ID), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to enqueue deployment executor")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit deployment tx", "deployment_id", uuidToString(dep.ID), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit deployment")
		return
	}

	emitEvent(ctx, h.eventBus, events.DeploymentCreated, "deployment", uuidToString(dep.ID), tenantID, dep)

	WriteJSON(w, http.StatusCreated, createDeploymentResponse{
		deploymentResponse: toDeploymentResponse(dep),
		TargetCount:        targetCount,
	})
}

// List handles GET /api/v1/deployments.
func (h *DeploymentHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	cursorTime, cursorID, err := DecodeCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "invalid pagination cursor")
		return
	}
	limit := ParseLimit(r.URL.Query().Get("limit"))

	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var cursorTS pgtype.Timestamptz
	var cursorUUID pgtype.UUID
	if !cursorTime.IsZero() {
		cursorTS = pgtype.Timestamptz{Time: cursorTime, Valid: true}
		cursorUUID, err = scanUUID(cursorID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "invalid pagination cursor: cursor ID is not a valid UUID")
			return
		}
	}

	var policyID pgtype.UUID
	if pid := r.URL.Query().Get("policy_id"); pid != "" {
		policyID, err = scanUUID(pid)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_POLICY_ID", "invalid policy_id: not a valid UUID")
			return
		}
	}

	var createdAfter, createdBefore pgtype.Timestamptz
	if ca := r.URL.Query().Get("created_after"); ca != "" {
		t, parseErr := time.Parse(time.RFC3339, ca)
		if parseErr != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_CREATED_AFTER", "created_after must be RFC3339 format")
			return
		}
		createdAfter = pgtype.Timestamptz{Time: t, Valid: true}
	}
	if cb := r.URL.Query().Get("created_before"); cb != "" {
		t, parseErr := time.Parse(time.RFC3339, cb)
		if parseErr != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_CREATED_BEFORE", "created_before must be RFC3339 format")
			return
		}
		createdBefore = pgtype.Timestamptz{Time: t, Valid: true}
	}

	params := sqlcgen.ListDeploymentsByTenantFilteredParams{
		TenantID:        tid,
		Status:          r.URL.Query().Get("status"),
		PolicyID:        policyID,
		CreatedAfter:    createdAfter,
		CreatedBefore:   createdBefore,
		CursorCreatedAt: cursorTS,
		CursorID:        cursorUUID,
		PageLimit:       limit,
	}

	deployments, err := h.q.ListDeploymentsByTenantFiltered(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "list deployments", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list deployments")
		return
	}

	var nextCursor string
	if len(deployments) == int(limit) {
		last := deployments[len(deployments)-1]
		nextCursor = EncodeCursor(last.CreatedAt.Time, uuidToString(last.ID))
	}

	// Fetch status counts for filter pills.
	statusRows, scErr := h.q.CountDeploymentsByStatus(ctx, tid)
	if scErr != nil {
		slog.ErrorContext(ctx, "count deployments by status", "tenant_id", tenantID, "error", scErr)
		// Non-fatal — return empty counts.
		statusRows = nil
	}
	statusCounts := make(map[string]int, len(statusRows))
	var totalCount int64
	for _, row := range statusRows {
		statusCounts[row.Status] = int(row.Count)
		totalCount += int64(row.Count)
	}

	// Build policy name cache for the deployments in this page.
	policyNames := make(map[string]string)
	for _, d := range deployments {
		if d.PolicyID.Valid {
			pidStr := uuidToString(d.PolicyID)
			if _, seen := policyNames[pidStr]; !seen {
				p, pErr := h.q.GetPolicyByID(ctx, sqlcgen.GetPolicyByIDParams{ID: d.PolicyID, TenantID: tid})
				if pErr == nil {
					policyNames[pidStr] = p.Name
				}
			}
		}
	}

	// Convert to clean API response types.
	depResponses := make([]deploymentResponse, len(deployments))
	for i, d := range deployments {
		depResponses[i] = toDeploymentResponse(d)
		depResponses[i].PolicyName = policyNames[depResponses[i].PolicyID]
	}

	listResp := deploymentListResponse{
		Data:         depResponses,
		TotalCount:   totalCount,
		StatusCounts: statusCounts,
	}
	if nextCursor != "" {
		listResp.NextCursor = &nextCursor
	}
	WriteJSON(w, http.StatusOK, listResp)
}

// Get handles GET /api/v1/deployments/{id}.
func (h *DeploymentHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid deployment ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	dep, err := h.q.GetDeploymentByID(ctx, sqlcgen.GetDeploymentByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "deployment not found")
			return
		}
		slog.ErrorContext(ctx, "get deployment", "deployment_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get deployment")
		return
	}

	targets, err := h.q.ListDeploymentTargetsWithHostname(ctx, sqlcgen.ListDeploymentTargetsWithHostnameParams{
		DeploymentID: id,
		TenantID:     tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list deployment targets", "deployment_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list deployment targets")
		return
	}

	targetResponses := make([]deploymentTargetResponse, len(targets))
	for i, t := range targets {
		targetResponses[i] = toDeploymentTargetWithHostnameResponse(t)
	}

	WriteJSON(w, http.StatusOK, deploymentDetailResponse{
		deploymentResponse: toDeploymentResponse(dep),
		Targets:            targetResponses,
	})
}

// waveResponse is the clean API response type for a deployment wave.
type waveResponse struct {
	ID                string     `json:"id"`
	DeploymentID      string     `json:"deployment_id"`
	WaveNumber        int32      `json:"wave_number"`
	Status            string     `json:"status"`
	Percentage        int32      `json:"percentage"`
	TargetCount       int32      `json:"target_count"`
	SuccessCount      int32      `json:"success_count"`
	FailedCount       int32      `json:"failed_count"`
	DelayAfterMinutes int32      `json:"delay_after_minutes"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	EligibleAt        *time.Time `json:"eligible_at,omitempty"`
}

// toWaveResponse converts a sqlcgen.DeploymentWave to a clean API response.
func toWaveResponse(w sqlcgen.DeploymentWave) waveResponse {
	resp := waveResponse{
		ID:                uuidToString(w.ID),
		DeploymentID:      uuidToString(w.DeploymentID),
		WaveNumber:        w.WaveNumber,
		Status:            w.Status,
		Percentage:        w.Percentage,
		TargetCount:       w.TargetCount,
		SuccessCount:      w.SuccessCount,
		FailedCount:       w.FailedCount,
		DelayAfterMinutes: w.DelayAfterMinutes,
	}
	if w.StartedAt.Valid {
		resp.StartedAt = &w.StartedAt.Time
	}
	if w.CompletedAt.Valid {
		resp.CompletedAt = &w.CompletedAt.Time
	}
	if w.EligibleAt.Valid {
		resp.EligibleAt = &w.EligibleAt.Time
	}
	return resp
}

// GetWaves handles GET /api/v1/deployments/{id}/waves.
func (h *DeploymentHandler) GetWaves(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid deployment ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	waves, err := h.q.ListDeploymentWaves(ctx, sqlcgen.ListDeploymentWavesParams{
		DeploymentID: id,
		TenantID:     tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list deployment waves", "deployment_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list deployment waves")
		return
	}

	resp := make([]waveResponse, len(waves))
	for i, dw := range waves {
		resp[i] = toWaveResponse(dw)
	}

	WriteJSON(w, http.StatusOK, resp)
}

// Cancel handles POST /api/v1/deployments/{id}/cancel.
func (h *DeploymentHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid deployment ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	cancelQ, commit, rollback, txErr := h.beginCancelTx(ctx, tenantID)
	if txErr != nil {
		slog.ErrorContext(ctx, "begin cancel deployment tx", "tenant_id", tenantID, "error", txErr)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to begin transaction")
		return
	}
	defer func() {
		if err := rollback(); err != nil {
			slog.WarnContext(ctx, "rollback deployment tx", "error", err)
		}
	}()

	dep, pendingEvents, err := h.sm.CancelDeployment(ctx, cancelQ, id, tid)
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "deployment not found")
			return
		}
		slog.ErrorContext(ctx, "cancel deployment", "deployment_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to cancel deployment")
		return
	}

	if err := commit(); err != nil {
		slog.ErrorContext(ctx, "commit cancel deployment tx", "deployment_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit cancellation")
		return
	}

	// Post-commit event emission is best-effort: the DB state is authoritative.
	deployment.EmitBestEffort(ctx, h.eventBus, pendingEvents)

	WriteJSON(w, http.StatusOK, toDeploymentResponse(dep))
}

// beginCancelTx returns a CancelQuerier with commit/rollback. If cancelTxFactory
// is set (tests), it delegates there. Otherwise, it delegates to beginTx.
func (h *DeploymentHandler) beginCancelTx(ctx context.Context, tenantID string) (deployment.CancelQuerier, func() error, func() error, error) {
	if h.cancelTxFactory != nil {
		return h.cancelTxFactory(ctx, tenantID)
	}
	return h.beginTx(ctx, tenantID)
}

// beginTx opens a tenant-scoped transaction and returns the full sqlcgen.Queries.
// Use this for handlers that need query methods beyond CancelQuerier.
func (h *DeploymentHandler) beginTx(ctx context.Context, tenantID string) (*sqlcgen.Queries, func() error, func() error, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("begin tx: %w", err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			slog.WarnContext(ctx, "beginTx: rollback after set_config failure", "error", rbErr)
		}
		return nil, nil, nil, fmt.Errorf("set tenant context: %w", err)
	}
	rollback := func() error {
		err := tx.Rollback(ctx)
		if errors.Is(err, pgx.ErrTxClosed) {
			return nil
		}
		return err
	}
	return sqlcgen.New(tx), func() error { return tx.Commit(ctx) }, rollback, nil
}

// Retry handles POST /api/v1/deployments/{id}/retry.
func (h *DeploymentHandler) Retry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid deployment ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	q, commit, rollback, txErr := h.beginTx(ctx, tenantID)
	if txErr != nil {
		slog.ErrorContext(ctx, "begin retry deployment tx", "tenant_id", tenantID, "error", txErr)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to begin transaction")
		return
	}
	defer func() {
		if err := rollback(); err != nil {
			slog.WarnContext(ctx, "rollback retry deployment tx", "error", err)
		}
	}()

	dep, pendingEvents, err := h.sm.RetryDeployment(ctx, q, id, tid)
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "deployment not found or not in retryable state")
			return
		}
		slog.ErrorContext(ctx, "retry deployment", "deployment_id", chi.URLParam(r, "id"), "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retry deployment")
		return
	}

	if err := commit(); err != nil {
		slog.ErrorContext(ctx, "commit retry deployment tx", "deployment_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit retry")
		return
	}

	deployment.EmitBestEffort(ctx, h.eventBus, pendingEvents)
	WriteJSON(w, http.StatusOK, toDeploymentResponse(dep))
}

// Rollback handles POST /api/v1/deployments/{id}/rollback.
func (h *DeploymentHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid deployment ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	q, commit, rollback, txErr := h.beginTx(ctx, tenantID)
	if txErr != nil {
		slog.ErrorContext(ctx, "begin rollback deployment tx", "tenant_id", tenantID, "error", txErr)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to begin transaction")
		return
	}
	defer func() {
		if err := rollback(); err != nil {
			slog.WarnContext(ctx, "rollback deployment tx", "error", err)
		}
	}()

	dep, pendingEvents, err := h.sm.RollbackDeployment(ctx, q, id, tid)
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "deployment not found or not in rollbackable state")
			return
		}
		slog.ErrorContext(ctx, "rollback deployment", "deployment_id", chi.URLParam(r, "id"), "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to rollback deployment")
		return
	}

	if err := commit(); err != nil {
		slog.ErrorContext(ctx, "commit rollback deployment tx", "deployment_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit rollback")
		return
	}

	deployment.EmitBestEffort(ctx, h.eventBus, pendingEvents)
	WriteJSON(w, http.StatusOK, toDeploymentResponse(dep))
}

// deploymentPatchSummaryResponse is the JSON response for GET /deployments/{id}/patches.
type deploymentPatchSummaryResponse struct {
	PatchID       string `json:"patch_id"`
	PatchTitle    string `json:"patch_title"`
	PatchVersion  string `json:"patch_version"`
	PatchSeverity string `json:"patch_severity"`
	TotalTargets  int    `json:"total_targets"`
	SuccessCount  int    `json:"success_count"`
	FailedCount   int    `json:"failed_count"`
}

// GetPatchSummary handles GET /api/v1/deployments/{id}/patches.
func (h *DeploymentHandler) GetPatchSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid deployment ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.ListDeploymentPatchSummary(ctx, sqlcgen.ListDeploymentPatchSummaryParams{
		DeploymentID: id,
		TenantID:     tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list deployment patch summary", "deployment_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list patch summary")
		return
	}

	resp := make([]deploymentPatchSummaryResponse, len(rows))
	for i, row := range rows {
		resp[i] = deploymentPatchSummaryResponse{
			PatchID:       uuidToString(row.PatchID),
			PatchTitle:    row.PatchTitle,
			PatchVersion:  row.PatchVersion,
			PatchSeverity: row.PatchSeverity,
			TotalTargets:  int(row.TotalTargets),
			SuccessCount:  int(row.SuccessCount),
			FailedCount:   int(row.FailedCount),
		}
	}

	WriteJSON(w, http.StatusOK, resp)
}
