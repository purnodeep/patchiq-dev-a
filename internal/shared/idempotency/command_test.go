package idempotency_test

import (
	"context"
	"errors"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/idempotency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeduplicator_NewCommand(t *testing.T) {
	store := idempotency.NewMemoryCommandStore()
	d := idempotency.NewDeduplicator(store)
	ctx := context.Background()

	called := 0
	result, err := d.Execute(ctx, "cmd-001", func(ctx context.Context) (idempotency.CommandResult, error) {
		called++
		return idempotency.CommandResult{Data: []byte(`"ok"`)}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, called)
	assert.Equal(t, []byte(`"ok"`), result.Data)
	assert.Empty(t, result.Error)
}

func TestDeduplicator_DuplicateCommand(t *testing.T) {
	store := idempotency.NewMemoryCommandStore()
	d := idempotency.NewDeduplicator(store)
	ctx := context.Background()

	called := 0
	handler := func(ctx context.Context) (idempotency.CommandResult, error) {
		called++
		return idempotency.CommandResult{Data: []byte(`"first"`)}, nil
	}

	result1, err1 := d.Execute(ctx, "cmd-002", handler)
	require.NoError(t, err1)

	result2, err2 := d.Execute(ctx, "cmd-002", handler)
	require.NoError(t, err2)

	assert.Equal(t, 1, called, "handler must be called only once")
	assert.Equal(t, result1.Data, result2.Data)
}

func TestDeduplicator_DifferentCommands(t *testing.T) {
	store := idempotency.NewMemoryCommandStore()
	d := idempotency.NewDeduplicator(store)
	ctx := context.Background()

	called := 0
	handler := func(ctx context.Context) (idempotency.CommandResult, error) {
		called++
		return idempotency.CommandResult{Data: []byte(`"done"`)}, nil
	}

	_, err1 := d.Execute(ctx, "cmd-003", handler)
	require.NoError(t, err1)

	_, err2 := d.Execute(ctx, "cmd-004", handler)
	require.NoError(t, err2)

	assert.Equal(t, 2, called, "each distinct command_id must call the handler")
}

func TestDeduplicator_HandlerError(t *testing.T) {
	store := idempotency.NewMemoryCommandStore()
	d := idempotency.NewDeduplicator(store)
	ctx := context.Background()

	called := 0
	handler := func(ctx context.Context) (idempotency.CommandResult, error) {
		called++
		return idempotency.CommandResult{}, errors.New("install failed")
	}

	result1, err1 := d.Execute(ctx, "cmd-005", handler)
	assert.Error(t, err1, "first call must return the handler error")
	assert.Equal(t, "install failed", result1.Error)

	// Second call returns cached result with nil Go error.
	result2, err2 := d.Execute(ctx, "cmd-005", handler)
	require.NoError(t, err2, "second call must return nil error (error is in Error field)")
	assert.Equal(t, "install failed", result2.Error)
	assert.Equal(t, 1, called, "handler must not be called again")
}

// failingCommandStore is a CommandStore that returns errors on configured paths,
// used to test error propagation in Deduplicator.Execute.
type failingCommandStore struct {
	failHas  bool
	failMark bool
}

func (f *failingCommandStore) HasExecuted(_ context.Context, _ string) (idempotency.CommandResult, bool, error) {
	if f.failHas {
		return idempotency.CommandResult{}, false, errors.New("db locked")
	}
	return idempotency.CommandResult{}, false, nil
}

func (f *failingCommandStore) MarkExecuted(_ context.Context, _ string, _ idempotency.CommandResult) error {
	if f.failMark {
		return errors.New("db locked")
	}
	return nil
}

func TestDeduplicator_HasExecutedError(t *testing.T) {
	store := &failingCommandStore{failHas: true}
	d := idempotency.NewDeduplicator(store)
	ctx := context.Background()

	called := 0
	_, err := d.Execute(ctx, "cmd-006", func(ctx context.Context) (idempotency.CommandResult, error) {
		called++
		return idempotency.CommandResult{Data: []byte(`"ok"`)}, nil
	})

	require.Error(t, err, "HasExecuted error must be returned to the caller")
	assert.Equal(t, 0, called, "handler must not be called when HasExecuted errors")
}

func TestDeduplicator_MarkExecutedError(t *testing.T) {
	store := &failingCommandStore{failMark: true}
	d := idempotency.NewDeduplicator(store)
	ctx := context.Background()

	called := 0
	_, err := d.Execute(ctx, "cmd-007", func(ctx context.Context) (idempotency.CommandResult, error) {
		called++
		return idempotency.CommandResult{Data: []byte(`"ok"`)}, nil
	})

	require.Error(t, err, "MarkExecuted error must be returned to the caller")
	assert.Equal(t, 1, called, "handler was called before MarkExecuted failed")
}
