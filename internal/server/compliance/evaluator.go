package compliance

import (
	"math"
	"time"
)

// EvalState represents the compliance state of a CVE.
type EvalState string

const (
	StateCompliant       EvalState = "COMPLIANT"
	StateAtRisk          EvalState = "AT_RISK"
	StateNonCompliant    EvalState = "NON_COMPLIANT"
	StateLateRemediation EvalState = "LATE_REMEDIATION"
)

// ComputeSLADeadline returns the SLA deadline for a CVE, or nil if slaDays is nil.
func ComputeSLADeadline(slaStartTime time.Time, slaDays *int) *time.Time {
	if slaDays == nil {
		return nil
	}
	deadline := slaStartTime.AddDate(0, 0, *slaDays)
	return &deadline
}

// DaysRemaining returns the number of days between now and the deadline.
// Positive means days left, negative means overdue.
func DaysRemaining(deadline time.Time, now time.Time) int {
	diff := deadline.Sub(now)
	return int(math.Round(diff.Hours() / 24))
}

// EvaluateCVEState determines the compliance state of a CVE based on SLA and remediation status.
func EvaluateCVEState(slaStartTime time.Time, slaDays *int, remediatedAt *time.Time, now time.Time, atRiskThreshold float64) EvalState {
	if slaDays == nil {
		return StateCompliant
	}

	deadline := ComputeSLADeadline(slaStartTime, slaDays)

	if remediatedAt != nil {
		if remediatedAt.Before(*deadline) || remediatedAt.Equal(*deadline) {
			return StateCompliant
		}
		return StateLateRemediation
	}

	// Not patched
	if now.After(*deadline) {
		return StateNonCompliant
	}

	// Within SLA — check at-risk threshold
	totalDuration := deadline.Sub(slaStartTime).Hours()
	if totalDuration <= 0 {
		return StateNonCompliant
	}
	elapsed := now.Sub(slaStartTime).Hours()
	fractionElapsed := elapsed / totalDuration

	if fractionElapsed >= atRiskThreshold {
		return StateAtRisk
	}

	return StateCompliant
}
