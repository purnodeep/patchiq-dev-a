package store

import (
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

func TestStatusProvider_Defaults(t *testing.T) {
	db := openTestDB(t)
	state := comms.NewAgentState(db)
	sp := NewStatusProvider(state, "1.0.0", "localhost:50051", db)

	info := sp.Status()

	if info.AgentVersion != "1.0.0" {
		t.Fatalf("expected version 1.0.0, got %q", info.AgentVersion)
	}
	if info.ServerURL != "localhost:50051" {
		t.Fatalf("expected server url localhost:50051, got %q", info.ServerURL)
	}
	if info.EnrollmentStatus != "pending" {
		t.Fatalf("expected default enrollment_status 'pending', got %q", info.EnrollmentStatus)
	}
	if info.AgentID != "" {
		t.Fatalf("expected empty agent_id, got %q", info.AgentID)
	}
	if info.LastHeartbeat != nil {
		t.Fatalf("expected nil last_heartbeat, got %v", info.LastHeartbeat)
	}
	if info.UptimeSeconds < 0 {
		t.Fatalf("expected non-negative uptime, got %d", info.UptimeSeconds)
	}
}

func TestStatusProvider_SetLastHeartbeat(t *testing.T) {
	db := openTestDB(t)
	state := comms.NewAgentState(db)
	sp := NewStatusProvider(state, "1.0.0", "localhost:50051", db)

	now := time.Now().Truncate(time.Second)
	sp.SetLastHeartbeat(now)

	info := sp.Status()
	if info.LastHeartbeat == nil {
		t.Fatal("expected non-nil last_heartbeat after SetLastHeartbeat")
	}
	if !info.LastHeartbeat.Equal(now) {
		t.Fatalf("expected last_heartbeat %v, got %v", now, *info.LastHeartbeat)
	}
}

func TestStatusProviderCounts(t *testing.T) {
	db := openTestDB(t)

	// Insert test data
	_, _ = db.Exec(`INSERT INTO pending_patches (id, name, version, severity, status, queued_at)
		VALUES ('p1','pkg','1.0','high','queued','2026-03-10T10:00:00Z')`)
	_, _ = db.Exec(`INSERT INTO patch_history (id, patch_name, patch_version, action, result, completed_at)
		VALUES ('h1','pkg','1.0','install','success','2026-03-10T10:00:00Z')`)
	_, _ = db.Exec(`INSERT INTO patch_history (id, patch_name, patch_version, action, result, completed_at)
		VALUES ('h2','pkg','1.0','install','failed','2026-03-10T11:00:00Z')`)

	state := comms.NewAgentState(db)
	sp := NewStatusProvider(state, "1.0.0", "localhost:50051", db)
	info := sp.Status()

	if info.PendingPatchCount != 1 {
		t.Errorf("want 1 pending, got %d", info.PendingPatchCount)
	}
	if info.InstalledCount != 1 {
		t.Errorf("want 1 installed, got %d", info.InstalledCount)
	}
	if info.FailedCount != 1 {
		t.Errorf("want 1 failed, got %d", info.FailedCount)
	}
}
