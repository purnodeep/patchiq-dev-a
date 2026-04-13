package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// StatusInfo holds agent status information returned by GET /api/v1/status.
type StatusInfo struct {
	AgentID           string     `json:"agent_id"`
	Hostname          string     `json:"hostname"`
	OSFamily          string     `json:"os_family"`
	OSVersion         string     `json:"os_version"`
	AgentVersion      string     `json:"agent_version"`
	EnrollmentStatus  string     `json:"enrollment_status"`
	ServerURL         string     `json:"server_url"`
	LastHeartbeat     *time.Time `json:"last_heartbeat"`
	UptimeSeconds     int64      `json:"uptime_seconds"`
	PendingPatchCount int64      `json:"pending_patch_count"`
	InstalledCount    int64      `json:"installed_count"`
	FailedCount       int64      `json:"failed_count"`
	LastCollectionAt  *time.Time `json:"last_collection_at,omitempty"`
	CollectorCount    int        `json:"collector_count"`
	CollectionErrors  []string   `json:"collection_errors,omitempty"`
}

// MarshalJSON implements custom JSON marshaling so that last_heartbeat and
// last_collection_at serialize as RFC3339 strings or null (not the default
// time.Time format).
func (s StatusInfo) MarshalJSON() ([]byte, error) {
	type Alias StatusInfo

	var hb *string
	if s.LastHeartbeat != nil {
		v := s.LastHeartbeat.Format(time.RFC3339)
		hb = &v
	}

	var lca *string
	if s.LastCollectionAt != nil {
		v := s.LastCollectionAt.Format(time.RFC3339)
		lca = &v
	}

	return json.Marshal(struct {
		Alias
		LastHeartbeat    *string `json:"last_heartbeat"`
		LastCollectionAt *string `json:"last_collection_at,omitempty"`
	}{
		Alias:            Alias(s),
		LastHeartbeat:    hb,
		LastCollectionAt: lca,
	})
}

// StatusProvider retrieves the current agent status.
type StatusProvider interface {
	Status() StatusInfo
}

// staticStatusProvider returns a fixed StatusInfo (useful for testing).
type staticStatusProvider struct {
	info StatusInfo
}

func (p staticStatusProvider) Status() StatusInfo { return p.info }

// StaticStatusProvider returns a StatusProvider that always returns the given info.
func StaticStatusProvider(info StatusInfo) StatusProvider {
	return staticStatusProvider{info: info}
}

// StatusHandler serves GET /api/v1/status.
type StatusHandler struct {
	provider StatusProvider
}

// NewStatusHandler creates a StatusHandler with the given provider.
func NewStatusHandler(provider StatusProvider) *StatusHandler {
	return &StatusHandler{provider: provider}
}

// Get handles GET /api/v1/status.
func (h *StatusHandler) Get(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, h.provider.Status())
}
