package comms_test

import (
	"context"
	"testing"
)

func TestAgentState_GetSet(t *testing.T) {
	_, _, state := openTestDBRaw(t)
	ctx := context.Background()

	if err := state.Set(ctx, "agent_id", "agent-123"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	val, err := state.Get(ctx, "agent_id")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "agent-123" {
		t.Errorf("expected 'agent-123', got %q", val)
	}
}

func TestAgentState_Get_Missing(t *testing.T) {
	_, _, state := openTestDBRaw(t)
	ctx := context.Background()

	val, err := state.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get missing key should not error: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}
}

func TestAgentState_Set_Upsert(t *testing.T) {
	_, _, state := openTestDBRaw(t)
	ctx := context.Background()

	state.Set(ctx, "key", "v1") //nolint:errcheck
	state.Set(ctx, "key", "v2") //nolint:errcheck
	val, _ := state.Get(ctx, "key")
	if val != "v2" {
		t.Errorf("expected 'v2', got %q", val)
	}
}
