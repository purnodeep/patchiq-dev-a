package v1_test

import (
	"testing"
	"time"

	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursorRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	id := "550e8400-e29b-41d4-a716-446655440000"

	encoded := v1.EncodeCursor(ts, id)
	assert.NotEmpty(t, encoded)

	gotTime, gotID, err := v1.DecodeCursor(encoded)
	require.NoError(t, err)
	assert.True(t, ts.Equal(gotTime))
	assert.Equal(t, id, gotID)
}

func TestDecodeCursor_empty(t *testing.T) {
	ts, id, err := v1.DecodeCursor("")
	require.NoError(t, err)
	assert.True(t, ts.IsZero())
	assert.Empty(t, id)
}

func TestDecodeCursor_invalid(t *testing.T) {
	_, _, err := v1.DecodeCursor("not-valid-base64-cursor")
	assert.Error(t, err)
}

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int32
	}{
		{"default when empty", "", 50},
		{"valid value", "25", 25},
		{"below minimum", "0", 1},
		{"above maximum", "500", 200},
		{"non-numeric", "abc", 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, v1.ParseLimit(tt.input))
		})
	}
}
