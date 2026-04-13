package comms_test

import (
	"context"
	"testing"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

type mockEnroller struct {
	resp   *pb.EnrollResponse
	err    error
	called bool
}

func (m *mockEnroller) Enroll(ctx context.Context, req *pb.EnrollRequest) (*pb.EnrollResponse, error) {
	m.called = true
	return m.resp, m.err
}

func TestEnroll_Success(t *testing.T) {
	_, _, state := openTestDBRaw(t)
	ctx := context.Background()

	mock := &mockEnroller{
		resp: &pb.EnrollResponse{
			AgentId:                   "agent-abc",
			NegotiatedProtocolVersion: 1,
			Config:                    &pb.AgentConfig{HeartbeatIntervalSeconds: 30},
			ErrorCode:                 pb.EnrollmentErrorCode_ENROLLMENT_ERROR_CODE_UNSPECIFIED,
		},
	}

	meta := comms.AgentMeta{AgentVersion: "1.0.0", ProtocolVersion: 1, Capabilities: []string{"inventory"}}
	endpoint := &pb.EndpointInfo{Hostname: "host-1", OsFamily: pb.OsFamily_OS_FAMILY_LINUX, OsVersion: "ubuntu-22.04"}

	result, err := comms.Enroll(ctx, mock, state, "tok-123", meta, endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.called {
		t.Error("expected mock to be called")
	}
	if result.AgentID != "agent-abc" {
		t.Errorf("AgentID: got %q, want %q", result.AgentID, "agent-abc")
	}
	if result.NegotiatedVersion != 1 {
		t.Errorf("NegotiatedVersion: got %d, want 1", result.NegotiatedVersion)
	}
	if result.Config == nil || result.Config.HeartbeatIntervalSeconds != 30 {
		t.Errorf("Config: got %+v, want HeartbeatIntervalSeconds=30", result.Config)
	}

	// Verify agent_id persisted in state
	stored, err := state.Get(ctx, "agent_id")
	if err != nil {
		t.Fatalf("state.Get: %v", err)
	}
	if stored != "agent-abc" {
		t.Errorf("stored agent_id: got %q, want %q", stored, "agent-abc")
	}

	// Verify negotiated_protocol_version persisted
	ver, err := state.Get(ctx, "negotiated_protocol_version")
	if err != nil {
		t.Fatalf("state.Get: %v", err)
	}
	if ver != "1" {
		t.Errorf("stored negotiated_protocol_version: got %q, want %q", ver, "1")
	}
}

func TestEnroll_SkipsIfAgentIDExists(t *testing.T) {
	_, _, state := openTestDBRaw(t)
	ctx := context.Background()

	// Pre-set agent_id
	if err := state.Set(ctx, "agent_id", "existing-agent"); err != nil {
		t.Fatalf("state.Set: %v", err)
	}
	if err := state.Set(ctx, "negotiated_protocol_version", "1"); err != nil {
		t.Fatalf("state.Set: %v", err)
	}

	mock := &mockEnroller{}
	meta := comms.AgentMeta{AgentVersion: "1.0.0", ProtocolVersion: 1}
	endpoint := &pb.EndpointInfo{Hostname: "host-1", OsFamily: pb.OsFamily_OS_FAMILY_LINUX}

	result, err := comms.Enroll(ctx, mock, state, "tok-123", meta, endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.called {
		t.Error("expected mock NOT to be called when agent_id already exists")
	}
	if result.AgentID != "existing-agent" {
		t.Errorf("AgentID: got %q, want %q", result.AgentID, "existing-agent")
	}
	if result.NegotiatedVersion != 1 {
		t.Errorf("NegotiatedVersion: got %d, want 1", result.NegotiatedVersion)
	}
}

func TestEnroll_ErrorCodeReturnsError(t *testing.T) {
	_, _, state := openTestDBRaw(t)
	ctx := context.Background()

	mock := &mockEnroller{
		resp: &pb.EnrollResponse{
			ErrorCode:    pb.EnrollmentErrorCode_ENROLLMENT_ERROR_CODE_INVALID_TOKEN,
			ErrorMessage: "token is invalid",
		},
	}

	meta := comms.AgentMeta{AgentVersion: "1.0.0", ProtocolVersion: 1}
	endpoint := &pb.EndpointInfo{Hostname: "host-1", OsFamily: pb.OsFamily_OS_FAMILY_LINUX}

	_, err := comms.Enroll(ctx, mock, state, "tok-123", meta, endpoint)
	if err == nil {
		t.Fatal("expected error for INVALID_TOKEN error code")
	}

	// Verify agent_id NOT stored
	stored, stateErr := state.Get(ctx, "agent_id")
	if stateErr != nil {
		t.Fatalf("state.Get: %v", stateErr)
	}
	if stored != "" {
		t.Errorf("agent_id should not be stored on error, got %q", stored)
	}
}

func TestNegotiateProtocolVersion(t *testing.T) {
	tests := []struct {
		name      string
		agentVer  uint32
		serverVer uint32
		serverMin uint32
		wantVer   uint32
		wantErr   bool
	}{
		{"same version", 1, 1, 1, 1, false},
		{"agent lower", 3, 5, 3, 3, false},
		{"server lower", 5, 3, 1, 3, false},
		{"agent too old", 2, 5, 3, 0, true},
		{"both zero", 0, 0, 0, 0, true},
		{"agent version zero", 0, 5, 1, 0, true},
		{"server version zero", 3, 0, 1, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := comms.NegotiateProtocolVersion(tt.agentVer, tt.serverVer, tt.serverMin)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantVer {
				t.Errorf("expected version %d, got %d", tt.wantVer, got)
			}
		})
	}
}

func TestBuildEnrollRequest(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		info     comms.AgentMeta
		endpoint *pb.EndpointInfo
		wantErr  bool
	}{
		{
			name:  "populates all fields including endpoint info",
			token: "tok-123",
			info: comms.AgentMeta{
				AgentVersion:    "1.0.0",
				ProtocolVersion: 1,
				Capabilities:    []string{"inventory"},
			},
			endpoint: &pb.EndpointInfo{
				Hostname:  "host-1",
				OsFamily:  pb.OsFamily_OS_FAMILY_LINUX,
				OsVersion: "linux/amd64",
			},
		},
		{
			name:  "empty token returns error",
			token: "",
			info: comms.AgentMeta{
				AgentVersion:    "1.0.0",
				ProtocolVersion: 1,
			},
			endpoint: &pb.EndpointInfo{
				Hostname: "host-1",
				OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
			},
			wantErr: true,
		},
		{
			name:  "nil endpoint info returns error",
			token: "tok-123",
			info: comms.AgentMeta{
				AgentVersion:    "1.0.0",
				ProtocolVersion: 1,
			},
			endpoint: nil,
			wantErr:  true,
		},
		{
			name:  "empty agent version returns error",
			token: "tok-123",
			info: comms.AgentMeta{
				AgentVersion:    "",
				ProtocolVersion: 1,
			},
			endpoint: &pb.EndpointInfo{
				Hostname: "host-1",
				OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
			},
			wantErr: true,
		},
		{
			name:  "zero protocol version returns error",
			token: "tok-123",
			info: comms.AgentMeta{
				AgentVersion:    "1.0.0",
				ProtocolVersion: 0,
			},
			endpoint: &pb.EndpointInfo{
				Hostname: "host-1",
				OsFamily: pb.OsFamily_OS_FAMILY_LINUX,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := comms.BuildEnrollRequest(tt.token, tt.info, tt.endpoint)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if req.EnrollmentToken != tt.token {
				t.Errorf("token: got %q, want %q", req.EnrollmentToken, tt.token)
			}
			if req.AgentInfo == nil {
				t.Fatal("agent_info is nil")
			}
			if req.AgentInfo.AgentVersion != tt.info.AgentVersion {
				t.Errorf("agent_version: got %q, want %q", req.AgentInfo.AgentVersion, tt.info.AgentVersion)
			}
			if req.AgentInfo.ProtocolVersion != tt.info.ProtocolVersion {
				t.Errorf("protocol_version: got %d, want %d", req.AgentInfo.ProtocolVersion, tt.info.ProtocolVersion)
			}
			if req.EndpointInfo != tt.endpoint {
				t.Errorf("endpoint_info: got %p, want %p (pointer identity)", req.EndpointInfo, tt.endpoint)
			}
			if len(req.AgentInfo.Capabilities) != len(tt.info.Capabilities) {
				t.Errorf("capabilities length: got %d, want %d", len(req.AgentInfo.Capabilities), len(tt.info.Capabilities))
			}
		})
	}
}
