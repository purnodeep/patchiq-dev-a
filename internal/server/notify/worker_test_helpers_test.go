package notify_test

import (
	"context"

	"github.com/skenzeriq/patchiq/internal/server/notify"
)

// MockHistoryRecorder is a test double for HistoryRecorder.
type MockHistoryRecorder struct {
	Records []notify.HistoryRecord
	Err     error
}

func (m *MockHistoryRecorder) Record(_ context.Context, rec notify.HistoryRecord) error {
	m.Records = append(m.Records, rec)
	return m.Err
}
