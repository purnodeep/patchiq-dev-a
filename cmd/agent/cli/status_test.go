package cli

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

func TestStatusParseFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    statusOpts
		wantErr bool
	}{
		{
			name: "defaults",
			args: []string{},
			want: statusOpts{},
		},
		{
			name: "watch flag",
			args: []string{"--watch"},
			want: statusOpts{watch: true},
		},
		{
			name: "json flag",
			args: []string{"--json"},
			want: statusOpts{jsonOutput: true},
		},
		{
			name: "config and data-dir",
			args: []string{"--config", "/etc/patchiq.yaml", "--data-dir", "/tmp/data"},
			want: statusOpts{configPath: "/etc/patchiq.yaml", dataDir: "/tmp/data"},
		},
		{
			name: "all flags",
			args: []string{"--watch", "--json", "--config", "c.yaml", "--data-dir", "/d"},
			want: statusOpts{watch: true, jsonOutput: true, configPath: "c.yaml", dataDir: "/d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStatusFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseStatusFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseStatusFlags() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCollectStatusInfo(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "agent.db")
	db, err := comms.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	state := comms.NewAgentState(db)
	outbox := comms.NewOutbox(db)

	// Set state values.
	if err := state.Set(ctx, "agent_id", "agt-123"); err != nil {
		t.Fatalf("Set agent_id: %v", err)
	}
	if err := state.Set(ctx, "last_heartbeat", "2026-03-05T10:00:00Z"); err != nil {
		t.Fatalf("Set last_heartbeat: %v", err)
	}
	if err := state.Set(ctx, "last_scan", "2026-03-05T09:45:00Z"); err != nil {
		t.Fatalf("Set last_scan: %v", err)
	}

	// Add a pending outbox item.
	if _, err := outbox.Add(ctx, "inventory", []byte(`{}`)); err != nil {
		t.Fatalf("Outbox.Add: %v", err)
	}

	info, err := collectStatusInfo(ctx, state, outbox)
	if err != nil {
		t.Fatalf("collectStatusInfo: %v", err)
	}

	if info.AgentID != "agt-123" {
		t.Errorf("AgentID = %q, want %q", info.AgentID, "agt-123")
	}
	if info.Connection != "connected" {
		t.Errorf("Connection = %q, want %q", info.Connection, "connected")
	}
	if info.LastHeartbeat != "2026-03-05T10:00:00Z" {
		t.Errorf("LastHeartbeat = %q, want %q", info.LastHeartbeat, "2026-03-05T10:00:00Z")
	}
	if info.LastScan != "2026-03-05T09:45:00Z" {
		t.Errorf("LastScan = %q, want %q", info.LastScan, "2026-03-05T09:45:00Z")
	}
	if info.QueueDepth != 1 {
		t.Errorf("QueueDepth = %d, want %d", info.QueueDepth, 1)
	}

	// Test disconnected state: fresh DB with no state.
	db2Path := filepath.Join(dir, "agent2.db")
	db2, err := comms.OpenDB(db2Path)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db2.Close()

	state2 := comms.NewAgentState(db2)
	outbox2 := comms.NewOutbox(db2)

	info2, err := collectStatusInfo(ctx, state2, outbox2)
	if err != nil {
		t.Fatalf("collectStatusInfo (empty): %v", err)
	}
	if info2.AgentID != "(not enrolled)" {
		t.Errorf("AgentID = %q, want %q", info2.AgentID, "(not enrolled)")
	}
	if info2.Connection != "disconnected" {
		t.Errorf("Connection = %q, want %q", info2.Connection, "disconnected")
	}
	if info2.LastHeartbeat != "(never)" {
		t.Errorf("LastHeartbeat = %q, want %q", info2.LastHeartbeat, "(never)")
	}
	if info2.LastScan != "(never)" {
		t.Errorf("LastScan = %q, want %q", info2.LastScan, "(never)")
	}
}

func TestStatusInfoJSON(t *testing.T) {
	info := StatusInfo{
		AgentID:       "agt-456",
		Connection:    "connected",
		LastHeartbeat: "2026-03-05T10:00:00Z",
		LastScan:      "2026-03-05T09:00:00Z",
		QueueDepth:    3,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	expectedKeys := []string{"agent_id", "connection", "last_heartbeat", "last_scan", "queue_depth"}
	for _, key := range expectedKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("missing JSON key %q", key)
		}
	}

	if m["agent_id"] != "agt-456" {
		t.Errorf("agent_id = %v, want %q", m["agent_id"], "agt-456")
	}
	// queue_depth is float64 from JSON unmarshaling.
	if m["queue_depth"] != float64(3) {
		t.Errorf("queue_depth = %v, want %v", m["queue_depth"], 3)
	}
}
