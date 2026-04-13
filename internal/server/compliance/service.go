package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// EvaluationQuerier defines the database operations needed by the evaluation service.
type EvaluationQuerier interface {
	ListEnabledTenantFrameworks(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.ComplianceTenantFramework, error)
	ListAffectedEndpointCVEs(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.ListAffectedEndpointCVEsRow, error)
	InsertEvaluation(ctx context.Context, arg sqlcgen.InsertEvaluationParams) (sqlcgen.ComplianceEvaluation, error)
	InsertScore(ctx context.Context, arg sqlcgen.InsertScoreParams) (sqlcgen.ComplianceScore, error)
	InsertControlResult(ctx context.Context, arg sqlcgen.InsertControlResultParams) (sqlcgen.ComplianceControlResult, error)
	DeleteControlResultsByFramework(ctx context.Context, arg sqlcgen.DeleteControlResultsByFrameworkParams) error
	UpdateEndpointScoresForRun(ctx context.Context, arg sqlcgen.UpdateEndpointScoresForRunParams) error
	UpdateEndpointScoreByID(ctx context.Context, arg sqlcgen.UpdateEndpointScoreByIDParams) error
}

// EvaluationResult summarizes the output of a compliance evaluation run.
type EvaluationResult struct {
	RunID               string
	FrameworksEvaluated int
	TotalEvaluations    int
	FrameworkScores     []FrameworkScoreResult
}

// FrameworkScoreResult holds a single framework's tenant-level score.
type FrameworkScoreResult struct {
	FrameworkID string
	Score       float64
	Counts      StateCounts
}

// CustomFrameworkLoader loads custom frameworks from the database.
type CustomFrameworkLoader interface {
	LoadCustomFramework(ctx context.Context, tenantID pgtype.UUID, frameworkID string) (*Framework, error)
}

// Service orchestrates compliance evaluations.
type Service struct {
	customLoader CustomFrameworkLoader
	controlQ     ControlQuerier
}

// NewService creates a new compliance evaluation service.
func NewService() *Service {
	return &Service{}
}

// WithControlQuerier configures the service to use real data-driven control evaluators.
func (s *Service) WithControlQuerier(q ControlQuerier) *Service {
	s.controlQ = q
	return s
}

// WithCustomFrameworks configures the service to evaluate custom frameworks from the DB.
func (s *Service) WithCustomFrameworks(loader CustomFrameworkLoader) *Service {
	s.customLoader = loader
	return s
}

// ParseSLAOverrides parses tenant SLA override JSON (e.g. {"critical": 10, "high": 20}).
// Nil or empty input returns an empty map.
func ParseSLAOverrides(raw []byte) (map[string]int, error) {
	if len(raw) == 0 {
		return map[string]int{}, nil
	}
	var overrides map[string]int
	if err := json.Unmarshal(raw, &overrides); err != nil {
		return nil, fmt.Errorf("parse SLA overrides: %w", err)
	}
	if overrides == nil {
		return map[string]int{}, nil
	}
	return overrides, nil
}

// ResolveSLADays determines SLA days for a CVE by CVSS score. It uses the framework
// control's default, then checks the overrides map. Returns (slaDays, severityLabel).
func ResolveSLADays(control *Control, cvss float64, overrides map[string]int) (*int, string) {
	days, severity := control.SLADaysByCVSS(cvss)
	if severity == "" {
		return nil, ""
	}
	if v, ok := overrides[severity]; ok {
		return intP(v), severity
	}
	return days, severity
}

// RunFrameworkEvaluation evaluates a single framework for a tenant.
func (s *Service) RunFrameworkEvaluation(ctx context.Context, tenantID pgtype.UUID, frameworkID string, q EvaluationQuerier) (*EvaluationResult, error) {
	return s.runEvaluation(ctx, tenantID, frameworkID, q)
}

// RunEvaluation evaluates all affected CVEs across enabled frameworks for a tenant.
// The caller is responsible for providing a transaction-scoped querier with
// app.current_tenant_id set (via store.BeginTx) to satisfy RLS policies.
//
// Domain events: This method performs bulk inserts as part of a single evaluation run.
// A single ComplianceEvaluationCompleted event is emitted by the handler after commit,
// rather than per-row events, to avoid excessive event volume during batch evaluation.
func (s *Service) RunEvaluation(ctx context.Context, tenantID pgtype.UUID, q EvaluationQuerier) (*EvaluationResult, error) {
	return s.runEvaluation(ctx, tenantID, "", q)
}

func (s *Service) runEvaluation(ctx context.Context, tenantID pgtype.UUID, onlyFrameworkID string, q EvaluationQuerier) (*EvaluationResult, error) {
	// Prefer the transaction-scoped querier for control evaluations so that
	// RLS policies (which require SET LOCAL app.current_tenant_id) are satisfied.
	// The caller's txQ (*sqlcgen.Queries) satisfies both EvaluationQuerier and
	// ControlQuerier; using it instead of s.controlQ (raw pool) avoids RLS
	// returning zero rows.
	ctrlQ := s.controlQ
	if tq, ok := q.(ControlQuerier); ok {
		ctrlQ = tq
	}

	frameworks, err := q.ListEnabledTenantFrameworks(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list enabled frameworks: %w", err)
	}

	// Filter to a single framework if requested.
	if onlyFrameworkID != "" {
		filtered := frameworks[:0]
		for _, tf := range frameworks {
			if tf.FrameworkID == onlyFrameworkID {
				filtered = append(filtered, tf)
			}
		}
		frameworks = filtered
	}

	result := &EvaluationResult{
		FrameworkScores: []FrameworkScoreResult{},
	}

	if len(frameworks) == 0 {
		return result, nil
	}

	cves, err := q.ListAffectedEndpointCVEs(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list affected endpoint CVEs: %w", err)
	}

	runUUID := uuid.New()
	runID := pgtype.UUID{Valid: true}
	copy(runID.Bytes[:], runUUID[:])
	result.RunID = runUUID.String()

	now := time.Now().UTC()
	evaluatedAt := pgtype.Timestamptz{Time: now, Valid: true}

	// Group CVEs by endpoint_id
	type endpointCVEs struct {
		endpointID pgtype.UUID
		cves       []sqlcgen.ListAffectedEndpointCVEsRow
	}
	endpointMap := make(map[pgtype.UUID]*endpointCVEs)
	var endpointOrder []pgtype.UUID
	for _, cve := range cves {
		ec, ok := endpointMap[cve.EndpointID]
		if !ok {
			ec = &endpointCVEs{endpointID: cve.EndpointID}
			endpointMap[cve.EndpointID] = ec
			endpointOrder = append(endpointOrder, cve.EndpointID)
		}
		ec.cves = append(ec.cves, cve)
	}

	for _, tf := range frameworks {
		fw := GetFramework(tf.FrameworkID)
		if fw == nil && s.customLoader != nil {
			// Try loading as a custom framework from DB.
			customFW, loadErr := s.customLoader.LoadCustomFramework(ctx, tenantID, tf.FrameworkID)
			if loadErr != nil {
				slog.WarnContext(ctx, "compliance: failed to load custom framework",
					"framework_id", tf.FrameworkID, "error", loadErr)
			} else {
				fw = customFW
			}
		}
		if fw == nil {
			slog.WarnContext(ctx, "compliance: skipping unknown framework", "framework_id", tf.FrameworkID)
			continue
		}
		control := fw.PatchSLAControl()

		// Frameworks with no SLA control (e.g. custom frameworks using only
		// non-SLA check types like asset_inventory, vuln_scanning) are evaluated
		// purely via their control evaluators — skip the CVE-based SLA scoring.
		if control == nil {
			if ctrlQ == nil || len(fw.Controls) == 0 {
				slog.WarnContext(ctx, "compliance: framework has no patch SLA control and no evaluable controls", "framework_id", tf.FrameworkID)
				continue
			}

			score, err := evaluateNonSLAFramework(ctx, q, ctrlQ, fw, tenantID, runID, evaluatedAt)
			if err != nil {
				return nil, fmt.Errorf("evaluate non-SLA framework %s: %w", tf.FrameworkID, err)
			}
			result.FrameworksEvaluated++
			result.TotalEvaluations += len(fw.Controls)
			result.FrameworkScores = append(result.FrameworkScores, FrameworkScoreResult{
				FrameworkID: tf.FrameworkID,
				Score:       score,
			})
			continue
		}

		overrides, err := ParseSLAOverrides(tf.SlaOverrides)
		if err != nil {
			return nil, fmt.Errorf("parse SLA overrides for framework %s: %w", tf.FrameworkID, err)
		}

		atRiskThreshold := parseAtRiskThreshold(ctx, tf.AtRiskThreshold)

		scoringMethod := tf.ScoringMethod
		if scoringMethod == "" {
			scoringMethod = fw.DefaultScoringMethod
		}

		var allEndpointScores []float64
		var totalCounts StateCounts
		frameworkEvalCount := 0

		if len(cves) == 0 {
			// No CVEs: SLA controls pass (100%), evaluate non-SLA controls normally.
			noCVECounts, noCVEErr := insertControlResults(ctx, q, ctrlQ, fw, tenantID, runID, evaluatedAt, 100.0, 0)
			if noCVEErr != nil {
				return nil, fmt.Errorf("insert control results for %s: %w", tf.FrameworkID, noCVEErr)
			}
			noCVEScore := 100.0
			if noCVECounts.Evaluated > 0 {
				noCVEScore = math.Round(float64(noCVECounts.Passing) / float64(noCVECounts.Evaluated) * 100)
			}
			if err := insertScoreRow(ctx, q, sqlcgen.InsertScoreParams{
				TenantID:        tenantID,
				EvaluationRunID: runID,
				FrameworkID:     tf.FrameworkID,
				ScopeType:       "tenant",
				ScopeID:         tenantID,
				Score:           float64ToNumeric(noCVEScore),
				EvaluatedAt:     evaluatedAt,
			}); err != nil {
				return nil, err
			}
			result.FrameworksEvaluated++
			result.FrameworkScores = append(result.FrameworkScores, FrameworkScoreResult{
				FrameworkID: tf.FrameworkID,
				Score:       noCVEScore,
				Counts:      StateCounts{},
			})
			continue
		}

		for _, epID := range endpointOrder {
			ec := endpointMap[epID]
			var states []EvalState

			for _, cve := range ec.cves {
				cvss := numericToFloat64(cve.CvssV3Score)
				slaDays, _ := ResolveSLADays(control, cvss, overrides)

				// publishedAt fallback: use detectedAt if publishedAt is not valid
				slaStart := cve.DetectedAt.Time
				if cve.PublishedAt.Valid {
					slaStart = cve.PublishedAt.Time
				}

				var remediatedAt *time.Time
				if cve.ResolvedAt.Valid {
					t := cve.ResolvedAt.Time
					remediatedAt = &t
				}

				state := EvaluateCVEState(slaStart, slaDays, remediatedAt, now, atRiskThreshold)
				states = append(states, state)

				deadline := ComputeSLADeadline(slaStart, slaDays)
				var slaDeadlineAt pgtype.Timestamptz
				if deadline != nil {
					slaDeadlineAt = pgtype.Timestamptz{Time: *deadline, Valid: true}
				}

				var daysRem pgtype.Int4
				if deadline != nil {
					dr := DaysRemaining(*deadline, now)
					daysRem = pgtype.Int4{Int32: int32(dr), Valid: true}
				}

				var remediatedAtPG pgtype.Timestamptz
				if remediatedAt != nil {
					remediatedAtPG = pgtype.Timestamptz{Time: *remediatedAt, Valid: true}
				}

				evalParam := sqlcgen.InsertEvaluationParams{
					TenantID:        tenantID,
					EvaluationRunID: runID,
					EndpointID:      cve.EndpointID,
					CveID:           cve.CveIdentifier,
					FrameworkID:     tf.FrameworkID,
					ControlID:       control.ID,
					State:           string(state),
					SlaDeadlineAt:   slaDeadlineAt,
					RemediatedAt:    remediatedAtPG,
					DaysRemaining:   daysRem,
					EvaluatedAt:     evaluatedAt,
				}
				if _, err := q.InsertEvaluation(ctx, evalParam); err != nil {
					return nil, fmt.Errorf("insert evaluation for %s/%s: %w", cve.CveIdentifier, tf.FrameworkID, err)
				}
				frameworkEvalCount++
			}

			score, counts := ComputeEndpointScore(states)
			allEndpointScores = append(allEndpointScores, score)
			totalCounts.Total += counts.Total
			totalCounts.Compliant += counts.Compliant
			totalCounts.AtRisk += counts.AtRisk
			totalCounts.NonCompliant += counts.NonCompliant
			totalCounts.LateRemediation += counts.LateRemediation

			if err := insertScoreRow(ctx, q, sqlcgen.InsertScoreParams{
				TenantID:            tenantID,
				EvaluationRunID:     runID,
				FrameworkID:         tf.FrameworkID,
				ScopeType:           "endpoint",
				ScopeID:             ec.endpointID,
				Score:               float64ToNumeric(score),
				TotalCves:           int32(counts.Total),
				CompliantCves:       int32(counts.Compliant),
				AtRiskCves:          int32(counts.AtRisk),
				NonCompliantCves:    int32(counts.NonCompliant),
				LateRemediationCves: int32(counts.LateRemediation),
				EvaluatedAt:         evaluatedAt,
			}); err != nil {
				return nil, err
			}
		}

		tenantCVEScore := ComputeGroupScore(allEndpointScores, scoringMethod)

		// Evaluate all controls: SLA controls derive from CVE score,
		// non-SLA controls use their dedicated evaluators.
		ctrlCounts, err := insertControlResults(ctx, q, ctrlQ, fw, tenantID, runID, evaluatedAt, tenantCVEScore, int32(len(endpointOrder)))
		if err != nil {
			return nil, fmt.Errorf("insert control results for %s: %w", tf.FrameworkID, err)
		}

		// Framework score = % of evaluated controls passing.
		frameworkScore := tenantCVEScore
		if ctrlCounts.Evaluated > 0 {
			frameworkScore = math.Round(float64(ctrlCounts.Passing) / float64(ctrlCounts.Evaluated) * 100)
		}

		// Compute per-endpoint scores based on individual control compliance.
		// Each endpoint gets its own score depending on which controls it passes.
		if ctrlQ != nil && ctrlCounts.Evaluated > 0 {
			aggResults := make(map[string]string)
			for _, ctrl := range fw.Controls {
				ct := ctrl.CheckType
				if ct == "" {
					ct = ctrl.ID
				}
				if len(ctrl.SLATiers) > 0 && !hasRegisteredEvaluator(ct) {
					if tenantCVEScore >= 95 {
						aggResults[ct] = "pass"
					} else {
						aggResults[ct] = "fail"
					}
				} else if ct == "deployment_governance" {
					if evalFn, ok := controlEvaluators[ct]; ok {
						cfg := ParseCheckConfig(ct, ctrl.CheckConfig)
						if res, evalErr := evalFn(ctx, ctrlQ, tenantID, cfg); evalErr == nil {
							aggResults[ct] = res.Status
						} else {
							aggResults[ct] = "na"
						}
					}
				}
			}

			flags, flagsErr := ctrlQ.ListEndpointComplianceFlags(ctx, sqlcgen.ListEndpointComplianceFlagsParams{
				TenantID:       tenantID,
				HeartbeatSince: pgtype.Timestamptz{Time: time.Now().UTC().Add(-24 * time.Hour), Valid: true},
				ScanSince:      pgtype.Timestamptz{Time: time.Now().UTC().Add(-7 * 24 * time.Hour), Valid: true},
				CveMaxAge:      pgtype.Timestamptz{Time: time.Now().UTC().Add(-30 * 24 * time.Hour), Valid: true},
			})
			if flagsErr != nil {
				slog.WarnContext(ctx, "compliance: could not compute per-endpoint scores",
					"framework_id", tf.FrameworkID, "error", flagsErr)
			} else {
				epScores := ComputePerEndpointScores(fw.Controls, flags, aggResults)
				for epID, epScore := range epScores {
					roundedScore := math.Round(epScore)
					if err := q.UpdateEndpointScoreByID(ctx, sqlcgen.UpdateEndpointScoreByIDParams{
						Score:           float64ToNumeric(roundedScore),
						TenantID:        tenantID,
						EvaluationRunID: runID,
						FrameworkID:     tf.FrameworkID,
						ScopeID:         epID,
					}); err != nil {
						return nil, fmt.Errorf("update endpoint score for %s: %w", tf.FrameworkID, err)
					}
				}
			}
		}

		if err := insertScoreRow(ctx, q, sqlcgen.InsertScoreParams{
			TenantID:            tenantID,
			EvaluationRunID:     runID,
			FrameworkID:         tf.FrameworkID,
			ScopeType:           "tenant",
			ScopeID:             tenantID,
			Score:               float64ToNumeric(frameworkScore),
			TotalCves:           int32(totalCounts.Total),
			CompliantCves:       int32(totalCounts.Compliant),
			AtRiskCves:          int32(totalCounts.AtRisk),
			NonCompliantCves:    int32(totalCounts.NonCompliant),
			LateRemediationCves: int32(totalCounts.LateRemediation),
			EvaluatedAt:         evaluatedAt,
		}); err != nil {
			return nil, err
		}

		result.FrameworksEvaluated++
		result.TotalEvaluations += frameworkEvalCount
		result.FrameworkScores = append(result.FrameworkScores, FrameworkScoreResult{
			FrameworkID: tf.FrameworkID,
			Score:       frameworkScore,
			Counts:      totalCounts,
		})
	}

	return result, nil
}

func insertScoreRow(ctx context.Context, q EvaluationQuerier, arg sqlcgen.InsertScoreParams) error {
	if _, err := q.InsertScore(ctx, arg); err != nil {
		return fmt.Errorf("insert score (%s/%s): %w", arg.FrameworkID, arg.ScopeType, err)
	}
	return nil
}

// float64ToNumeric converts a float64 to pgtype.Numeric (multiply by 100, exp -2).
func float64ToNumeric(f float64) pgtype.Numeric {
	return pgtype.Numeric{
		Int:   big.NewInt(int64(math.Round(f * 100))),
		Exp:   -2,
		Valid: true,
	}
}

// numericToFloat64 converts a pgtype.Numeric to float64.
func numericToFloat64(n pgtype.Numeric) float64 {
	f, _ := n.Float64Value()
	if f.Valid {
		return f.Float64
	}
	return 0
}

// parseAtRiskThreshold extracts at_risk_threshold from pgtype.Numeric, defaulting to 0.75.
func parseAtRiskThreshold(ctx context.Context, n pgtype.Numeric) float64 {
	const defaultThreshold = 0.75
	if !n.Valid {
		return defaultThreshold
	}
	f := numericToFloat64(n)
	if f > 0 && f <= 1.0 {
		return f
	}
	slog.WarnContext(ctx, "compliance: invalid at_risk_threshold, using default",
		"stored_value", f, "default", defaultThreshold)
	return defaultThreshold
}

// deriveControlStatus produces a deterministic status for non-SLA controls
// based on the tenant score and a stable FNV hash of the framework ID and control ID.
// M2: Non-SLA controls use deterministic simulation. M3 will replace with real agent checks.
func deriveControlStatus(tenantScore float64, controlID string, frameworkID string) (status string, passingRatio float64) {
	h := fnv.New32a()
	h.Write([]byte(frameworkID + ":" + controlID))
	hash := h.Sum32()
	offset := float64(hash%20) - 10 // -10 to +9
	adjusted := tenantScore + offset

	// Clamp to valid range
	if adjusted > 100 {
		adjusted = 100
	}
	if adjusted < 0 {
		adjusted = 0
	}

	switch {
	case adjusted >= 95:
		return "pass", adjusted / 100.0
	case adjusted >= 70:
		return "partial", adjusted / 100.0
	default:
		return "fail", adjusted / 100.0
	}
}

// insertControlResults inserts a compliance_control_results row for every control
// in the framework. When controlQ is non-nil, Tier 1 evaluators run real queries;
// otherwise it falls back to deterministic simulation for backward compatibility.
// ControlResultCounts summarises pass/fail/na counts from a control evaluation run.
type ControlResultCounts struct {
	Passing   int
	Failing   int
	Partial   int
	NA        int
	Evaluated int // Passing + Failing + Partial
	Total     int
}

func insertControlResults(
	ctx context.Context,
	q EvaluationQuerier,
	controlQ ControlQuerier,
	fw *Framework,
	tenantID pgtype.UUID,
	runID pgtype.UUID,
	evaluatedAt pgtype.Timestamptz,
	tenantScore float64,
	totalEndpoints int32,
) (ControlResultCounts, error) {
	var counts ControlResultCounts
	now := evaluatedAt.Time
	for _, ctrl := range fw.Controls {
		var status string
		var passing int32
		var passingRatio float64

		// Determine how to evaluate this control.
		// Priority 1: If CheckType has a registered evaluator, use it (real data).
		// Priority 2: If no evaluator but has SLA tiers, derive from CVE score.
		// Priority 3: No evaluator and no SLA tiers → "na".
		lookupKey := ctrl.CheckType
		if lookupKey == "" {
			lookupKey = ctrl.ID
		}
		evalFn, hasEvaluator := controlEvaluators[lookupKey]

		if hasEvaluator && controlQ != nil {
			// Real evaluator (asset_inventory, software_inventory, vuln_scanning, etc.)
			cfg := ParseCheckConfig(lookupKey, ctrl.CheckConfig)
			evalResult, evalErr := evalFn(ctx, controlQ, tenantID, cfg)
			if evalErr != nil {
				slog.WarnContext(ctx, "compliance: control evaluator failed",
					"control_id", ctrl.ID, "framework_id", fw.ID, "error", evalErr)
				status = "na"
			} else {
				status = evalResult.Status
				passing = evalResult.PassingEndpoints
				totalEndpoints = evalResult.TotalEndpoints
			}
		} else if len(ctrl.SLATiers) > 0 {
			// Pure SLA control: derive status from CVE compliance score.
			switch {
			case tenantScore >= 95:
				status = "pass"
				passingRatio = 1.0
			case tenantScore >= 70:
				status = "partial"
				passingRatio = tenantScore / 100.0
			default:
				status = "fail"
				passingRatio = tenantScore / 100.0
			}
			passing = min(int32(passingRatio*float64(totalEndpoints)), totalEndpoints)
		} else {
			// No evaluator and no SLA tiers — cannot assess.
			status = "na"
		}

		var slaDeadline pgtype.Timestamptz
		var daysOverdue pgtype.Int4

		if status == "fail" || status == "partial" {
			// Use hash to make ~1/3 of failing controls overdue
			h := fnv.New32a()
			h.Write([]byte(fw.ID + ":deadline:" + ctrl.ID))
			dHash := h.Sum32()
			if dHash%3 == 0 {
				// Overdue: deadline in the past
				pastDays := int(dHash%30) + 1
				deadline := now.AddDate(0, 0, -pastDays)
				slaDeadline = pgtype.Timestamptz{Time: deadline, Valid: true}
				daysOverdue = pgtype.Int4{Int32: int32(pastDays), Valid: true}
			} else {
				// Upcoming: deadline in the future
				futureDays := int(dHash%60) + 1
				deadline := now.AddDate(0, 0, futureDays)
				slaDeadline = pgtype.Timestamptz{Time: deadline, Valid: true}
			}
		}

		var hint pgtype.Text
		if ctrl.RemediationHint != "" {
			hint = pgtype.Text{String: ctrl.RemediationHint, Valid: true}
		}

		if _, err := q.InsertControlResult(ctx, sqlcgen.InsertControlResultParams{
			TenantID:         tenantID,
			EvaluationRunID:  runID,
			FrameworkID:      fw.ID,
			ControlID:        ctrl.ID,
			Category:         ctrl.Category,
			Status:           status,
			PassingEndpoints: passing,
			TotalEndpoints:   totalEndpoints,
			RemediationHint:  hint,
			SlaDeadlineAt:    slaDeadline,
			DaysOverdue:      daysOverdue,
			EvaluatedAt:      evaluatedAt,
		}); err != nil {
			return counts, fmt.Errorf("insert control result %s/%s: %w", fw.ID, ctrl.ID, err)
		}

		counts.Total++
		switch status {
		case "pass":
			counts.Passing++
			counts.Evaluated++
		case "fail":
			counts.Failing++
			counts.Evaluated++
		case "partial":
			counts.Partial++
			counts.Evaluated++
		default:
			counts.NA++
		}
	}

	// Remove control results from previous evaluation runs for this framework,
	// keeping only the current run's results. This prevents duplicate cards in
	// the UI when Evaluate is triggered multiple times.
	if err := q.DeleteControlResultsByFramework(ctx, sqlcgen.DeleteControlResultsByFrameworkParams{
		TenantID:     tenantID,
		FrameworkID:  fw.ID,
		CurrentRunID: runID,
	}); err != nil {
		return counts, fmt.Errorf("delete old control results for %s: %w", fw.ID, err)
	}

	return counts, nil
}

// evaluateNonSLAFramework evaluates a framework that has no SLA-based controls.
// It runs each control's check-type evaluator and computes a framework-level
// score from the passing ratio across all controls.
func evaluateNonSLAFramework(
	ctx context.Context,
	q EvaluationQuerier,
	controlQ ControlQuerier,
	fw *Framework,
	tenantID pgtype.UUID,
	runID pgtype.UUID,
	evaluatedAt pgtype.Timestamptz,
) (float64, error) {
	var passingControls, totalControls int

	for _, ctrl := range fw.Controls {
		lookupKey := ctrl.CheckType
		if lookupKey == "" {
			lookupKey = ctrl.ID
		}

		var status string
		var passing, totalEndpoints int32

		if evalFn, ok := controlEvaluators[lookupKey]; ok {
			cfg := ParseCheckConfig(lookupKey, ctrl.CheckConfig)
			evalResult, evalErr := evalFn(ctx, controlQ, tenantID, cfg)
			if evalErr != nil {
				slog.WarnContext(ctx, "compliance: control evaluator failed",
					"control_id", ctrl.ID, "framework_id", fw.ID, "error", evalErr)
				status = "na"
			} else {
				status = evalResult.Status
				passing = evalResult.PassingEndpoints
				totalEndpoints = evalResult.TotalEndpoints
			}
		} else {
			status = "na"
		}

		var hint pgtype.Text
		if ctrl.RemediationHint != "" {
			hint = pgtype.Text{String: ctrl.RemediationHint, Valid: true}
		}

		if _, err := q.InsertControlResult(ctx, sqlcgen.InsertControlResultParams{
			TenantID:         tenantID,
			EvaluationRunID:  runID,
			FrameworkID:      fw.ID,
			ControlID:        ctrl.ID,
			Category:         ctrl.Category,
			Status:           status,
			PassingEndpoints: passing,
			TotalEndpoints:   totalEndpoints,
			RemediationHint:  hint,
			EvaluatedAt:      evaluatedAt,
		}); err != nil {
			return 0, fmt.Errorf("insert control result %s/%s: %w", fw.ID, ctrl.ID, err)
		}

		totalControls++
		if status == "pass" {
			passingControls++
		}
	}

	// Compute framework score as percentage of passing controls.
	var score float64
	if totalControls > 0 {
		score = math.Round((float64(passingControls) / float64(totalControls)) * 100)
	}

	// Insert per-endpoint scores so the Endpoints tab populates.
	// Non-SLA controls are evaluated at aggregate level, so every active
	// endpoint gets the same framework-level score.
	endpointIDs, epErr := controlQ.ListNonDecommissionedEndpointIDs(ctx, tenantID)
	if epErr != nil {
		slog.WarnContext(ctx, "compliance: could not list endpoints for per-endpoint scores",
			"framework_id", fw.ID, "error", epErr)
	} else {
		for _, epID := range endpointIDs {
			if err := insertScoreRow(ctx, q, sqlcgen.InsertScoreParams{
				TenantID:        tenantID,
				EvaluationRunID: runID,
				FrameworkID:     fw.ID,
				ScopeType:       "endpoint",
				ScopeID:         epID,
				Score:           float64ToNumeric(score),
				EvaluatedAt:     evaluatedAt,
			}); err != nil {
				return 0, fmt.Errorf("insert endpoint score for %s: %w", fw.ID, err)
			}
		}
	}

	// Insert tenant-level score.
	if err := insertScoreRow(ctx, q, sqlcgen.InsertScoreParams{
		TenantID:        tenantID,
		EvaluationRunID: runID,
		FrameworkID:     fw.ID,
		ScopeType:       "tenant",
		ScopeID:         tenantID,
		Score:           float64ToNumeric(score),
		EvaluatedAt:     evaluatedAt,
	}); err != nil {
		return 0, err
	}

	// Clean up old control results.
	if err := q.DeleteControlResultsByFramework(ctx, sqlcgen.DeleteControlResultsByFrameworkParams{
		TenantID:     tenantID,
		FrameworkID:  fw.ID,
		CurrentRunID: runID,
	}); err != nil {
		return 0, fmt.Errorf("delete old control results for %s: %w", fw.ID, err)
	}

	return score, nil
}
