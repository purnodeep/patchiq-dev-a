package v1_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	v1.WriteJSON(rec, http.StatusOK, map[string]string{"key": "value"})

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "value", body["key"])
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	v1.WriteError(rec, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "VALIDATION_ERROR", body["code"])
	assert.Equal(t, "name is required", body["message"])
}

func TestWriteList(t *testing.T) {
	rec := httptest.NewRecorder()
	items := []string{"a", "b"}
	v1.WriteList(rec, items, "next123", 42)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body["data"], 2)
	assert.Equal(t, "next123", body["next_cursor"])
	assert.Equal(t, float64(42), body["total_count"])
}

func TestWriteList_lastPage(t *testing.T) {
	rec := httptest.NewRecorder()
	v1.WriteList(rec, []string{}, "", 0)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Nil(t, body["next_cursor"])
	assert.Equal(t, float64(0), body["total_count"])
}
