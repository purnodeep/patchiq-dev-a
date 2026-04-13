package compliance

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// ControlQuerier defines the database operations needed by Tier 1 control evaluators.
// The generated *sqlcgen.Queries struct satisfies this interface.
type ControlQuerier interface {
	CountActiveEndpoints(ctx context.Context, tenantID pgtype.UUID) (int64, error)
	CountEndpointsWithRecentInventory(ctx context.Context, arg sqlcgen.CountEndpointsWithRecentInventoryParams) (int64, error)
	CountEndpointsWithRecentHeartbeat(ctx context.Context, arg sqlcgen.CountEndpointsWithRecentHeartbeatParams) (int64, error)
	CountEndpointsWithHardwareInfo(ctx context.Context, tenantID pgtype.UUID) (int64, error)
	CountEndpointsWithKEVVulnerabilities(ctx context.Context, tenantID pgtype.UUID) (int64, error)
	CountEndpointsScannedForCVEs(ctx context.Context, tenantID pgtype.UUID) (int64, error)
	CountEndpointsWithStaleCriticalCVEs(ctx context.Context, arg sqlcgen.CountEndpointsWithStaleCriticalCVEsParams) (int64, error)
	CountEndpointsWithStaleCriticalOnlyCVEs(ctx context.Context, arg sqlcgen.CountEndpointsWithStaleCriticalOnlyCVEsParams) (int64, error)
	GetRecentDeploymentStats(ctx context.Context, arg sqlcgen.GetRecentDeploymentStatsParams) (sqlcgen.GetRecentDeploymentStatsRow, error)
	ListNonDecommissionedEndpointIDs(ctx context.Context, tenantID pgtype.UUID) ([]pgtype.UUID, error)
	ListEndpointComplianceFlags(ctx context.Context, arg sqlcgen.ListEndpointComplianceFlagsParams) ([]sqlcgen.ListEndpointComplianceFlagsRow, error)
}

// ControlEvalResult holds the outcome of a single control evaluation.
type ControlEvalResult struct {
	Status           string // "pass", "fail", "partial", "na"
	PassingEndpoints int32
	TotalEndpoints   int32
}

// ControlEvalFunc evaluates a single compliance control against real data.
type ControlEvalFunc func(ctx context.Context, q ControlQuerier, tenantID pgtype.UUID, config CheckConfig) (ControlEvalResult, error)

// controlEvaluators maps control IDs and check type keys to their evaluator functions.
var controlEvaluators = map[string]ControlEvalFunc{
	// By check type (for custom frameworks)
	"asset_inventory":           evalAssetInventory,
	"software_inventory":        evalSoftwareInventory,
	"vuln_scanning":             evalVulnScanning,
	"kev_compliance":            evalKEVCompliance,
	"deployment_governance":     evalDeploymentGovernance,
	"agent_monitoring":          evalAgentMonitoring,
	"critical_vuln_remediation": evalCriticalVulnRemediation,

	// By control ID (for built-in frameworks — backward compat)
	// Asset inventory
	"CIS-1.1":  evalAssetInventory,
	"ISO-A8.1": evalAssetInventory,

	// Software inventory
	"CIS-2.1": evalSoftwareInventory,

	// Vulnerability scanning
	"RA-5":       evalVulnScanning,
	"11.3.1":     evalVulnScanning,
	"SOC2-CC7.1": evalVulnScanning,

	// KEV / security management
	"HIPAA-164.308a1": evalKEVCompliance,

	// Deployment governance
	"CM-3":            evalDeploymentGovernance,
	"CIS-4.1":         evalDeploymentGovernance,
	"SOC2-CC8.1":      evalDeploymentGovernance,
	"ISO-A8.9":        evalDeploymentGovernance,
	"HIPAA-164.312c1": evalDeploymentGovernance,

	// Agent monitoring / health
	"HIPAA-164.312b": evalAgentMonitoring,
	"ISO-A8.16":      evalAgentMonitoring,
	"SOC2-A1.1":      evalAgentMonitoring,

	// Critical vuln remediation
	"ISO-A8.8": evalCriticalVulnRemediation,
}

// CheckTypeInfo describes an available check type for custom controls.
type CheckTypeInfo struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AvailableCheckTypes lists check types that can be assigned to custom controls.
var AvailableCheckTypes = []CheckTypeInfo{
	{Type: "sla", Name: "SLA Compliance", Description: "Evaluate patch remediation against CVSS-based SLA deadlines"},
	{Type: "asset_inventory", Name: "Asset Inventory", Description: "Verify endpoints are enrolled with hardware data and recent heartbeat"},
	{Type: "software_inventory", Name: "Software Inventory", Description: "Verify endpoints have completed package scans within 7 days"},
	{Type: "vuln_scanning", Name: "Vulnerability Scanning", Description: "Verify endpoints are covered by CVE vulnerability scanning"},
	{Type: "kev_compliance", Name: "CISA KEV Compliance", Description: "Verify no endpoints have CISA Known Exploited Vulnerabilities"},
	{Type: "deployment_governance", Name: "Deployment Governance", Description: "Verify patch deployments have acceptable success rates (>80%)"},
	{Type: "agent_monitoring", Name: "Agent Monitoring", Description: "Verify 95%+ of endpoints have recent agent heartbeats"},
	{Type: "critical_vuln_remediation", Name: "Critical Vulnerability Remediation", Description: "Verify no critical/high CVEs are unpatched for over 30 days"},
}

// evalAssetInventory evaluates CIS-1.1, ISO-A8.1: asset inventory completeness.
func evalAssetInventory(ctx context.Context, q ControlQuerier, tenantID pgtype.UUID, config CheckConfig) (ControlEvalResult, error) {
	total, err := q.CountActiveEndpoints(ctx, tenantID)
	if err != nil {
		return ControlEvalResult{}, fmt.Errorf("eval asset inventory: count active: %w", err)
	}
	if total == 0 {
		return ControlEvalResult{Status: "na"}, nil
	}

	counts := []int64{total}

	if config.IsEnabled("has_hardware_data") {
		hwCount, err := q.CountEndpointsWithHardwareInfo(ctx, tenantID)
		if err != nil {
			return ControlEvalResult{}, fmt.Errorf("eval asset inventory: count hardware: %w", err)
		}
		counts = append(counts, hwCount)
	}

	if config.IsEnabled("heartbeat_freshness") {
		hours := config.GetValue("heartbeat_freshness", 24)
		since := pgtype.Timestamptz{Time: time.Now().UTC().Add(-time.Duration(hours) * time.Hour), Valid: true}
		hbCount, err := q.CountEndpointsWithRecentHeartbeat(ctx, sqlcgen.CountEndpointsWithRecentHeartbeatParams{
			TenantID: tenantID,
			Since:    since,
		})
		if err != nil {
			return ControlEvalResult{}, fmt.Errorf("eval asset inventory: count heartbeat: %w", err)
		}
		counts = append(counts, hbCount)
	}

	passing := minSlice(counts)
	ratio := float64(passing) / float64(total)
	status := thresholdStatus(ratio, config.PassThreshold/100.0, config.PartialThreshold/100.0)

	return ControlEvalResult{
		Status:           status,
		PassingEndpoints: int32(passing),
		TotalEndpoints:   int32(total),
	}, nil
}

// evalSoftwareInventory evaluates CIS-2.1: software inventory scan recency.
func evalSoftwareInventory(ctx context.Context, q ControlQuerier, tenantID pgtype.UUID, config CheckConfig) (ControlEvalResult, error) {
	total, err := q.CountActiveEndpoints(ctx, tenantID)
	if err != nil {
		return ControlEvalResult{}, fmt.Errorf("eval software inventory: count active: %w", err)
	}
	if total == 0 {
		return ControlEvalResult{Status: "na"}, nil
	}

	scanDays := config.GetValue("scan_max_age", 7)
	since := pgtype.Timestamptz{Time: time.Now().UTC().Add(-time.Duration(scanDays) * 24 * time.Hour), Valid: true}
	scanned, err := q.CountEndpointsWithRecentInventory(ctx, sqlcgen.CountEndpointsWithRecentInventoryParams{
		TenantID: tenantID,
		Since:    since,
	})
	if err != nil {
		return ControlEvalResult{}, fmt.Errorf("eval software inventory: count inventory: %w", err)
	}

	ratio := float64(scanned) / float64(total)
	status := thresholdStatus(ratio, config.PassThreshold/100.0, config.PartialThreshold/100.0)

	return ControlEvalResult{
		Status:           status,
		PassingEndpoints: int32(scanned),
		TotalEndpoints:   int32(total),
	}, nil
}

// evalVulnScanning evaluates RA-5, 11.3.1, SOC2-CC7.1: vulnerability scanning coverage.
func evalVulnScanning(ctx context.Context, q ControlQuerier, tenantID pgtype.UUID, config CheckConfig) (ControlEvalResult, error) {
	total, err := q.CountActiveEndpoints(ctx, tenantID)
	if err != nil {
		return ControlEvalResult{}, fmt.Errorf("eval vuln scanning: count active: %w", err)
	}
	if total == 0 {
		return ControlEvalResult{Status: "na"}, nil
	}

	scanned, err := q.CountEndpointsScannedForCVEs(ctx, tenantID)
	if err != nil {
		return ControlEvalResult{}, fmt.Errorf("eval vuln scanning: count scanned: %w", err)
	}

	ratio := float64(scanned) / float64(total)
	status := thresholdStatus(ratio, config.PassThreshold/100.0, config.PartialThreshold/100.0)

	return ControlEvalResult{
		Status:           status,
		PassingEndpoints: int32(scanned),
		TotalEndpoints:   int32(total),
	}, nil
}

// evalKEVCompliance evaluates HIPAA-164.308a1: endpoints with CISA KEV vulnerabilities.
func evalKEVCompliance(ctx context.Context, q ControlQuerier, tenantID pgtype.UUID, config CheckConfig) (ControlEvalResult, error) {
	total, err := q.CountActiveEndpoints(ctx, tenantID)
	if err != nil {
		return ControlEvalResult{}, fmt.Errorf("eval KEV compliance: count active: %w", err)
	}
	if total == 0 {
		return ControlEvalResult{Status: "na"}, nil
	}

	kevCount, err := q.CountEndpointsWithKEVVulnerabilities(ctx, tenantID)
	if err != nil {
		return ControlEvalResult{}, fmt.Errorf("eval KEV compliance: count KEV: %w", err)
	}

	passing := total - kevCount
	if passing < 0 {
		passing = 0
	}

	ratio := float64(passing) / float64(total)
	status := thresholdStatus(ratio, config.PassThreshold/100.0, config.PartialThreshold/100.0)

	return ControlEvalResult{
		Status:           status,
		PassingEndpoints: int32(passing),
		TotalEndpoints:   int32(total),
	}, nil
}

// evalDeploymentGovernance evaluates CM-3, CIS-4.1, SOC2-CC8.1, ISO-A8.9, HIPAA-164.312c1.
// Based on deployment success rate over a configurable lookback period.
func evalDeploymentGovernance(ctx context.Context, q ControlQuerier, tenantID pgtype.UUID, config CheckConfig) (ControlEvalResult, error) {
	total, err := q.CountActiveEndpoints(ctx, tenantID)
	if err != nil {
		return ControlEvalResult{}, fmt.Errorf("eval deployment governance: count active: %w", err)
	}

	lookbackDays := config.GetValue("lookback_days", 30)
	since := pgtype.Timestamptz{Time: time.Now().UTC().Add(-time.Duration(lookbackDays) * 24 * time.Hour), Valid: true}
	stats, err := q.GetRecentDeploymentStats(ctx, sqlcgen.GetRecentDeploymentStatsParams{
		TenantID: tenantID,
		Since:    since,
	})
	if err != nil {
		return ControlEvalResult{}, fmt.Errorf("eval deployment governance: get stats: %w", err)
	}

	if stats.Total == 0 {
		return ControlEvalResult{
			Status:           "na",
			PassingEndpoints: 0,
			TotalEndpoints:   int32(total),
		}, nil
	}

	successRate := float64(stats.Succeeded) / float64(stats.Total)
	status := thresholdStatus(successRate, config.PassThreshold/100.0, config.PartialThreshold/100.0)

	// Approximate passing endpoints based on success rate
	passing := int32(successRate * float64(total))
	if passing > int32(total) {
		passing = int32(total)
	}

	return ControlEvalResult{
		Status:           status,
		PassingEndpoints: passing,
		TotalEndpoints:   int32(total),
	}, nil
}

// evalAgentMonitoring evaluates HIPAA-164.312b, ISO-A8.16, SOC2-A1.1: agent health.
func evalAgentMonitoring(ctx context.Context, q ControlQuerier, tenantID pgtype.UUID, config CheckConfig) (ControlEvalResult, error) {
	total, err := q.CountActiveEndpoints(ctx, tenantID)
	if err != nil {
		return ControlEvalResult{}, fmt.Errorf("eval agent monitoring: count active: %w", err)
	}
	if total == 0 {
		return ControlEvalResult{Status: "na"}, nil
	}

	hours := config.GetValue("heartbeat_freshness", 24)
	since := pgtype.Timestamptz{Time: time.Now().UTC().Add(-time.Duration(hours) * time.Hour), Valid: true}
	healthy, err := q.CountEndpointsWithRecentHeartbeat(ctx, sqlcgen.CountEndpointsWithRecentHeartbeatParams{
		TenantID: tenantID,
		Since:    since,
	})
	if err != nil {
		return ControlEvalResult{}, fmt.Errorf("eval agent monitoring: count heartbeat: %w", err)
	}

	ratio := float64(healthy) / float64(total)
	status := thresholdStatus(ratio, config.PassThreshold/100.0, config.PartialThreshold/100.0)

	return ControlEvalResult{
		Status:           status,
		PassingEndpoints: int32(healthy),
		TotalEndpoints:   int32(total),
	}, nil
}

// evalCriticalVulnRemediation evaluates ISO-A8.8: critical/high CVEs unpatched beyond a configurable window.
func evalCriticalVulnRemediation(ctx context.Context, q ControlQuerier, tenantID pgtype.UUID, config CheckConfig) (ControlEvalResult, error) {
	total, err := q.CountActiveEndpoints(ctx, tenantID)
	if err != nil {
		return ControlEvalResult{}, fmt.Errorf("eval critical vuln remediation: count active: %w", err)
	}
	if total == 0 {
		return ControlEvalResult{Status: "na"}, nil
	}

	maxAgeDays := config.GetValue("max_age_days", 30)
	maxAge := pgtype.Timestamptz{Time: time.Now().UTC().Add(-time.Duration(maxAgeDays) * 24 * time.Hour), Valid: true}

	var stale int64
	includeHigh := config.IsEnabled("include_high")
	if includeHigh {
		stale, err = q.CountEndpointsWithStaleCriticalCVEs(ctx, sqlcgen.CountEndpointsWithStaleCriticalCVEsParams{
			TenantID: tenantID,
			MaxAge:   maxAge,
		})
	} else {
		stale, err = q.CountEndpointsWithStaleCriticalOnlyCVEs(ctx, sqlcgen.CountEndpointsWithStaleCriticalOnlyCVEsParams{
			TenantID: tenantID,
			MaxAge:   maxAge,
		})
	}
	if err != nil {
		return ControlEvalResult{}, fmt.Errorf("eval critical vuln remediation: count stale: %w", err)
	}

	passing := total - stale
	if passing < 0 {
		passing = 0
	}

	ratio := float64(passing) / float64(total)
	status := thresholdStatus(ratio, config.PassThreshold/100.0, config.PartialThreshold/100.0)

	return ControlEvalResult{
		Status:           status,
		PassingEndpoints: int32(passing),
		TotalEndpoints:   int32(total),
	}, nil
}

// thresholdStatus returns "pass", "partial", or "fail" based on ratio thresholds.
func thresholdStatus(ratio, passThreshold, partialThreshold float64) string {
	switch {
	case ratio >= passThreshold:
		return "pass"
	case ratio >= partialThreshold:
		return "partial"
	default:
		return "fail"
	}
}

// minSlice returns the minimum value in a non-empty int64 slice.
func minSlice(s []int64) int64 {
	if len(s) == 0 {
		return 0
	}
	m := s[0]
	for _, v := range s[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// hasRegisteredEvaluator returns true if the check type has a real evaluator function.
func hasRegisteredEvaluator(checkType string) bool {
	_, ok := controlEvaluators[checkType]
	return ok
}

// endpointPassesCheck returns true if the given endpoint passes a specific check type
// based on its compliance flags.
func endpointPassesCheck(checkType string, flags sqlcgen.ListEndpointComplianceFlagsRow) bool {
	switch checkType {
	case "asset_inventory":
		return flags.HasHardware && flags.HasRecentHeartbeat
	case "software_inventory":
		return flags.HasRecentScan
	case "vuln_scanning":
		return flags.HasCveData
	case "kev_compliance":
		return flags.KevClean.Valid && flags.KevClean.Bool
	case "agent_monitoring":
		return flags.HasRecentHeartbeat
	case "critical_vuln_remediation":
		return flags.NoStaleCritical.Valid && flags.NoStaleCritical.Bool
	case "deployment_governance":
		// Deployment governance is aggregate — applies equally to all endpoints.
		// Caller should pass the aggregate result.
		return false
	default:
		return false
	}
}

// ComputePerEndpointScores computes individual endpoint scores based on which
// controls each endpoint passes. Returns a map of endpoint ID → score (0-100).
func ComputePerEndpointScores(
	controls []Control,
	flags []sqlcgen.ListEndpointComplianceFlagsRow,
	aggregateResults map[string]string, // control check_type → aggregate status ("pass"/"fail"/"na")
) map[pgtype.UUID]float64 {
	scores := make(map[pgtype.UUID]float64, len(flags))

	// Determine which controls are evaluated (not "na").
	type evalCtrl struct {
		checkType   string
		isAggregate bool // deployment_governance etc — same result for all endpoints
		aggPasses   bool
	}
	var evaluated []evalCtrl
	for _, ctrl := range controls {
		ct := ctrl.CheckType
		if ct == "" {
			ct = ctrl.ID
		}
		aggStatus, hasAgg := aggregateResults[ct]
		if hasAgg && aggStatus == "na" {
			continue // not evaluated
		}
		// Check if this is an aggregate-only check type
		isAgg := ct == "deployment_governance" || ct == "sla"
		if !hasAgg && !isAgg {
			// No evaluator result → check if we have a per-endpoint check for it
			if _, ok := controlEvaluators[ct]; !ok {
				continue // no evaluator at all
			}
		}
		evaluated = append(evaluated, evalCtrl{
			checkType:   ct,
			isAggregate: isAgg || (len(ctrl.SLATiers) > 0 && ct != "asset_inventory" && ct != "software_inventory" && ct != "vuln_scanning" && ct != "kev_compliance" && ct != "agent_monitoring" && ct != "critical_vuln_remediation"),
			aggPasses:   aggStatus == "pass",
		})
	}

	if len(evaluated) == 0 {
		for _, f := range flags {
			scores[f.EndpointID] = 0
		}
		return scores
	}

	for _, f := range flags {
		passing := 0
		for _, ec := range evaluated {
			if ec.isAggregate {
				if ec.aggPasses {
					passing++
				}
			} else {
				if endpointPassesCheck(ec.checkType, f) {
					passing++
				}
			}
		}
		scores[f.EndpointID] = float64(passing) / float64(len(evaluated)) * 100
	}

	return scores
}
