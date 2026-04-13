package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent"
	"google.golang.org/protobuf/proto"
)

func TestModuleMetadata(t *testing.T) {
	m := New()
	if m.Name() != "executor" {
		t.Errorf("Name() = %q, want %q", m.Name(), "executor")
	}
	if m.Version() != "0.1.0" {
		t.Errorf("Version() = %q, want %q", m.Version(), "0.1.0")
	}
	wantCmds := []string{"run_script"}
	gotCmds := m.SupportedCommands()
	if len(gotCmds) != len(wantCmds) {
		t.Fatalf("SupportedCommands() = %v, want %v", gotCmds, wantCmds)
	}
	for i, want := range wantCmds {
		if gotCmds[i] != want {
			t.Errorf("SupportedCommands()[%d] = %q, want %q", i, gotCmds[i], want)
		}
	}
	wantCaps := []string{"script_execution"}
	gotCaps := m.Capabilities()
	if len(gotCaps) != len(wantCaps) {
		t.Fatalf("Capabilities() = %v, want %v", gotCaps, wantCaps)
	}
	for i, want := range wantCaps {
		if gotCaps[i] != want {
			t.Errorf("Capabilities()[%d] = %q, want %q", i, gotCaps[i], want)
		}
	}
	if m.CollectInterval() != 0 {
		t.Errorf("CollectInterval() = %v, want 0", m.CollectInterval())
	}
}

func TestRunShellScript(t *testing.T) {
	m := newTestModule()

	payload := &pb.RunScriptPayload{
		InlineScript: "echo hello",
		ScriptType:   pb.ScriptType_SCRIPT_TYPE_SHELL,
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID:      "cmd-1",
		Type:    "run_script",
		Payload: payloadBytes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage != "" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}

	var output pb.RunScriptOutput
	if err := proto.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.Stdout != "hello\n" {
		t.Errorf("stdout = %q, want %q", output.Stdout, "hello\n")
	}
	if output.ExitCode != 0 {
		t.Errorf("exit_code = %d, want 0", output.ExitCode)
	}
	if output.TimedOut {
		t.Error("expected timed_out = false")
	}
}

func TestRunScriptFailure(t *testing.T) {
	m := newTestModule()

	payload := &pb.RunScriptPayload{
		InlineScript: "exit 42",
		ScriptType:   pb.ScriptType_SCRIPT_TYPE_SHELL,
	}
	payloadBytes, _ := proto.Marshal(payload)

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID:      "cmd-fail",
		Type:    "run_script",
		Payload: payloadBytes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message for non-zero exit code")
	}

	var output pb.RunScriptOutput
	if err := proto.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.ExitCode != 42 {
		t.Errorf("exit_code = %d, want 42", output.ExitCode)
	}
}

func TestRunScriptTimeout(t *testing.T) {
	m := newTestModule()

	payload := &pb.RunScriptPayload{
		InlineScript:   "sleep 60",
		ScriptType:     pb.ScriptType_SCRIPT_TYPE_SHELL,
		TimeoutSeconds: 1,
	}
	payloadBytes, _ := proto.Marshal(payload)

	start := time.Now()
	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID:      "cmd-timeout",
		Type:    "run_script",
		Payload: payloadBytes,
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message for timeout")
	}

	var output pb.RunScriptOutput
	if err := proto.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if !output.TimedOut {
		t.Error("expected timed_out = true")
	}
	if elapsed > 5*time.Second {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

func TestRunScriptOutputTruncation(t *testing.T) {
	m := newTestModule()

	// Generate ~2KB of output, truncate to 100 bytes
	payload := &pb.RunScriptPayload{
		InlineScript:   `for i in $(seq 1 200); do echo "line number $i of output data padding"; done`,
		ScriptType:     pb.ScriptType_SCRIPT_TYPE_SHELL,
		MaxOutputBytes: 100,
	}
	payloadBytes, _ := proto.Marshal(payload)

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID:      "cmd-trunc",
		Type:    "run_script",
		Payload: payloadBytes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output pb.RunScriptOutput
	if err := proto.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Stdout) > 100 {
		t.Errorf("stdout length = %d, want <= 100", len(output.Stdout))
	}
	if len(output.Stdout) == 0 {
		t.Error("expected non-empty stdout after truncation")
	}
}

func TestRunScriptEmptyPayload(t *testing.T) {
	m := newTestModule()

	payload := &pb.RunScriptPayload{
		InlineScript: "",
		ScriptType:   pb.ScriptType_SCRIPT_TYPE_SHELL,
	}
	payloadBytes, _ := proto.Marshal(payload)

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID:      "cmd-empty",
		Type:    "run_script",
		Payload: payloadBytes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message for empty script")
	}
}

func TestRunScriptWithEnvVars(t *testing.T) {
	m := newTestModule()

	payload := &pb.RunScriptPayload{
		InlineScript: "echo $MY_VAR",
		ScriptType:   pb.ScriptType_SCRIPT_TYPE_SHELL,
		Env: map[string]string{
			"MY_VAR": "hello_from_env",
		},
	}
	payloadBytes, _ := proto.Marshal(payload)

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID:      "cmd-env",
		Type:    "run_script",
		Payload: payloadBytes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ErrorMessage != "" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}

	var output pb.RunScriptOutput
	if err := proto.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if !strings.Contains(output.Stdout, "hello_from_env") {
		t.Errorf("stdout = %q, want to contain %q", output.Stdout, "hello_from_env")
	}
}

func TestRunScriptInvalidPayload(t *testing.T) {
	m := newTestModule()

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID:      "cmd-invalid",
		Type:    "run_script",
		Payload: []byte("not-a-protobuf-message"),
	})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message for invalid payload")
	}
}
