package system

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent"
	"google.golang.org/protobuf/proto"

	_ "modernc.org/sqlite"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS agent_state (key TEXT PRIMARY KEY, value TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func initModule(t *testing.T, m *Module, db *sql.DB) {
	t.Helper()
	err := m.Init(context.Background(), agent.ModuleDeps{
		Logger:         slog.Default(),
		LocalDB:        db,
		ConfigProvider: agent.NoopConfigProvider{},
		EventEmitter:   agent.NoopEventEmitter{},
		FileCache:      agent.NoopFileCache{},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestModuleMetadata(t *testing.T) {
	m := New()
	if m.Name() != "system" {
		t.Errorf("Name() = %q, want %q", m.Name(), "system")
	}
	if m.Version() != "0.1.0" {
		t.Errorf("Version() = %q, want %q", m.Version(), "0.1.0")
	}
	cmds := m.SupportedCommands()
	want := map[string]bool{"reboot": true, "update_config": true}
	for _, c := range cmds {
		if !want[c] {
			t.Errorf("unexpected command %q", c)
		}
		delete(want, c)
	}
	for c := range want {
		t.Errorf("missing command %q", c)
	}
	caps := m.Capabilities()
	if len(caps) != 1 || caps[0] != "system_management" {
		t.Errorf("Capabilities() = %v, want [system_management]", caps)
	}
	if m.CollectInterval() != 0 {
		t.Errorf("CollectInterval() = %v, want 0", m.CollectInterval())
	}
}

func TestHandleUpdateConfig(t *testing.T) {
	db := testDB(t)
	m := New()
	initModule(t, m, db)

	payload, _ := proto.Marshal(&pb.UpdateConfigPayload{
		Settings: map[string]string{"log_level": "debug", "scan_interval": "3600"},
	})

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID:      "cmd-1",
		Type:    "update_config",
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("HandleCommand error: %v", err)
	}
	if result.ErrorMessage != "" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}

	// Verify settings were written to DB.
	var val string
	err = db.QueryRow("SELECT value FROM agent_state WHERE key = ?", "log_level").Scan(&val)
	if err != nil {
		t.Fatalf("query log_level: %v", err)
	}
	if val != "debug" {
		t.Errorf("log_level = %q, want %q", val, "debug")
	}

	err = db.QueryRow("SELECT value FROM agent_state WHERE key = ?", "scan_interval").Scan(&val)
	if err != nil {
		t.Fatalf("query scan_interval: %v", err)
	}
	if val != "3600" {
		t.Errorf("scan_interval = %q, want %q", val, "3600")
	}
}

func TestHandleRebootImmediate(t *testing.T) {
	db := testDB(t)
	m := New()
	var rebootCalled bool
	var gotMode pb.RebootMode
	m.rebootFunc = func(_ context.Context, mode pb.RebootMode, _ int32, _ string) error {
		rebootCalled = true
		gotMode = mode
		return nil
	}
	initModule(t, m, db)

	payload, _ := proto.Marshal(&pb.RebootPayload{
		Mode:           pb.RebootMode_REBOOT_MODE_IMMEDIATE,
		PostRebootScan: true,
	})

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID:      "cmd-2",
		Type:    "reboot",
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("HandleCommand error: %v", err)
	}
	if result.ErrorMessage != "" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}
	if !rebootCalled {
		t.Error("reboot function was not called")
	}
	if gotMode != pb.RebootMode_REBOOT_MODE_IMMEDIATE {
		t.Errorf("mode = %v, want IMMEDIATE", gotMode)
	}

	// Verify reboot_pending_scan flag was set.
	var val string
	err = db.QueryRow("SELECT value FROM agent_state WHERE key = ?", "reboot_pending_scan").Scan(&val)
	if err != nil {
		t.Fatalf("query reboot_pending_scan: %v", err)
	}
	if val != "true" {
		t.Errorf("reboot_pending_scan = %q, want %q", val, "true")
	}
}

func TestHandleRebootGraceful(t *testing.T) {
	db := testDB(t)
	m := New()
	var gotGracePeriod int32
	var gotMode pb.RebootMode
	m.rebootFunc = func(_ context.Context, mode pb.RebootMode, gracePeriod int32, _ string) error {
		gotMode = mode
		gotGracePeriod = gracePeriod
		return nil
	}
	initModule(t, m, db)

	payload, _ := proto.Marshal(&pb.RebootPayload{
		Mode:               pb.RebootMode_REBOOT_MODE_GRACEFUL,
		GracePeriodSeconds: 300,
		Message:            "scheduled maintenance",
	})

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID:      "cmd-graceful",
		Type:    "reboot",
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("HandleCommand error: %v", err)
	}
	if result.ErrorMessage != "" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}
	if gotMode != pb.RebootMode_REBOOT_MODE_GRACEFUL {
		t.Errorf("mode = %v, want GRACEFUL", gotMode)
	}
	if gotGracePeriod != 300 {
		t.Errorf("grace_period = %d, want 300", gotGracePeriod)
	}
}

func TestHandleRebootDeferred(t *testing.T) {
	db := testDB(t)
	m := New()
	var gotMode pb.RebootMode
	m.rebootFunc = func(_ context.Context, mode pb.RebootMode, _ int32, _ string) error {
		gotMode = mode
		return nil
	}
	initModule(t, m, db)

	payload, _ := proto.Marshal(&pb.RebootPayload{
		Mode: pb.RebootMode_REBOOT_MODE_DEFERRED,
	})

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID:      "cmd-deferred",
		Type:    "reboot",
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("HandleCommand error: %v", err)
	}
	if result.ErrorMessage != "" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}
	if gotMode != pb.RebootMode_REBOOT_MODE_DEFERRED {
		t.Errorf("mode = %v, want DEFERRED", gotMode)
	}
}

func TestHandleUnsupportedCommand(t *testing.T) {
	m := New()
	initModule(t, m, nil)

	_, err := m.HandleCommand(context.Background(), agent.Command{
		ID:   "cmd-3",
		Type: "unknown",
	})
	if err == nil {
		t.Error("expected error for unsupported command")
	}
}

func TestHandleInvalidPayload(t *testing.T) {
	m := New()
	initModule(t, m, nil)

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID:      "cmd-4",
		Type:    "reboot",
		Payload: []byte("invalid"),
	})
	if err != nil {
		t.Fatalf("should not return Go error for invalid payload: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message for invalid payload")
	}
}

func TestHandleUpdateConfigNoDatabase(t *testing.T) {
	m := New()
	initModule(t, m, nil) // no DB

	payload, _ := proto.Marshal(&pb.UpdateConfigPayload{
		Settings: map[string]string{"log_level": "debug"},
	})

	result, err := m.HandleCommand(context.Background(), agent.Command{
		ID:      "cmd-nodb",
		Type:    "update_config",
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("should not return Go error: %v", err)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message when DB is nil")
	}
}
