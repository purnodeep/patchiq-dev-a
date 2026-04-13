package cli

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/skenzeriq/patchiq/cmd/agent/sysinfo"
	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	yamlv3 "gopkg.in/yaml.v3"
)

type installOpts struct {
	server         string
	token          string
	nonInteractive bool
	configPath     string
	dataDir        string
	resetConfig    bool
}

// envEnrollmentToken is the environment variable name for passing the enrollment
// token without exposing it on the process command line (visible via ps/top).
const envEnrollmentToken = "PATCHIQ_ENROLLMENT_TOKEN"

func parseInstallFlags(args []string) (installOpts, error) {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	var opts installOpts
	opts.configPath = defaultConfigPath

	fs.StringVar(&opts.server, "server", "", "Patch Manager address (host:port)")
	fs.StringVar(&opts.token, "token", "", "Enrollment token (or set "+envEnrollmentToken+")")
	fs.BoolVar(&opts.nonInteractive, "non-interactive", false, "Run in non-interactive (headless) mode")
	fs.StringVar(&opts.configPath, "config", opts.configPath, "Path to agent config file")
	fs.StringVar(&opts.dataDir, "data-dir", "", "Data directory for agent storage")
	fs.BoolVar(&opts.resetConfig, "reset-config", false, "Delete any existing agent.yaml + agent.db before installing (forces a fresh enrollment)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: patchiq-agent install [flags]

Guided first-run setup. Validates server connectivity, runs enrollment,
and writes the agent configuration file.

The enrollment token can also be supplied via the %s
environment variable to avoid exposing it in the process list.

Examples:
  patchiq-agent install
  patchiq-agent install --server 10.0.0.1:50051 --token abc123 --non-interactive
  PATCHIQ_ENROLLMENT_TOKEN=abc123 patchiq-agent install --server 10.0.0.1:50051 --non-interactive

Flags:
`, envEnrollmentToken)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return installOpts{}, fmt.Errorf("install: parse flags: %w", err)
	}

	// If token was not supplied via flag, fall back to the environment variable.
	// This avoids leaking the token in the process list.
	if opts.token == "" {
		opts.token = os.Getenv(envEnrollmentToken)
	}

	return opts, nil
}

// validateInstallOpts checks invariants for the install command and resolves
// the server address using flag > env > ldflags-baked default precedence.
// Returns the resolved opts (with server address filled in) and an error if
// any required field is missing.
func validateInstallOpts(opts installOpts) (installOpts, error) {
	if opts.nonInteractive {
		if opts.token == "" {
			return opts, fmt.Errorf("install: --token is required in non-interactive mode")
		}
		envServer := os.Getenv("PATCHIQ_AGENT_SERVER_ADDRESS")
		resolved, err := resolveServerAddress(opts.server, envServer, DefaultServerAddress)
		if err != nil {
			return opts, fmt.Errorf("install: %w", err)
		}
		opts.server = resolved
	}
	return opts, nil
}

func writeAgentConfig(path string, cfg AgentConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("install: create config directory %s: %w", dir, err)
	}

	data, err := yamlv3.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("install: marshal agent config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("install: write config file %s: %w", path, err)
	}
	return nil
}

// RunInstall implements the "install" subcommand.
func RunInstall(args []string) int {
	if !isAdmin() {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  PatchIQ Agent installer must be run as Administrator.")
		fmt.Fprintln(os.Stderr, "  Right-click patchiq-agent.exe and select \"Run as administrator\".")
		fmt.Fprintln(os.Stderr, "")
		return ExitError
	}

	opts, err := parseInstallFlags(args)
	if err != nil {
		slog.Error("install: failed to parse flags", "error", err)
		return ExitError
	}

	resolved, err := validateInstallOpts(opts)
	if err != nil {
		slog.Error("install: validation failed", "error", err)
		return ExitError
	}
	opts = resolved

	// --reset-config: delete any existing config file and local database
	// BEFORE enrollment/install. Lets an operator recover deterministically
	// from a prior failed attempt without hand-deleting files on the box.
	if opts.resetConfig {
		if err := resetExistingConfig(opts); err != nil {
			slog.Error("install: reset-config failed", "error", err)
			return ExitError
		}
	}

	if !opts.nonInteractive {
		return runInstallTUI(opts)
	}

	return runInstallHeadless(opts)
}

// resetExistingConfig removes the agent.yaml + agent.db from both the
// resolved configPath/dataDir and the platform-default fallback path.
// Missing files are not an error — the whole point of reset is to tolerate
// any prior state. Only reports real I/O errors (permission denied etc).
func resetExistingConfig(opts installOpts) error {
	dataDir := opts.dataDir
	if dataDir == "" {
		dataDir = DefaultDataDir()
	}
	configPath := opts.configPath
	if configPath == "" {
		configPath = defaultConfigPath
	}

	paths := []string{
		configPath,
		filepath.Join(dataDir, "agent.db"),
	}
	// Also wipe the platform-default paths in case a prior non-elevated
	// run wrote to the fallback (%UserHome%\.patchiq on Windows).
	defaultDir := DefaultDataDir()
	if defaultDir != dataDir {
		paths = append(paths,
			filepath.Join(defaultDir, "agent.yaml"),
			filepath.Join(defaultDir, "agent.db"),
		)
	}

	for _, p := range paths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("reset-config: remove %s: %w", p, err)
		}
		slog.Info("reset-config: removed (or already absent)", "path", p)
	}
	return nil
}

func runInstallHeadless(opts installOpts) int {
	slog.Info("starting headless install", "server", opts.server, "config", opts.configPath)

	dataDir := opts.dataDir
	if dataDir == "" {
		dataDir = DefaultDataDir()
	}

	// Open agent database.
	dbPath := filepath.Join(dataDir, "agent.db")
	db, err := comms.OpenDB(dbPath)
	if err != nil {
		slog.Error("install: failed to open database", "path", dbPath, "error", err)
		return ExitError
	}
	defer db.Close()

	// Dial the Patch Manager gRPC server.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := dialServer(ctx, opts.server)
	if err != nil {
		slog.Error("install: failed to connect to server", "server", opts.server, "error", err)
		return ExitConnectionError
	}
	defer conn.Close()

	// Enroll the agent.
	enroller := &grpcEnroller{client: pb.NewAgentServiceClient(conn)}
	state := comms.NewAgentState(db)
	result, err := doEnrollment(ctx, enroller, state, opts.token)
	if err != nil {
		slog.Error("install: enrollment failed", "error", err)
		return ExitConnectionError
	}

	slog.Info("enrollment successful", "agent_id", result.AgentID, "negotiated_version", result.NegotiatedVersion)

	cfg := AgentConfig{
		ServerAddress: opts.server,
		DataDir:       dataDir,
		LogLevel:      "info",
		ScanInterval:  6 * time.Hour,
	}

	if err := writeAgentConfig(opts.configPath, cfg); err != nil {
		slog.Error("install: failed to write config", "error", err)
		return ExitError
	}

	slog.Info("agent config written", "path", opts.configPath)

	// Install and start the platform service so the agent runs as a daemon.
	if err := installAndStartService(); err != nil {
		slog.Warn("headless install: service installation failed (agent config written but service not running)", "error", err)
		// Non-fatal: config is written, user can manually run `service install` + `service start`.
	} else {
		slog.Info("headless install: service installed and running")
	}

	return ExitOK
}

// dialServer creates a gRPC client handle for the Patch Manager server.
// Connection is established lazily on first RPC call.
// TODO(PIQ-116): replace insecure credentials with mTLS.
func dialServer(_ context.Context, addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial server %s: %w", addr, err)
	}
	return conn, nil
}

// doEnrollment builds agent metadata and endpoint info, then calls comms.Enroll.
func doEnrollment(ctx context.Context, client comms.Enroller, state *comms.AgentState, token string) (comms.EnrollResult, error) {
	meta := comms.AgentMeta{
		AgentVersion:    "dev",
		ProtocolVersion: 1,
		Capabilities:    []string{"inventory"},
	}
	endpoint := sysinfo.BuildEndpointInfo(slog.Default())
	result, err := comms.Enroll(ctx, client, state, token, meta, endpoint)
	if err != nil {
		return comms.EnrollResult{}, fmt.Errorf("install enrollment: %w", err)
	}
	return result, nil
}

// grpcEnroller adapts pb.AgentServiceClient to the comms.Enroller interface.
type grpcEnroller struct {
	client pb.AgentServiceClient
}

func (e *grpcEnroller) Enroll(ctx context.Context, req *pb.EnrollRequest) (*pb.EnrollResponse, error) {
	return e.client.Enroll(ctx, req)
}

// runInstallTUI launches the interactive Bubble Tea install wizard.
func runInstallTUI(opts installOpts) int {
	model := newInstallModel(opts)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		slog.Error("install TUI error", "error", err)
		return ExitError
	}
	m, ok := finalModel.(installModel)
	if !ok {
		slog.Error("install: unexpected TUI model type", "type", fmt.Sprintf("%T", finalModel))
		return ExitError
	}
	if m.err != nil {
		return ExitConnectionError
	}
	return ExitOK
}
