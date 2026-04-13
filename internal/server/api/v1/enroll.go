package v1

import (
	"fmt"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// ValidateEnrollRequest checks that an EnrollRequest contains all required
// fields, including EndpointInfo with a non-zero os_family.
func ValidateEnrollRequest(req *pb.EnrollRequest) error {
	if req == nil {
		return fmt.Errorf("validate enroll request: enroll request is nil")
	}
	if req.AgentInfo == nil {
		return fmt.Errorf("validate enroll request: agent_info is required")
	}
	if req.AgentInfo.AgentVersion == "" {
		return fmt.Errorf("validate enroll request: agent_info.agent_version is required")
	}
	if req.AgentInfo.ProtocolVersion == 0 {
		return fmt.Errorf("validate enroll request: agent_info.protocol_version must be > 0")
	}
	if req.EnrollmentToken == "" {
		return fmt.Errorf("validate enroll request: enrollment_token is required")
	}
	if req.EndpointInfo == nil {
		return fmt.Errorf("validate enroll request: endpoint_info is required")
	}
	if req.EndpointInfo.Hostname == "" {
		return fmt.Errorf("validate enroll request: endpoint_info.hostname is required")
	}
	if req.EndpointInfo.OsFamily == pb.OsFamily_OS_FAMILY_UNSPECIFIED {
		return fmt.Errorf("validate enroll request: endpoint_info.os_family is required")
	}
	return nil
}
