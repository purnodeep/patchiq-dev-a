package v1_test

import (
	"testing"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEnrollRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.EnrollRequest
		wantErr string
	}{
		{
			name: "valid request with endpoint info",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "1.0.0",
					ProtocolVersion: 1,
				},
				EnrollmentToken: "tok-123",
				EndpointInfo: &pb.EndpointInfo{
					Hostname: "host-1",
					OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
				},
			},
		},
		{
			name:    "nil request",
			req:     nil,
			wantErr: "enroll request is nil",
		},
		{
			name: "missing endpoint info",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "1.0.0",
					ProtocolVersion: 1,
				},
				EnrollmentToken: "tok-123",
			},
			wantErr: "endpoint_info is required",
		},
		{
			name: "missing agent info",
			req: &pb.EnrollRequest{
				EnrollmentToken: "tok-123",
				EndpointInfo: &pb.EndpointInfo{
					Hostname: "host-1",
					OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
				},
			},
			wantErr: "agent_info is required",
		},
		{
			name: "empty enrollment token",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "1.0.0",
					ProtocolVersion: 1,
				},
				EndpointInfo: &pb.EndpointInfo{
					Hostname: "host-1",
					OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
				},
			},
			wantErr: "enrollment_token is required",
		},
		{
			name: "empty agent version",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "",
					ProtocolVersion: 1,
				},
				EnrollmentToken: "tok-123",
				EndpointInfo: &pb.EndpointInfo{
					Hostname: "host-1",
					OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
				},
			},
			wantErr: "agent_info.agent_version is required",
		},
		{
			name: "zero protocol version",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "1.0.0",
					ProtocolVersion: 0,
				},
				EnrollmentToken: "tok-123",
				EndpointInfo: &pb.EndpointInfo{
					Hostname: "host-1",
					OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
				},
			},
			wantErr: "agent_info.protocol_version must be > 0",
		},
		{
			name: "empty hostname",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "1.0.0",
					ProtocolVersion: 1,
				},
				EnrollmentToken: "tok-123",
				EndpointInfo: &pb.EndpointInfo{
					Hostname: "",
					OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
				},
			},
			wantErr: "endpoint_info.hostname is required",
		},
		{
			name: "unspecified os family",
			req: &pb.EnrollRequest{
				AgentInfo: &pb.AgentInfo{
					AgentVersion:    "1.0.0",
					ProtocolVersion: 1,
				},
				EnrollmentToken: "tok-123",
				EndpointInfo: &pb.EndpointInfo{
					Hostname: "host-1",
					OsFamily: pb.OsFamily_OS_FAMILY_UNSPECIFIED,
				},
			},
			wantErr: "endpoint_info.os_family is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v1.ValidateEnrollRequest(tt.req)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
