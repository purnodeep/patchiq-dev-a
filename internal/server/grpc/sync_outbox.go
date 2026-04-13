package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/server/cve"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// SyncOutbox streams queued messages from agent to server, acknowledging each.
func (s *AgentServiceServer) SyncOutbox(stream grpc.BidiStreamingServer[pb.OutboxMessage, pb.OutboxAck]) error {
	ctx := stream.Context()

	// 1. Extract agent_id from gRPC metadata.
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.InvalidArgument, "sync outbox: missing metadata")
	}
	agentIDVals := md.Get("x-agent-id")
	if len(agentIDVals) == 0 || agentIDVals[0] == "" {
		return status.Error(codes.InvalidArgument, "sync outbox: missing x-agent-id metadata")
	}
	agentIDStr := agentIDVals[0]

	agentUUID, err := uuid.Parse(agentIDStr)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "sync outbox: invalid agent_id: %v", err)
	}

	logger := s.logger.With("agent_id", agentIDStr)

	// 2. Lookup endpoint to resolve tenant (bypasses RLS — tenant unknown yet).
	queries := sqlcgen.New(s.store.BypassPool())
	endpoint, err := queries.LookupEndpointByID(ctx, pgtype.UUID{Bytes: agentUUID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.WarnContext(ctx, "sync outbox: agent not enrolled")
			return status.Error(codes.NotFound, "sync outbox: agent not enrolled")
		}
		return status.Errorf(codes.Internal, "sync outbox: lookup endpoint: %v", err)
	}

	// 3. Set tenant context.
	tenantIDStr := uuid.UUID(endpoint.TenantID.Bytes).String()
	ctx = tenant.WithTenantID(ctx, tenantIDStr)
	logger = logger.With("tenant_id", tenantIDStr)

	logger.InfoContext(ctx, "sync outbox: stream started")

	// 4. Receive loop: process each OutboxMessage and send an OutboxAck.
	for {
		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.InfoContext(ctx, "sync outbox: stream closed by agent")
				return nil
			}
			return status.Errorf(codes.Internal, "sync outbox: recv: %v", err)
		}

		msgLogger := logger.With(
			"message_id", msg.GetMessageId(),
			"message_type", msg.GetType().String(),
		)

		switch msg.GetType() {
		case pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_INVENTORY:
			ack, processErr := s.processInventory(ctx, msg, endpoint.ID, tenantIDStr, agentIDStr)
			if processErr != nil {
				msgLogger.ErrorContext(ctx, "sync outbox: process inventory failed", "error", processErr)
			}
			if err := stream.Send(ack); err != nil {
				return status.Errorf(codes.Internal, "sync outbox: send ack: %v", err)
			}

		case pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_COMMAND_RESULT:
			if err := s.processCommandResult(ctx, tenantIDStr, agentIDStr, msg); err != nil {
				if errors.Is(err, errPayloadInvalid) {
					msgLogger.ErrorContext(ctx, "sync outbox: command result payload invalid", "error", err)
					ack := rejectedAck(msg.GetMessageId(),
						pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_PAYLOAD_INVALID,
						"payload invalid: unmarshal failed",
					)
					if err := stream.Send(ack); err != nil {
						return status.Errorf(codes.Internal, "sync outbox: send rejection: %v", err)
					}
					continue
				}
				// Event bus failure — reject so the agent retries.
				// CommandResultReceived events drive the ResultHandler which
				// updates deployment counters and state transitions. Accepting
				// here would silently drop the result and stall deployments.
				msgLogger.ErrorContext(ctx, "sync outbox: command result event emission failed", "error", err)
				ack := rejectedAck(msg.GetMessageId(),
					pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_SERVER_OVERLOADED,
					"event emission failed",
				)
				if err := stream.Send(ack); err != nil {
					return status.Errorf(codes.Internal, "sync outbox: send rejection: %v", err)
				}
				continue
			}
			if err := stream.Send(acceptedAck(msg.GetMessageId())); err != nil {
				return status.Errorf(codes.Internal, "sync outbox: send ack: %v", err)
			}

		default:
			msgLogger.WarnContext(ctx, "sync outbox: unknown message type, rejecting permanently")
			ack := rejectedAck(msg.GetMessageId(),
				pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_UNKNOWN_TYPE,
				fmt.Sprintf("unsupported message type: %s", msg.GetType().String()),
			)
			if err := stream.Send(ack); err != nil {
				return status.Errorf(codes.Internal, "sync outbox: send rejection: %v", err)
			}
		}
	}
}

// processInventory unmarshals, validates, persists, and emits events for an
// inventory report received via the outbox stream.
func (s *AgentServiceServer) processInventory(
	ctx context.Context,
	msg *pb.OutboxMessage,
	endpointID pgtype.UUID,
	tenantIDStr, agentIDStr string,
) (*pb.OutboxAck, error) {
	msgID := msg.GetMessageId()

	// 1. Unmarshal the payload.
	var report pb.InventoryReport
	if err := proto.Unmarshal(msg.GetPayload(), &report); err != nil {
		return rejectedAck(msgID, pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_PAYLOAD_INVALID,
			fmt.Sprintf("unmarshal inventory report: %v", err),
		), fmt.Errorf("process inventory: unmarshal: %w", err)
	}

	// 2. Validate: reject only if zero packages AND zero collection errors.
	// A report with collection_errors but no packages is a valid partial report —
	// the agent tried but some collectors failed.
	if len(report.GetInstalledPackages()) == 0 && len(report.GetCollectionErrors()) == 0 {
		return rejectedAck(msgID, pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_PAYLOAD_INVALID,
			"inventory report contains no packages and no collection errors",
		), fmt.Errorf("process inventory: empty report")
	}

	if len(report.GetCollectionErrors()) > 0 {
		s.logger.WarnContext(ctx, "process inventory: partial report received",
			"agent_id", agentIDStr,
			"package_count", len(report.GetInstalledPackages()),
			"collection_errors", len(report.GetCollectionErrors()),
		)
	}

	scannedAt := report.GetCollectedAt().AsTime()

	// 3. Persist inventory + packages in a single transaction.
	tenantUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return rejectedAck(msgID, pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_SERVER_OVERLOADED,
			"invalid tenant id",
		), fmt.Errorf("process inventory: parse tenant id: %w", err)
	}
	tenantPgUUID := pgtype.UUID{Bytes: tenantUUID, Valid: true}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return rejectedAck(msgID, pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_SERVER_OVERLOADED,
			"begin transaction failed",
		), fmt.Errorf("process inventory: begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.WarnContext(ctx, "process inventory: rollback",
				"agent_id", agentIDStr, "message_id", msgID, "error", err)
		}
	}()

	// Marshal collection errors to JSONB for storage. If marshal fails we
	// reject the message so the agent retries — silently defaulting to "[]"
	// would destroy the very data the partial-report path exists to preserve.
	collectionErrorsJSON := []byte("[]")
	if len(report.GetCollectionErrors()) > 0 {
		type collError struct {
			Collector string `json:"collector"`
			Error     string `json:"error"`
		}
		errs := make([]collError, len(report.GetCollectionErrors()))
		for i, ce := range report.GetCollectionErrors() {
			errs[i] = collError{Collector: ce.GetCollector(), Error: ce.GetErrorMessage()}
		}
		marshaled, marshalErr := json.Marshal(errs)
		if marshalErr != nil {
			return rejectedAck(msgID, pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_PAYLOAD_INVALID,
				"marshal collection errors failed",
			), fmt.Errorf("process inventory: marshal collection errors: agent_id=%s msg_id=%s: %w", agentIDStr, msgID, marshalErr)
		}
		collectionErrorsJSON = marshaled
	}

	qtx := sqlcgen.New(tx)
	inv, err := qtx.CreateEndpointInventory(ctx, sqlcgen.CreateEndpointInventoryParams{
		TenantID:         tenantPgUUID,
		EndpointID:       endpointID,
		ScannedAt:        pgtype.Timestamptz{Time: scannedAt, Valid: true},
		PackageCount:     int32(len(report.GetInstalledPackages())),
		CollectionErrors: collectionErrorsJSON,
	})
	if err != nil {
		return rejectedAck(msgID, pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_SERVER_OVERLOADED,
			"create inventory record failed",
		), fmt.Errorf("process inventory: create inventory: %w", err)
	}

	_, err = s.store.BulkInsertEndpointPackages(ctx, tx, tenantPgUUID, endpointID, inv.ID, report.GetInstalledPackages())
	if err != nil {
		return rejectedAck(msgID, pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_SERVER_OVERLOADED,
			"insert packages failed",
		), fmt.Errorf("process inventory: bulk insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return rejectedAck(msgID, pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_SERVER_OVERLOADED,
			"commit failed",
		), fmt.Errorf("process inventory: commit: %w", err)
	}

	// 4. Store deep hardware details and software summary (best-effort, post-commit).
	s.storeHardwareAndSoftware(ctx, &report, endpointID, tenantPgUUID, agentIDStr)

	// 5. Upsert network interfaces from hardware data (best-effort, post-commit).
	s.upsertNetworkInterfaces(ctx, &report, endpointID, tenantPgUUID, agentIDStr)

	// 6. Enqueue CVE match job for this endpoint (best-effort, post-commit).
	// Triggers immediate vulnerability correlation after fresh inventory data arrives,
	// rather than waiting for the next periodic NVD sync cycle (24h).
	if s.cveJobInserter != nil {
		if _, err := s.cveJobInserter.Insert(ctx, cve.EndpointMatchJobArgs{
			TenantID:   tenantIDStr,
			EndpointID: uuid.UUID(endpointID.Bytes).String(),
		}, nil); err != nil {
			s.logger.WarnContext(ctx, "process inventory: enqueue CVE match job failed",
				"agent_id", agentIDStr, "error", err)
		}
	}

	// 6b. Process agent-detected CVEs (best-effort, post-commit).
	// Agents may detect CVEs locally; these supplement server-side NVD correlation.
	if detectedCVEs := report.GetDetectedCves(); len(detectedCVEs) > 0 {
		s.processDetectedCVEs(ctx, detectedCVEs, endpointID, tenantPgUUID, tenantIDStr, agentIDStr)
	}

	// 7. Emit domain events.
	// Post-commit event emission is best-effort: the DB state is authoritative.
	// Inventory inserts are append-only; a duplicate row from a retry is harmless
	// since only the latest inventory is used for queries. Returning an error would
	// waste bandwidth.
	// TODO(PIQ-145): add event emission failure counter to surface silent drops.
	eventPayload := map[string]any{
		"agent_id":           agentIDStr,
		"endpoint_id":        uuid.UUID(endpointID.Bytes).String(),
		"inventory_id":       uuid.UUID(inv.ID.Bytes).String(),
		"package_count":      len(report.GetInstalledPackages()),
		"detected_cve_count": len(report.GetDetectedCves()),
		"scanned_at":         scannedAt.Format(time.RFC3339),
	}

	evt1 := domain.NewSystemEvent(events.InventoryReceived, tenantIDStr, "endpoint", agentIDStr, "received", eventPayload)
	evt2 := domain.NewSystemEvent(events.InventoryScanCompleted, tenantIDStr, "endpoint", agentIDStr, "scan_completed", eventPayload)
	deployment.EmitBestEffort(ctx, s.eventBus, []domain.DomainEvent{evt1, evt2})

	return acceptedAck(msgID), nil
}

// storeHardwareAndSoftware extracts hardware_json from the inventory report tags
// and computes a software_summary, then persists both as JSONB on the endpoint.
// This is best-effort: failures are logged but do not reject the inventory message.
func (s *AgentServiceServer) storeHardwareAndSoftware(
	ctx context.Context,
	report *pb.InventoryReport,
	endpointID, tenantID pgtype.UUID,
	agentIDStr string,
) {
	tags := report.GetEndpointInfo().GetTags()

	// Extract hardware_json from tags.
	var hardwareJSON []byte
	if raw, ok := tags["hardware_json"]; ok && raw != "" {
		// Validate it is valid JSON before storing.
		if json.Valid([]byte(raw)) {
			hardwareJSON = []byte(raw)
		} else {
			s.logger.WarnContext(ctx, "process inventory: hardware_json tag is not valid JSON",
				"agent_id", agentIDStr)
		}
	}
	if hardwareJSON == nil {
		hardwareJSON = []byte("{}")
	}

	// Compute software_summary from installed packages.
	pkgsBySource := make(map[string]int)
	for _, pkg := range report.GetInstalledPackages() {
		src := pkg.GetSource()
		if src == "" {
			src = "unknown"
		}
		pkgsBySource[src]++
	}
	summary := map[string]any{
		"total_packages":     len(report.GetInstalledPackages()),
		"packages_by_source": pkgsBySource,
	}
	softwareJSON, err := json.Marshal(summary)
	if err != nil {
		s.logger.ErrorContext(ctx, "process inventory: marshal software summary",
			"agent_id", agentIDStr, "error", err)
		softwareJSON = []byte("{}")
	}

	// Persist in a separate short transaction (best-effort).
	hwTx, err := s.store.BeginTx(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "process inventory: begin hardware details tx",
			"agent_id", agentIDStr, "error", err)
		return
	}
	defer func() {
		if rbErr := hwTx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			s.logger.WarnContext(ctx, "process inventory: rollback hardware details tx",
				"agent_id", agentIDStr, "error", rbErr)
		}
	}()

	// Sanitize JSON for PostgreSQL JSONB: strip \u0000 null bytes and invalid Unicode
	// escape sequences that Windows WMI/PowerShell output may contain (SQLSTATE 22P05).
	hardwareJSON = sanitizeJSONBBytes(hardwareJSON)
	softwareJSON = sanitizeJSONBBytes(softwareJSON)

	hwQ := sqlcgen.New(hwTx)
	if err := hwQ.UpdateEndpointHardwareDetails(ctx, sqlcgen.UpdateEndpointHardwareDetailsParams{
		HardwareDetails: hardwareJSON,
		SoftwareSummary: softwareJSON,
		ID:              endpointID,
		TenantID:        tenantID,
	}); err != nil {
		s.logger.ErrorContext(ctx, "process inventory: update hardware details",
			"agent_id", agentIDStr, "error", err)
		// Don't return — still attempt to populate scalar summary fields below.
		s.logger.WarnContext(ctx, "process inventory: JSONB storage failed, scalar summary fields will still be populated",
			"agent_id", agentIDStr)
	} else if err := hwTx.Commit(ctx); err != nil {
		s.logger.ErrorContext(ctx, "process inventory: commit hardware details",
			"agent_id", agentIDStr, "error", err)
	} else {
		s.logger.InfoContext(ctx, "process inventory: stored hardware details and software summary",
			"agent_id", agentIDStr,
			"hardware_json_size", len(hardwareJSON),
			"total_packages", len(report.GetInstalledPackages()),
		)
	}

	// Extract summary fields from hardware_json and update top-level endpoint columns
	// (cpu_model, cpu_cores, memory_total_mb, disk_total_gb, ip_address, kernel_version).
	// These columns power the dashboard and list views. Runs even if JSONB storage failed.
	s.updateSummaryFieldsFromHardware(ctx, hardwareJSON, report.GetEndpointInfo(), endpointID, tenantID, agentIDStr)

	// Emit hardware updated event (best-effort).
	evt := domain.NewSystemEvent(events.EndpointUpdated, uuid.UUID(tenantID.Bytes).String(),
		"endpoint", uuid.UUID(endpointID.Bytes).String(), "hardware_updated",
		map[string]any{
			"agent_id":           agentIDStr,
			"hardware_json_size": len(hardwareJSON),
			"total_packages":     len(report.GetInstalledPackages()),
		},
	)
	deployment.EmitBestEffort(ctx, s.eventBus, []domain.DomainEvent{evt})
}

// updateSummaryFieldsFromHardware parses the hardware_json blob and populates the
// top-level endpoint summary columns (cpu_model, cpu_cores, memory_total_mb,
// disk_total_gb, ip_address, kernel_version) that power dashboard and list views.
// This is best-effort: failures are logged but do not reject the inventory message.
func (s *AgentServiceServer) updateSummaryFieldsFromHardware(
	ctx context.Context,
	hardwareJSON []byte,
	endpointInfo *pb.EndpointInfo,
	endpointID, tenantID pgtype.UUID,
	agentIDStr string,
) {
	// Parse the hardware JSON structure (mirrors agent's HardwareInfo).
	var hw struct {
		CPU struct {
			ModelName    string `json:"model_name"`
			TotalLogical int    `json:"total_logical_cpus"`
		} `json:"cpu"`
		Memory struct {
			TotalBytes     uint64 `json:"total_bytes"`
			AvailableBytes uint64 `json:"available_bytes"`
		} `json:"memory"`
		Storage []struct {
			SizeBytes uint64 `json:"size_bytes"`
		} `json:"storage"`
		GPU []struct {
			Model string `json:"model"`
		} `json:"gpu"`
		Network []struct {
			Name          string `json:"name"`
			State         string `json:"state"`
			IPv4Addresses []struct {
				Address string `json:"address"`
			} `json:"ipv4_addresses"`
		} `json:"network"`
	}
	if err := json.Unmarshal(hardwareJSON, &hw); err != nil {
		s.logger.WarnContext(ctx, "process inventory: unmarshal hardware_json for summary fields",
			"agent_id", agentIDStr, "error", err)
		return
	}

	params := sqlcgen.UpdateEndpointHardwareParams{
		ID:       endpointID,
		TenantID: tenantID,
	}

	// CPU model.
	if hw.CPU.ModelName != "" {
		params.CpuModel = pgtype.Text{String: hw.CPU.ModelName, Valid: true}
	}

	// CPU cores (total logical CPUs).
	if hw.CPU.TotalLogical > 0 {
		params.CpuCores = pgtype.Int4{Int32: int32(hw.CPU.TotalLogical), Valid: true}
	}

	// Memory total in MB.
	if hw.Memory.TotalBytes > 0 {
		params.MemoryTotalMb = pgtype.Int8{Int64: int64(hw.Memory.TotalBytes / (1024 * 1024)), Valid: true}
	}

	// Memory used in MB (total - available).
	if hw.Memory.TotalBytes > 0 && hw.Memory.AvailableBytes > 0 && hw.Memory.TotalBytes >= hw.Memory.AvailableBytes {
		usedMB := int64((hw.Memory.TotalBytes - hw.Memory.AvailableBytes) / (1024 * 1024))
		params.MemoryUsedMb = pgtype.Int8{Int64: usedMB, Valid: true}
	}

	// GPU model from first GPU entry.
	if len(hw.GPU) > 0 && hw.GPU[0].Model != "" {
		params.GpuModel = pgtype.Text{String: hw.GPU[0].Model, Valid: true}
	}

	// Disk total in GB (sum of all storage devices).
	var totalDiskBytes uint64
	for _, disk := range hw.Storage {
		totalDiskBytes += disk.SizeBytes
	}
	if totalDiskBytes > 0 {
		params.DiskTotalGb = pgtype.Int8{Int64: int64(totalDiskBytes / (1024 * 1024 * 1024)), Valid: true}
	}

	// IP address: first non-loopback IPv4 from network interfaces.
	for _, iface := range hw.Network {
		if len(iface.IPv4Addresses) > 0 {
			addr := iface.IPv4Addresses[0].Address
			if addr != "" && addr != "127.0.0.1" {
				params.IpAddress = pgtype.Text{String: addr, Valid: true}
				break
			}
		}
	}

	// Kernel version from EndpointInfo tags (agent sets this during inventory collection).
	if endpointInfo != nil {
		if tags := endpointInfo.GetTags(); tags != nil {
			if kv, ok := tags["kernel_version"]; ok && kv != "" {
				params.KernelVersion = pgtype.Text{String: kv, Valid: true}
			}
		}
		// Arch from tags.
		if tags := endpointInfo.GetTags(); tags != nil {
			if v, ok := tags["arch"]; ok && v != "" {
				params.Arch = pgtype.Text{String: v, Valid: true}
			}
		}
		// Also pick up cpu_cores and disk_total_gb from tags as fallback
		// (agent may set these directly).
		if tags := endpointInfo.GetTags(); tags != nil {
			if !params.CpuCores.Valid {
				if v, ok := tags["cpu_cores"]; ok {
					if cores, err := strconv.ParseInt(v, 10, 32); err == nil && cores > 0 {
						params.CpuCores = pgtype.Int4{Int32: int32(cores), Valid: true}
					}
				}
			}
			if !params.DiskTotalGb.Valid {
				if v, ok := tags["disk_total_gb"]; ok {
					if gb, err := strconv.ParseInt(v, 10, 64); err == nil && gb > 0 {
						params.DiskTotalGb = pgtype.Int8{Int64: gb, Valid: true}
					}
				}
			}
		}
	}

	// Only update if we have at least one valid field.
	hasData := params.CpuModel.Valid || params.CpuCores.Valid ||
		params.MemoryTotalMb.Valid || params.MemoryUsedMb.Valid ||
		params.DiskTotalGb.Valid || params.GpuModel.Valid ||
		params.IpAddress.Valid || params.KernelVersion.Valid
	if !hasData {
		return
	}

	sumTx, err := s.store.BeginTx(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "process inventory: begin summary fields tx",
			"agent_id", agentIDStr, "error", err)
		return
	}
	defer func() {
		if rbErr := sumTx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			s.logger.WarnContext(ctx, "process inventory: rollback summary fields tx",
				"agent_id", agentIDStr, "error", rbErr)
		}
	}()

	sumQ := sqlcgen.New(sumTx)
	if _, err := sumQ.UpdateEndpointHardware(ctx, params); err != nil {
		s.logger.ErrorContext(ctx, "process inventory: update summary fields from hardware",
			"agent_id", agentIDStr, "error", err)
		return
	}

	if err := sumTx.Commit(ctx); err != nil {
		s.logger.ErrorContext(ctx, "process inventory: commit summary fields",
			"agent_id", agentIDStr, "error", err)
		return
	}

	s.logger.InfoContext(ctx, "process inventory: updated endpoint summary fields from hardware",
		"agent_id", agentIDStr,
		"cpu_model", params.CpuModel.String,
		"cpu_cores", params.CpuCores.Int32,
		"memory_total_mb", params.MemoryTotalMb.Int64,
		"memory_used_mb", params.MemoryUsedMb.Int64,
		"disk_total_gb", params.DiskTotalGb.Int64,
		"gpu_model", params.GpuModel.String,
		"ip_address", params.IpAddress.String,
		"kernel_version", params.KernelVersion.String,
	)
}

// upsertNetworkInterfaces extracts network interface data from the hardware_json
// tag in the inventory report and upserts them into the endpoint_network_interfaces table.
func (s *AgentServiceServer) upsertNetworkInterfaces(
	ctx context.Context,
	report *pb.InventoryReport,
	endpointID, tenantID pgtype.UUID,
	agentIDStr string,
) {
	tags := report.GetEndpointInfo().GetTags()
	raw, ok := tags["hardware_json"]
	if !ok || raw == "" {
		return
	}

	// Parse the hardware JSON to extract network interfaces.
	// The agent sends network as a flat array with ipv4_addresses (array of
	// {address, prefix_len}) and state (not ip_address / status).
	var hw struct {
		Network []struct {
			Name          string `json:"name"`
			MACAddress    string `json:"mac_address"`
			State         string `json:"state"`
			IPv4Addresses []struct {
				Address   string `json:"address"`
				PrefixLen int    `json:"prefix_len"`
			} `json:"ipv4_addresses"`
		} `json:"network"`
	}
	if err := json.Unmarshal([]byte(raw), &hw); err != nil {
		s.logger.WarnContext(ctx, "process inventory: unmarshal hardware_json for network interfaces",
			"agent_id", agentIDStr, "error", err)
		return
	}

	if len(hw.Network) == 0 {
		return
	}

	nicTx, err := s.store.BeginTx(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "process inventory: begin network interfaces tx",
			"agent_id", agentIDStr, "error", err)
		return
	}
	defer func() {
		if rbErr := nicTx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			s.logger.WarnContext(ctx, "process inventory: rollback network interfaces tx",
				"agent_id", agentIDStr, "error", rbErr)
		}
	}()

	nicQ := sqlcgen.New(nicTx)
	activeNames := make([]string, 0, len(hw.Network))

	for _, iface := range hw.Network {
		if iface.Name == "" {
			continue
		}
		nicStatus := iface.State
		if nicStatus == "" {
			nicStatus = "up"
		}
		// Normalize status to match the CHECK constraint (up/down).
		if nicStatus != "up" && nicStatus != "down" {
			nicStatus = "up"
		}

		// Extract first IPv4 address if available.
		var ipAddress string
		if len(iface.IPv4Addresses) > 0 {
			ipAddress = iface.IPv4Addresses[0].Address
		}

		if err := nicQ.UpsertEndpointNetworkInterface(ctx, sqlcgen.UpsertEndpointNetworkInterfaceParams{
			TenantID:   tenantID,
			EndpointID: endpointID,
			Name:       iface.Name,
			IpAddress:  pgtype.Text{String: ipAddress, Valid: ipAddress != ""},
			MacAddress: pgtype.Text{String: iface.MACAddress, Valid: iface.MACAddress != ""},
			Status:     nicStatus,
		}); err != nil {
			s.logger.ErrorContext(ctx, "process inventory: upsert network interface",
				"agent_id", agentIDStr, "interface", iface.Name, "error", err)
			return
		}
		activeNames = append(activeNames, iface.Name)
	}

	// Remove stale interfaces that are no longer reported.
	if err := nicQ.DeleteStaleNetworkInterfaces(ctx, sqlcgen.DeleteStaleNetworkInterfacesParams{
		TenantID:    tenantID,
		EndpointID:  endpointID,
		ActiveNames: activeNames,
	}); err != nil {
		s.logger.ErrorContext(ctx, "process inventory: delete stale network interfaces",
			"agent_id", agentIDStr, "error", err)
		return
	}

	if err := nicTx.Commit(ctx); err != nil {
		s.logger.ErrorContext(ctx, "process inventory: commit network interfaces",
			"agent_id", agentIDStr, "error", err)
		return
	}

	s.logger.InfoContext(ctx, "process inventory: upserted network interfaces",
		"agent_id", agentIDStr, "count", len(activeNames))
}

// errPayloadInvalid is a sentinel error indicating a permanent unmarshal failure.
var errPayloadInvalid = errors.New("payload invalid")

// processCommandResult unmarshals the command result payload and publishes a CommandResultReceived event.
func (s *AgentServiceServer) processCommandResult(ctx context.Context, tenantID, agentID string, msg *pb.OutboxMessage) error {
	var cmdResp pb.CommandResponse
	if err := proto.Unmarshal(msg.GetPayload(), &cmdResp); err != nil {
		return fmt.Errorf("%w: unmarshal command response: %w", errPayloadInvalid, err)
	}

	// Extract human-readable stdout, stderr, and exit code from the structured
	// output bytes. CommandResponse.output is a serialized proto (InstallPatchOutput
	// or RunScriptOutput), not plain text — storing it directly produces garbled bytes.
	stdout, stderr, exitCode := extractCommandOutput(cmdResp.GetOutput())

	evt := domain.NewSystemEvent(
		events.CommandResultReceived,
		tenantID,
		"command",
		cmdResp.GetCommandId(),
		"received",
		events.CommandResultPayload{
			CommandID:    cmdResp.GetCommandId(),
			AgentID:      agentID,
			Succeeded:    cmdResp.GetStatus() == pb.CommandStatus_COMMAND_STATUS_SUCCEEDED,
			Output:       stdout,
			Stderr:       stderr,
			ErrorMessage: cmdResp.GetErrorMessage(),
			ExitCode:     exitCode,
		},
	)
	return s.eventBus.Emit(ctx, evt)
}

// extractCommandOutput deserializes the raw proto bytes from CommandResponse.output
// into human-readable stdout, stderr, and exit code. It tries InstallPatchOutput
// first (most common), then RunScriptOutput. If neither succeeds, returns empty
// strings (the raw bytes are not useful as text).
func extractCommandOutput(raw []byte) (stdout, stderr string, exitCode *int32) {
	if len(raw) == 0 {
		return "", "", nil
	}

	// Try InstallPatchOutput first (INSTALL_PATCH and ROLLBACK_PATCH commands).
	var installOutput pb.InstallPatchOutput
	if err := proto.Unmarshal(raw, &installOutput); err == nil && len(installOutput.Results) > 0 {
		ec := installOutput.Results[0].ExitCode
		exitCode = &ec

		var stdoutParts, stderrParts []string
		for _, r := range installOutput.Results {
			if r.Stdout != "" {
				stdoutParts = append(stdoutParts, fmt.Sprintf("[%s] %s", r.PackageName, r.Stdout))
			}
			if r.Stderr != "" {
				stderrParts = append(stderrParts, fmt.Sprintf("[%s] %s", r.PackageName, r.Stderr))
			}
		}
		if installOutput.PreScriptOutput != "" {
			stdoutParts = append([]string{"[pre-script] " + installOutput.PreScriptOutput}, stdoutParts...)
		}
		if installOutput.PostScriptOutput != "" {
			stdoutParts = append(stdoutParts, "[post-script] "+installOutput.PostScriptOutput)
		}
		return strings.Join(stdoutParts, "\n"), strings.Join(stderrParts, "\n"), exitCode
	}

	// Try RunScriptOutput (RUN_SCRIPT commands).
	var scriptOutput pb.RunScriptOutput
	if err := proto.Unmarshal(raw, &scriptOutput); err == nil &&
		(scriptOutput.Stdout != "" || scriptOutput.Stderr != "" || scriptOutput.ExitCode != 0) {
		ec := scriptOutput.ExitCode
		exitCode = &ec
		return scriptOutput.Stdout, scriptOutput.Stderr, exitCode
	}

	// Unknown output type — return empty. The raw proto bytes are not human-readable.
	return "", "", nil
}

// acceptedAck returns an OutboxAck indicating the message was accepted.
// RejectionCode UNSPECIFIED (zero value) means accepted.
func acceptedAck(messageID string) *pb.OutboxAck {
	return &pb.OutboxAck{MessageId: messageID}
}

// rejectedAck returns an OutboxAck with a rejection code and detail.
func rejectedAck(messageID string, code pb.OutboxRejectionCode, detail string) *pb.OutboxAck {
	return &pb.OutboxAck{
		MessageId:       messageID,
		RejectionCode:   code,
		RejectionDetail: detail,
	}
}

// sanitizeJSONBBytes removes characters that PostgreSQL JSONB rejects (SQLSTATE 22P05).
// Windows agents produce \u0000 null bytes and raw 0x00 bytes in WMI/PowerShell output.
func sanitizeJSONBBytes(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	data = bytes.ReplaceAll(data, []byte("\\u0000"), []byte(""))
	data = bytes.ReplaceAll(data, []byte{0x00}, []byte{})
	return data
}

// processDetectedCVEs upserts agent-detected CVEs and creates endpoint-CVE associations.
// This is best-effort: failures are logged but do not reject the inventory message.
func (s *AgentServiceServer) processDetectedCVEs(
	ctx context.Context,
	detectedCVEs []*pb.CVEInfo,
	endpointID pgtype.UUID,
	tenantID pgtype.UUID,
	tenantIDStr, agentIDStr string,
) {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "process detected CVEs: begin tx failed",
			"agent_id", agentIDStr, "error", err)
		return
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.WarnContext(ctx, "process detected CVEs: rollback", "error", err)
		}
	}()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantIDStr); err != nil {
		s.logger.ErrorContext(ctx, "process detected CVEs: set tenant context failed",
			"agent_id", agentIDStr, "error", err)
		return
	}

	qtx := sqlcgen.New(tx)
	upserted := 0

	for _, cveInfo := range detectedCVEs {
		if cveInfo.GetCveId() == "" {
			continue
		}

		severity := protoSeverityToString(cveInfo.GetSeverity())
		cvssScore := float64ToNumeric(cveInfo.GetCvssScore())

		result, err := qtx.UpsertAgentCVE(ctx, sqlcgen.UpsertAgentCVEParams{
			TenantID:    tenantID,
			CveID:       cveInfo.GetCveId(),
			Severity:    severity,
			Description: pgtype.Text{String: cveInfo.GetDescription(), Valid: cveInfo.GetDescription() != ""},
			CvssV3Score: cvssScore,
			Source:      "agent",
		})
		if err != nil {
			s.logger.WarnContext(ctx, "process detected CVEs: upsert CVE failed",
				"agent_id", agentIDStr, "cve_id", cveInfo.GetCveId(), "error", err)
			continue
		}

		_, err = qtx.UpsertEndpointCVE(ctx, sqlcgen.UpsertEndpointCVEParams{
			TenantID:   tenantID,
			EndpointID: endpointID,
			CveID:      result.ID,
			Status:     "detected",
			DetectedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			RiskScore:  cvssScore,
		})
		if err != nil {
			s.logger.WarnContext(ctx, "process detected CVEs: upsert endpoint CVE failed",
				"agent_id", agentIDStr, "cve_id", cveInfo.GetCveId(), "error", err)
			continue
		}
		upserted++
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.ErrorContext(ctx, "process detected CVEs: commit failed",
			"agent_id", agentIDStr, "error", err)
		return
	}

	s.logger.InfoContext(ctx, "process detected CVEs: completed",
		"agent_id", agentIDStr, "detected", len(detectedCVEs), "upserted", upserted)
}

// float64ToNumeric converts a float64 to a pgtype.Numeric for database storage.
func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

// protoSeverityToString converts a protobuf Severity enum to the DB severity string.
func protoSeverityToString(s pb.Severity) string {
	switch s {
	case pb.Severity_SEVERITY_CRITICAL:
		return "critical"
	case pb.Severity_SEVERITY_HIGH:
		return "high"
	case pb.Severity_SEVERITY_MEDIUM:
		return "medium"
	case pb.Severity_SEVERITY_LOW:
		return "low"
	case pb.Severity_SEVERITY_NONE:
		return "none"
	default:
		return "none"
	}
}
