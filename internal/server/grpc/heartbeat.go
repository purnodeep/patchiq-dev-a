package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Heartbeat implements bidirectional streaming: receives agent status updates,
// persists them, emits domain events, and responds with server timestamp.
func (s *AgentServiceServer) Heartbeat(stream pb.AgentService_HeartbeatServer) error {
	ctx := stream.Context()

	// 1. Receive first message to identify the agent.
	first, err := stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			s.logger.WarnContext(ctx, "heartbeat: stream closed immediately without sending any message")
			return nil
		}
		return status.Errorf(codes.Unknown, "heartbeat: recv first message: %v", err)
	}

	if first.AgentId == "" {
		return status.Errorf(codes.InvalidArgument, "heartbeat: agent_id is required")
	}

	agentUUID, err := uuid.Parse(first.AgentId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "heartbeat: invalid agent_id: %v", err)
	}

	pgAgentID := pgtype.UUID{Bytes: agentUUID, Valid: true}

	// 2. Lookup endpoint (bypasses RLS — tenant unknown yet).
	queries := sqlcgen.New(s.store.BypassPool())
	endpoint, err := queries.LookupEndpointByID(ctx, pgAgentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return status.Errorf(codes.NotFound, "heartbeat: unknown agent_id %s", first.AgentId)
		}
		return status.Errorf(codes.Internal, "heartbeat: lookup endpoint: %v", err)
	}

	tenantIDStr := uuid.UUID(endpoint.TenantID.Bytes).String()
	ctx = tenant.WithTenantID(ctx, tenantIDStr)

	logger := s.logger.With(
		"agent_id", first.AgentId,
		"tenant_id", tenantIDStr,
	)
	logger.InfoContext(ctx, "heartbeat: stream opened")

	// 3. Process first message, then loop.
	if err := s.processHeartbeat(ctx, stream, first, pgAgentID, endpoint.TenantID, tenantIDStr, logger); err != nil {
		return err
	}

	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.InfoContext(ctx, "heartbeat: stream closed by agent")
				return nil
			}
			return status.Errorf(codes.Unknown, "heartbeat: recv: %v", err)
		}

		if err := s.processHeartbeat(ctx, stream, req, pgAgentID, endpoint.TenantID, tenantIDStr, logger); err != nil {
			return err
		}
	}
}

// processHeartbeat handles a single heartbeat message: updates the DB, emits an event, sends a response.
func (s *AgentServiceServer) processHeartbeat(
	ctx context.Context,
	stream pb.AgentService_HeartbeatServer,
	req *pb.HeartbeatRequest,
	agentID pgtype.UUID,
	tenantID pgtype.UUID,
	tenantIDStr string,
	logger *slog.Logger,
) error {
	dbStatus := mapAgentStatus(req.Status)

	// Compute resource usage for persistence.
	var uptimeSeconds pgtype.Int8
	if req.UptimeSeconds > 0 {
		uptimeSeconds = pgtype.Int8{Int64: req.UptimeSeconds, Valid: true}
	}
	var memUsedMb pgtype.Int8
	if req.ResourceUsage != nil && req.ResourceUsage.MemoryBytes > 0 {
		memUsedMb = pgtype.Int8{Int64: int64(req.ResourceUsage.MemoryBytes / (1024 * 1024)), Valid: true}
	}
	var diskUsedGb pgtype.Int8
	if req.ResourceUsage != nil && req.ResourceUsage.DiskBytes > 0 {
		diskUsedGb = pgtype.Int8{Int64: int64(req.ResourceUsage.DiskBytes / (1024 * 1024 * 1024)), Valid: true}
	}
	var cpuUsagePct pgtype.Int2
	if req.ResourceUsage != nil && req.ResourceUsage.CpuPercent >= 0 {
		cpu := req.ResourceUsage.CpuPercent
		if cpu > 100 || cpu != cpu { // cpu != cpu catches NaN
			logger.WarnContext(ctx, "heartbeat: cpu_percent out of valid range, discarding",
				"agent_id", req.AgentId,
				"cpu_percent", cpu,
			)
		} else {
			cpuUsagePct = pgtype.Int2{Int16: int16(cpu), Valid: true}
		}
	}

	// Update endpoint status + hardware metrics in a tenant-scoped transaction.
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "heartbeat: begin tx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := sqlcgen.New(tx)
	_, err = qtx.UpdateEndpointHeartbeat(ctx, sqlcgen.UpdateEndpointHeartbeatParams{
		ID:              agentID,
		Status:          dbStatus,
		UptimeSeconds:   uptimeSeconds,
		MemoryUsedMb:    memUsedMb,
		DiskUsedGb:      diskUsedGb,
		CpuUsagePercent: cpuUsagePct,
		TenantID:        tenantID,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "heartbeat: update endpoint status: %v", err)
	}

	// Count pending commands for this agent (must be before tx.Commit since qtx wraps the tx).
	pendingCount, countErr := qtx.CountPendingCommandsByAgent(ctx, sqlcgen.CountPendingCommandsByAgentParams{
		AgentID:  agentID,
		TenantID: tenantID,
	})
	if countErr != nil {
		return status.Errorf(codes.Internal, "heartbeat: count pending commands: %v", countErr)
	}

	if err := tx.Commit(ctx); err != nil {
		return status.Errorf(codes.Internal, "heartbeat: commit tx: %v", err)
	}

	// Emit domain event (non-fatal on error).
	evt := domain.NewSystemEvent(
		events.HeartbeatReceived,
		tenantIDStr,
		"endpoint",
		req.AgentId,
		dbStatus,
		map[string]string{
			"agent_status": req.Status.String(),
		},
	)
	if err := s.eventBus.Emit(ctx, evt); err != nil {
		logger.ErrorContext(ctx, "heartbeat: emit event failed", "error", err.Error())
	}

	// Check for pending config push (H-I5). Uses a new query since tx is committed.
	configUpdate := s.checkConfigUpdate(ctx, agentID, tenantID, tenantIDStr, req.AgentId, logger)

	// Send response.
	resp := &pb.HeartbeatResponse{
		ServerTimestamp: timestamppb.Now(),
		CommandsPending: uint32(pendingCount),
		ConfigUpdate:    configUpdate,
	}
	if err := stream.Send(resp); err != nil {
		return fmt.Errorf("heartbeat: send response: %w", err)
	}

	return nil
}

// commsConfig represents the JSON structure stored in config_overrides.config for module "comms".
type commsConfig struct {
	HeartbeatIntervalSeconds uint32 `json:"heartbeat_interval_seconds"`
	InventoryIntervalSeconds uint32 `json:"inventory_interval_seconds"`
	MaxRetryAttempts         uint32 `json:"max_retry_attempts"`
}

// checkConfigUpdate looks for a comms config override updated since the endpoint's
// last config push. Returns nil if no update is needed.
func (s *AgentServiceServer) checkConfigUpdate(
	ctx context.Context,
	agentID pgtype.UUID,
	tenantID pgtype.UUID,
	tenantIDStr string,
	agentIDStr string,
	logger *slog.Logger,
) *pb.AgentConfig {
	// Use bypass pool — RLS is not set in the heartbeat stream context, and
	// the tenant ID has already been verified via the endpoint lookup.
	bypassQ := sqlcgen.New(s.store.BypassPool())

	// threshold is the endpoint's config_pushed_at; if never pushed (NULL/invalid),
	// the SQL query's IS NULL branch ensures any config override is detected.
	endpoint, err := bypassQ.LookupEndpointByID(ctx, agentID)
	if err != nil {
		logger.ErrorContext(ctx, "heartbeat: config push: lookup endpoint failed", "error", err)
		return nil
	}

	threshold := endpoint.ConfigPushedAt // zero if never pushed

	override, err := bypassQ.GetCommsConfigForEndpoint(ctx, sqlcgen.GetCommsConfigForEndpointParams{
		TenantID:     tenantID,
		ScopeID:      agentID,
		UpdatedAfter: threshold,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // no updated config
		}
		logger.ErrorContext(ctx, "heartbeat: config push: query failed", "error", err)
		return nil
	}

	// Parse the JSONB config into commsConfig.
	var cc commsConfig
	if err := json.Unmarshal(override.Config, &cc); err != nil {
		logger.ErrorContext(ctx, "heartbeat: config push: unmarshal failed", "error", err)
		return nil
	}

	// Validate that at least one field is non-zero to avoid pushing empty/corrupt config.
	if cc.HeartbeatIntervalSeconds == 0 && cc.InventoryIntervalSeconds == 0 && cc.MaxRetryAttempts == 0 {
		logger.ErrorContext(ctx, "heartbeat: config push: all config fields are zero, skipping push")
		return nil
	}

	// Mark config as pushed via tenant-scoped transaction. If we can't mark it,
	// don't push the config — otherwise the agent gets the same update every heartbeat.
	tx, txErr := s.store.BeginTx(ctx)
	if txErr != nil {
		logger.ErrorContext(ctx, "heartbeat: config push: begin tx failed", "error", txErr)
		return nil
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := sqlcgen.New(tx)
	if err := qtx.UpdateEndpointConfigPushedAt(ctx, sqlcgen.UpdateEndpointConfigPushedAtParams{
		ID: agentID, TenantID: tenantID,
	}); err != nil {
		logger.ErrorContext(ctx, "heartbeat: config push: update pushed_at failed", "error", err)
		return nil
	}
	if err := tx.Commit(ctx); err != nil {
		logger.ErrorContext(ctx, "heartbeat: config push: commit failed", "error", err)
		return nil
	}

	// Emit domain event for the config push write (non-fatal on error).
	evt := domain.NewSystemEvent(
		events.EndpointConfigPushed,
		tenantIDStr,
		"endpoint",
		agentIDStr,
		"config_pushed",
		nil,
	)
	if err := s.eventBus.Emit(ctx, evt); err != nil {
		logger.ErrorContext(ctx, "heartbeat: config push: emit event failed", "error", err.Error())
	}

	logger.InfoContext(ctx, "heartbeat: pushing config update to agent")
	return &pb.AgentConfig{
		HeartbeatIntervalSeconds: cc.HeartbeatIntervalSeconds,
		InventoryIntervalSeconds: cc.InventoryIntervalSeconds,
		MaxRetryAttempts:         cc.MaxRetryAttempts,
	}
}

// mapAgentStatus converts a protobuf AgentStatus to a database status string.
// Valid endpoint statuses per chk_endpoints_status: pending, online, offline, stale.
func mapAgentStatus(s pb.AgentStatus) string {
	switch s {
	case pb.AgentStatus_AGENT_STATUS_IDLE, pb.AgentStatus_AGENT_STATUS_BUSY, pb.AgentStatus_AGENT_STATUS_UPDATING:
		return "online"
	case pb.AgentStatus_AGENT_STATUS_ERROR:
		return "offline"
	default:
		slog.Warn("heartbeat: unknown agent status, defaulting to online", "status", s.String())
		return "online"
	}
}
