package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

// performEnroll executes the full enrollment flow: open DB, dial server, enroll,
// and write the agent config file. logStatus is called at each phase boundary to
// report progress. Returns the agent ID on success.
func performEnroll(ctx context.Context, opts installOpts, logStatus func(string)) (string, error) {
	dataDir := opts.dataDir
	if dataDir == "" {
		dataDir = DefaultDataDir()
	}

	logStatus("Opening database...")
	dbPath := filepath.Join(dataDir, "agent.db")
	db, err := comms.OpenDB(dbPath)
	if err != nil {
		return "", fmt.Errorf("install: open database %s: %w", dbPath, err)
	}
	defer db.Close()

	logStatus("Connecting to server...")
	dialCtx, dialCancel := context.WithTimeout(ctx, 30*time.Second)
	defer dialCancel()

	conn, err := dialServer(dialCtx, opts.server)
	if err != nil {
		return "", fmt.Errorf("install: connect to server %s: %w", opts.server, err)
	}
	defer conn.Close()

	logStatus("Enrolling...")
	enroller := &grpcEnroller{client: pb.NewAgentServiceClient(conn)}
	state := comms.NewAgentState(db)
	result, err := doEnrollment(ctx, enroller, state, opts.token)
	if err != nil {
		return "", err
	}

	logStatus("Writing configuration...")
	configPath := opts.configPath
	if configPath == "" {
		configPath = defaultConfigPath
	}
	cfg := AgentConfig{
		ServerAddress: opts.server,
		DataDir:       dataDir,
		LogLevel:      "info",
		ScanInterval:  6 * time.Hour,
	}
	if err := writeAgentConfig(configPath, cfg); err != nil {
		return "", err
	}

	// Install and start systemd service (Linux only; no-op on other platforms).
	logStatus("Installing service...")
	if err := installService(configPath, logStatus); err != nil {
		return "", fmt.Errorf("install service: %w", err)
	}

	return result.AgentID, nil
}
