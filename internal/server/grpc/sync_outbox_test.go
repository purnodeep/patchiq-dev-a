package grpc_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/server/events"
	servergrpc "github.com/skenzeriq/patchiq/internal/server/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSyncOutbox_MissingMetadata(t *testing.T) {
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())
	client, cleanup := setupBufconn(t, svc)
	defer cleanup()

	// No metadata at all.
	stream, err := client.SyncOutbox(context.Background())
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	// Send any message to trigger server processing.
	// Ignore send error — stream may already be broken by server-side validation.
	_ = stream.Send(&pb.OutboxMessage{
		MessageId: "msg-1",
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_INVENTORY,
	})

	_, err = stream.Recv()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v: %s", st.Code(), st.Message())
	}
}

func TestSyncOutbox_EmptyAgentID(t *testing.T) {
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())
	client, cleanup := setupBufconn(t, svc)
	defer cleanup()

	md := metadata.Pairs("x-agent-id", "")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	stream, err := client.SyncOutbox(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	_ = stream.Send(&pb.OutboxMessage{
		MessageId: "msg-1",
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_INVENTORY,
	})

	_, err = stream.Recv()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v: %s", st.Code(), st.Message())
	}
}

func TestSyncOutbox_InvalidAgentID(t *testing.T) {
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())
	client, cleanup := setupBufconn(t, svc)
	defer cleanup()

	md := metadata.Pairs("x-agent-id", "not-a-uuid")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	stream, err := client.SyncOutbox(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	_ = stream.Send(&pb.OutboxMessage{
		MessageId: "msg-1",
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_INVENTORY,
	})

	_, err = stream.Recv()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v: %s", st.Code(), st.Message())
	}
}

func TestSyncOutbox_Inventory_InvalidPayload(t *testing.T) {
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())

	endpointUUID := uuid.New()
	endpointID := pgtype.UUID{Bytes: endpointUUID, Valid: true}
	tenantIDStr := uuid.New().String()
	agentIDStr := endpointUUID.String()

	msg := &pb.OutboxMessage{
		MessageId: "msg-inv-1",
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_INVENTORY,
		Payload:   []byte("this is not a valid protobuf"),
	}

	ack, err := svc.ExportedProcessInventory(context.Background(), msg, endpointID, tenantIDStr, agentIDStr)
	if err == nil {
		t.Fatal("expected error for invalid payload, got nil")
	}
	if ack == nil {
		t.Fatal("expected non-nil ack even on error")
	}
	if ack.GetMessageId() != "msg-inv-1" {
		t.Errorf("ack message_id = %q, want %q", ack.GetMessageId(), "msg-inv-1")
	}
	if ack.GetRejectionCode() != pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_PAYLOAD_INVALID {
		t.Errorf("rejection_code = %v, want PAYLOAD_INVALID", ack.GetRejectionCode())
	}
}

func TestSyncOutbox_Inventory_EmptyPackagesAndNoErrors_Rejected(t *testing.T) {
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())

	endpointUUID := uuid.New()
	endpointID := pgtype.UUID{Bytes: endpointUUID, Valid: true}
	tenantIDStr := uuid.New().String()
	agentIDStr := endpointUUID.String()

	report := &pb.InventoryReport{
		AgentId:           agentIDStr,
		ProtocolVersion:   1,
		InstalledPackages: []*pb.PackageInfo{}, // empty
		CollectedAt:       timestamppb.Now(),
	}
	payload, err := proto.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	msg := &pb.OutboxMessage{
		MessageId: "msg-inv-2",
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_INVENTORY,
		Payload:   payload,
	}

	ack, processErr := svc.ExportedProcessInventory(context.Background(), msg, endpointID, tenantIDStr, agentIDStr)
	if processErr == nil {
		t.Fatal("expected error for empty packages, got nil")
	}
	if ack == nil {
		t.Fatal("expected non-nil ack even on error")
	}
	if ack.GetMessageId() != "msg-inv-2" {
		t.Errorf("ack message_id = %q, want %q", ack.GetMessageId(), "msg-inv-2")
	}
	if ack.GetRejectionCode() != pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_PAYLOAD_INVALID {
		t.Errorf("rejection_code = %v, want PAYLOAD_INVALID", ack.GetRejectionCode())
	}
}

// A partial report (zero packages BUT with collection_errors) must bypass the
// early PAYLOAD_INVALID guard: the agent tried to collect, some collectors
// failed, and that outcome is valid data the server must persist. Because this
// test uses a nil-pool store, the flow will reach BeginTx and fail there —
// that's the signal we got past validation.
func TestSyncOutbox_Inventory_PartialReportPassesValidation(t *testing.T) {
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())

	endpointUUID := uuid.New()
	endpointID := pgtype.UUID{Bytes: endpointUUID, Valid: true}
	tenantIDStr := uuid.New().String()
	agentIDStr := endpointUUID.String()

	report := &pb.InventoryReport{
		AgentId:           agentIDStr,
		ProtocolVersion:   1,
		InstalledPackages: []*pb.PackageInfo{},
		CollectionErrors: []*pb.InventoryCollectionError{
			{Collector: "wua", ErrorMessage: "COM init failed"},
		},
		CollectedAt: timestamppb.Now(),
	}
	payload, err := proto.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	msg := &pb.OutboxMessage{
		MessageId: "msg-inv-partial",
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_INVENTORY,
		Payload:   payload,
	}

	ack, processErr := svc.ExportedProcessInventory(context.Background(), msg, endpointID, tenantIDStr, agentIDStr)
	if processErr == nil {
		t.Fatal("expected DB error downstream of validation, got nil")
	}
	if ack == nil {
		t.Fatal("expected non-nil ack")
	}
	// The partial report must NOT be rejected as PAYLOAD_INVALID — it should
	// reach the DB path and fail there with SERVER_OVERLOADED (our nil-pool
	// sentinel). Any other rejection code means we regressed the guard.
	if ack.GetRejectionCode() == pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_PAYLOAD_INVALID {
		t.Errorf("partial report was rejected as PAYLOAD_INVALID; validation regressed")
	}
}

func TestSyncOutbox_CommandResult_EmitFailure_ReturnsError(t *testing.T) {
	emitErr := errors.New("event bus unavailable")
	svc := servergrpc.NewAgentServiceServer(testStore(t), failingEventBus{err: emitErr}, slog.Default())

	tenantIDStr := uuid.New().String()

	cmdResp := &pb.CommandResponse{
		CommandId: uuid.New().String(),
		Status:    pb.CommandStatus_COMMAND_STATUS_SUCCEEDED,
		Output:    []byte("patch applied"),
	}
	payload, err := proto.Marshal(cmdResp)
	if err != nil {
		t.Fatalf("marshal command response: %v", err)
	}

	msg := &pb.OutboxMessage{
		MessageId: "msg-cmd-fail-1",
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_COMMAND_RESULT,
		Payload:   payload,
	}

	err = svc.ExportedProcessCommandResult(context.Background(), tenantIDStr, uuid.New().String(), msg)
	if err == nil {
		t.Fatal("expected error when event bus fails, got nil")
	}
	if !errors.Is(err, emitErr) {
		t.Errorf("expected error to wrap %v, got %v", emitErr, err)
	}
}

func TestSyncOutbox_CommandResult_InvalidPayload_ReturnsPayloadInvalidError(t *testing.T) {
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())

	msg := &pb.OutboxMessage{
		MessageId: "msg-cmd-fail-2",
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_COMMAND_RESULT,
		Payload:   []byte("not valid protobuf"),
	}

	err := svc.ExportedProcessCommandResult(context.Background(), uuid.New().String(), uuid.New().String(), msg)
	if err == nil {
		t.Fatal("expected error for invalid payload, got nil")
	}
	if !errors.Is(err, servergrpc.ExportedErrPayloadInvalid) {
		t.Errorf("expected error to wrap errPayloadInvalid, got %v", err)
	}
}

func TestSyncOutbox_CommandResult_EmitFailure_IsNotPayloadInvalid(t *testing.T) {
	// Verifies that when processCommandResult fails due to event bus failure
	// (not a payload issue), the error does NOT wrap errPayloadInvalid. This
	// ensures the COMMAND_RESULT case in SyncOutbox treats it as a transient
	// failure (SERVER_OVERLOADED rejection, agent retries) rather than a
	// permanent rejection (PAYLOAD_INVALID).
	emitErr := errors.New("event bus unavailable")
	svc := servergrpc.NewAgentServiceServer(testStore(t), failingEventBus{err: emitErr}, slog.Default())

	cmdResp := &pb.CommandResponse{
		CommandId: uuid.New().String(),
		Status:    pb.CommandStatus_COMMAND_STATUS_SUCCEEDED,
		Output:    []byte("ok"),
	}
	payload, err := proto.Marshal(cmdResp)
	if err != nil {
		t.Fatalf("marshal command response: %v", err)
	}

	msg := &pb.OutboxMessage{
		MessageId: "msg-cmd-bus-fail",
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_COMMAND_RESULT,
		Payload:   payload,
	}

	err = svc.ExportedProcessCommandResult(context.Background(), uuid.New().String(), uuid.New().String(), msg)
	if err == nil {
		t.Fatal("expected error when event bus fails, got nil")
	}
	if errors.Is(err, servergrpc.ExportedErrPayloadInvalid) {
		t.Error("event bus failure should NOT wrap errPayloadInvalid")
	}
}

func TestSyncOutbox_Inventory_ServerOverloaded_OnTenantIDParseFailure(t *testing.T) {
	// Verifies processInventory returns a SERVER_OVERLOADED rejection ACK when
	// the tenant ID is not a valid UUID. The invalid tenant ID triggers a parse
	// failure before any DB or event bus access, producing a SERVER_OVERLOADED
	// rejection code.
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())

	endpointUUID := uuid.New()
	endpointID := pgtype.UUID{Bytes: endpointUUID, Valid: true}
	agentIDStr := endpointUUID.String()

	report := &pb.InventoryReport{
		AgentId:         agentIDStr,
		ProtocolVersion: 1,
		InstalledPackages: []*pb.PackageInfo{
			{Name: "openssl", Version: "3.0.2", Architecture: "amd64"},
		},
		CollectedAt: timestamppb.Now(),
	}
	payload, err := proto.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	msg := &pb.OutboxMessage{
		MessageId: "msg-inv-evt-fail",
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_INVENTORY,
		Payload:   payload,
	}

	ack, processErr := svc.ExportedProcessInventory(context.Background(), msg, endpointID, "not-a-uuid", agentIDStr)
	if processErr == nil {
		t.Fatal("expected error, got nil")
	}
	if ack == nil {
		t.Fatal("expected non-nil ack even on error")
	}
	if ack.GetMessageId() != "msg-inv-evt-fail" {
		t.Errorf("ack message_id = %q, want %q", ack.GetMessageId(), "msg-inv-evt-fail")
	}
	if ack.GetRejectionCode() != pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_SERVER_OVERLOADED {
		t.Errorf("rejection_code = %v, want SERVER_OVERLOADED", ack.GetRejectionCode())
	}
}

// NOTE: Stream-level tests for the COMMAND_RESULT accept/reject branching require
// a real DB (LookupEndpointByID). The unit tests below cover processCommandResult
// directly via ExportedProcessCommandResult, which exercises the same code path.
// Full stream-level coverage is provided by integration tests.

func TestSyncOutbox_CommandResult_Success(t *testing.T) {
	bus := &capturingEventBus{}
	svc := servergrpc.NewAgentServiceServer(testStore(t), bus, slog.Default())

	agentID := uuid.New().String()

	// Build a proper InstallPatchOutput proto as the CommandResponse.Output.
	installOutput := &pb.InstallPatchOutput{
		Results: []*pb.InstallResultDetail{
			{
				PackageName: "curl",
				Version:     "7.88.1",
				Succeeded:   true,
				ExitCode:    0,
				Stdout:      "patch applied",
			},
		},
	}
	outputBytes, err := proto.Marshal(installOutput)
	if err != nil {
		t.Fatalf("marshal install output: %v", err)
	}

	cmdResp := &pb.CommandResponse{
		CommandId:    uuid.New().String(),
		Status:       pb.CommandStatus_COMMAND_STATUS_SUCCEEDED,
		Output:       outputBytes,
		ErrorMessage: "",
	}
	payload, err := proto.Marshal(cmdResp)
	if err != nil {
		t.Fatalf("marshal command response: %v", err)
	}

	msg := &pb.OutboxMessage{
		MessageId: "msg-cmd-ok",
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_COMMAND_RESULT,
		Payload:   payload,
	}

	err = svc.ExportedProcessCommandResult(context.Background(), uuid.New().String(), agentID, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	emitted := bus.Events()
	if len(emitted) != 1 {
		t.Fatalf("expected 1 emitted event, got %d", len(emitted))
	}
	resultPayload, ok := emitted[0].Payload.(events.CommandResultPayload)
	if !ok {
		t.Fatalf("expected CommandResultPayload, got %T", emitted[0].Payload)
	}
	if resultPayload.AgentID != agentID {
		t.Errorf("emitted event AgentID = %q, want %q", resultPayload.AgentID, agentID)
	}
	if resultPayload.Output != "[curl] patch applied" {
		t.Errorf("emitted event Output = %q, want %q", resultPayload.Output, "[curl] patch applied")
	}
	if resultPayload.ExitCode == nil || *resultPayload.ExitCode != 0 {
		t.Errorf("emitted event ExitCode = %v, want 0", resultPayload.ExitCode)
	}
}

func TestSyncOutbox_CommandResult_Failed(t *testing.T) {
	bus := &capturingEventBus{}
	svc := servergrpc.NewAgentServiceServer(testStore(t), bus, slog.Default())

	agentID := uuid.New().String()

	// Build a proper InstallPatchOutput proto with failure details.
	installOutput := &pb.InstallPatchOutput{
		Results: []*pb.InstallResultDetail{
			{
				PackageName: "openssl",
				Version:     "3.0.2",
				Succeeded:   false,
				ExitCode:    1,
				Stderr:      "dependency conflict",
			},
		},
	}
	outputBytes, err := proto.Marshal(installOutput)
	if err != nil {
		t.Fatalf("marshal install output: %v", err)
	}

	cmdResp := &pb.CommandResponse{
		CommandId:    uuid.New().String(),
		Status:       pb.CommandStatus_COMMAND_STATUS_FAILED,
		Output:       outputBytes,
		ErrorMessage: "install failed: dependency conflict",
	}
	payload, err := proto.Marshal(cmdResp)
	if err != nil {
		t.Fatalf("marshal command response: %v", err)
	}

	msg := &pb.OutboxMessage{
		MessageId: "msg-cmd-fail",
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_COMMAND_RESULT,
		Payload:   payload,
	}

	err = svc.ExportedProcessCommandResult(context.Background(), uuid.New().String(), agentID, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	emitted := bus.Events()
	if len(emitted) != 1 {
		t.Fatalf("expected 1 emitted event, got %d", len(emitted))
	}
	resultPayload, ok := emitted[0].Payload.(events.CommandResultPayload)
	if !ok {
		t.Fatalf("expected CommandResultPayload, got %T", emitted[0].Payload)
	}
	if resultPayload.AgentID != agentID {
		t.Errorf("emitted event AgentID = %q, want %q", resultPayload.AgentID, agentID)
	}
	if resultPayload.Stderr != "[openssl] dependency conflict" {
		t.Errorf("emitted event Stderr = %q, want %q", resultPayload.Stderr, "[openssl] dependency conflict")
	}
	if resultPayload.ExitCode == nil || *resultPayload.ExitCode != 1 {
		t.Errorf("emitted event ExitCode = %v, want 1", resultPayload.ExitCode)
	}
}

func TestSyncOutbox_UnknownMessageType_RejectedAck(t *testing.T) {
	// Verifies the rejectedAck helper produces a correct UNKNOWN_TYPE rejection.
	// The full stream path requires a DB (for endpoint lookup), so this tests the
	// ack construction directly.
	msgID := "msg-unknown-1"
	ack := servergrpc.ExportedRejectedAck(msgID,
		pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_UNKNOWN_TYPE,
		"unsupported message type: OUTBOX_MESSAGE_TYPE_UNSPECIFIED",
	)
	if ack.GetMessageId() != msgID {
		t.Errorf("ack message_id = %q, want %q", ack.GetMessageId(), msgID)
	}
	if ack.GetRejectionCode() != pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_UNKNOWN_TYPE {
		t.Errorf("rejection_code = %v, want UNKNOWN_TYPE", ack.GetRejectionCode())
	}
	if ack.GetRejectionDetail() == "" {
		t.Error("expected non-empty rejection detail")
	}
}
