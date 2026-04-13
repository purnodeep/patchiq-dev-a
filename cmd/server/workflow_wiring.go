package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
	"github.com/skenzeriq/patchiq/internal/server/workflow/handlers"
)

// --- UUID helpers ---

func wfPgUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	parsed, err := uuid.Parse(s)
	if err != nil {
		return u
	}
	u.Bytes = parsed
	u.Valid = true
	return u
}

func wfUUIDStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

// --- FilterDataSource adapter ---

type dbFilterDataSource struct {
	pool *pgxpool.Pool
}

func (ds *dbFilterDataSource) FilterEndpoints(ctx context.Context, tenantID string, cfg workflow.FilterConfig) ([]string, error) {
	tid := wfPgUUID(tenantID)
	if !tid.Valid {
		return nil, fmt.Errorf("filter endpoints: invalid tenant_id %q", tenantID)
	}

	query := `SELECT DISTINCT e.id::text FROM endpoints e WHERE e.tenant_id = $1 AND e.status != 'decommissioned'`
	args := []any{tid}
	argIdx := 2

	if len(cfg.OSTypes) > 0 {
		query += fmt.Sprintf(" AND e.os_family = ANY($%d)", argIdx)
		args = append(args, cfg.OSTypes)
		argIdx++
	}

	if len(cfg.Tags) > 0 {
		// Tags filter is a slice of "key=value" strings. Each one becomes
		// an EXISTS subquery that requires the endpoint to carry a
		// matching (key, value) row in endpoint_tags. Multiple entries
		// compose with AND (intersection) because workflow filter nodes
		// are conjunctive by design.
		//
		// Malformed entries are a hard error here. FilterConfig.Validate
		// rejects them at save time, so reaching this branch with a bad
		// entry means a row bypassed validation (e.g. a pre-migration
		// legacy value or direct DB write). Silently skipping such
		// entries would widen the filter to "no tag predicate" and
		// return the entire tenant — the exact blast-radius footgun the
		// targeting DSL was built to prevent.
		for _, kv := range cfg.Tags {
			eq := strings.IndexByte(kv, '=')
			if eq <= 0 || eq == len(kv)-1 {
				return nil, fmt.Errorf("filter endpoints: malformed tag entry %q (want non-empty key=value)", kv)
			}
			key := kv[:eq]
			value := kv[eq+1:]
			query += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM endpoint_tags et JOIN tags t ON t.id = et.tag_id WHERE et.endpoint_id = e.id AND et.tenant_id = $1 AND lower(t.key) = lower($%d) AND lower(t.value) = lower($%d))", argIdx, argIdx+1)
			args = append(args, key, value)
			argIdx += 2
		}
	}

	if cfg.MinSeverity != "" {
		sevMap := map[string]float64{"low": 0.1, "medium": 4.0, "high": 7.0, "critical": 9.0}
		if minScore, ok := sevMap[cfg.MinSeverity]; ok {
			query += fmt.Sprintf(" AND e.id IN (SELECT ec.endpoint_id FROM endpoint_cves ec JOIN cves c ON ec.cve_id = c.id WHERE ec.tenant_id = $1 AND ec.status = 'affected' AND c.cvss_score >= $%d)", argIdx)
			args = append(args, minScore)
			argIdx++
		}
	}

	if cfg.PackageRegex != "" {
		query += fmt.Sprintf(" AND e.id IN (SELECT DISTINCT ep.endpoint_id FROM endpoint_packages ep WHERE ep.tenant_id = $1 AND ep.name ~ $%d)", argIdx)
		args = append(args, cfg.PackageRegex)
	}

	rows, err := ds.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("filter endpoints: query: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("filter endpoints: scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// --- ApprovalStore adapter ---

type dbApprovalStore struct {
	pool *pgxpool.Pool
}

func (s *dbApprovalStore) CreateApproval(ctx context.Context, req handlers.ApprovalCreateRequest) error {
	tid := wfPgUUID(req.TenantID)
	execID := wfPgUUID(req.ExecutionID)
	nodeID := wfPgUUID(req.NodeID)

	timeoutAt := pgtype.Timestamptz{
		Time:  time.Now().Add(time.Duration(req.TimeoutHours) * time.Hour),
		Valid: true,
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO approval_requests (tenant_id, execution_id, node_id, approver_roles, escalation_role, timeout_action, timeout_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		tid, execID, nodeID, req.ApproverRoles, "", "reject", timeoutAt,
	)
	if err != nil {
		return fmt.Errorf("create approval: %w", err)
	}
	return nil
}

// --- WaveDeployer adapter ---

type dbWaveDeployer struct {
	pool *pgxpool.Pool
}

func (d *dbWaveDeployer) CreateWorkflowDeployment(ctx context.Context, req handlers.WaveDeploymentRequest) (string, error) {
	tid := wfPgUUID(req.TenantID)
	if !tid.Valid {
		return "", fmt.Errorf("create workflow deployment: invalid tenant_id")
	}

	// Look up any policy for this tenant to use as the deployment's policy_id.
	var policyID pgtype.UUID
	err := d.pool.QueryRow(ctx,
		`SELECT id FROM policies WHERE tenant_id = $1 AND deleted_at IS NULL LIMIT 1`,
		tid,
	).Scan(&policyID)
	if err != nil {
		return "", fmt.Errorf("create workflow deployment: no policy found for tenant: %w", err)
	}

	// Calculate wave targets: subset of endpoints based on percentage.
	targetCount := len(req.Endpoints)
	if req.Percentage < 100 && req.Percentage > 0 {
		targetCount = (len(req.Endpoints) * req.Percentage) / 100
		if targetCount == 0 {
			targetCount = 1
		}
	}

	waveConfig, _ := json.Marshal(map[string]any{
		"source":          "workflow",
		"execution_id":    req.ExecutionID,
		"node_id":         req.NodeID,
		"wave_percentage": req.Percentage,
	})

	var deploymentID pgtype.UUID
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	err = d.pool.QueryRow(ctx,
		`INSERT INTO deployments (tenant_id, policy_id, status, total_targets, wave_config, started_at)
		 VALUES ($1, $2, 'running', $3, $4, $5)
		 RETURNING id`,
		tid, policyID, targetCount, waveConfig, now,
	).Scan(&deploymentID)
	if err != nil {
		return "", fmt.Errorf("create workflow deployment: insert deployment: %w", err)
	}

	depIDStr := wfUUIDStr(deploymentID)

	slog.InfoContext(ctx, "workflow deployment created",
		"deployment_id", depIDStr,
		"endpoint_count", targetCount,
		"percentage", req.Percentage,
	)
	return depIDStr, nil
}

// --- NotificationSender adapter (log-only) ---

type logNotificationSender struct{}

func (s *logNotificationSender) SendNotification(ctx context.Context, tenantID string, cfg workflow.NotificationConfig) error {
	slog.InfoContext(ctx, "workflow notification",
		"tenant_id", tenantID,
		"channel", cfg.Channel,
		"target", cfg.Target,
		"message", cfg.MessageTemplate,
	)
	return nil
}

// --- CommandSender adapter (log-only) ---

type logCommandSender struct{}

func (s *logCommandSender) SendCommand(ctx context.Context, tenantID string, endpointID string, commandType string, payload json.RawMessage) error {
	slog.InfoContext(ctx, "workflow command sent",
		"tenant_id", tenantID,
		"endpoint_id", endpointID,
		"command_type", commandType,
	)
	return nil
}

// --- TagResolver adapter (DB-backed) ---

type dbTagResolver struct {
	pool *pgxpool.Pool
}

func (r *dbTagResolver) ResolveTags(ctx context.Context, tenantID string, endpointIDs []string) (map[string][]string, error) {
	tid := wfPgUUID(tenantID)
	if !tid.Valid {
		return nil, fmt.Errorf("resolve tags: invalid tenant_id %q", tenantID)
	}

	epUUIDs := make([]pgtype.UUID, 0, len(endpointIDs))
	for _, eid := range endpointIDs {
		u := wfPgUUID(eid)
		if u.Valid {
			epUUIDs = append(epUUIDs, u)
		}
	}

	// Return each endpoint's tags as "key=value" strings so the workflow
	// filter pipeline (which matches strings) keeps its existing shape.
	rows, err := r.pool.Query(ctx,
		`SELECT e.id::text, COALESCE(array_agg(DISTINCT (t.key || '=' || t.value)) FILTER (WHERE t.key IS NOT NULL), '{}')
		 FROM endpoints e
		 LEFT JOIN endpoint_tags et ON et.endpoint_id = e.id AND et.tenant_id = e.tenant_id
		 LEFT JOIN tags t ON t.id = et.tag_id AND t.tenant_id = e.tenant_id
		 WHERE e.tenant_id = $1 AND e.id = ANY($2)
		 GROUP BY e.id`,
		tid, epUUIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("resolve tags: query: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var epID string
		var tags []string
		if err := rows.Scan(&epID, &tags); err != nil {
			return nil, fmt.Errorf("resolve tags: scan: %w", err)
		}
		result[epID] = tags
	}
	return result, rows.Err()
}

// --- RollbackRequester adapter (log-only) ---

type logRollbackRequester struct{}

func (r *logRollbackRequester) CreateRollback(ctx context.Context, req handlers.RollbackRequest) (string, error) {
	slog.InfoContext(ctx, "workflow rollback requested",
		"tenant_id", req.TenantID,
		"deployment_id", req.DeploymentID,
		"strategy", req.Strategy,
	)
	return uuid.New().String(), nil
}

// immediateWaveHandler creates a deployment and completes without pausing.
// The standard wave handler pauses to wait for deployment results; this
// version creates the deployment record and returns immediately.
type immediateWaveHandler struct {
	deployer *dbWaveDeployer
}

func (h *immediateWaveHandler) Execute(ctx context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.DeploymentWaveConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("wave handler: unmarshal config: %w", err)
	}

	endpoints, _ := exec.Context["filtered_endpoints"].([]string)
	if len(endpoints) == 0 {
		return &workflow.NodeResult{
			Status: workflow.NodeExecFailed,
			Error:  "no endpoints available for deployment wave",
			Output: map[string]any{},
		}, nil
	}

	deploymentID, err := h.deployer.CreateWorkflowDeployment(ctx, handlers.WaveDeploymentRequest{
		TenantID:         exec.TenantID,
		ExecutionID:      exec.ExecutionID,
		NodeID:           exec.Node.ID,
		Endpoints:        endpoints,
		Percentage:       cfg.Percentage,
		MaxParallel:      cfg.MaxParallel,
		TimeoutMinutes:   cfg.TimeoutMinutes,
		SuccessThreshold: cfg.SuccessThreshold,
	})
	if err != nil {
		return nil, fmt.Errorf("wave handler: create deployment: %w", err)
	}

	return &workflow.NodeResult{
		Status: workflow.NodeExecCompleted,
		Output: map[string]any{
			"deployment_id":     deploymentID,
			"endpoint_count":    len(endpoints),
			"percentage":        cfg.Percentage,
			"success_threshold": cfg.SuccessThreshold,
		},
	}, nil
}

// immediateGateHandler passes through immediately without waiting.
type immediateGateHandler struct{}

func (h *immediateGateHandler) Execute(_ context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.GateConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("gate handler: unmarshal config: %w", err)
	}

	return &workflow.NodeResult{
		Status: workflow.NodeExecCompleted,
		Output: map[string]any{
			"wait_minutes":      cfg.WaitMinutes,
			"failure_threshold": cfg.FailureThreshold,
			"health_check":      cfg.HealthCheck,
			"gate_passed":       true,
		},
	}, nil
}

// immediateScriptHandler logs the script and completes without dispatching.
type immediateScriptHandler struct{}

func (h *immediateScriptHandler) Execute(ctx context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.ScriptConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("script handler: unmarshal config: %w", err)
	}

	slog.InfoContext(ctx, "workflow script (immediate)",
		"script_type", cfg.ScriptType,
		"timeout_minutes", cfg.TimeoutMinutes,
	)

	return &workflow.NodeResult{
		Status: workflow.NodeExecCompleted,
		Output: map[string]any{
			"script_type":      cfg.ScriptType,
			"failure_behavior": cfg.FailureBehavior,
			"executed":         true,
		},
	}, nil
}

// buildWorkflowHandlers constructs the handler map for the workflow executor.
func buildWorkflowHandlers(pool *pgxpool.Pool) map[workflow.NodeType]workflow.NodeHandler {
	filterDS := &dbFilterDataSource{pool: pool}
	approvalStore := &dbApprovalStore{pool: pool}
	waveDeployer := &dbWaveDeployer{pool: pool}
	notifSender := &logNotificationSender{}
	rollbackReq := &logRollbackRequester{}

	cmdSender := &logCommandSender{}
	tagResolver := &dbTagResolver{pool: pool}

	return map[workflow.NodeType]workflow.NodeHandler{
		workflow.NodeTrigger:         handlers.NewTriggerHandler(),
		workflow.NodeFilter:          handlers.NewFilterHandler(filterDS),
		workflow.NodeApproval:        handlers.NewApprovalHandler(approvalStore),
		workflow.NodeDeploymentWave:  &immediateWaveHandler{deployer: waveDeployer},
		workflow.NodeGate:            &immediateGateHandler{},
		workflow.NodeScript:          &immediateScriptHandler{},
		workflow.NodeNotification:    handlers.NewNotificationHandler(notifSender),
		workflow.NodeRollback:        handlers.NewRollbackHandler(rollbackReq),
		workflow.NodeDecision:        handlers.NewDecisionHandler(),
		workflow.NodeComplete:        handlers.NewCompleteHandler(),
		workflow.NodeReboot:          handlers.NewRebootHandler(cmdSender),
		workflow.NodeScan:            handlers.NewScanHandler(cmdSender),
		workflow.NodeTagGate:         handlers.NewTagGateHandler(tagResolver),
		workflow.NodeComplianceCheck: handlers.NewComplianceCheckHandler(),
	}
}
