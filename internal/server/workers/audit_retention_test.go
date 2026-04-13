package workers_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/server/workers"
)

// fakeRetentionQuerier implements workers.RetentionQuerier for testing.
type fakeRetentionQuerier struct {
	policies []sqlcgen.ListTenantRetentionPoliciesRow
}

func (f *fakeRetentionQuerier) ListTenantRetentionPolicies(ctx context.Context) ([]sqlcgen.ListTenantRetentionPoliciesRow, error) {
	return f.policies, nil
}

// fakePartitionDropper implements workers.PartitionDropper for testing.
type fakePartitionDropper struct {
	partitions []string
	dropped    []string
}

func (f *fakePartitionDropper) ListAuditPartitions(ctx context.Context) ([]string, error) {
	return f.partitions, nil
}

func (f *fakePartitionDropper) DropPartition(ctx context.Context, name string) error {
	f.dropped = append(f.dropped, name)
	return nil
}

func TestAuditRetentionPurger_NoPolicies(t *testing.T) {
	// No tenant policies → use default 365 days.
	// Current date is 2026-03, so a partition from 2024_01 (>1 year ago) should be dropped.
	dropper := &fakePartitionDropper{
		partitions: []string{"audit_events_2024_01", "audit_events_2026_01"},
	}
	purger := workers.NewAuditRetentionPurger(
		&fakeRetentionQuerier{policies: nil},
		dropper,
	)

	if err := purger.Purge(context.Background()); err != nil {
		t.Fatalf("Purge() error: %v", err)
	}

	if len(dropper.dropped) != 1 {
		t.Fatalf("expected 1 partition dropped, got %d: %v", len(dropper.dropped), dropper.dropped)
	}
	if dropper.dropped[0] != "audit_events_2024_01" {
		t.Errorf("expected audit_events_2024_01 dropped, got %s", dropper.dropped[0])
	}
}

func TestAuditRetentionPurger_RespectsLongestRetention(t *testing.T) {
	// Two tenants: 90 days and 730 days (2 years).
	// A partition from ~1 year ago should NOT be dropped (within the 730-day window).
	dropper := &fakePartitionDropper{
		partitions: []string{"audit_events_2025_03", "audit_events_2026_02"},
	}
	purger := workers.NewAuditRetentionPurger(
		&fakeRetentionQuerier{policies: []sqlcgen.ListTenantRetentionPoliciesRow{
			{TenantID: pgtype.UUID{Valid: true}, AuditRetentionDays: 90},
			{TenantID: pgtype.UUID{Valid: true}, AuditRetentionDays: 730},
		}},
		dropper,
	)

	if err := purger.Purge(context.Background()); err != nil {
		t.Fatalf("Purge() error: %v", err)
	}

	if len(dropper.dropped) != 0 {
		t.Fatalf("expected 0 partitions dropped, got %d: %v", len(dropper.dropped), dropper.dropped)
	}
}

func TestAuditRetentionPurger_DropsExpiredPartitions(t *testing.T) {
	// One tenant with 365 days retention.
	// 2024_01 and 2024_06 should be dropped, 2026_01 should be kept.
	// audit_events_default should never be dropped.
	dropper := &fakePartitionDropper{
		partitions: []string{
			"audit_events_2024_01",
			"audit_events_2024_06",
			"audit_events_2026_01",
			"audit_events_default",
		},
	}
	purger := workers.NewAuditRetentionPurger(
		&fakeRetentionQuerier{policies: []sqlcgen.ListTenantRetentionPoliciesRow{
			{TenantID: pgtype.UUID{Valid: true}, AuditRetentionDays: 365},
		}},
		dropper,
	)

	if err := purger.Purge(context.Background()); err != nil {
		t.Fatalf("Purge() error: %v", err)
	}

	if len(dropper.dropped) != 2 {
		t.Fatalf("expected 2 partitions dropped, got %d: %v", len(dropper.dropped), dropper.dropped)
	}

	expected := map[string]bool{
		"audit_events_2024_01": true,
		"audit_events_2024_06": true,
	}
	for _, name := range dropper.dropped {
		if !expected[name] {
			t.Errorf("unexpected partition dropped: %s", name)
		}
	}
}

func TestAuditRetentionPurger_SkipsDefaultPartition(t *testing.T) {
	// Only audit_events_default in partition list → nothing dropped.
	dropper := &fakePartitionDropper{
		partitions: []string{"audit_events_default"},
	}
	purger := workers.NewAuditRetentionPurger(
		&fakeRetentionQuerier{policies: []sqlcgen.ListTenantRetentionPoliciesRow{
			{TenantID: pgtype.UUID{Valid: true}, AuditRetentionDays: 30},
		}},
		dropper,
	)

	if err := purger.Purge(context.Background()); err != nil {
		t.Fatalf("Purge() error: %v", err)
	}

	if len(dropper.dropped) != 0 {
		t.Fatalf("expected 0 partitions dropped, got %d: %v", len(dropper.dropped), dropper.dropped)
	}
}
