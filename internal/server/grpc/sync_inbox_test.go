package grpc_test

import (
	"log/slog"
	"testing"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	servergrpc "github.com/skenzeriq/patchiq/internal/server/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSyncInbox_emptyAgentID(t *testing.T) {
	t.Parallel()
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())
	err := svc.SyncInbox(&pb.InboxRequest{AgentId: ""}, nil)
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestSyncInbox_invalidUUID(t *testing.T) {
	t.Parallel()
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())
	err := svc.SyncInbox(&pb.InboxRequest{AgentId: "not-a-uuid"}, nil)
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestMapCommandType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input deployment.CommandType
		want  pb.CommandType
	}{
		{deployment.CommandTypeInstallPatch, pb.CommandType_COMMAND_TYPE_INSTALL_PATCH},
		{deployment.CommandTypeRunScan, pb.CommandType_COMMAND_TYPE_RUN_SCAN},
		{deployment.CommandTypeUpdateConfig, pb.CommandType_COMMAND_TYPE_UPDATE_CONFIG},
		{deployment.CommandTypeReboot, pb.CommandType_COMMAND_TYPE_REBOOT},
		{deployment.CommandTypeRunScript, pb.CommandType_COMMAND_TYPE_RUN_SCRIPT},
		{"unknown_type", pb.CommandType_COMMAND_TYPE_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			t.Parallel()
			got := servergrpc.ExportedMapCommandType(tt.input)
			if got != tt.want {
				t.Fatalf("mapCommandType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
