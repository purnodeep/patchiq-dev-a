package comms

import (
	"context"
	"fmt"
	"strconv"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"

	"github.com/skenzeriq/patchiq/internal/shared/protocol"
)

// Enroller abstracts the gRPC Enroll RPC call.
type Enroller interface {
	Enroll(ctx context.Context, req *pb.EnrollRequest) (*pb.EnrollResponse, error)
}

// EnrollResult holds the outcome of a successful enrollment.
type EnrollResult struct {
	AgentID           string
	NegotiatedVersion uint32
	Config            *pb.AgentConfig
}

// Enroll performs agent enrollment via the gRPC Enroll RPC. If the agent is
// already enrolled (agent_id exists in state), it skips the RPC and returns
// the stored values.
func Enroll(ctx context.Context, client Enroller, state *AgentState, token string, meta AgentMeta, endpoint *pb.EndpointInfo) (EnrollResult, error) {
	// Check if already enrolled.
	agentID, err := state.Get(ctx, "agent_id")
	if err != nil {
		return EnrollResult{}, fmt.Errorf("enroll check state: %w", err)
	}
	if agentID != "" {
		verStr, err := state.Get(ctx, "negotiated_protocol_version")
		if err != nil {
			return EnrollResult{}, fmt.Errorf("enroll get negotiated version: %w", err)
		}
		ver, err := strconv.ParseUint(verStr, 10, 32)
		if err != nil && verStr != "" {
			return EnrollResult{}, fmt.Errorf("enroll parse negotiated version %q: %w", verStr, err)
		}
		return EnrollResult{
			AgentID:           agentID,
			NegotiatedVersion: uint32(ver),
		}, nil
	}

	// Build and send the enrollment request.
	req, err := BuildEnrollRequest(token, meta, endpoint)
	if err != nil {
		return EnrollResult{}, fmt.Errorf("enroll: %w", err)
	}

	resp, err := client.Enroll(ctx, req)
	if err != nil {
		return EnrollResult{}, fmt.Errorf("enroll rpc: %w", err)
	}

	// Check for server-side error codes.
	if resp.ErrorCode != pb.EnrollmentErrorCode_ENROLLMENT_ERROR_CODE_UNSPECIFIED {
		return EnrollResult{}, fmt.Errorf("enroll rejected: %s: %s", resp.ErrorCode.String(), resp.ErrorMessage)
	}

	// Persist enrollment state.
	if err := state.Set(ctx, "agent_id", resp.AgentId); err != nil {
		return EnrollResult{}, fmt.Errorf("enroll store agent_id: %w", err)
	}
	if err := state.Set(ctx, "negotiated_protocol_version", strconv.FormatUint(uint64(resp.NegotiatedProtocolVersion), 10)); err != nil {
		return EnrollResult{}, fmt.Errorf("enroll store negotiated_protocol_version: %w", err)
	}

	return EnrollResult{
		AgentID:           resp.AgentId,
		NegotiatedVersion: resp.NegotiatedProtocolVersion,
		Config:            resp.Config,
	}, nil
}

// AgentMeta holds agent version and capability metadata for enrollment.
// AgentID is omitted because the server assigns it in EnrollResponse.
type AgentMeta struct {
	AgentVersion    string
	ProtocolVersion uint32
	Capabilities    []string
}

// BuildEnrollRequest constructs an EnrollRequest from the given token, agent metadata, and endpoint info.
func BuildEnrollRequest(token string, meta AgentMeta, endpoint *pb.EndpointInfo) (*pb.EnrollRequest, error) {
	if token == "" {
		return nil, fmt.Errorf("build enroll request: enrollment token is required")
	}
	if meta.AgentVersion == "" {
		return nil, fmt.Errorf("build enroll request: agent_version is required")
	}
	if meta.ProtocolVersion == 0 {
		return nil, fmt.Errorf("build enroll request: protocol_version must be > 0")
	}
	if endpoint == nil {
		return nil, fmt.Errorf("build enroll request: endpoint info is required")
	}
	return &pb.EnrollRequest{
		AgentInfo: &pb.AgentInfo{
			AgentVersion:    meta.AgentVersion,
			ProtocolVersion: meta.ProtocolVersion,
			Capabilities:    meta.Capabilities,
		},
		EnrollmentToken: token,
		EndpointInfo:    endpoint,
	}, nil
}

// NegotiateProtocolVersion delegates to the shared protocol package.
var NegotiateProtocolVersion = protocol.NegotiateProtocolVersion
