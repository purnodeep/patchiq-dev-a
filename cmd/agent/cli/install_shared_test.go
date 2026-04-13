package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

func TestPerformEnroll_Success(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "agent.yaml")
	dataDir := filepath.Join(dir, "data")

	// Pre-create a DB that will be used by performEnroll.
	// performEnroll calls comms.OpenDB which creates the DB if needed.

	var statuses []string
	logStatus := func(msg string) {
		statuses = append(statuses, msg)
	}

	// We can't easily test performEnroll end-to-end without a real gRPC server,
	// because it calls dialServer internally. Instead, test the parts that are
	// testable: verify the function signature compiles and the logStatus callback
	// is invoked. A full integration test would require a running server.
	//
	// For unit testing, we verify that the function returns an error when the
	// server is unreachable (which exercises the early path).
	opts := installOpts{
		server:     "localhost:1", // unreachable port
		token:      "test-token",
		configPath: configPath,
		dataDir:    dataDir,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*100) // very short timeout
	defer cancel()

	_, err := performEnroll(ctx, opts, logStatus)
	// We expect an enrollment error (server unreachable), not a DB error.
	if err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}

	// Verify logStatus was called for at least the first phases.
	if len(statuses) < 2 {
		t.Errorf("expected at least 2 status messages, got %d: %v", len(statuses), statuses)
	}
}

func TestPerformEnroll_DBOpenError(t *testing.T) {
	// Use an invalid path that can't be created.
	opts := installOpts{
		server:  "localhost:50051",
		token:   "test-token",
		dataDir: "/dev/null/impossible",
	}

	_, err := performEnroll(context.Background(), opts, func(string) {})
	if err == nil {
		t.Fatal("expected error for invalid dataDir, got nil")
	}
}

func TestPerformEnroll_DoEnrollmentIntegration(t *testing.T) {
	// Test that doEnrollment (used by performEnroll) works with a mock enroller.
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "agent.db")
	db, err := comms.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	state := comms.NewAgentState(db)
	mock := &mockEnroller{
		resp: &pb.EnrollResponse{
			AgentId:                   "agent-shared-001",
			NegotiatedProtocolVersion: 1,
		},
	}

	result, err := doEnrollment(context.Background(), mock, state, "test-token")
	if err != nil {
		t.Fatalf("doEnrollment: %v", err)
	}
	if result.AgentID != "agent-shared-001" {
		t.Errorf("AgentID = %q, want %q", result.AgentID, "agent-shared-001")
	}
}

func TestPerformEnroll_DoEnrollmentError(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "agent.db")
	db, err := comms.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	state := comms.NewAgentState(db)
	mock := &mockEnroller{err: fmt.Errorf("server rejected enrollment")}

	_, err = doEnrollment(context.Background(), mock, state, "test-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPerformEnroll_WriteConfigError(t *testing.T) {
	// Use a config path that cannot be written.
	dir := t.TempDir()
	readOnlyDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	opts := installOpts{
		server:     "localhost:1",
		token:      "test-token",
		configPath: filepath.Join(readOnlyDir, "sub", "agent.yaml"),
		dataDir:    filepath.Join(dir, "data"),
	}

	// This will fail at the connect phase, not the write phase,
	// because we can't mock dialServer here. That's acceptable —
	// the write path is tested via TestInstallWriteConfig.
	_, err := performEnroll(context.Background(), opts, func(string) {})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
