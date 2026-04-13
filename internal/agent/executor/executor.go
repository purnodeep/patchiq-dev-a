package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent"
	"google.golang.org/protobuf/proto"
)

const (
	defaultMaxOutputBytes = 1024 * 1024 // 1MB
	defaultTimeoutSeconds = 300         // 5 minutes
)

// Module implements the executor agent module for run_script commands.
type Module struct {
	logger *slog.Logger
}

// New creates a new executor module.
func New() *Module {
	return &Module{}
}

// newTestModule creates a module with a real logger for testing.
func newTestModule() *Module {
	m := New()
	m.logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	return m
}

func (m *Module) Name() string                   { return "executor" }
func (m *Module) Version() string                { return "0.1.0" }
func (m *Module) Capabilities() []string         { return []string{"script_execution"} }
func (m *Module) SupportedCommands() []string    { return []string{"run_script"} }
func (m *Module) CollectInterval() time.Duration { return 0 }

func (m *Module) Init(_ context.Context, deps agent.ModuleDeps) error {
	m.logger = deps.Logger
	if m.logger == nil {
		m.logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return nil
}

func (m *Module) Start(_ context.Context) error       { return nil }
func (m *Module) Stop(_ context.Context) error        { return nil }
func (m *Module) HealthCheck(_ context.Context) error { return nil }

func (m *Module) Collect(_ context.Context) ([]agent.OutboxItem, error) { return nil, nil }

func (m *Module) HandleCommand(ctx context.Context, cmd agent.Command) (agent.Result, error) {
	switch cmd.Type {
	case "run_script":
		return m.handleRunScript(ctx, cmd)
	default:
		return agent.Result{}, fmt.Errorf("executor: unsupported command type %q", cmd.Type)
	}
}

func (m *Module) handleRunScript(ctx context.Context, cmd agent.Command) (agent.Result, error) {
	var payload pb.RunScriptPayload
	if err := proto.Unmarshal(cmd.Payload, &payload); err != nil {
		return agent.Result{ErrorMessage: fmt.Sprintf("executor: unmarshal payload: %v", err)}, nil
	}

	if payload.InlineScript == "" {
		return agent.Result{ErrorMessage: "executor: inline_script is empty"}, nil
	}

	// Resolve timeout.
	timeoutSec := payload.TimeoutSeconds
	if timeoutSec <= 0 {
		timeoutSec = defaultTimeoutSeconds
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Resolve max output bytes.
	maxOutput := int(payload.MaxOutputBytes)
	if maxOutput <= 0 {
		maxOutput = defaultMaxOutputBytes
	}

	// Resolve interpreter.
	interpreter, interpreterArgs := resolveInterpreter(payload.ScriptType)

	// Write script to temp file.
	tmpFile, err := os.CreateTemp("", "patchiq-script-*")
	if err != nil {
		return agent.Result{ErrorMessage: fmt.Sprintf("executor: create temp file: %v", err)}, nil
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(payload.InlineScript); err != nil {
		tmpFile.Close()
		return agent.Result{ErrorMessage: fmt.Sprintf("executor: write script: %v", err)}, nil
	}
	tmpFile.Close()

	// Make executable for shell scripts.
	if err := os.Chmod(tmpPath, 0700); err != nil {
		return agent.Result{ErrorMessage: fmt.Sprintf("executor: chmod temp file: %v", err)}, nil
	}

	args := append(interpreterArgs, tmpPath)
	execCmd := exec.CommandContext(ctx, interpreter, args...)

	// Set up platform-specific process group and kill behavior so we can
	// kill all descendants (not just the direct child) on timeout.
	setProcGroup(execCmd)
	execCmd.Cancel = cancelFunc(execCmd)
	execCmd.WaitDelay = 2 * time.Second

	// Inject environment variables.
	if len(payload.Env) > 0 {
		execCmd.Env = os.Environ()
		for k, v := range payload.Env {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	runErr := execCmd.Run()
	timedOut := ctx.Err() == context.DeadlineExceeded

	output := &pb.RunScriptOutput{}

	if timedOut {
		output.TimedOut = true
		output.Stdout = truncate(stdout.String(), maxOutput)
		output.Stderr = truncate(stderr.String(), maxOutput)

		outputBytes, marshalErr := proto.Marshal(output)
		if marshalErr != nil {
			return agent.Result{}, fmt.Errorf("executor: marshal output: %w", marshalErr)
		}
		return agent.Result{
			Output:       outputBytes,
			ErrorMessage: fmt.Sprintf("executor: script timed out after %ds", timeoutSec),
		}, nil
	}

	// Extract exit code.
	exitCode := 0
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		} else {
			// Non-exit error (e.g., binary not found).
			return agent.Result{ErrorMessage: fmt.Sprintf("executor: run script: %v", runErr)}, nil
		}
	}

	output.Stdout = truncate(stdout.String(), maxOutput)
	output.Stderr = truncate(stderr.String(), maxOutput)
	output.ExitCode = int32(exitCode)

	outputBytes, marshalErr := proto.Marshal(output)
	if marshalErr != nil {
		return agent.Result{}, fmt.Errorf("executor: marshal output: %w", marshalErr)
	}

	result := agent.Result{Output: outputBytes}
	if exitCode != 0 {
		result.ErrorMessage = fmt.Sprintf("executor: script exited with code %d", exitCode)
	}

	m.logger.InfoContext(ctx, "executor: script completed",
		"command_id", cmd.ID,
		"exit_code", exitCode,
		"stdout_len", len(output.Stdout),
		"stderr_len", len(output.Stderr),
	)

	return result, nil
}

// resolveInterpreter returns the interpreter binary and arguments for the given script type.
func resolveInterpreter(st pb.ScriptType) (string, []string) {
	switch st {
	case pb.ScriptType_SCRIPT_TYPE_POWERSHELL:
		return "powershell", []string{"-File"}
	case pb.ScriptType_SCRIPT_TYPE_PYTHON:
		return "python3", nil
	default:
		// SHELL or UNSPECIFIED
		if runtime.GOOS == "windows" {
			return "powershell", []string{"-File"}
		}
		return "sh", nil
	}
}

// truncate trims s to at most maxBytes bytes.
func truncate(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes]
}

// Verify Module satisfies the interface at compile time.
var _ agent.Module = (*Module)(nil)
