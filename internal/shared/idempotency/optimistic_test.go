package idempotency_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// simulateUpdate models the SQL pattern:
//
//	UPDATE deployments SET status = $new WHERE id = $id AND status = $old
//
// Returns 1 if the row matched (update succeeded) or 0 if it didn't (conflict).
func simulateUpdate(currentStatus, expectedOld, newStatus string) (rowsAffected int, newCurrent string) {
	if currentStatus == expectedOld {
		return 1, newStatus
	}
	return 0, currentStatus
}

func TestOptimisticLocking_SuccessfulTransition(t *testing.T) {
	current := "pending"

	rows, current := simulateUpdate(current, "pending", "running")
	assert.Equal(t, 1, rows, "should update when status matches")
	assert.Equal(t, "running", current)
}

func TestOptimisticLocking_DetectsConcurrentModification(t *testing.T) {
	current := "pending"

	// First writer transitions pending -> running
	rows, current := simulateUpdate(current, "pending", "running")
	assert.Equal(t, 1, rows)
	assert.Equal(t, "running", current)

	// Second writer also tries pending -> running (stale read)
	rows, current = simulateUpdate(current, "pending", "running")
	assert.Equal(t, 0, rows, "should detect concurrent modification: expected pending but found running")
	assert.Equal(t, "running", current, "status should remain unchanged")
}

func TestOptimisticLocking_SequentialTransitions(t *testing.T) {
	current := "pending"

	rows, current := simulateUpdate(current, "pending", "running")
	assert.Equal(t, 1, rows)

	rows, current = simulateUpdate(current, "running", "completed")
	assert.Equal(t, 1, rows)
	assert.Equal(t, "completed", current)

	// Can't go back
	rows, _ = simulateUpdate(current, "running", "failed")
	assert.Equal(t, 0, rows, "should not allow transition from completed via running")
}
