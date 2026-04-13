package grpc_test

import (
	"context"
	"log/slog"
	"testing"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	servergrpc "github.com/skenzeriq/patchiq/internal/server/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestEnroll_Validation(t *testing.T) {
	// Create server — validation runs before any DB access.
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())

	tests := []struct {
		name string
		req  *pb.EnrollRequest
	}{
		{
			name: "nil request",
			req:  nil,
		},
		{
			name: "missing agent_info",
			req: &pb.EnrollRequest{
				EnrollmentToken: "tok",
				EndpointInfo: &pb.EndpointInfo{
					Hostname: "host",
					OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
				},
			},
		},
		{
			name: "empty token",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "1.0.0",
					ProtocolVersion: 1,
				},
				EnrollmentToken: "",
				EndpointInfo: &pb.EndpointInfo{
					Hostname: "host",
					OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
				},
			},
		},
		{
			name: "missing endpoint_info",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "1.0.0",
					ProtocolVersion: 1,
				},
				EnrollmentToken: "tok",
			},
		},
		{
			name: "missing hostname",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "1.0.0",
					ProtocolVersion: 1,
				},
				EnrollmentToken: "tok",
				EndpointInfo: &pb.EndpointInfo{
					OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
				},
			},
		},
		{
			name: "missing os_family",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "1.0.0",
					ProtocolVersion: 1,
				},
				EnrollmentToken: "tok",
				EndpointInfo: &pb.EndpointInfo{
					Hostname: "host",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.Enroll(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got %T: %v", err, err)
			}
			if st.Code() != codes.InvalidArgument {
				t.Errorf("expected InvalidArgument, got %v", st.Code())
			}
			if resp != nil {
				t.Errorf("expected nil response, got %+v", resp)
			}
		})
	}
}
