package notify_test

import (
	"context"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/notify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockSender_RecordsCalls(t *testing.T) {
	mock := &notify.MockSender{}
	err := mock.Send(context.Background(), "slack://hook", "test message")
	require.NoError(t, err)
	assert.Equal(t, 1, len(mock.Calls))
	assert.Equal(t, "slack://hook", mock.Calls[0])
	assert.Equal(t, "test message", mock.Messages[0])
}

func TestMockSender_ReturnsConfiguredError(t *testing.T) {
	mock := &notify.MockSender{Err: assert.AnError}
	err := mock.Send(context.Background(), "slack://hook", "test")
	assert.ErrorIs(t, err, assert.AnError)
}
