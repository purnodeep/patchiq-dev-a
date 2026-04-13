//go:build integration

package comms_test

import (
	"context"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockAgentService struct {
	pb.UnimplementedAgentServiceServer
	enrollResp     *pb.EnrollResponse
	mu             sync.Mutex
	heartbeatCount int
}

func (s *mockAgentService) Enroll(_ context.Context, _ *pb.EnrollRequest) (*pb.EnrollResponse, error) {
	return s.enrollResp, nil
}

func (s *mockAgentService) Heartbeat(stream pb.AgentService_HeartbeatServer) error {
	for {
		_, err := stream.Recv()
		if err != nil {
			return nil
		}
		s.mu.Lock()
		s.heartbeatCount++
		s.mu.Unlock()
		if err := stream.Send(&pb.HeartbeatResponse{
			ServerTimestamp: timestamppb.Now(),
		}); err != nil {
			return err
		}
	}
}

func (s *mockAgentService) getHeartbeatCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.heartbeatCount
}

func TestIntegration_EnrollAndHeartbeat(t *testing.T) {
	// 1. Start in-process gRPC server via bufconn.
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	mockSvc := &mockAgentService{
		enrollResp: &pb.EnrollResponse{
			AgentId:                   "int-agent-001",
			NegotiatedProtocolVersion: 1,
			Config:                    &pb.AgentConfig{HeartbeatIntervalSeconds: 1},
			ErrorCode:                 pb.EnrollmentErrorCode_ENROLLMENT_ERROR_CODE_UNSPECIFIED,
		},
	}
	pb.RegisterAgentServiceServer(srv, mockSvc)
	go srv.Serve(lis) //nolint:errcheck // test server
	defer srv.Stop()

	// 2. Create client connection via bufconn.
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	defer conn.Close()

	agentClient := pb.NewAgentServiceClient(conn)
	ctx := context.Background()

	// 3. Set up agent state with SQLite.
	dir := t.TempDir()
	db, err := comms.OpenDB(filepath.Join(dir, "integration.db"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()
	state := comms.NewAgentState(db)

	// 4. Test enrollment.
	enroller := comms.NewGRPCEnroller(agentClient)
	meta := comms.AgentMeta{AgentVersion: "1.0.0", ProtocolVersion: 1, Capabilities: []string{"inventory"}}
	endpoint := &pb.EndpointInfo{Hostname: "int-host", OsFamily: pb.OsFamily_OS_FAMILY_LINUX, OsVersion: "ubuntu-22.04"}

	result, err := comms.Enroll(ctx, enroller, state, "tok-int", meta, endpoint)
	if err != nil {
		t.Fatalf("Enroll: %v", err)
	}
	if result.AgentID != "int-agent-001" {
		t.Errorf("AgentID: got %q, want %q", result.AgentID, "int-agent-001")
	}
	if result.NegotiatedVersion != 1 {
		t.Errorf("NegotiatedVersion: got %d, want 1", result.NegotiatedVersion)
	}

	// Verify enrollment persisted.
	stored, err := state.Get(ctx, "agent_id")
	if err != nil {
		t.Fatalf("state.Get agent_id: %v", err)
	}
	if stored != "int-agent-001" {
		t.Errorf("stored agent_id: got %q, want %q", stored, "int-agent-001")
	}

	// 5. Test heartbeat.
	outbox := comms.NewOutbox(db)

	streamer := comms.NewGRPCHeartbeatStreamer(agentClient)
	cfg := comms.HeartbeatConfig{
		Interval:        500 * time.Millisecond,
		AgentID:         result.AgentID,
		ProtocolVersion: result.NegotiatedVersion,
		StartTime:       time.Now(),
	}

	hbCtx, hbCancel := context.WithTimeout(ctx, 3*time.Second)
	defer hbCancel()

	err = comms.RunHeartbeat(hbCtx, streamer, cfg, outbox)
	// Should exit due to context timeout.
	if err != nil && ctx.Err() == nil {
		// Accept context deadline exceeded from the heartbeat context.
		if hbCtx.Err() == nil {
			t.Fatalf("RunHeartbeat unexpected error: %v", err)
		}
	}

	// Verify mock received at least 2 heartbeats.
	count := mockSvc.getHeartbeatCount()
	if count < 2 {
		t.Errorf("heartbeat count: got %d, want >= 2", count)
	}
	t.Logf("heartbeat exchanges: %d", count)
}
