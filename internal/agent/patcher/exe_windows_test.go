//go:build windows

package patcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tempEXE creates a temporary .exe file and returns its path.
func tempEXE(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "setup.exe")
	if err := os.WriteFile(path, []byte("MZ-fake-binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEXEInstaller_Name(t *testing.T) {
	inst := &exeInstaller{}
	assert.Equal(t, "exe", inst.Name())
}

func TestEXEInstaller_Install_Success(t *testing.T) {
	exePath := tempEXE(t)
	inst := &exeInstaller{
		executor: &mockCmdExecutor{
			result: ExecResult{ExitCode: 0, Stdout: []byte("ok")},
		},
		logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	result, err := inst.Install(context.Background(), PatchTarget{
		Name:    exePath,
		Version: "122.0",
	}, false)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.False(t, result.RebootRequired)
}

func TestEXEInstaller_Install_RebootRequired(t *testing.T) {
	exePath := tempEXE(t)
	inst := &exeInstaller{
		executor: &mockCmdExecutor{
			result: ExecResult{ExitCode: 3010},
		},
		logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	result, err := inst.Install(context.Background(), PatchTarget{Name: exePath}, false)
	require.NoError(t, err)
	assert.True(t, result.RebootRequired)
	assert.Equal(t, 3010, result.ExitCode)
}

func TestEXEInstaller_Install_RebootInitiated(t *testing.T) {
	exePath := tempEXE(t)
	inst := &exeInstaller{
		executor: &mockCmdExecutor{
			result: ExecResult{ExitCode: 1641},
		},
		logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	result, err := inst.Install(context.Background(), PatchTarget{Name: exePath}, false)
	require.NoError(t, err)
	assert.True(t, result.RebootRequired)
	assert.Equal(t, 1641, result.ExitCode)
}

func TestEXEInstaller_Install_WithSilentArgs(t *testing.T) {
	exePath := tempEXE(t)
	executor := &recordingCmdExecutor{}
	inst := &exeInstaller{
		executor:   executor,
		silentArgs: "/S /D=C:\\Program Files\\App",
		logger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	_, err := inst.Install(context.Background(), PatchTarget{Name: exePath}, false)
	require.NoError(t, err)
	assert.Contains(t, executor.lastArgs, "/S")
}

func TestEXEInstaller_Install_NoSilentArgs(t *testing.T) {
	exePath := tempEXE(t)
	executor := &recordingCmdExecutor{}
	inst := &exeInstaller{
		executor: executor,
		logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	_, err := inst.Install(context.Background(), PatchTarget{Name: exePath}, false)
	require.NoError(t, err)
	assert.Empty(t, executor.lastArgs)
}

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"/S", []string{"/S"}},
		{"/S /quiet", []string{"/S", "/quiet"}},
		{`/D="C:\Program Files\App"`, []string{`/D=C:\Program Files\App`}},
		{"", nil},
	}
	for _, tt := range tests {
		got := splitArgs(tt.input)
		assert.Equal(t, tt.want, got, "splitArgs(%q)", tt.input)
	}
}

func TestEXEInstaller_Install_FileNotFound(t *testing.T) {
	inst := &exeInstaller{
		executor: &mockCmdExecutor{},
		logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
	_, err := inst.Install(context.Background(), PatchTarget{Name: "/nonexistent/file.exe"}, false)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "binary not found") {
		t.Errorf("error = %v, want containing 'binary not found'", err)
	}
}

type mockCmdExecutor struct {
	result ExecResult
	err    error
}

func (m *mockCmdExecutor) Execute(_ context.Context, _ string, _ ...string) (ExecResult, error) {
	return m.result, m.err
}

type recordingCmdExecutor struct {
	lastBinary string
	lastArgs   []string
}

func (r *recordingCmdExecutor) Execute(_ context.Context, binary string, args ...string) (ExecResult, error) {
	r.lastBinary = binary
	r.lastArgs = args
	return ExecResult{ExitCode: 0}, nil
}
