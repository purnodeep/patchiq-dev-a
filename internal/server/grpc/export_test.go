package grpc

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"google.golang.org/grpc"
)

// ExportedLoggingUnaryInterceptor exposes the interceptor for testing.
var ExportedLoggingUnaryInterceptor grpc.UnaryServerInterceptor = loggingUnaryInterceptor

// ExportedMapAgentStatus exposes mapAgentStatus for testing.
var ExportedMapAgentStatus = mapAgentStatus

// ExportedMapCommandType exposes mapCommandType for testing.
func ExportedMapCommandType(dbType deployment.CommandType) pb.CommandType {
	return mapCommandType(slog.Default(), context.Background(), dbType)
}

// ExportedProcessInventory exposes processInventory for unit testing.
func (s *AgentServiceServer) ExportedProcessInventory(
	ctx context.Context,
	msg *pb.OutboxMessage,
	endpointID pgtype.UUID,
	tenantIDStr, agentIDStr string,
) (*pb.OutboxAck, error) {
	return s.processInventory(ctx, msg, endpointID, tenantIDStr, agentIDStr)
}

// ExportedProcessCommandResult exposes processCommandResult for unit testing.
func (s *AgentServiceServer) ExportedProcessCommandResult(
	ctx context.Context,
	tenantID, agentID string,
	msg *pb.OutboxMessage,
) error {
	return s.processCommandResult(ctx, tenantID, agentID, msg)
}

// ExportedErrPayloadInvalid exposes errPayloadInvalid for unit testing.
var ExportedErrPayloadInvalid = errPayloadInvalid

// ExportedRejectedAck exposes rejectedAck for unit testing.
var ExportedRejectedAck = rejectedAck

// ExportedCheckConfigUpdate exposes checkConfigUpdate for unit testing.
func (s *AgentServiceServer) ExportedCheckConfigUpdate(
	ctx context.Context,
	agentID pgtype.UUID,
	tenantID pgtype.UUID,
	tenantIDStr string,
	agentIDStr string,
	logger *slog.Logger,
) *pb.AgentConfig {
	return s.checkConfigUpdate(ctx, agentID, tenantID, tenantIDStr, agentIDStr, logger)
}
