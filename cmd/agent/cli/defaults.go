package cli

import "fmt"

// DefaultServerAddress is the Patch Manager gRPC address baked into the
// binary at build time via -ldflags. Empty in development builds; set to
// the public address (e.g. "patchiq.skenzer.com:3013") in release builds.
//
// Override at build time:
//
//	go build -ldflags "-X github.com/skenzeriq/patchiq/cmd/agent/cli.DefaultServerAddress=patchiq.example.com:3013" ./cmd/agent
var DefaultServerAddress = ""

// resolveServerAddress picks the server address using the precedence:
// explicit flag > environment variable > ldflags-baked default.
// Returns an error if all three are empty.
func resolveServerAddress(flag, env, baked string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if env != "" {
		return env, nil
	}
	if baked != "" {
		return baked, nil
	}
	return "", fmt.Errorf("no server address: pass --server, set PATCHIQ_AGENT_SERVER_ADDRESS, or build the release binary with a baked-in DefaultServerAddress")
}
