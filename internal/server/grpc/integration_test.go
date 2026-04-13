//go:build integration

package grpc_test

import (
	"context"
	"log/slog"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	servergrpc "github.com/skenzeriq/patchiq/internal/server/grpc"
)

func TestIntegration_Enroll(t *testing.T) {
	// Start a gRPC server for validation-only testing.
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()

	srv := servergrpc.NewGRPCServer(servergrpc.ServerConfig{})
	agentSvc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())
	pb.RegisterAgentServiceServer(srv, agentSvc)

	go srv.Serve(lis)
	defer srv.GracefulStop()

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewAgentServiceClient(conn)

	tests := []struct {
		name     string
		req      *pb.EnrollRequest
		wantCode codes.Code
	}{
		{
			name: "missing agent_info returns InvalidArgument",
			req: &pb.EnrollRequest{
				EnrollmentToken: "test-token",
				EndpointInfo: &pb.EndpointInfo{
					Hostname: "test-host",
					OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing endpoint_info returns InvalidArgument",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "1.0.0",
					ProtocolVersion: 1,
				},
				EnrollmentToken: "test-token",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing enrollment_token returns InvalidArgument",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "1.0.0",
					ProtocolVersion: 1,
				},
				EndpointInfo: &pb.EndpointInfo{
					Hostname: "test-host",
					OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing os_family returns InvalidArgument",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "1.0.0",
					ProtocolVersion: 1,
				},
				EnrollmentToken: "test-token",
				EndpointInfo: &pb.EndpointInfo{
					Hostname: "test-host",
				},
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Enroll(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got: %v", err)
			}
			if st.Code() != tt.wantCode {
				t.Errorf("expected code %v, got %v: %s", tt.wantCode, st.Code(), st.Message())
			}
		})
	}
}
