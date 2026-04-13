package grpc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/protocol"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Enroll handles agent enrollment via a registration token.
func (s *AgentServiceServer) Enroll(ctx context.Context, req *pb.EnrollRequest) (*pb.EnrollResponse, error) {
	// 1. Validate request fields.
	if err := v1.ValidateEnrollRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "enroll: %s", err)
	}

	logger := s.logger.With("hostname", req.EndpointInfo.Hostname)

	// 2. Lookup registration token (bypasses RLS — tenant unknown yet).
	queries := sqlcgen.New(s.store.BypassPool())
	reg, err := queries.LookupRegistrationByToken(ctx, req.EnrollmentToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.WarnContext(ctx, "enroll: invalid registration token")
			return &pb.EnrollResponse{
				ErrorCode:    pb.EnrollmentErrorCode_ENROLLMENT_ERROR_CODE_INVALID_TOKEN,
				ErrorMessage: "invalid registration token",
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "enroll: lookup registration: %v", err)
	}

	// 3. Check token is still pending.
	if reg.Status != "pending" {
		logger.WarnContext(ctx, "enroll: registration token already used", "status", reg.Status)
		return &pb.EnrollResponse{
			ErrorCode:    pb.EnrollmentErrorCode_ENROLLMENT_ERROR_CODE_TOKEN_ALREADY_USED,
			ErrorMessage: "registration token already used",
		}, nil
	}

	// 3b. Check token has not expired (H-I1).
	if reg.ExpiresAt.Valid && time.Now().After(reg.ExpiresAt.Time) {
		logger.WarnContext(ctx, "enroll: registration token expired",
			"expires_at", reg.ExpiresAt.Time.Format(time.RFC3339),
		)
		return &pb.EnrollResponse{
			ErrorCode:    pb.EnrollmentErrorCode_ENROLLMENT_ERROR_CODE_INVALID_TOKEN,
			ErrorMessage: "registration token has expired",
		}, nil
	}

	// 4. Extract tenant ID and set in context.
	tenantIDStr := uuid.UUID(reg.TenantID.Bytes).String()
	ctx = tenant.WithTenantID(ctx, tenantIDStr)
	logger = logger.With("tenant_id", tenantIDStr)

	// 5. Negotiate protocol version.
	negotiated, err := protocol.NegotiateProtocolVersion(
		req.AgentInfo.ProtocolVersion,
		ServerProtocolVersion,
		ServerMinProtocolVersion,
	)
	if err != nil {
		logger.WarnContext(ctx, "enroll: protocol version incompatible",
			"agent_version", req.AgentInfo.ProtocolVersion,
		)
		return &pb.EnrollResponse{
			ErrorCode:    pb.EnrollmentErrorCode_ENROLLMENT_ERROR_CODE_PROTOCOL_VERSION_INCOMPATIBLE,
			ErrorMessage: fmt.Sprintf("protocol version incompatible: %s", err),
		}, nil
	}

	// 6. Begin tenant-scoped transaction.
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "enroll: begin tx: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := sqlcgen.New(tx)

	// 7. Check for existing endpoint (idempotent re-enrollment).
	osFamilyStr := normalizeOSFamily(req.EndpointInfo.OsFamily)
	existing, err := qtx.GetEndpointByHostnameAndOS(ctx, sqlcgen.GetEndpointByHostnameAndOSParams{
		TenantID: reg.TenantID,
		Hostname: req.EndpointInfo.Hostname,
		OsFamily: osFamilyStr,
	})
	if err == nil {
		// Endpoint already exists — update hardware and return its ID (idempotent).
		agentID := uuid.UUID(existing.ID.Bytes).String()
		logger.InfoContext(ctx, "enroll: returning existing endpoint, updating hardware", "agent_id", agentID)

		hwp := buildHardwareParams(existing.ID, reg.TenantID, req)
		if _, hwErr := qtx.UpdateEndpointHardware(ctx, hwp); hwErr != nil {
			logger.WarnContext(ctx, "enroll: update hardware on re-enroll failed (non-fatal)", "error", hwErr)
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			logger.WarnContext(ctx, "enroll: commit hardware update failed (non-fatal)", "error", commitErr)
		}

		return &pb.EnrollResponse{
			AgentId:                   agentID,
			NegotiatedProtocolVersion: negotiated,
			Config:                    defaultAgentConfig(),
		}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Errorf(codes.Internal, "enroll: lookup existing endpoint: %v", err)
	}

	// 8. Create new endpoint with hardware info from EndpointInfo.
	var ipAddr pgtype.Text
	if addrs := req.EndpointInfo.GetIpAddresses(); len(addrs) > 0 {
		ipAddr = pgtype.Text{String: addrs[0], Valid: true}
	}

	var archText pgtype.Text
	// Prefer arch from tags map (set by agent collector); fall back to "amd64".
	// TODO(PIQ-ARCH): add an explicit `arch` field to EndpointInfo proto so agents
	// can report architecture without relying on the tags side-channel.
	if tags := req.EndpointInfo.GetTags(); tags != nil {
		if v, ok := tags["arch"]; ok && v != "" {
			archText = pgtype.Text{String: v, Valid: true}
		}
	}
	if !archText.Valid {
		archText = pgtype.Text{String: "amd64", Valid: true}
	}
	// Determine kernel version: on macOS, HardwareModel is the hardware name (e.g. "MacBook Air"),
	// not the kernel — use tags["kernel_version"] instead. On Linux, HardwareModel carries /proc/version.
	var kernelText pgtype.Text
	if tags := req.EndpointInfo.GetTags(); tags != nil {
		if kv, ok := tags["kernel_version"]; ok && kv != "" {
			kernelText = pgtype.Text{String: kv, Valid: true}
		}
	}
	if !kernelText.Valid && req.EndpointInfo.HardwareModel != "" && req.EndpointInfo.OsFamily != pb.OsFamily_OS_FAMILY_MACOS {
		// On Linux, HardwareModel carries the kernel version from /proc/version
		kernelText = pgtype.Text{String: req.EndpointInfo.HardwareModel, Valid: true}
	}

	endpoint, err := qtx.CreateEndpoint(ctx, sqlcgen.CreateEndpointParams{
		TenantID:      reg.TenantID,
		Hostname:      req.EndpointInfo.Hostname,
		OsFamily:      osFamilyStr,
		OsVersion:     req.EndpointInfo.OsVersion,
		AgentVersion:  pgtype.Text{String: req.AgentInfo.AgentVersion, Valid: true},
		Status:        "online",
		IpAddress:     ipAddr,
		Arch:          archText,
		KernelVersion: kernelText,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "enroll: create endpoint: %v", err)
	}

	// 8b. Set hardware fields from EndpointInfo (cpu, memory, disk).
	hwParams := buildHardwareParams(endpoint.ID, reg.TenantID, req)
	if _, err := qtx.UpdateEndpointHardware(ctx, hwParams); err != nil {
		logger.WarnContext(ctx, "enroll: update hardware failed (non-fatal)", "error", err)
	}

	// 9. Claim the registration token.
	_, err = qtx.ClaimRegistration(ctx, sqlcgen.ClaimRegistrationParams{
		ID:         reg.ID,
		TenantID:   reg.TenantID,
		EndpointID: endpoint.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "enroll: claim registration: %v", err)
	}

	// 10. Commit transaction.
	if err := tx.Commit(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "enroll: commit tx: %v", err)
	}

	agentID := uuid.UUID(endpoint.ID.Bytes).String()
	logger.InfoContext(ctx, "enroll: endpoint enrolled", "agent_id", agentID)

	// 11. Emit domain event. Failure is logged but non-fatal to avoid failing
	// the enrollment after the endpoint was already committed.
	evt := domain.NewSystemEvent(
		events.EndpointEnrolled,
		tenantIDStr,
		"endpoint",
		agentID,
		"enrolled",
		map[string]string{
			"hostname": req.EndpointInfo.Hostname,
		},
	)
	if err := s.eventBus.Emit(ctx, evt); err != nil {
		logger.ErrorContext(ctx, "enroll: emit event failed", "error", err.Error())
	}

	// 12. Return response.
	// TODO(PIQ-331): populate MtlsCertificate field (H-I2) — requires RSA cert generation
	// from internal/shared/crypto and wiring into enrollment flow.
	return &pb.EnrollResponse{
		AgentId:                   agentID,
		NegotiatedProtocolVersion: negotiated,
		Config:                    defaultAgentConfig(),
	}, nil
}

// normalizeOSFamily converts a protobuf OsFamily enum to the lowercase string
// used in the database (e.g. "linux", "windows", "darwin").
func normalizeOSFamily(f pb.OsFamily) string {
	switch f {
	case pb.OsFamily_OS_FAMILY_LINUX:
		return "linux"
	case pb.OsFamily_OS_FAMILY_WINDOWS:
		return "windows"
	case pb.OsFamily_OS_FAMILY_MACOS:
		return "darwin"
	default:
		return "unknown"
	}
}

func defaultAgentConfig() *pb.AgentConfig {
	return &pb.AgentConfig{
		HeartbeatIntervalSeconds: 60,
		InventoryIntervalSeconds: 3600,
		MaxRetryAttempts:         3,
	}
}

// buildHardwareParams constructs UpdateEndpointHardwareParams from the EnrollRequest.
func buildHardwareParams(endpointID, tenantID pgtype.UUID, req *pb.EnrollRequest) sqlcgen.UpdateEndpointHardwareParams {
	params := sqlcgen.UpdateEndpointHardwareParams{
		ID:       endpointID,
		TenantID: tenantID,
	}
	// Always refresh os_version and agent_version on (re-)enrollment.
	if req.EndpointInfo.OsVersion != "" {
		params.OsVersion = req.EndpointInfo.OsVersion
	}
	if req.AgentInfo.AgentVersion != "" {
		params.AgentVersion = pgtype.Text{String: req.AgentInfo.AgentVersion, Valid: true}
	}
	if addrs := req.EndpointInfo.GetIpAddresses(); len(addrs) > 0 {
		params.IpAddress = pgtype.Text{String: addrs[0], Valid: true}
	}
	// Determine kernel version: on macOS, HardwareModel is the hardware name (e.g. "MacBook Air"),
	// not the kernel — use tags["kernel_version"] instead. On Linux, HardwareModel carries /proc/version.
	if tags := req.EndpointInfo.GetTags(); tags != nil {
		if kv, ok := tags["kernel_version"]; ok && kv != "" {
			params.KernelVersion = pgtype.Text{String: kv, Valid: true}
		}
	}
	if !params.KernelVersion.Valid && req.EndpointInfo.HardwareModel != "" && req.EndpointInfo.OsFamily != pb.OsFamily_OS_FAMILY_MACOS {
		params.KernelVersion = pgtype.Text{String: req.EndpointInfo.HardwareModel, Valid: true}
	}
	if req.EndpointInfo.CpuType != "" {
		params.CpuModel = pgtype.Text{String: req.EndpointInfo.CpuType, Valid: true}
	}
	if req.EndpointInfo.MemoryBytes > 0 {
		params.MemoryTotalMb = pgtype.Int8{Int64: int64(req.EndpointInfo.MemoryBytes / (1024 * 1024)), Valid: true}
	}
	// Prefer arch from tags map (set by agent collector); fall back to "amd64".
	// TODO(PIQ-ARCH): add an explicit `arch` field to EndpointInfo proto so agents
	// can report architecture without relying on the tags side-channel.
	if tags := req.EndpointInfo.GetTags(); tags != nil {
		if v, ok := tags["arch"]; ok && v != "" {
			params.Arch = pgtype.Text{String: v, Valid: true}
		}
	}
	if !params.Arch.Valid {
		params.Arch = pgtype.Text{String: "amd64", Valid: true}
	}
	// Read extended hardware info from tags (fields not in proto schema).
	if tags := req.EndpointInfo.GetTags(); tags != nil {
		if v, ok := tags["cpu_cores"]; ok {
			if cores, err := strconv.ParseInt(v, 10, 32); err == nil && cores > 0 {
				params.CpuCores = pgtype.Int4{Int32: int32(cores), Valid: true}
			}
		}
		if v, ok := tags["disk_total_gb"]; ok {
			if gb, err := strconv.ParseInt(v, 10, 64); err == nil && gb > 0 {
				params.DiskTotalGb = pgtype.Int8{Int64: gb, Valid: true}
			}
		}
	}
	return params
}
