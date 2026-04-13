package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// CVEQuerier defines the sqlc queries needed by CVEHandler.
type CVEQuerier interface {
	ListCVEsFiltered(ctx context.Context, arg sqlcgen.ListCVEsFilteredParams) ([]sqlcgen.ListCVEsFilteredRow, error)
	CountCVEsFiltered(ctx context.Context, arg sqlcgen.CountCVEsFilteredParams) (int64, error)
	GetCVEByID(ctx context.Context, arg sqlcgen.GetCVEByIDParams) (sqlcgen.CVE, error)
	ListAffectedEndpointsForCVE(ctx context.Context, arg sqlcgen.ListAffectedEndpointsForCVEParams) ([]sqlcgen.ListAffectedEndpointsForCVERow, error)
	CountAffectedEndpointsForCVE(ctx context.Context, arg sqlcgen.CountAffectedEndpointsForCVEParams) (int64, error)
	ListPatchesForCVEDetail(ctx context.Context, arg sqlcgen.ListPatchesForCVEDetailParams) ([]sqlcgen.ListPatchesForCVEDetailRow, error)
	CountCVEsBySeverity(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.CountCVEsBySeverityRow, error)
	CountCVEsKEV(ctx context.Context, tenantID pgtype.UUID) (int32, error)
	CountCVEsExploit(ctx context.Context, tenantID pgtype.UUID) (int32, error)
	ListRelatedCVEsForCVE(ctx context.Context, arg sqlcgen.ListRelatedCVEsForCVEParams) ([]sqlcgen.ListRelatedCVEsForCVERow, error)
}

// CVEHandler serves CVE REST API endpoints (read-only).
type CVEHandler struct {
	q CVEQuerier
}

// NewCVEHandler creates a CVEHandler.
func NewCVEHandler(q CVEQuerier) *CVEHandler {
	if q == nil {
		panic("cves: NewCVEHandler called with nil querier")
	}
	return &CVEHandler{q: q}
}

// List handles GET /api/v1/cves with pagination and filters.
func (h *CVEHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	cursorTime, cursorID, err := DecodeCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "invalid pagination cursor")
		return
	}
	limit := ParseLimit(r.URL.Query().Get("limit"))

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var cursorTS pgtype.Timestamptz
	var cursorUUID pgtype.UUID
	if !cursorTime.IsZero() {
		cursorTS = pgtype.Timestamptz{Time: cursorTime, Valid: true}
		cursorUUID, err = scanUUID(cursorID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "invalid pagination cursor: cursor ID is not a valid UUID")
			return
		}
	}

	var publishedAfter pgtype.Timestamptz
	if pa := r.URL.Query().Get("published_after"); pa != "" {
		t, parseErr := time.Parse(time.RFC3339, pa)
		if parseErr != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_FILTER", "published_after must be RFC3339 format (e.g. 2024-01-01T00:00:00Z)")
			return
		}
		publishedAfter = pgtype.Timestamptz{Time: t, Valid: true}
	}

	params := sqlcgen.ListCVEsFilteredParams{
		TenantID:         tid,
		Severity:         r.URL.Query().Get("severity"),
		CisaKev:          r.URL.Query().Get("cisa_kev"),
		ExploitAvailable: r.URL.Query().Get("exploit_available"),
		AttackVector:     r.URL.Query().Get("attack_vector"),
		Search:           r.URL.Query().Get("search"),
		PublishedAfter:   publishedAfter,
		HasPatch:         r.URL.Query().Get("has_patch"),
		CursorCreatedAt:  cursorTS,
		CursorID:         cursorUUID,
		PageLimit:        limit,
	}

	cves, err := h.q.ListCVEsFiltered(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "list cves", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list CVEs")
		return
	}

	countParams := sqlcgen.CountCVEsFilteredParams{
		TenantID:         tid,
		Severity:         params.Severity,
		CisaKev:          params.CisaKev,
		ExploitAvailable: params.ExploitAvailable,
		AttackVector:     params.AttackVector,
		Search:           params.Search,
		PublishedAfter:   params.PublishedAfter,
		HasPatch:         params.HasPatch,
	}
	total, err := h.q.CountCVEsFiltered(ctx, countParams)
	if err != nil {
		slog.ErrorContext(ctx, "count cves", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count CVEs")
		return
	}

	var nextCursor string
	if len(cves) == int(limit) {
		last := cves[len(cves)-1]
		nextCursor = EncodeCursor(last.CreatedAt.Time, uuidToString(last.ID))
	}

	WriteList(w, cves, nextCursor, total)
}

// cveSummaryResponse is the JSON body for GET /cves/summary.
type cveSummaryResponse struct {
	Total        int            `json:"total"`
	BySeverity   map[string]int `json:"by_severity"`
	KEVCount     int            `json:"kev_count"`
	ExploitCount int            `json:"exploit_count"`
}

// cveDetailResponse is the JSON body for GET /cves/{id}.
type cveDetailResponse struct {
	ID                 string                `json:"id"`
	TenantID           string                `json:"tenant_id"`
	CveID              string                `json:"cve_id"`
	Severity           string                `json:"severity"`
	Description        *string               `json:"description"`
	PublishedAt        string                `json:"published_at"`
	CreatedAt          string                `json:"created_at"`
	UpdatedAt          string                `json:"updated_at"`
	CvssV3Score        *float64              `json:"cvss_v3_score"`
	CvssV3Vector       *string               `json:"cvss_v3_vector"`
	CisaKevDueDate     *string               `json:"cisa_kev_due_date"`
	ExploitAvailable   bool                  `json:"exploit_available"`
	NvdLastModified    string                `json:"nvd_last_modified"`
	AttackVector       *string               `json:"attack_vector"`
	CweID              *string               `json:"cwe_id,omitempty"`
	Source             string                `json:"source"`
	ExternalReferences []externalRefResp     `json:"external_references"`
	AffectedEndpoints  affectedEndpointsResp `json:"affected_endpoints"`
	Patches            []cvePatchResp        `json:"patches"`
	RelatedCVEs        []relatedCVEResp      `json:"related_cves"`
}

type relatedCVEResp struct {
	ID          string   `json:"id"`
	CveID       string   `json:"cve_id"`
	Severity    string   `json:"severity"`
	CvssV3Score *float64 `json:"cvss_v3_score"`
}

type externalRefResp struct {
	URL    string `json:"url"`
	Source string `json:"source"`
}

type affectedEndpointsResp struct {
	Count   int64                  `json:"count"`
	Items   []affectedEndpointItem `json:"items"`
	HasMore bool                   `json:"has_more"`
}

type affectedEndpointItem struct {
	ID           string  `json:"id"`
	Hostname     string  `json:"hostname"`
	OsFamily     string  `json:"os_family"`
	OsVersion    string  `json:"os_version"`
	IpAddress    *string `json:"ip_address"`
	Status       string  `json:"status"`
	AgentVersion *string `json:"agent_version"`
	LastSeen     *string `json:"last_seen"`
	GroupNames   *string `json:"group_names"`
}

type cvePatchResp struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Version          string `json:"version"`
	Severity         string `json:"severity"`
	OsFamily         string `json:"os_family"`
	ReleasedAt       string `json:"released_at"`
	EndpointsCovered int32  `json:"endpoints_covered"`
	EndpointsPatched int32  `json:"endpoints_patched"`
}

// Summary handles GET /api/v1/cves/summary.
func (h *CVEHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	severityRows, err := h.q.CountCVEsBySeverity(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "count cves by severity", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count CVEs by severity")
		return
	}

	kevCount, err := h.q.CountCVEsKEV(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "count cves kev", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count KEV CVEs")
		return
	}

	exploitCount, err := h.q.CountCVEsExploit(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "count cves exploit", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count exploit CVEs")
		return
	}

	bySeverity := make(map[string]int, len(severityRows))
	total := 0
	for _, row := range severityRows {
		bySeverity[row.Severity] = int(row.Count)
		total += int(row.Count)
	}

	WriteJSON(w, http.StatusOK, cveSummaryResponse{
		Total:        total,
		BySeverity:   bySeverity,
		KEVCount:     int(kevCount),
		ExploitCount: int(exploitCount),
	})
}

// Get handles GET /api/v1/cves/{id}.
func (h *CVEHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid CVE ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	cve, err := h.q.GetCVEByID(ctx, sqlcgen.GetCVEByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "CVE not found")
			return
		}
		slog.ErrorContext(ctx, "get cve", "cve_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get CVE")
		return
	}

	affected, err := h.q.ListAffectedEndpointsForCVE(ctx, sqlcgen.ListAffectedEndpointsForCVEParams{CveID: id, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "list affected endpoints for cve", "cve_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list affected endpoints")
		return
	}

	affectedCount, err := h.q.CountAffectedEndpointsForCVE(ctx, sqlcgen.CountAffectedEndpointsForCVEParams{CveID: id, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "count affected endpoints for cve", "cve_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count affected endpoints")
		return
	}

	patches, err := h.q.ListPatchesForCVEDetail(ctx, sqlcgen.ListPatchesForCVEDetailParams{CveID: id, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "list patches for cve", "cve_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list patches for CVE")
		return
	}

	items := make([]affectedEndpointItem, len(affected))
	for i, a := range affected {
		var lastSeen *string
		if a.LastSeen.Valid {
			s := a.LastSeen.Time.Format(time.RFC3339)
			lastSeen = &s
		}
		var groupNames *string
		if len(a.GroupNames) > 0 {
			s := string(a.GroupNames)
			groupNames = &s
		}
		items[i] = affectedEndpointItem{
			ID:           uuidToString(a.ID),
			Hostname:     a.Hostname,
			OsFamily:     a.OsFamily,
			OsVersion:    a.OsVersion,
			IpAddress:    nullableText(a.IpAddress),
			Status:       a.Status,
			AgentVersion: nullableText(a.AgentVersion),
			LastSeen:     lastSeen,
			GroupNames:   groupNames,
		}
	}

	patchItems := make([]cvePatchResp, len(patches))
	for i, p := range patches {
		patchItems[i] = cvePatchResp{
			ID:               uuidToString(p.ID),
			Name:             p.Name,
			Version:          p.Version,
			Severity:         p.Severity,
			OsFamily:         p.OsFamily,
			ReleasedAt:       p.CreatedAt.Time.Format(time.RFC3339),
			EndpointsCovered: p.EndpointsCovered,
			EndpointsPatched: p.EndpointsPatched,
		}
	}

	relatedRows, err := h.q.ListRelatedCVEsForCVE(ctx, sqlcgen.ListRelatedCVEsForCVEParams{CveID: id, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "list related cves", "cve_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		// Non-fatal — continue with empty related CVEs
		relatedRows = nil
	}

	relatedCVEs := make([]relatedCVEResp, len(relatedRows))
	for i, rc := range relatedRows {
		var score *float64
		if fv, err := rc.CvssV3Score.Float64Value(); err == nil && fv.Valid {
			score = &fv.Float64
		}
		relatedCVEs[i] = relatedCVEResp{
			ID:          uuidToString(rc.ID),
			CveID:       rc.CveID,
			Severity:    rc.Severity,
			CvssV3Score: score,
		}
	}

	var cvssScore *float64
	fv, convErr := cve.CvssV3Score.Float64Value()
	if convErr != nil {
		slog.ErrorContext(ctx, "convert cvss_v3_score", "cve_id", chi.URLParam(r, "id"), "error", convErr)
	} else if fv.Valid {
		cvssScore = &fv.Float64
	}

	var kevDueDate *string
	if cve.CisaKevDueDate.Valid {
		s := cve.CisaKevDueDate.Time.Format("2006-01-02")
		kevDueDate = &s
	}

	var refs []externalRefResp
	if len(cve.ExternalReferences) > 0 {
		if err := json.Unmarshal(cve.ExternalReferences, &refs); err != nil {
			slog.ErrorContext(ctx, "unmarshal external references", "cve_id", chi.URLParam(r, "id"), "error", err)
		}
	}
	if refs == nil {
		refs = []externalRefResp{}
	}

	nvdModified := ""
	if cve.NvdLastModified.Valid && !cve.NvdLastModified.Time.IsZero() {
		nvdModified = cve.NvdLastModified.Time.Format(time.RFC3339)
	}

	resp := cveDetailResponse{
		ID:                 uuidToString(cve.ID),
		TenantID:           uuidToString(cve.TenantID),
		CveID:              cve.CveID,
		Severity:           cve.Severity,
		Description:        nullableText(cve.Description),
		PublishedAt:        cve.PublishedAt.Time.Format(time.RFC3339),
		CreatedAt:          cve.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:          cve.UpdatedAt.Time.Format(time.RFC3339),
		CvssV3Score:        cvssScore,
		CvssV3Vector:       nullableText(cve.CvssV3Vector),
		CisaKevDueDate:     kevDueDate,
		ExploitAvailable:   cve.ExploitAvailable,
		NvdLastModified:    nvdModified,
		AttackVector:       nullableText(cve.AttackVector),
		CweID:              nullableText(cve.CweID),
		Source:             cve.Source,
		ExternalReferences: refs,
		AffectedEndpoints: affectedEndpointsResp{
			Count:   affectedCount,
			Items:   items,
			HasMore: affectedCount > int64(len(affected)),
		},
		Patches:     patchItems,
		RelatedCVEs: relatedCVEs,
	}

	WriteJSON(w, http.StatusOK, resp)
}
