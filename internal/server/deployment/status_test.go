package deployment

import "testing"

func TestDeploymentStatuses_AllDefined(t *testing.T) {
	statuses := []DeploymentStatus{
		StatusCreated, StatusRunning, StatusCompleted, StatusFailed,
		StatusCancelled, StatusScheduled, StatusRollingBack, StatusRolledBack, StatusRollbackFailed,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty deployment status constant")
		}
	}
}

func TestWaveStatuses_AllDefined(t *testing.T) {
	statuses := []WaveStatus{
		WavePending, WaveRunning, WaveCompleted, WaveFailed, WaveCancelled,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty wave status constant")
		}
	}
}
