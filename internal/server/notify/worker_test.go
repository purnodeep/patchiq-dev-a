package notify_test

import (
	"context"
	"testing"

	"github.com/riverqueue/river"
	"github.com/skenzeriq/patchiq/internal/server/notify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendWorker_Success(t *testing.T) {
	mock := &notify.MockSender{}
	recorder := &MockHistoryRecorder{}
	worker := notify.NewSendWorker(mock, recorder, nil)

	args := notify.SendJobArgs{
		TenantID:    "tenant-1",
		UserID:      "user-1",
		TriggerType: "deployment.failed",
		ChannelID:   "chan-1",
		ShoutrrrURL: "slack://token@channel",
		Message:     "[PatchIQ] Deployment abc failed: timeout",
	}
	job := &river.Job[notify.SendJobArgs]{Args: args}

	err := worker.Work(context.Background(), job)
	require.NoError(t, err)

	require.Equal(t, 1, len(mock.Calls))
	assert.Equal(t, "slack://token@channel", mock.Calls[0])

	require.Equal(t, 1, len(recorder.Records))
	assert.Equal(t, "sent", recorder.Records[0].Status)
}

func TestSendWorker_RecordsChannelType(t *testing.T) {
	mock := &notify.MockSender{}
	recorder := &MockHistoryRecorder{}
	worker := notify.NewSendWorker(mock, recorder, nil)

	args := notify.SendJobArgs{
		TenantID:    "tenant-1",
		UserID:      "user-1",
		TriggerType: "deployment.failed",
		ChannelID:   "chan-1",
		ShoutrrrURL: "mailto://user:pass@example.com",
		Message:     "test",
		ChannelType: "email",
		Recipient:   "ops@example.com",
		Subject:     "[PatchIQ] Deployment failed",
	}
	job := &river.Job[notify.SendJobArgs]{Args: args}

	err := worker.Work(context.Background(), job)
	require.NoError(t, err)

	require.Equal(t, 1, len(recorder.Records))
	assert.Equal(t, "email", recorder.Records[0].ChannelType)
	assert.Equal(t, "ops@example.com", recorder.Records[0].Recipient)
	assert.Equal(t, "[PatchIQ] Deployment failed", recorder.Records[0].Subject)
}

func TestSendWorker_Failure(t *testing.T) {
	mock := &notify.MockSender{Err: assert.AnError}
	recorder := &MockHistoryRecorder{}
	worker := notify.NewSendWorker(mock, recorder, nil)

	args := notify.SendJobArgs{
		TenantID:    "tenant-1",
		UserID:      "user-1",
		TriggerType: "deployment.failed",
		ChannelID:   "chan-1",
		ShoutrrrURL: "slack://token@channel",
		Message:     "test",
	}
	job := &river.Job[notify.SendJobArgs]{Args: args}

	err := worker.Work(context.Background(), job)
	assert.Error(t, err)

	require.Equal(t, 1, len(recorder.Records))
	assert.Equal(t, "failed", recorder.Records[0].Status)
}

func TestSendWorker_RecorderFailure(t *testing.T) {
	mock := &notify.MockSender{}
	recorder := &MockHistoryRecorder{Err: assert.AnError}
	worker := notify.NewSendWorker(mock, recorder, nil)

	args := notify.SendJobArgs{
		TenantID:    "tenant-1",
		UserID:      "user-1",
		TriggerType: "deployment.failed",
		ChannelID:   "chan-1",
		ShoutrrrURL: "slack://token@channel",
		Message:     "test",
	}
	job := &river.Job[notify.SendJobArgs]{Args: args}

	err := worker.Work(context.Background(), job)
	// Recorder failures are best-effort and must not propagate as a Work error.
	require.NoError(t, err)
}
