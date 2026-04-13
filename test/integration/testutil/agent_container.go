//go:build integration

package testutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
)

// AgentContainerConfig holds the configuration needed to start an agent container.
type AgentContainerConfig struct {
	// ServerAddr is the gRPC server address the agent connects to.
	ServerAddr string
	// EnrollToken is the enrollment token for agent registration.
	EnrollToken string
	// TLSBundle is the TLS bundle for mTLS (currently unused by the agent,
	// which uses insecure gRPC credentials; retained for future mTLS support).
	TLSBundle *TLSBundle
	// ScanInterval overrides the agent scan interval. Zero means agent default.
	ScanInterval time.Duration
	// DataDir is the agent data directory inside the container.
	// Defaults to /var/lib/patchiq if empty.
	DataDir string
}

// AgentContainer wraps a running testcontainers container for the PatchIQ agent.
type AgentContainer struct {
	container testcontainers.Container
	t         *testing.T
}

// BuildAgentBinary cross-compiles the PatchIQ agent binary for linux/amd64 and
// returns the absolute path to the resulting binary. The binary is placed in a
// temporary directory that is cleaned up when the test ends.
func BuildAgentBinary(t *testing.T) string {
	t.Helper()

	repoRoot := findProjectRoot(t)
	outputDir := t.TempDir()
	binaryPath := filepath.Join(outputDir, "patchiq-agent")

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/agent/")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build agent binary: %v\noutput: %s", err, output)
	}

	return binaryPath
}

// StartAgentContainer builds the agent Docker image and starts a container with
// the given configuration. The container is automatically terminated when the
// test ends via t.Cleanup.
func StartAgentContainer(t *testing.T, cfg AgentContainerConfig) *AgentContainer {
	t.Helper()
	ctx := context.Background()

	agentBinary := BuildAgentBinary(t)

	dataDir := cfg.DataDir
	if dataDir == "" {
		dataDir = "/var/lib/patchiq"
	}

	// Build a temporary Docker context directory containing the Dockerfile
	// and the compiled agent binary.
	buildCtxDir := t.TempDir()

	dockerfileSrc := findDockerfileAgent(t)
	dockerfileContent, err := os.ReadFile(dockerfileSrc)
	if err != nil {
		t.Fatalf("read Dockerfile.agent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(buildCtxDir, "Dockerfile"), dockerfileContent, 0o644); err != nil {
		t.Fatalf("write Dockerfile to build context: %v", err)
	}

	binaryContent, err := os.ReadFile(agentBinary)
	if err != nil {
		t.Fatalf("read agent binary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(buildCtxDir, "patchiq-agent"), binaryContent, 0o755); err != nil {
		t.Fatalf("write agent binary to build context: %v", err)
	}

	env := map[string]string{
		"PATCHIQ_AGENT_SERVER_ADDRESS":   cfg.ServerAddr,
		"PATCHIQ_AGENT_ENROLLMENT_TOKEN": cfg.EnrollToken,
		"PATCHIQ_AGENT_DATA_DIR":         dataDir,
		"PATCHIQ_AGENT_LOG_LEVEL":        "debug",
	}

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    buildCtxDir,
			Dockerfile: "Dockerfile",
		},
		Env:         env,
		NetworkMode: "host",
		WaitingFor:  wait.ForLog("agent running").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start agent container: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("terminate agent container: %v", err)
		}
	})

	return &AgentContainer{
		container: container,
		t:         t,
	}
}

// Exec runs a command inside the agent container and returns the combined
// stdout/stderr output. Uses tcexec.Multiplexed() to strip Docker stream
// framing headers from the output.
func (ac *AgentContainer) Exec(ctx context.Context, cmd []string) (string, error) {
	exitCode, reader, err := ac.container.Exec(ctx, cmd, tcexec.Multiplexed())
	if err != nil {
		return "", fmt.Errorf("exec %v: %w", cmd, err)
	}

	buf := new(strings.Builder)
	if reader != nil {
		b := make([]byte, 4096)
		for {
			n, readErr := reader.Read(b)
			if n > 0 {
				buf.Write(b[:n])
			}
			if readErr != nil {
				break
			}
		}
	}

	output := buf.String()
	if exitCode != 0 {
		return output, fmt.Errorf("exec %v: exit code %d: %s", cmd, exitCode, output)
	}
	return output, nil
}

// OutboxCount returns the number of pending items in the agent outbox table
// by querying the SQLite database inside the container.
func (ac *AgentContainer) OutboxCount(ctx context.Context) (int, error) {
	output, err := ac.Exec(ctx, []string{
		"sqlite3", "/var/lib/patchiq/agent.db",
		"SELECT COUNT(*) FROM outbox WHERE status = 'pending';",
	})
	if err != nil {
		return 0, fmt.Errorf("outbox count query: %w", err)
	}

	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return 0, fmt.Errorf("parse outbox count %q: %w", strings.TrimSpace(output), err)
	}
	return count, nil
}

// IsPackageInstalled checks whether a Debian package is installed inside the
// agent container using dpkg.
func (ac *AgentContainer) IsPackageInstalled(ctx context.Context, pkg string) (bool, error) {
	_, err := ac.Exec(ctx, []string{"dpkg", "-s", pkg})
	if err != nil {
		// dpkg -s returns non-zero if the package is not installed.
		return false, nil
	}
	return true, nil
}

// Logs returns the container logs (stdout + stderr).
func (ac *AgentContainer) Logs(ctx context.Context) (string, error) {
	reader, err := ac.container.Logs(ctx)
	if err != nil {
		return "", fmt.Errorf("get agent container logs: %w", err)
	}
	defer reader.Close()

	buf := new(strings.Builder)
	b := make([]byte, 4096)
	for {
		n, readErr := reader.Read(b)
		if n > 0 {
			buf.Write(b[:n])
		}
		if readErr != nil {
			break
		}
	}
	return buf.String(), nil
}

// findProjectRoot locates the repository root by walking up from this source
// file's directory until go.mod is found.
func findProjectRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed: cannot determine source file path")
	}

	// filename = .../test/integration/testutil/agent_container.go
	// Up 3 levels = repo root
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")

	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err != nil {
		t.Fatalf("project root not found at %s (no go.mod): %v", repoRoot, err)
	}
	return repoRoot
}

// findDockerfileAgent returns the absolute path to Dockerfile.agent, which
// lives alongside this source file.
func findDockerfileAgent(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed: cannot determine source file path")
	}

	dockerfilePath := filepath.Join(filepath.Dir(filename), "Dockerfile.agent")
	if _, err := os.Stat(dockerfilePath); err != nil {
		t.Fatalf("Dockerfile.agent not found at %s: %v", dockerfilePath, err)
	}
	return dockerfilePath
}
