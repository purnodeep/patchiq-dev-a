package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/hub/workers"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// FeedQuerier defines the queries needed by FeedHandler.
type FeedQuerier interface {
	ListFeedSourcesWithSyncState(ctx context.Context) ([]sqlcgen.ListFeedSourcesWithSyncStateRow, error)
	GetFeedSourceByID(ctx context.Context, id pgtype.UUID) (sqlcgen.FeedSource, error)
	UpdateFeedSource(ctx context.Context, arg sqlcgen.UpdateFeedSourceParams) (sqlcgen.FeedSource, error)
	GetFeedSourceWithSyncStateByID(ctx context.Context, id pgtype.UUID) (sqlcgen.GetFeedSourceWithSyncStateByIDRow, error)
	ListFeedSyncHistory(ctx context.Context, arg sqlcgen.ListFeedSyncHistoryParams) ([]sqlcgen.FeedSyncHistory, error)
	CountFeedSyncHistory(ctx context.Context, feedSourceID pgtype.UUID) (int64, error)
	ListRecentFeedSyncStatus(ctx context.Context, feedSourceID pgtype.UUID) ([]sqlcgen.ListRecentFeedSyncStatusRow, error)
	GetFeedNewThisWeek(ctx context.Context, feedSourceID pgtype.UUID) (int64, error)
	GetFeedErrorRate(ctx context.Context, feedSourceID pgtype.UUID) (sqlcgen.GetFeedErrorRateRow, error)
}

// RiverEnqueuer abstracts the River client for testability.
type RiverEnqueuer interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// FeedHandler serves feed status endpoints.
type FeedHandler struct {
	queries     FeedQuerier
	riverClient RiverEnqueuer
	eventBus    domain.EventBus
}

// NewFeedHandler creates a new FeedHandler.
func NewFeedHandler(queries FeedQuerier, riverClient RiverEnqueuer, eventBus domain.EventBus) *FeedHandler {
	return &FeedHandler{queries: queries, riverClient: riverClient, eventBus: eventBus}
}

// recentSyncStatus represents a single sync event for sparkline display.
type recentSyncStatus struct {
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

// feedResponse is the JSON representation of a feed source with sync state.
type feedResponse struct {
	ID                  string             `json:"id"`
	Name                string             `json:"name"`
	DisplayName         string             `json:"display_name"`
	Enabled             bool               `json:"enabled"`
	SyncIntervalSeconds int32              `json:"sync_interval_seconds"`
	Url                 string             `json:"url"`
	AuthType            string             `json:"auth_type"`
	LastSyncAt          *time.Time         `json:"last_sync_at"`
	NextSyncAt          *time.Time         `json:"next_sync_at"`
	Status              string             `json:"status"`
	ErrorCount          int32              `json:"error_count"`
	LastError           *string            `json:"last_error"`
	EntriesIngested     int64              `json:"entries_ingested"`
	RecentHistory       []recentSyncStatus `json:"recent_history,omitempty"`
	NewThisWeek         int64              `json:"new_this_week,omitempty"`
	ErrorRate           float64            `json:"error_rate,omitempty"`
}

// List handles GET /api/v1/feeds.
func (h *FeedHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.ListFeedSourcesWithSyncState(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list feed sources", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list feed sources: internal error")
		return
	}

	feeds := make([]feedResponse, len(rows))
	for i, row := range rows {
		status := "never_synced"
		if row.Status.Valid {
			status = row.Status.String
		}
		f := feedResponse{
			ID:                  uuidToString(row.ID),
			Name:                row.Name,
			DisplayName:         row.DisplayName,
			Enabled:             row.Enabled,
			SyncIntervalSeconds: row.SyncIntervalSeconds,
			Url:                 row.Url,
			AuthType:            row.AuthType,
			Status:              status,
			ErrorCount:          row.ErrorCount.Int32,
			EntriesIngested:     row.EntriesIngested.Int64,
		}
		if row.LastSyncAt.Valid {
			t := row.LastSyncAt.Time.UTC()
			f.LastSyncAt = &t
		}
		if row.NextSyncAt.Valid {
			t := row.NextSyncAt.Time.UTC()
			f.NextSyncAt = &t
		}
		if row.LastError.Valid {
			f.LastError = &row.LastError.String
		}

		// Enrich with sparkline, new this week, error rate.
		// N+1 queries per feed — acceptable for small feed count (typically 6-10).
		recentHistory, err := h.queries.ListRecentFeedSyncStatus(r.Context(), row.ID)
		if err != nil {
			slog.ErrorContext(r.Context(), "list recent feed sync status", "feed_id", uuidToString(row.ID), "error", err)
		} else {
			items := make([]recentSyncStatus, len(recentHistory))
			for j, rh := range recentHistory {
				items[j] = recentSyncStatus{
					Status:    rh.Status,
					StartedAt: rh.StartedAt.Time,
				}
			}
			f.RecentHistory = items
		}

		newThisWeek, err := h.queries.GetFeedNewThisWeek(r.Context(), row.ID)
		if err != nil {
			slog.ErrorContext(r.Context(), "get feed new this week", "feed_id", uuidToString(row.ID), "error", err)
		} else {
			f.NewThisWeek = newThisWeek
		}

		errRate, err := h.queries.GetFeedErrorRate(r.Context(), row.ID)
		if err != nil {
			slog.ErrorContext(r.Context(), "get feed error rate", "feed_id", uuidToString(row.ID), "error", err)
		} else if errRate.TotalCount > 0 {
			f.ErrorRate = float64(errRate.FailedCount) / float64(errRate.TotalCount)
		}

		feeds[i] = f
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(feeds); err != nil {
		slog.ErrorContext(r.Context(), "encode feed list response", "error", err)
	}
}

// Get handles GET /api/v1/feeds/{id}.
func (h *FeedHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse feed id: %s", err))
		return
	}

	feed, err := h.queries.GetFeedSourceWithSyncStateByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "feed source not found")
			return
		}
		slog.ErrorContext(r.Context(), "get feed source with sync state", "feed_id", uuidToString(id), "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get feed source: internal error")
		return
	}

	var severityMapping map[string]string
	if len(feed.SeverityMapping) > 0 {
		if err := json.Unmarshal(feed.SeverityMapping, &severityMapping); err != nil {
			slog.ErrorContext(r.Context(), "unmarshal severity mapping", "error", err)
		}
	}

	resp := map[string]any{
		"id":                    uuidToString(feed.ID),
		"name":                  feed.Name,
		"display_name":          feed.DisplayName,
		"enabled":               feed.Enabled,
		"sync_interval_seconds": feed.SyncIntervalSeconds,
		"url":                   feed.Url,
		"auth_type":             feed.AuthType,
		"severity_filter":       feed.SeverityFilter,
		"os_filter":             feed.OsFilter,
		"severity_mapping":      severityMapping,
		"created_at":            feed.CreatedAt,
		"updated_at":            feed.UpdatedAt,
		"status":                feed.Status.String,
		"error_count":           feed.ErrorCount.Int32,
		"entries_ingested":      feed.EntriesIngested.Int64,
	}

	if feed.LastSyncAt.Valid {
		resp["last_sync_at"] = feed.LastSyncAt.Time.UTC()
	} else {
		resp["last_sync_at"] = nil
	}
	if feed.NextSyncAt.Valid {
		resp["next_sync_at"] = feed.NextSyncAt.Time.UTC()
	} else {
		resp["next_sync_at"] = nil
	}
	if feed.LastError.Valid {
		resp["last_error"] = feed.LastError.String
	} else {
		resp["last_error"] = nil
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(r.Context(), "encode get feed response", "error", err)
	}
}

// History handles GET /api/v1/feeds/{id}/history.
func (h *FeedHandler) History(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse feed id: %s", err))
		return
	}

	limit := queryParamInt(r, "limit", 20)
	limit = min(limit, 100)
	if limit < 1 {
		limit = 20
	}
	offset := queryParamInt(r, "offset", 0)

	runs, err := h.queries.ListFeedSyncHistory(r.Context(), sqlcgen.ListFeedSyncHistoryParams{
		FeedSourceID: id,
		Limit:        int32(limit),
		Offset:       int32(offset),
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "list feed sync history", "feed_id", uuidToString(id), "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list feed sync history: internal error")
		return
	}

	total, err := h.queries.CountFeedSyncHistory(r.Context(), id)
	if err != nil {
		slog.ErrorContext(r.Context(), "count feed sync history", "feed_id", uuidToString(id), "error", err)
		writeJSONError(w, http.StatusInternalServerError, "count feed sync history: internal error")
		return
	}

	if runs == nil {
		runs = []sqlcgen.FeedSyncHistory{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"runs":   runs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode feed history response", "error", err)
	}
}

type updateFeedRequest struct {
	Enabled             *bool             `json:"enabled"`
	SyncIntervalSeconds *int32            `json:"sync_interval_seconds"`
	Url                 *string           `json:"url"`
	AuthType            *string           `json:"auth_type"`
	SeverityFilter      []string          `json:"severity_filter"`
	OsFilter            []string          `json:"os_filter"`
	SeverityMapping     map[string]string `json:"severity_mapping"`
}

var validAuthTypes = map[string]bool{
	"none": true, "api_key": true, "bearer": true, "basic": true,
}

func (r *updateFeedRequest) validate() error {
	if r.SyncIntervalSeconds != nil && *r.SyncIntervalSeconds <= 0 {
		return fmt.Errorf("sync_interval_seconds must be positive")
	}
	if r.AuthType != nil && !validAuthTypes[*r.AuthType] {
		return fmt.Errorf("invalid auth_type %q: must be none, api_key, bearer, or basic", *r.AuthType)
	}
	if r.Url != nil && *r.Url != "" {
		if len(*r.Url) > 2048 {
			return fmt.Errorf("url exceeds maximum length of 2048 characters")
		}
	}
	return nil
}

// Update handles PUT /api/v1/feeds/{id}.
func (h *FeedHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse feed id: %s", err))
		return
	}

	var req updateFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode request body: %s", err))
		return
	}

	if err := req.validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	params := sqlcgen.UpdateFeedSourceParams{ID: id}
	if req.Enabled != nil {
		params.Enabled = pgtype.Bool{Bool: *req.Enabled, Valid: true}
	}
	if req.SyncIntervalSeconds != nil {
		params.SyncIntervalSeconds = pgtype.Int4{Int32: *req.SyncIntervalSeconds, Valid: true}
	}
	if req.Url != nil {
		params.Url = pgtype.Text{String: *req.Url, Valid: true}
	}
	if req.AuthType != nil {
		params.AuthType = pgtype.Text{String: *req.AuthType, Valid: true}
	}
	if req.SeverityFilter != nil {
		params.SeverityFilter = req.SeverityFilter
	}
	if req.OsFilter != nil {
		params.OsFilter = req.OsFilter
	}
	if req.SeverityMapping != nil {
		b, err := json.Marshal(req.SeverityMapping)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("marshal severity_mapping: %s", err))
			return
		}
		params.SeverityMapping = b
	}

	feed, err := h.queries.UpdateFeedSource(r.Context(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "feed source not found")
			return
		}
		slog.ErrorContext(r.Context(), "update feed source", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "update feed source: internal error")
		return
	}

	feedIDStr := uuidToString(feed.ID)
	if h.eventBus != nil {
		tenantID := tenant.MustTenantID(r.Context())
		evt := domain.NewSystemEvent(events.FeedSourceUpdated, tenantID, "feed_sources", feedIDStr, "update", feed)
		if err := h.eventBus.Emit(r.Context(), evt); err != nil {
			slog.ErrorContext(r.Context(), "emit feed.source_updated event", "error", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"id":                    feedIDStr,
		"name":                  feed.Name,
		"display_name":          feed.DisplayName,
		"enabled":               feed.Enabled,
		"sync_interval_seconds": feed.SyncIntervalSeconds,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode update feed response", "error", err)
	}
}

// TriggerSync handles POST /api/v1/feeds/{id}/sync.
func (h *FeedHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse feed id: %s", err))
		return
	}

	feed, err := h.queries.GetFeedSourceByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "feed source not found")
			return
		}
		slog.ErrorContext(r.Context(), "get feed source for sync trigger", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get feed source: internal error")
		return
	}

	_, err = h.riverClient.Insert(r.Context(), workers.FeedSyncJobArgs{FeedName: feed.Name}, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "enqueue feed sync job", "feed", feed.Name, "error", err)
		writeJSONError(w, http.StatusInternalServerError, "enqueue feed sync job: internal error")
		return
	}

	slog.InfoContext(r.Context(), "feed sync triggered", "feed", feed.Name, "feed_id", uuidToString(id))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":    "accepted",
		"feed_name": feed.Name,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode trigger sync response", "error", err)
	}
}
