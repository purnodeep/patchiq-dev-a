package protocol

import "fmt"

// NegotiateProtocolVersion selects the negotiated protocol version.
// Returns min(agentVersion, serverVersion) if agentVersion >= serverMinVersion.
func NegotiateProtocolVersion(agentVersion, serverVersion, serverMinVersion uint32) (uint32, error) {
	if agentVersion == 0 || serverVersion == 0 {
		return 0, fmt.Errorf("protocol version must be > 0 (agent=%d, server=%d)", agentVersion, serverVersion)
	}
	if agentVersion < serverMinVersion {
		return 0, fmt.Errorf("agent protocol version %d is below server minimum %d", agentVersion, serverMinVersion)
	}
	return min(agentVersion, serverVersion), nil
}
