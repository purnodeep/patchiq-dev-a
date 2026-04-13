package compliance

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func testUUID(b byte) pgtype.UUID {
	var u pgtype.UUID
	u.Valid = true
	u.Bytes[0] = b
	return u
}

func testNumeric(f float64) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(int64(f * 100)), Exp: -2, Valid: true}
}

func testTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// ---------------------------------------------------------------------------
// Mock querier
// ---------------------------------------------------------------------------

type mockQuerier struct {
	enabledFrameworks      []sqlcgen.ComplianceTenantFramework
	affectedCVEs           []sqlcgen.ListAffectedEndpointCVEsRow
	insertedEvals          []sqlcgen.InsertEvaluationParams
	insertedScores         []sqlcgen.InsertScoreParams
	listFrameworksErr      error
	listCVEsErr            error
	insertEvaluationErr    error
	insertScoreErr         error
	insertControlResultErr error
}

func (m *mockQuerier) ListEnabledTenantFrameworks(_ context.Context, _ pgtype.UUID) ([]sqlcgen.ComplianceTenantFramework, error) {
	if m.listFrameworksErr != nil {
		return nil, m.listFrameworksErr
	}
	return m.enabledFrameworks, nil
}

func (m *mockQuerier) ListAffectedEndpointCVEs(_ context.Context, _ pgtype.UUID) ([]sqlcgen.ListAffectedEndpointCVEsRow, error) {
	if m.listCVEsErr != nil {
		return nil, m.listCVEsErr
	}
	return m.affectedCVEs, nil
}

func (m *mockQuerier) InsertEvaluation(_ context.Context, arg sqlcgen.InsertEvaluationParams) (sqlcgen.ComplianceEvaluation, error) {
	if m.insertEvaluationErr != nil {
		return sqlcgen.ComplianceEvaluation{}, m.insertEvaluationErr
	}
	m.insertedEvals = append(m.insertedEvals, arg)
	return sqlcgen.ComplianceEvaluation{}, nil
}

func (m *mockQuerier) InsertScore(_ context.Context, arg sqlcgen.InsertScoreParams) (sqlcgen.ComplianceScore, error) {
	if m.insertScoreErr != nil {
		return sqlcgen.ComplianceScore{}, m.insertScoreErr
	}
	m.insertedScores = append(m.insertedScores, arg)
	return sqlcgen.ComplianceScore{}, nil
}

func (m *mockQuerier) InsertControlResult(_ context.Context, _ sqlcgen.InsertControlResultParams) (sqlcgen.ComplianceControlResult, error) {
	if m.insertControlResultErr != nil {
		return sqlcgen.ComplianceControlResult{}, m.insertControlResultErr
	}
	return sqlcgen.ComplianceControlResult{}, nil
}

func (m *mockQuerier) DeleteControlResultsByFramework(_ context.Context, _ sqlcgen.DeleteControlResultsByFrameworkParams) error {
	return nil
}

func (m *mockQuerier) UpdateEndpointScoresForRun(_ context.Context, _ sqlcgen.UpdateEndpointScoresForRunParams) error {
	return nil
}

func (m *mockQuerier) UpdateEndpointScoreByID(_ context.Context, _ sqlcgen.UpdateEndpointScoreByIDParams) error {
	return nil
}

// ---------------------------------------------------------------------------
// Mock ControlQuerier for evaluator priority tests
// ---------------------------------------------------------------------------

type mockControlQuerier struct {
	activeEndpoints   int64
	hwEndpoints       int64
	recentHeartbeat   int64
	recentInventory   int64
	scannedForCVEs    int64
	kevEndpoints      int64
	staleCritical     int64
	staleCriticalOnly int64
	deployStats       sqlcgen.GetRecentDeploymentStatsRow
}

func (m *mockControlQuerier) CountActiveEndpoints(_ context.Context, _ pgtype.UUID) (int64, error) {
	return m.activeEndpoints, nil
}
func (m *mockControlQuerier) CountEndpointsWithRecentInventory(_ context.Context, _ sqlcgen.CountEndpointsWithRecentInventoryParams) (int64, error) {
	return m.recentInventory, nil
}
func (m *mockControlQuerier) CountEndpointsWithRecentHeartbeat(_ context.Context, _ sqlcgen.CountEndpointsWithRecentHeartbeatParams) (int64, error) {
	return m.recentHeartbeat, nil
}
func (m *mockControlQuerier) CountEndpointsWithHardwareInfo(_ context.Context, _ pgtype.UUID) (int64, error) {
	return m.hwEndpoints, nil
}
func (m *mockControlQuerier) CountEndpointsWithKEVVulnerabilities(_ context.Context, _ pgtype.UUID) (int64, error) {
	return m.kevEndpoints, nil
}
func (m *mockControlQuerier) CountEndpointsScannedForCVEs(_ context.Context, _ pgtype.UUID) (int64, error) {
	return m.scannedForCVEs, nil
}
func (m *mockControlQuerier) CountEndpointsWithStaleCriticalCVEs(_ context.Context, _ sqlcgen.CountEndpointsWithStaleCriticalCVEsParams) (int64, error) {
	return m.staleCritical, nil
}
func (m *mockControlQuerier) CountEndpointsWithStaleCriticalOnlyCVEs(_ context.Context, _ sqlcgen.CountEndpointsWithStaleCriticalOnlyCVEsParams) (int64, error) {
	return m.staleCriticalOnly, nil
}
func (m *mockControlQuerier) GetRecentDeploymentStats(_ context.Context, _ sqlcgen.GetRecentDeploymentStatsParams) (sqlcgen.GetRecentDeploymentStatsRow, error) {
	return m.deployStats, nil
}
func (m *mockControlQuerier) ListNonDecommissionedEndpointIDs(_ context.Context, _ pgtype.UUID) ([]pgtype.UUID, error) {
	return nil, nil
}

func (m *mockControlQuerier) ListEndpointComplianceFlags(_ context.Context, _ sqlcgen.ListEndpointComplianceFlagsParams) ([]sqlcgen.ListEndpointComplianceFlagsRow, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Evaluator priority: CheckType evaluator overrides SLA tier derivation
// ---------------------------------------------------------------------------

func TestInsertControlResults_EvaluatorPriority(t *testing.T) {
	t.Parallel()

	// Framework with a control that has BOTH a check_type evaluator AND SLA tiers.
	// The evaluator should be used, NOT the SLA tier derivation.
	fw := &Framework{
		ID:   "test-priority",
		Name: "Evaluator Priority Test",
		Controls: []Control{
			{
				ID:        "SW-TEST",
				Name:      "Software Inventory Test",
				Category:  "Test",
				CheckType: "software_inventory", // has registered evaluator
				SLATiers: []SeverityTier{ // also has SLA tiers — should be IGNORED
					{Label: "critical", Days: intP(7), CVSSMin: 9, CVSSMax: 10},
				},
			},
			{
				ID:       "SLA-TEST",
				Name:     "Pure SLA Control",
				Category: "Test",
				// No CheckType — pure SLA control
				SLATiers: []SeverityTier{
					{Label: "critical", Days: intP(7), CVSSMin: 9, CVSSMax: 10},
				},
			},
		},
	}

	mq := &mockQuerier{}

	// Mock: 10 endpoints, only 1 with recent inventory scan.
	// software_inventory evaluator should return fail (1/10 = 10% < 95% threshold).
	cq := &mockControlQuerier{
		activeEndpoints: 10,
		recentInventory: 1,
	}

	tenantID := testUUID(0x01)
	runID := testUUID(0x02)
	now := time.Now().UTC()
	evaluatedAt := pgtype.Timestamptz{Time: now, Valid: true}

	// High CVE score (100%) — if SLA tiers were used, both controls would PASS.
	tenantCVEScore := 100.0

	counts, err := insertControlResults(
		context.Background(), mq, cq, fw,
		tenantID, runID, evaluatedAt,
		tenantCVEScore, 10,
	)
	if err != nil {
		t.Fatalf("insertControlResults: %v", err)
	}

	// SW-TEST: should use software_inventory evaluator → FAIL (1/10 < 95%)
	// SLA-TEST: no evaluator, has SLA tiers → derive from 100% score → PASS
	if counts.Passing != 1 {
		t.Errorf("passing: got %d, want 1 (only SLA-TEST should pass)", counts.Passing)
	}
	if counts.Failing != 1 {
		t.Errorf("failing: got %d, want 1 (SW-TEST should fail via evaluator)", counts.Failing)
	}
	if counts.Evaluated != 2 {
		t.Errorf("evaluated: got %d, want 2", counts.Evaluated)
	}
}

func TestInsertControlResults_PureSLADerivation(t *testing.T) {
	t.Parallel()

	// Framework with only pure SLA controls (no check_type).
	fw := &Framework{
		ID:   "test-sla-only",
		Name: "Pure SLA Test",
		Controls: []Control{
			{
				ID:       "SLA-1",
				Name:     "SLA Control 1",
				Category: "Patch Management",
				SLATiers: []SeverityTier{
					{Label: "critical", Days: intP(7), CVSSMin: 9, CVSSMax: 10},
				},
			},
		},
	}

	mq := &mockQuerier{}
	cq := &mockControlQuerier{activeEndpoints: 10}
	tenantID := testUUID(0x01)
	runID := testUUID(0x02)
	evaluatedAt := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}

	// Low CVE score → SLA control should FAIL.
	counts, err := insertControlResults(
		context.Background(), mq, cq, fw,
		tenantID, runID, evaluatedAt,
		50.0, 10,
	)
	if err != nil {
		t.Fatalf("insertControlResults: %v", err)
	}

	if counts.Passing != 0 {
		t.Errorf("passing: got %d, want 0 (50%% score → fail)", counts.Passing)
	}
	if counts.Failing != 1 {
		t.Errorf("failing: got %d, want 1", counts.Failing)
	}
}

// ---------------------------------------------------------------------------
// Task 3: ParseSLAOverrides tests
// ---------------------------------------------------------------------------

func TestParseSLAOverrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    map[string]int
		wantErr bool
	}{
		{
			name:  "nil input returns empty map",
			input: nil,
			want:  map[string]int{},
		},
		{
			name:  "empty bytes returns empty map",
			input: []byte{},
			want:  map[string]int{},
		},
		{
			name:  "empty JSON object returns empty map",
			input: []byte(`{}`),
			want:  map[string]int{},
		},
		{
			name:  "valid overrides",
			input: []byte(`{"critical": 10, "high": 20}`),
			want:  map[string]int{"critical": 10, "high": 20},
		},
		{
			name:    "invalid JSON returns error",
			input:   []byte(`not json`),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseSLAOverrides(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d entries, want %d", len(got), len(tc.want))
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("key %q: got %d, want %d", k, got[k], v)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Task 3: ResolveSLADays tests
// ---------------------------------------------------------------------------

func TestResolveSLADays(t *testing.T) {
	t.Parallel()

	nist := GetFramework(FrameworkNIST80053)
	control := nist.PatchSLAControl()

	tests := []struct {
		name         string
		cvss         float64
		overrides    map[string]int
		wantDays     *int
		wantSeverity string
	}{
		{
			name:         "critical no override uses framework default 15",
			cvss:         9.8,
			overrides:    map[string]int{},
			wantDays:     intP(15),
			wantSeverity: "critical",
		},
		{
			name:         "critical with override uses override 7",
			cvss:         9.8,
			overrides:    map[string]int{"critical": 7},
			wantDays:     intP(7),
			wantSeverity: "critical",
		},
		{
			name:         "high no override uses framework default 30",
			cvss:         7.5,
			overrides:    map[string]int{},
			wantDays:     intP(30),
			wantSeverity: "high",
		},
		{
			name:         "low no SLA returns nil days",
			cvss:         2.0,
			overrides:    map[string]int{},
			wantDays:     nil,
			wantSeverity: "low",
		},
		{
			name:         "low with override adds SLA",
			cvss:         2.0,
			overrides:    map[string]int{"low": 180},
			wantDays:     intP(180),
			wantSeverity: "low",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			days, severity := ResolveSLADays(control, tc.cvss, tc.overrides)
			if severity != tc.wantSeverity {
				t.Errorf("severity: got %q, want %q", severity, tc.wantSeverity)
			}
			if tc.wantDays == nil {
				if days != nil {
					t.Errorf("days: got %d, want nil", *days)
				}
			} else {
				if days == nil {
					t.Fatalf("days: got nil, want %d", *tc.wantDays)
					return
				}
				if *days != *tc.wantDays {
					t.Errorf("days: got %d, want %d", *days, *tc.wantDays)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Task 4: RunEvaluation tests
// ---------------------------------------------------------------------------

func TestRunEvaluation_NoFrameworks(t *testing.T) {
	t.Parallel()

	mq := &mockQuerier{
		enabledFrameworks: []sqlcgen.ComplianceTenantFramework{},
	}
	svc := NewService()
	result, err := svc.RunEvaluation(context.Background(), testUUID(0x01), mq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FrameworksEvaluated != 0 {
		t.Errorf("FrameworksEvaluated: got %d, want 0", result.FrameworksEvaluated)
	}
	if result.TotalEvaluations != 0 {
		t.Errorf("TotalEvaluations: got %d, want 0", result.TotalEvaluations)
	}
	if len(mq.insertedEvals) != 0 {
		t.Errorf("expected 0 eval inserts, got %d", len(mq.insertedEvals))
	}
	if len(mq.insertedScores) != 0 {
		t.Errorf("expected 0 score inserts, got %d", len(mq.insertedScores))
	}
}

func TestRunEvaluation_SingleFrameworkSingleEndpoint(t *testing.T) {
	t.Parallel()

	tenantID := testUUID(0x01)
	endpointID := testUUID(0x10)
	published := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	mq := &mockQuerier{
		enabledFrameworks: []sqlcgen.ComplianceTenantFramework{
			{
				ID:            testUUID(0xA0),
				TenantID:      tenantID,
				FrameworkID:   FrameworkNIST80053,
				Enabled:       true,
				SlaOverrides:  nil,
				ScoringMethod: "strictest",
			},
		},
		affectedCVEs: []sqlcgen.ListAffectedEndpointCVEsRow{
			{
				EndpointCveID: testUUID(0xC0),
				EndpointID:    endpointID,
				CveRefID:      testUUID(0xD0),
				Status:        "affected",
				DetectedAt:    testTimestamptz(published),
				CveIdentifier: "CVE-2026-0001",
				Severity:      "critical",
				CvssV3Score:   testNumeric(9.8),
				PublishedAt:   testTimestamptz(published),
				Hostname:      "host-1",
				OsFamily:      "linux",
			},
		},
	}

	svc := NewService()
	result, err := svc.RunEvaluation(context.Background(), tenantID, mq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FrameworksEvaluated != 1 {
		t.Errorf("FrameworksEvaluated: got %d, want 1", result.FrameworksEvaluated)
	}
	if result.TotalEvaluations != 1 {
		t.Errorf("TotalEvaluations: got %d, want 1", result.TotalEvaluations)
	}

	// 1 eval insert
	if len(mq.insertedEvals) != 1 {
		t.Fatalf("expected 1 eval insert, got %d", len(mq.insertedEvals))
	}
	eval := mq.insertedEvals[0]
	if eval.FrameworkID != FrameworkNIST80053 {
		t.Errorf("eval framework: got %q, want %q", eval.FrameworkID, FrameworkNIST80053)
	}
	if eval.State != string(StateNonCompliant) {
		t.Errorf("eval state: got %q, want %q", eval.State, StateNonCompliant)
	}
	if eval.CveID != "CVE-2026-0001" {
		t.Errorf("eval cve_id: got %q, want CVE-2026-0001", eval.CveID)
	}

	// 2 score inserts (1 endpoint + 1 tenant)
	if len(mq.insertedScores) != 2 {
		t.Fatalf("expected 2 score inserts, got %d", len(mq.insertedScores))
	}

	// Find endpoint and tenant scores
	var endpointScore, tenantScore *sqlcgen.InsertScoreParams
	for i := range mq.insertedScores {
		switch mq.insertedScores[i].ScopeType {
		case "endpoint":
			endpointScore = &mq.insertedScores[i]
		case "tenant":
			tenantScore = &mq.insertedScores[i]
		}
	}
	if endpointScore == nil {
		t.Fatal("missing endpoint score")
		return
	}
	if tenantScore == nil {
		t.Fatal("missing tenant score")
		return
	}
	if endpointScore.NonCompliantCves != 1 {
		t.Errorf("endpoint non-compliant: got %d, want 1", endpointScore.NonCompliantCves)
	}
}

func TestRunEvaluation_WithSLAOverride(t *testing.T) {
	t.Parallel()

	tenantID := testUUID(0x01)
	endpointID := testUUID(0x10)
	published := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	mq := &mockQuerier{
		enabledFrameworks: []sqlcgen.ComplianceTenantFramework{
			{
				ID:            testUUID(0xA0),
				TenantID:      tenantID,
				FrameworkID:   FrameworkNIST80053,
				Enabled:       true,
				SlaOverrides:  []byte(`{"critical": 7}`),
				ScoringMethod: "strictest",
			},
		},
		affectedCVEs: []sqlcgen.ListAffectedEndpointCVEsRow{
			{
				EndpointCveID: testUUID(0xC0),
				EndpointID:    endpointID,
				CveRefID:      testUUID(0xD0),
				Status:        "affected",
				DetectedAt:    testTimestamptz(published),
				CveIdentifier: "CVE-2026-0001",
				Severity:      "critical",
				CvssV3Score:   testNumeric(9.8),
				PublishedAt:   testTimestamptz(published),
				Hostname:      "host-1",
				OsFamily:      "linux",
			},
		},
	}

	svc := NewService()
	result, err := svc.RunEvaluation(context.Background(), tenantID, mq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalEvaluations != 1 {
		t.Fatalf("expected 1 evaluation, got %d", result.TotalEvaluations)
	}

	eval := mq.insertedEvals[0]
	// With 7-day SLA from Jan 1, deadline should be Jan 8
	expectedDeadline := published.AddDate(0, 0, 7)
	if !eval.SlaDeadlineAt.Valid {
		t.Fatal("SLA deadline should be set")
	}
	if !eval.SlaDeadlineAt.Time.Equal(expectedDeadline) {
		t.Errorf("SLA deadline: got %v, want %v", eval.SlaDeadlineAt.Time, expectedDeadline)
	}
}

func TestRunEvaluation_NoCVEs(t *testing.T) {
	t.Parallel()

	tenantID := testUUID(0x01)
	mq := &mockQuerier{
		enabledFrameworks: []sqlcgen.ComplianceTenantFramework{
			{
				ID:            testUUID(0xA0),
				TenantID:      tenantID,
				FrameworkID:   FrameworkNIST80053,
				Enabled:       true,
				ScoringMethod: "strictest",
			},
		},
		affectedCVEs: []sqlcgen.ListAffectedEndpointCVEsRow{},
	}

	svc := NewService()
	result, err := svc.RunEvaluation(context.Background(), tenantID, mq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalEvaluations != 0 {
		t.Errorf("TotalEvaluations: got %d, want 0", result.TotalEvaluations)
	}
	if len(mq.insertedEvals) != 0 {
		t.Errorf("expected 0 eval inserts, got %d", len(mq.insertedEvals))
	}
	// Should still insert 1 tenant-level score of 100.0
	if len(mq.insertedScores) != 1 {
		t.Fatalf("expected 1 score insert (tenant), got %d", len(mq.insertedScores))
	}
	ts := mq.insertedScores[0]
	if ts.ScopeType != "tenant" {
		t.Errorf("scope_type: got %q, want tenant", ts.ScopeType)
	}
	// Score should be 100.0 — verify via numeric: Int=10000, Exp=-2
	scoreF := numericToFloat64(ts.Score)
	if scoreF != 100.0 {
		t.Errorf("tenant score: got %f, want 100.0", scoreF)
	}
	if ts.TotalCves != 0 {
		t.Errorf("total_cves: got %d, want 0", ts.TotalCves)
	}

	// Framework score in result
	if len(result.FrameworkScores) != 1 {
		t.Fatalf("expected 1 framework score, got %d", len(result.FrameworkScores))
	}
	if result.FrameworkScores[0].Score != 100.0 {
		t.Errorf("framework score: got %f, want 100.0", result.FrameworkScores[0].Score)
	}
}

func TestRunEvaluation_MultipleFrameworksMultipleEndpoints(t *testing.T) {
	t.Parallel()

	tenantID := testUUID(0x01)
	ep1 := testUUID(0x10)
	ep2 := testUUID(0x20)
	published := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	mq := &mockQuerier{
		enabledFrameworks: []sqlcgen.ComplianceTenantFramework{
			{
				ID:            testUUID(0xA0),
				TenantID:      tenantID,
				FrameworkID:   FrameworkNIST80053,
				Enabled:       true,
				ScoringMethod: "strictest",
			},
			{
				ID:            testUUID(0xA1),
				TenantID:      tenantID,
				FrameworkID:   FrameworkPCIDSSv4,
				Enabled:       true,
				ScoringMethod: "worst_case",
			},
		},
		affectedCVEs: []sqlcgen.ListAffectedEndpointCVEsRow{
			{
				EndpointCveID: testUUID(0xC0),
				EndpointID:    ep1,
				CveRefID:      testUUID(0xD0),
				Status:        "affected",
				DetectedAt:    testTimestamptz(published),
				CveIdentifier: "CVE-2026-0001",
				Severity:      "critical",
				CvssV3Score:   testNumeric(9.8),
				PublishedAt:   testTimestamptz(published),
				Hostname:      "host-1",
				OsFamily:      "linux",
			},
			{
				EndpointCveID: testUUID(0xC1),
				EndpointID:    ep2,
				CveRefID:      testUUID(0xD1),
				Status:        "affected",
				DetectedAt:    testTimestamptz(published),
				CveIdentifier: "CVE-2026-0002",
				Severity:      "high",
				CvssV3Score:   testNumeric(7.5),
				PublishedAt:   testTimestamptz(published),
				Hostname:      "host-2",
				OsFamily:      "windows",
			},
		},
	}

	svc := NewService()
	result, err := svc.RunEvaluation(context.Background(), tenantID, mq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FrameworksEvaluated != 2 {
		t.Errorf("FrameworksEvaluated: got %d, want 2", result.FrameworksEvaluated)
	}
	// 2 frameworks x 2 CVEs = 4 evaluations
	if result.TotalEvaluations != 4 {
		t.Errorf("TotalEvaluations: got %d, want 4", result.TotalEvaluations)
	}
	if len(mq.insertedEvals) != 4 {
		t.Errorf("eval inserts: got %d, want 4", len(mq.insertedEvals))
	}
	// 2 frameworks x (2 endpoints + 1 tenant) = 6 scores
	if len(mq.insertedScores) != 6 {
		t.Errorf("score inserts: got %d, want 6", len(mq.insertedScores))
	}
}

// ---------------------------------------------------------------------------
// Error path tests
// ---------------------------------------------------------------------------

func TestRunEvaluation_ListFrameworksError(t *testing.T) {
	t.Parallel()

	mq := &mockQuerier{
		listFrameworksErr: errors.New("db connection lost"),
	}
	svc := NewService()
	_, err := svc.RunEvaluation(context.Background(), testUUID(0x01), mq)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, mq.listFrameworksErr) {
		t.Errorf("expected wrapped db error, got: %v", err)
	}
}

func TestRunEvaluation_InsertEvaluationError(t *testing.T) {
	t.Parallel()

	tenantID := testUUID(0x01)
	published := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	mq := &mockQuerier{
		enabledFrameworks: []sqlcgen.ComplianceTenantFramework{
			{
				ID:            testUUID(0xA0),
				TenantID:      tenantID,
				FrameworkID:   FrameworkNIST80053,
				Enabled:       true,
				ScoringMethod: "strictest",
			},
		},
		affectedCVEs: []sqlcgen.ListAffectedEndpointCVEsRow{
			{
				EndpointCveID: testUUID(0xC0),
				EndpointID:    testUUID(0x10),
				CveRefID:      testUUID(0xD0),
				Status:        "affected",
				DetectedAt:    testTimestamptz(published),
				CveIdentifier: "CVE-2026-0001",
				Severity:      "critical",
				CvssV3Score:   testNumeric(9.8),
				PublishedAt:   testTimestamptz(published),
				Hostname:      "host-1",
				OsFamily:      "linux",
			},
		},
		insertEvaluationErr: errors.New("disk full"),
	}
	svc := NewService()
	_, err := svc.RunEvaluation(context.Background(), tenantID, mq)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, mq.insertEvaluationErr) {
		t.Errorf("expected wrapped disk error, got: %v", err)
	}
}

func TestRunEvaluation_InsertScoreError(t *testing.T) {
	t.Parallel()

	tenantID := testUUID(0x01)
	mq := &mockQuerier{
		enabledFrameworks: []sqlcgen.ComplianceTenantFramework{
			{
				ID:            testUUID(0xA0),
				TenantID:      tenantID,
				FrameworkID:   FrameworkNIST80053,
				Enabled:       true,
				ScoringMethod: "strictest",
			},
		},
		affectedCVEs:   []sqlcgen.ListAffectedEndpointCVEsRow{},
		insertScoreErr: errors.New("constraint violation"),
	}
	svc := NewService()
	_, err := svc.RunEvaluation(context.Background(), tenantID, mq)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, mq.insertScoreErr) {
		t.Errorf("expected wrapped constraint error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// deriveControlStatus tests
// ---------------------------------------------------------------------------

func TestDeriveControlStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		tenantScore float64
		controlID   string
		frameworkID string
		wantStatus  string
	}{
		{"high score deterministic", 100, "SI-2", "nist_800_53", ""},
		{"low score fail", 20, "SI-2", "nist_800_53", "fail"},
		{"deterministic", 85, "CIS-7.1", "cis", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status1, ratio1 := deriveControlStatus(tt.tenantScore, tt.controlID, tt.frameworkID)
			status2, ratio2 := deriveControlStatus(tt.tenantScore, tt.controlID, tt.frameworkID)
			// Determinism check
			if status1 != status2 || ratio1 != ratio2 {
				t.Errorf("non-deterministic: got %s/%f then %s/%f", status1, ratio1, status2, ratio2)
			}
			if tt.wantStatus != "" && status1 != tt.wantStatus {
				t.Errorf("want status %s, got %s", tt.wantStatus, status1)
			}
			// Ratio bounds
			if ratio1 < 0 || ratio1 > 1 {
				t.Errorf("ratio out of bounds: %f", ratio1)
			}
		})
	}
}
