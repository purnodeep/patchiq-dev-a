package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
	yamlv3 "gopkg.in/yaml.v3"
)

type mockEnroller struct {
	resp *pb.EnrollResponse
	err  error
}

func (m *mockEnroller) Enroll(_ context.Context, _ *pb.EnrollRequest) (*pb.EnrollResponse, error) {
	return m.resp, m.err
}

func TestInstallParseFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    installOpts
		wantErr bool
	}{
		{
			name: "headless with all flags",
			args: []string{
				"--server", "pm.example.com:9090",
				"--token", "abc123",
				"--non-interactive",
				"--config", "/tmp/agent.yaml",
				"--data-dir", "/tmp/patchiq-data",
			},
			want: installOpts{
				server:         "pm.example.com:9090",
				token:          "abc123",
				nonInteractive: true,
				configPath:     "/tmp/agent.yaml",
				dataDir:        "/tmp/patchiq-data",
			},
		},
		{
			name: "custom config path only",
			args: []string{"--config", "/opt/patchiq/agent.yaml"},
			want: installOpts{
				configPath: "/opt/patchiq/agent.yaml",
			},
		},
		{
			name: "defaults applied",
			args: []string{},
			want: installOpts{
				configPath: "/etc/patchiq/agent.yaml",
			},
		},
		{
			name:    "unknown flag",
			args:    []string{"--bogus"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInstallFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseInstallFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("parseInstallFlags() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestInstallWriteConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "nested", "agent.yaml")

	cfg := AgentConfig{
		ServerAddress: "pm.example.com:9090",
		DataDir:       "/var/lib/patchiq",
		LogLevel:      "debug",
		ScanInterval:  10 * time.Minute,
	}

	if err := writeAgentConfig(cfgPath, cfg); err != nil {
		t.Fatalf("writeAgentConfig() error = %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read written config: %v", err)
	}

	var got AgentConfig
	if err := yamlv3.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal written config: %v", err)
	}

	if got != cfg {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, cfg)
	}
}

func TestInstallHeadless_MissingServer(t *testing.T) {
	orig := DefaultServerAddress
	t.Cleanup(func() { DefaultServerAddress = orig })
	DefaultServerAddress = ""

	opts := installOpts{
		nonInteractive: true,
		token:          "abc123",
	}
	_, err := validateInstallOpts(opts)
	if err == nil {
		t.Fatal("expected error for missing server")
	}
	if got := err.Error(); !strings.Contains(got, "server") {
		t.Errorf("error should mention 'server', got: %s", got)
	}
}

func TestDoEnrollment_Success(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "agent.db")
	db, err := comms.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	state := comms.NewAgentState(db)
	mock := &mockEnroller{
		resp: &pb.EnrollResponse{
			AgentId:                   "agent-001",
			NegotiatedProtocolVersion: 1,
		},
	}

	result, err := doEnrollment(context.Background(), mock, state, "test-token")
	if err != nil {
		t.Fatalf("doEnrollment() error = %v", err)
	}
	if result.AgentID != "agent-001" {
		t.Errorf("AgentID = %q, want %q", result.AgentID, "agent-001")
	}
}

func TestDoEnrollment_ConnectionError(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "agent.db")
	db, err := comms.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	state := comms.NewAgentState(db)
	mock := &mockEnroller{err: fmt.Errorf("connection refused")}

	_, err = doEnrollment(context.Background(), mock, state, "test-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error should contain 'connection refused', got: %s", err.Error())
	}
}

func TestInstallHeadless_MissingToken(t *testing.T) {
	opts := installOpts{
		nonInteractive: true,
		server:         "pm.example.com:9090",
	}
	_, err := validateInstallOpts(opts)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if got := err.Error(); !strings.Contains(got, "token") {
		t.Errorf("error should mention 'token', got: %s", got)
	}
}

func TestValidateInstallOpts_UsesDefaultServerAddress(t *testing.T) {
	// Save & restore the package-level baked default.
	orig := DefaultServerAddress
	t.Cleanup(func() { DefaultServerAddress = orig })

	tests := []struct {
		name        string
		baked       string
		serverFlag  string
		envServer   string
		nonInteract bool
		token       string
		wantErr     bool
		wantServer  string
	}{
		{
			name:        "headless: baked default fills in missing --server",
			baked:       "patchiq.example.com:3013",
			serverFlag:  "",
			nonInteract: true,
			token:       "tok123",
			wantErr:     false,
			wantServer:  "patchiq.example.com:3013",
		},
		{
			name:        "headless: explicit --server overrides baked",
			baked:       "patchiq.example.com:3013",
			serverFlag:  "other.example:50051",
			nonInteract: true,
			token:       "tok123",
			wantErr:     false,
			wantServer:  "other.example:50051",
		},
		{
			name:        "headless: no flag, no env, no baked → error",
			baked:       "",
			serverFlag:  "",
			nonInteract: true,
			token:       "tok123",
			wantErr:     true,
		},
		{
			name:        "headless: env var fills in missing --server",
			baked:       "",
			envServer:   "env.example:50051",
			serverFlag:  "",
			nonInteract: true,
			token:       "tok123",
			wantErr:     false,
			wantServer:  "env.example:50051",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DefaultServerAddress = tt.baked
			if tt.envServer != "" {
				t.Setenv("PATCHIQ_AGENT_SERVER_ADDRESS", tt.envServer)
			}
			opts := installOpts{
				server:         tt.serverFlag,
				token:          tt.token,
				nonInteractive: tt.nonInteract,
			}
			resolved, err := validateInstallOpts(opts)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (resolved=%+v)", resolved)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolved.server != tt.wantServer {
				t.Errorf("server: got %q, want %q", resolved.server, tt.wantServer)
			}
		})
	}
}
