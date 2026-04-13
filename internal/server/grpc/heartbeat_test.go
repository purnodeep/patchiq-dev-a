package grpc_test

import (
	"context"
	"log/slog"
	"net"
	"testing"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	servergrpc "github.com/skenzeriq/patchiq/internal/server/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func setupBufconn(t *testing.T, svc pb.AgentServiceServer) (pb.AgentServiceClient, func()) {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	pb.RegisterAgentServiceServer(srv, svc)
	go srv.Serve(lis) //nolint:errcheck

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}

	client := pb.NewAgentServiceClient(conn)
	cleanup := func() {
		conn.Close()
		srv.Stop()
	}
	return client, cleanup
}

func TestHeartbeat_EmptyAgentID(t *testing.T) {
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())
	client, cleanup := setupBufconn(t, svc)
	defer cleanup()

	stream, err := client.Heartbeat(context.Background())
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	// Send a heartbeat with empty agent_id.
	err = stream.Send(&pb.HeartbeatRequest{
		AgentId: "",
		Status:  pb.AgentStatus_AGENT_STATUS_IDLE,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	// Receive should return InvalidArgument.
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

func TestHeartbeat_InvalidAgentID(t *testing.T) {
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())
	client, cleanup := setupBufconn(t, svc)
	defer cleanup()

	stream, err := client.Heartbeat(context.Background())
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	err = stream.Send(&pb.HeartbeatRequest{
		AgentId: "not-a-uuid",
		Status:  pb.AgentStatus_AGENT_STATUS_IDLE,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

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

func TestMapAgentStatus(t *testing.T) {
	tests := []struct {
		input pb.AgentStatus
		want  string
	}{
		{pb.AgentStatus_AGENT_STATUS_IDLE, "online"},
		{pb.AgentStatus_AGENT_STATUS_BUSY, "online"},
		{pb.AgentStatus_AGENT_STATUS_UPDATING, "online"},
		{pb.AgentStatus_AGENT_STATUS_ERROR, "offline"},
		{pb.AgentStatus_AGENT_STATUS_UNSPECIFIED, "online"},
	}

	for _, tc := range tests {
		t.Run(tc.input.String(), func(t *testing.T) {
			got := servergrpc.ExportedMapAgentStatus(tc.input)
			if got != tc.want {
				t.Errorf("mapAgentStatus(%s) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
