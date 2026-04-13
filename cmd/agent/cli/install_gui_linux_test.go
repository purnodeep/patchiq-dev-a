//go:build linux

package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadServerTxt(t *testing.T) {
	tests := []struct {
		name    string
		content *string // nil means don't create the file
		want    string
	}{
		{
			name:    "missing file",
			content: nil,
			want:    "",
		},
		{
			name:    "empty file",
			content: ptr(""),
			want:    "",
		},
		{
			name:    "whitespace only",
			content: ptr("   \n\t  "),
			want:    "",
		},
		{
			name:    "valid URL with trailing newline",
			content: ptr("pm.example.com:50051\n"),
			want:    "pm.example.com:50051",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			exePath := filepath.Join(dir, "patchiq-agent")
			if tt.content != nil {
				if err := os.WriteFile(filepath.Join(dir, "server.txt"), []byte(*tt.content), 0o644); err != nil {
					t.Fatalf("write server.txt: %v", err)
				}
			}
			got := readServerTxt(exePath)
			if got != tt.want {
				t.Errorf("readServerTxt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func ptr(s string) *string { return &s }

// fakeZenityRunner records calls and returns scripted responses.
type fakeZenityRunner struct {
	// responses maps the first arg (e.g. "--entry", "--info") to a sequence of responses.
	responses map[string][]fakeResponse
	calls     [][]string
}

type fakeResponse struct {
	stdout string
	err    error
}

func (f *fakeZenityRunner) Run(args ...string) (string, error) {
	f.calls = append(f.calls, args)
	key := ""
	if len(args) > 0 {
		key = args[0]
	}
	resps := f.responses[key]
	if len(resps) == 0 {
		return "", nil
	}
	resp := resps[0]
	f.responses[key] = resps[1:]
	return resp.stdout, resp.err
}

// nopWriteCloser wraps io.Discard to satisfy io.WriteCloser.
type nopWriteCloser struct{}

func (nopWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (nopWriteCloser) Close() error                { return nil }

func (f *fakeZenityRunner) Start(args ...string) (io.WriteCloser, func(), error) {
	f.calls = append(f.calls, args)
	return nopWriteCloser{}, func() {}, nil
}

func newFakeZenityWithWriteCloser() *fakeZenityRunner {
	return &fakeZenityRunner{
		responses: make(map[string][]fakeResponse),
	}
}

func TestGuiInstallerFlow_Success(t *testing.T) {
	fake := newFakeZenityWithWriteCloser()
	// Server entry dialog → user accepts default
	fake.responses["--entry"] = []fakeResponse{
		{stdout: "pm.example.com:50051"},
		{stdout: "test-token-123"},
	}
	// Info dialog for success
	fake.responses["--info"] = []fakeResponse{{stdout: ""}}

	installer := guiInstaller{
		runner: fake,
		enroll: func(_ context.Context, opts installOpts, logStatus func(string)) (string, error) {
			logStatus("testing...")
			if opts.server != "pm.example.com:50051" {
				t.Errorf("unexpected server: %s", opts.server)
			}
			if opts.token != "test-token-123" {
				t.Errorf("unexpected token: %s", opts.token)
			}
			return "agent-001", nil
		},
	}

	code := installer.run()
	if code != 0 {
		t.Errorf("run() = %d, want 0", code)
	}

	// Verify success dialog was shown.
	var foundInfo bool
	for _, call := range fake.calls {
		if len(call) > 0 && call[0] == "--info" {
			foundInfo = true
			joined := strings.Join(call, " ")
			if !strings.Contains(joined, "enrolled") {
				t.Errorf("success dialog should mention 'enrolled', got: %s", joined)
			}
		}
	}
	if !foundInfo {
		t.Error("expected info dialog to be shown on success")
	}
}

func TestGuiInstallerFlow_UserCancels(t *testing.T) {
	fake := newFakeZenityWithWriteCloser()
	// User cancels the server entry dialog.
	fake.responses["--entry"] = []fakeResponse{
		{stdout: "", err: fmt.Errorf("exit status 1")},
	}

	enrollCalled := false
	installer := guiInstaller{
		runner: fake,
		enroll: func(_ context.Context, _ installOpts, _ func(string)) (string, error) {
			enrollCalled = true
			return "", nil
		},
	}

	code := installer.run()
	if code != 1 {
		t.Errorf("run() = %d, want 1", code)
	}
	if enrollCalled {
		t.Error("enroll should not be called when user cancels")
	}
}

func TestGuiInstallerFlow_EnrollErrorRetries(t *testing.T) {
	fake := newFakeZenityWithWriteCloser()
	// 3 rounds of prompts (2 failures + 1 success).
	fake.responses["--entry"] = []fakeResponse{
		{stdout: "server:50051"}, {stdout: "token1"},
		{stdout: "server:50051"}, {stdout: "token2"},
		{stdout: "server:50051"}, {stdout: "token3"},
	}
	fake.responses["--error"] = []fakeResponse{
		{stdout: ""},
		{stdout: ""},
	}
	fake.responses["--info"] = []fakeResponse{{stdout: ""}}

	attempt := 0
	installer := guiInstaller{
		runner: fake,
		enroll: func(_ context.Context, _ installOpts, _ func(string)) (string, error) {
			attempt++
			if attempt < 3 {
				return "", fmt.Errorf("connection refused (attempt %d)", attempt)
			}
			return "agent-001", nil
		},
	}

	code := installer.run()
	if code != 0 {
		t.Errorf("run() = %d, want 0", code)
	}
	if attempt != 3 {
		t.Errorf("attempt = %d, want 3", attempt)
	}
}

func TestGuiInstallerFlow_ThreeFailuresExits(t *testing.T) {
	fake := newFakeZenityWithWriteCloser()
	// 3 rounds of prompts, all fail.
	fake.responses["--entry"] = []fakeResponse{
		{stdout: "server:50051"}, {stdout: "token1"},
		{stdout: "server:50051"}, {stdout: "token2"},
		{stdout: "server:50051"}, {stdout: "token3"},
	}
	fake.responses["--error"] = []fakeResponse{
		{stdout: ""}, {stdout: ""}, {stdout: ""},
	}

	installer := guiInstaller{
		runner: fake,
		enroll: func(_ context.Context, _ installOpts, _ func(string)) (string, error) {
			return "", fmt.Errorf("connection refused")
		},
	}

	code := installer.run()
	if code != 1 {
		t.Errorf("run() = %d, want 1", code)
	}
}
