package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SyncInbox streams pending commands to the agent.
func (s *AgentServiceServer) SyncInbox(req *pb.InboxRequest, stream grpc.ServerStreamingServer[pb.CommandRequest]) error {
	if req.GetAgentId() == "" {
		return status.Errorf(codes.InvalidArgument, "sync inbox: agent_id is required")
	}

	agentUUID, err := uuid.Parse(req.GetAgentId())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "sync inbox: invalid agent_id: %v", err)
	}

	ctx := stream.Context()
	queries := sqlcgen.New(s.store.BypassPool())
	pgUUID := pgtype.UUID{Bytes: agentUUID, Valid: true}
	endpoint, err := queries.LookupEndpointByID(ctx, pgUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return status.Errorf(codes.NotFound, "sync inbox: endpoint not found")
		}
		return status.Errorf(codes.Internal, "sync inbox: lookup endpoint: %v", err)
	}

	// Set tenant context for RLS-scoped queries.
	tenantIDStr := uuid.UUID(endpoint.TenantID.Bytes).String()
	ctx = tenant.WithTenantID(ctx, tenantIDStr)

	s.logger.InfoContext(ctx, "sync inbox: stream opened",
		"agent_id", req.GetAgentId(),
		"tenant_id", tenantIDStr,
	)

	// Begin a tenant-scoped transaction for listing and marking commands.
	// The commands table has RLS, so we must SET LOCAL app.current_tenant_id.
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "sync inbox: begin tx: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.DebugContext(ctx, "sync inbox: rollback tx", "error", err)
		}
	}()

	qtx := sqlcgen.New(tx)

	// List and stream pending commands within the tenant-scoped transaction.
	commands, err := qtx.ListPendingCommandsByAgent(ctx, sqlcgen.ListPendingCommandsByAgentParams{
		AgentID:  pgUUID,
		TenantID: endpoint.TenantID,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "sync inbox: list commands: %v", err)
	}

	// Track dispatched run_scan commands so we can emit scan.dispatched events
	// only after the transaction commits — emitting inside the loop would
	// produce phantom audit rows if the commit later rolls back (ADR-014).
	type dispatchedScan struct{ commandID, endpointID string }
	var dispatchedScans []dispatchedScan

	for _, cmd := range commands {
		pbCmd := &pb.CommandRequest{
			CommandId: uuid.UUID(cmd.ID.Bytes).String(),
			Type:      mapCommandType(s.logger, ctx, deployment.CommandType(cmd.Type)),
			Payload:   cmd.Payload,
			Priority:  uint32(cmd.Priority),
			AgentId:   req.GetAgentId(),
		}
		if cmd.Deadline.Valid {
			pbCmd.Deadline = timestamppb.New(cmd.Deadline.Time)
		}
		if err := stream.Send(pbCmd); err != nil {
			return fmt.Errorf("sync inbox: send command: %w", err)
		}

		// Mark as delivered after successful send. If this fails, the command may be
		// re-sent on the next SyncInbox call (at-least-once delivery). Agents must
		// handle duplicate commands idempotently.
		if _, markErr := qtx.MarkCommandDelivered(ctx, sqlcgen.MarkCommandDeliveredParams{
			ID:       cmd.ID,
			TenantID: cmd.TenantID,
		}); markErr != nil {
			s.logger.ErrorContext(ctx, "sync inbox: mark command delivered failed, command may be re-sent",
				"command_id", uuid.UUID(cmd.ID.Bytes).String(),
				"error", markErr,
			)
		}

		if cmd.Type == string(deployment.CommandTypeRunScan) {
			dispatchedScans = append(dispatchedScans, dispatchedScan{
				commandID:  uuid.UUID(cmd.ID.Bytes).String(),
				endpointID: uuid.UUID(cmd.AgentID.Bytes).String(),
			})
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return status.Errorf(codes.Internal, "sync inbox: commit tx: %v", err)
	}

	// Post-commit: emit scan.dispatched so the endpoint audit timeline shows
	// the pickup step between scan.triggered and scan.completed. Best-effort;
	// failures are logged but do not fail the RPC.
	for _, d := range dispatchedScans {
		evt := domain.NewSystemEvent(
			events.ScanDispatched,
			tenantIDStr,
			"endpoint",
			d.endpointID,
			events.ScanDispatched,
			events.ScanDispatchedPayload{
				CommandID:  d.commandID,
				EndpointID: d.endpointID,
			},
		)
		if emitErr := s.eventBus.Emit(ctx, evt); emitErr != nil {
			s.logger.ErrorContext(ctx, "sync inbox: emit scan.dispatched failed",
				"command_id", d.commandID,
				"endpoint_id", d.endpointID,
				"tenant_id", tenantIDStr,
				"error", emitErr)
		}
	}

	return nil
}

func mapCommandType(logger *slog.Logger, ctx context.Context, dbType deployment.CommandType) pb.CommandType {
	switch dbType {
	case deployment.CommandTypeInstallPatch:
		return pb.CommandType_COMMAND_TYPE_INSTALL_PATCH
	case deployment.CommandTypeRunScan:
		return pb.CommandType_COMMAND_TYPE_RUN_SCAN
	case deployment.CommandTypeUpdateConfig:
		return pb.CommandType_COMMAND_TYPE_UPDATE_CONFIG
	case deployment.CommandTypeReboot:
		return pb.CommandType_COMMAND_TYPE_REBOOT
	case deployment.CommandTypeRunScript:
		return pb.CommandType_COMMAND_TYPE_RUN_SCRIPT
	case deployment.CommandTypeRollbackPatch:
		return pb.CommandType_COMMAND_TYPE_ROLLBACK_PATCH
	default:
		logger.WarnContext(ctx, "sync inbox: unknown command type, mapping to UNSPECIFIED", "db_type", dbType)
		return pb.CommandType_COMMAND_TYPE_UNSPECIFIED
	}
}
