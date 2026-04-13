package v1

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/skenzeriq/patchiq/internal/server/cve"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// EndpointScanner creates scan commands for a single endpoint.
type EndpointScanner interface {
	ScanSingle(ctx context.Context, endpointID, tenantID pgtype.UUID, actorID, actorType string) (pgtype.UUID, error)
}

// EndpointQuerier defines the sqlc queries needed by EndpointHandler.
type EndpointQuerier interface {
	ListEndpoints(ctx context.Context, arg sqlcgen.ListEndpointsParams) ([]sqlcgen.ListEndpointsRow, error)
	CountEndpoints(ctx context.Context, arg sqlcgen.CountEndpointsParams) (int64, error)
	ListEndpointsForExport(ctx context.Context, arg sqlcgen.ListEndpointsForExportParams) ([]sqlcgen.ListEndpointsForExportRow, error)
	GetEndpointByID(ctx context.Context, arg sqlcgen.GetEndpointByIDParams) (sqlcgen.GetEndpointByIDRow, error)
	UpdateEndpoint(ctx context.Context, arg sqlcgen.UpdateEndpointParams) (sqlcgen.Endpoint, error)
	SoftDeleteEndpoint(ctx context.Context, arg sqlcgen.SoftDeleteEndpointParams) (sqlcgen.Endpoint, error)
	GetLatestEndpointInventory(ctx context.Context, arg sqlcgen.GetLatestEndpointInventoryParams) (sqlcgen.EndpointInventory, error)
	CountEndpointCVEsByStatus(ctx context.Context, arg sqlcgen.CountEndpointCVEsByStatusParams) ([]sqlcgen.CountEndpointCVEsByStatusRow, error)
	ListEndpointCVEsAffected(ctx context.Context, arg sqlcgen.ListEndpointCVEsAffectedParams) ([]sqlcgen.ListEndpointCVEsAffectedRow, error)
	ListEndpointNetworkInterfaces(ctx context.Context, arg sqlcgen.ListEndpointNetworkInterfacesParams) ([]sqlcgen.EndpointNetworkInterface, error)
	ListEndpointPackagesByEndpoint(ctx context.Context, arg sqlcgen.ListEndpointPackagesByEndpointParams) ([]sqlcgen.ListEndpointPackagesByEndpointRow, error)
	ListDeploymentTargetsByEndpoint(ctx context.Context, arg sqlcgen.ListDeploymentTargetsByEndpointParams) ([]sqlcgen.DeploymentTarget, error)
	ListPatchesForEndpoint(ctx context.Context, arg sqlcgen.ListPatchesForEndpointParams) ([]sqlcgen.ListPatchesForEndpointRow, error)
	ListAvailablePatchesForEndpointByOS(ctx context.Context, arg sqlcgen.ListAvailablePatchesForEndpointByOSParams) ([]sqlcgen.ListAvailablePatchesForEndpointByOSRow, error)
	ListTagsForEndpoint(ctx context.Context, arg sqlcgen.ListTagsForEndpointParams) ([]sqlcgen.Tag, error)
	ListAvailablePatchesForEndpointByPackage(ctx context.Context, arg sqlcgen.ListAvailablePatchesForEndpointByPackageParams) ([]sqlcgen.ListAvailablePatchesForEndpointByPackageRow, error)
	ListAuditEventsByEndpoint(ctx context.Context, arg sqlcgen.ListAuditEventsByEndpointParams) ([]sqlcgen.AuditEvent, error)
	CountAuditEventsByEndpoint(ctx context.Context, arg sqlcgen.CountAuditEventsByEndpointParams) (int64, error)
	GetActiveRunScanByAgent(ctx context.Context, arg sqlcgen.GetActiveRunScanByAgentParams) (sqlcgen.Command, error)
}

// CVEMatchInserter enqueues CVE endpoint match jobs.
type CVEMatchInserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// EndpointHandler serves endpoint REST API endpoints.
type EndpointHandler struct {
	q                EndpointQuerier
	eventBus         domain.EventBus
	scanScheduler    EndpointScanner
	cveMatchInserter CVEMatchInserter
}

// NewEndpointHandler creates an EndpointHandler.
func NewEndpointHandler(q EndpointQuerier, eventBus domain.EventBus, scanScheduler EndpointScanner) *EndpointHandler {
	if q == nil {
		panic("endpoints: NewEndpointHandler called with nil querier")
	}
	if eventBus == nil {
		panic("endpoints: NewEndpointHandler called with nil eventBus")
	}
	if scanScheduler == nil {
		panic("endpoints: NewEndpointHandler called with nil scanScheduler")
	}
	return &EndpointHandler{q: q, eventBus: eventBus, scanScheduler: scanScheduler}
}

// SetCVEMatchInserter configures the handler for CVE scan job enqueuing.
func (h *EndpointHandler) SetCVEMatchInserter(inserter CVEMatchInserter) {
	h.cveMatchInserter = inserter
}

// ScanCVEs handles POST /api/v1/endpoints/{id}/scan-cves.
func (h *EndpointHandler) ScanCVEs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid endpoint ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	if h.cveMatchInserter == nil {
		WriteError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "CVE scanning is not configured")
		return
	}

	// Verify endpoint exists.
	_, err = h.q.GetEndpointByID(ctx, sqlcgen.GetEndpointByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found")
			return
		}
		slog.ErrorContext(ctx, "get endpoint for CVE scan", "endpoint_id", chi.URLParam(r, "id"), "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get endpoint")
		return
	}

	_, err = h.cveMatchInserter.Insert(ctx, cve.EndpointMatchJobArgs{
		TenantID:   tenantID,
		EndpointID: uuidToString(id),
	}, nil)
	if err != nil {
		slog.ErrorContext(ctx, "enqueue CVE match job", "endpoint_id", chi.URLParam(r, "id"), "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to enqueue CVE scan")
		return
	}

	WriteJSON(w, http.StatusAccepted, map[string]string{"status": "cve_scan_requested"})
}

// List handles GET /api/v1/endpoints with pagination and filters.
func (h *EndpointHandler) List(w http.ResponseWriter, r *http.Request) {
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

	var tagID pgtype.UUID
	if tidParam := r.URL.Query().Get("tag_id"); tidParam != "" {
		var tidErr error
		tagID, tidErr = scanUUID(tidParam)
		if tidErr != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_TAG_ID", "invalid tag_id: not a valid UUID")
			return
		}
	}

	params := sqlcgen.ListEndpointsParams{
		TenantID:        tid,
		Status:          r.URL.Query().Get("status"),
		OsFamily:        r.URL.Query().Get("os_family"),
		Search:          r.URL.Query().Get("search"),
		TagID:           tagID,
		CursorCreatedAt: cursorTS,
		CursorID:        cursorUUID,
		PageLimit:       limit,
	}

	rows, err := h.q.ListEndpoints(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "list endpoints", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list endpoints")
		return
	}

	countParams := sqlcgen.CountEndpointsParams{
		TenantID: tid,
		Status:   params.Status,
		OsFamily: params.OsFamily,
		Search:   params.Search,
		TagID:    params.TagID,
	}
	total, err := h.q.CountEndpoints(ctx, countParams)
	if err != nil {
		slog.ErrorContext(ctx, "count endpoints", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count endpoints")
		return
	}

	var nextCursor string
	if len(rows) == int(limit) {
		last := rows[len(rows)-1]
		nextCursor = EncodeCursor(last.CreatedAt.Time, uuidToString(last.ID))
	}

	items := make([]endpointListItem, len(rows))
	for i, row := range rows {
		item := endpointListItem{
			ID:                  uuidToString(row.ID),
			Hostname:            row.Hostname,
			OsFamily:            row.OsFamily,
			OsVersion:           row.OsVersion,
			Status:              row.Status,
			CreatedAt:           row.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:           row.UpdatedAt.Time.Format(time.RFC3339),
			CveCount:            row.CveCount,
			CriticalCveCount:    row.CriticalCveCount,
			HighCveCount:        row.HighCveCount,
			MediumCveCount:      row.MediumCveCount,
			PendingPatchesCount: row.PendingPatchesCount,
			CriticalPatchCount:  row.CriticalPatchCount,
			HighPatchCount:      row.HighPatchCount,
			MediumPatchCount:    row.MediumPatchCount,
		}
		if row.AgentVersion.Valid {
			item.AgentVersion = &row.AgentVersion.String
		}
		if row.LastSeen.Valid {
			s := row.LastSeen.Time.Format(time.RFC3339)
			item.LastSeen = &s
		}
		if row.IpAddress.Valid {
			item.IpAddress = &row.IpAddress.String
		}
		if row.Arch.Valid {
			item.Arch = &row.Arch.String
		}
		if row.KernelVersion.Valid {
			item.KernelVersion = &row.KernelVersion.String
		}
		if row.CompliancePct.Valid {
			f := row.CompliancePct.Float64
			item.CompliancePct = &f
		}
		item.Tags = tagsToJSON(row.Tags)
		if row.CpuCores.Valid {
			v := row.CpuCores.Int32
			item.CpuCores = &v
		}
		if row.CpuUsagePercent.Valid {
			v := row.CpuUsagePercent.Int16
			item.CpuUsagePercent = &v
		}
		if row.MemoryTotalMb.Valid {
			v := row.MemoryTotalMb.Int64
			item.MemoryTotalMB = &v
		}
		if row.MemoryUsedMb.Valid {
			v := row.MemoryUsedMb.Int64
			item.MemoryUsedMB = &v
		}
		if row.DiskTotalGb.Valid {
			v := row.DiskTotalGb.Int64
			item.DiskTotalGB = &v
		}
		if row.DiskUsedGb.Valid {
			v := row.DiskUsedGb.Int64
			item.DiskUsedGB = &v
		}
		items[i] = item
	}

	WriteList(w, items, nextCursor, total)
}

// Get handles GET /api/v1/endpoints/{id}.
func (h *EndpointHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid endpoint ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	ep, err := h.q.GetEndpointByID(ctx, sqlcgen.GetEndpointByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found")
			return
		}
		slog.ErrorContext(ctx, "get endpoint", "endpoint_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get endpoint")
		return
	}

	detail := endpointDetail{
		ID:                  uuidToString(ep.ID),
		TenantID:            uuidToString(ep.TenantID),
		Hostname:            ep.Hostname,
		OsFamily:            ep.OsFamily,
		OsVersion:           ep.OsVersion,
		Status:              ep.Status,
		CreatedAt:           ep.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:           ep.UpdatedAt.Time.Format(time.RFC3339),
		CveCount:            ep.CveCount,
		PendingPatchesCount: ep.PendingPatchesCount,
		CriticalPatchCount:  ep.CriticalPatchCount,
		HighPatchCount:      ep.HighPatchCount,
		MediumPatchCount:    ep.MediumPatchCount,
		NetworkInterfaces:   []networkInterfaceItem{},
	}
	if ep.AgentVersion.Valid {
		detail.AgentVersion = &ep.AgentVersion.String
	}
	if ep.LastSeen.Valid {
		s := ep.LastSeen.Time.Format(time.RFC3339)
		detail.LastSeenStr = &s
	}
	if ep.IpAddress.Valid {
		detail.IpAddress = &ep.IpAddress.String
	}
	if ep.Arch.Valid {
		detail.Arch = &ep.Arch.String
	}
	if ep.KernelVersion.Valid {
		detail.KernelVersion = &ep.KernelVersion.String
	}
	if ep.CpuModel.Valid {
		detail.CpuModel = &ep.CpuModel.String
	}
	if ep.CpuCores.Valid {
		v := ep.CpuCores.Int32
		detail.CpuCores = &v
	}
	if ep.CpuUsagePercent.Valid {
		v := ep.CpuUsagePercent.Int16
		detail.CpuUsagePercent = &v
	}
	if ep.MemoryTotalMb.Valid {
		v := ep.MemoryTotalMb.Int64
		detail.MemoryTotalMb = &v
	}
	if ep.MemoryUsedMb.Valid {
		v := ep.MemoryUsedMb.Int64
		detail.MemoryUsedMb = &v
	}
	if ep.DiskTotalGb.Valid {
		v := ep.DiskTotalGb.Int64
		detail.DiskTotalGb = &v
	}
	if ep.DiskUsedGb.Valid {
		v := ep.DiskUsedGb.Int64
		detail.DiskUsedGb = &v
	}
	if ep.GpuModel.Valid {
		detail.GpuModel = &ep.GpuModel.String
	}
	if ep.UptimeSeconds.Valid {
		v := ep.UptimeSeconds.Int64
		detail.UptimeSeconds = &v
	}
	if ep.EnrolledAt.Valid {
		s := ep.EnrolledAt.Time.Format(time.RFC3339)
		detail.EnrolledAt = &s
	}
	if ep.LastHeartbeat.Valid {
		s := ep.LastHeartbeat.Time.Format(time.RFC3339)
		detail.LastHeartbeat = &s
	}
	if ep.CertExpiry.Valid {
		s := ep.CertExpiry.Time.Format(time.RFC3339)
		detail.CertExpiry = &s
	}
	if len(ep.HardwareDetails) > 0 && string(ep.HardwareDetails) != "{}" {
		detail.HardwareDetails = ep.HardwareDetails
	}
	if len(ep.SoftwareSummary) > 0 && string(ep.SoftwareSummary) != "{}" {
		detail.SoftwareSummary = ep.SoftwareSummary
	}

	// Populate package count and last scan time from the latest inventory.
	inv, err := h.q.GetLatestEndpointInventory(ctx, sqlcgen.GetLatestEndpointInventoryParams{
		EndpointID: id,
		TenantID:   tid,
	})
	if err != nil {
		if !isNotFound(err) {
			slog.ErrorContext(ctx, "get latest inventory for endpoint detail", "endpoint_id", chi.URLParam(r, "id"), "error", err)
		}
		// No inventory yet — leave defaults (0 / nil).
	} else {
		detail.PackageCount = int(inv.PackageCount)
		if inv.ScannedAt.Valid {
			t := inv.ScannedAt.Time
			detail.LastScan = &t
		}
	}

	// Populate vulnerable CVE count from endpoint_cves status counts.
	cveCounts, err := h.q.CountEndpointCVEsByStatus(ctx, sqlcgen.CountEndpointCVEsByStatusParams{
		EndpointID: id,
		TenantID:   tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "count endpoint CVEs for detail", "endpoint_id", chi.URLParam(r, "id"), "error", err)
	} else {
		for _, row := range cveCounts {
			if row.Status == "affected" {
				detail.VulnerableCVECount = int(row.Count)
				break
			}
		}
	}

	// Populate tags.
	tags, err := h.q.ListTagsForEndpoint(ctx, sqlcgen.ListTagsForEndpointParams{
		EndpointID: id,
		TenantID:   tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list tags for endpoint detail", "endpoint_id", chi.URLParam(r, "id"), "error", err)
	} else {
		tagObjs := make([]map[string]string, 0, len(tags))
		for _, t := range tags {
			tagObjs = append(tagObjs, map[string]string{
				"id":    uuidToString(t.ID),
				"key":   t.Key,
				"value": t.Value,
			})
		}
		if b, jerr := json.Marshal(tagObjs); jerr == nil {
			detail.Tags = json.RawMessage(b)
		}
	}
	if detail.Tags == nil {
		detail.Tags = json.RawMessage("[]")
	}

	// Populate network interfaces.
	nics, err := h.q.ListEndpointNetworkInterfaces(ctx, sqlcgen.ListEndpointNetworkInterfacesParams{
		TenantID:   tid,
		EndpointID: id,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list network interfaces for endpoint detail", "endpoint_id", chi.URLParam(r, "id"), "error", err)
	} else {
		for _, nic := range nics {
			item := networkInterfaceItem{
				ID:     uuidToString(nic.ID),
				Name:   nic.Name,
				Status: nic.Status,
			}
			if nic.IpAddress.Valid {
				item.IpAddress = &nic.IpAddress.String
			}
			if nic.MacAddress.Valid {
				item.MacAddress = &nic.MacAddress.String
			}
			detail.NetworkInterfaces = append(detail.NetworkInterfaces, item)
		}
	}

	WriteJSON(w, http.StatusOK, detail)
}

// endpointDetail is the JSON shape returned by GET /api/v1/endpoints/{id}.
type endpointDetail struct {
	ID            string  `json:"id"`
	TenantID      string  `json:"tenant_id"`
	Hostname      string  `json:"hostname"`
	OsFamily      string  `json:"os_family"`
	OsVersion     string  `json:"os_version"`
	AgentVersion  *string `json:"agent_version"`
	Status        string  `json:"status"`
	LastSeenStr   *string `json:"last_seen"`
	IpAddress     *string `json:"ip_address"`
	Arch          *string `json:"arch"`
	KernelVersion *string `json:"kernel_version"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	// Hardware
	CpuModel        *string `json:"cpu_model"`
	CpuCores        *int32  `json:"cpu_cores"`
	CpuUsagePercent *int16  `json:"cpu_usage_percent"`
	MemoryTotalMb   *int64  `json:"memory_total_mb"`
	MemoryUsedMb    *int64  `json:"memory_used_mb"`
	DiskTotalGb     *int64  `json:"disk_total_gb"`
	DiskUsedGb      *int64  `json:"disk_used_gb"`
	GpuModel        *string `json:"gpu_model"`
	UptimeSeconds   *int64  `json:"uptime_seconds"`
	// Agent
	EnrolledAt    *string `json:"enrolled_at"`
	LastHeartbeat *string `json:"last_heartbeat"`
	CertExpiry    *string `json:"cert_expiry"`
	// Deep hardware & software JSONB
	HardwareDetails json.RawMessage `json:"hardware_details,omitempty"`
	SoftwareSummary json.RawMessage `json:"software_summary,omitempty"`
	// Computed
	PackageCount        int        `json:"package_count"`
	LastScan            *time.Time `json:"last_scan"`
	VulnerableCVECount  int        `json:"vulnerable_cve_count"`
	CveCount            int64      `json:"cve_count"`
	PendingPatchesCount int64      `json:"pending_patches_count"`
	CriticalPatchCount  int64      `json:"critical_patch_count"`
	HighPatchCount      int64      `json:"high_patch_count"`
	MediumPatchCount    int64      `json:"medium_patch_count"`
	// Tags
	Tags json.RawMessage `json:"tags"`
	// Network
	NetworkInterfaces []networkInterfaceItem `json:"network_interfaces"`
}

// networkInterfaceItem is the JSON shape for a network interface in endpoint detail.
type networkInterfaceItem struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	IpAddress  *string `json:"ip_address"`
	MacAddress *string `json:"mac_address"`
	Status     string  `json:"status"`
}

// endpointListItem is the JSON shape returned by GET /api/v1/endpoints.
type endpointListItem struct {
	ID                  string          `json:"id"`
	Hostname            string          `json:"hostname"`
	OsFamily            string          `json:"os_family"`
	OsVersion           string          `json:"os_version"`
	AgentVersion        *string         `json:"agent_version"`
	Status              string          `json:"status"`
	LastSeen            *string         `json:"last_seen"`
	CreatedAt           string          `json:"created_at"`
	UpdatedAt           string          `json:"updated_at"`
	IpAddress           *string         `json:"ip_address"`
	Arch                *string         `json:"arch"`
	KernelVersion       *string         `json:"kernel_version"`
	CveCount            int64           `json:"cve_count"`
	CriticalCveCount    int64           `json:"critical_cve_count"`
	HighCveCount        int64           `json:"high_cve_count"`
	MediumCveCount      int64           `json:"medium_cve_count"`
	PendingPatchesCount int64           `json:"pending_patches_count"`
	CriticalPatchCount  int64           `json:"critical_patch_count"`
	HighPatchCount      int64           `json:"high_patch_count"`
	MediumPatchCount    int64           `json:"medium_patch_count"`
	CompliancePct       *float64        `json:"compliance_pct"`
	Tags                json.RawMessage `json:"tags"`
	CpuCores            *int32          `json:"cpu_cores,omitempty"`
	CpuUsagePercent     *int16          `json:"cpu_usage_percent,omitempty"`
	MemoryTotalMB       *int64          `json:"memory_total_mb,omitempty"`
	MemoryUsedMB        *int64          `json:"memory_used_mb,omitempty"`
	DiskTotalGB         *int64          `json:"disk_total_gb,omitempty"`
	DiskUsedGB          *int64          `json:"disk_used_gb,omitempty"`
}

// updateEndpointRequest is the JSON body for PUT /endpoints/{id}.
type updateEndpointRequest struct {
	Hostname      string `json:"hostname"`
	OsFamily      string `json:"os_family"`
	OsVersion     string `json:"os_version"`
	AgentVersion  string `json:"agent_version,omitempty"`
	IpAddress     string `json:"ip_address,omitempty"`
	Arch          string `json:"arch,omitempty"`
	KernelVersion string `json:"kernel_version,omitempty"`
}

// Update handles PUT /api/v1/endpoints/{id}.
func (h *EndpointHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid endpoint ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var body updateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}

	params := sqlcgen.UpdateEndpointParams{
		ID:            id,
		Hostname:      body.Hostname,
		OsFamily:      body.OsFamily,
		OsVersion:     body.OsVersion,
		AgentVersion:  textFromString(body.AgentVersion),
		IpAddress:     textFromString(body.IpAddress),
		Arch:          textFromString(body.Arch),
		KernelVersion: textFromString(body.KernelVersion),
		TenantID:      tid,
	}

	ep, err := h.q.UpdateEndpoint(ctx, params)
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found")
			return
		}
		slog.ErrorContext(ctx, "update endpoint", "endpoint_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update endpoint")
		return
	}

	emitEvent(ctx, h.eventBus, events.EndpointUpdated, "endpoint", uuidToString(ep.ID), tenantID, ep)
	WriteJSON(w, http.StatusOK, ep)
}

// Delete handles DELETE /api/v1/endpoints/{id} (soft delete).
func (h *EndpointHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid endpoint ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	ep, err := h.q.SoftDeleteEndpoint(ctx, sqlcgen.SoftDeleteEndpointParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found")
			return
		}
		slog.ErrorContext(ctx, "soft delete endpoint", "endpoint_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete endpoint")
		return
	}

	emitEvent(ctx, h.eventBus, events.EndpointDeleted, "endpoint", uuidToString(ep.ID), tenantID, ep)
	w.WriteHeader(http.StatusNoContent)
}

// Scan handles POST /api/v1/endpoints/{id}/scan.
func (h *EndpointHandler) Scan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid endpoint ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	// Verify endpoint exists before scheduling scan.
	_, err = h.q.GetEndpointByID(ctx, sqlcgen.GetEndpointByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found")
			return
		}
		slog.ErrorContext(ctx, "get endpoint for scan", "endpoint_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get endpoint")
		return
	}

	var actorID, actorType string
	if uid, ok := user.UserIDFromContext(ctx); ok && uid != "" {
		actorID = uid
		actorType = domain.ActorUser
	}

	cmdID, err := h.scanScheduler.ScanSingle(ctx, id, tid, actorID, actorType)
	if err != nil {
		slog.ErrorContext(ctx, "schedule endpoint scan", "endpoint_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to schedule scan")
		return
	}

	WriteJSON(w, http.StatusAccepted, map[string]string{
		"status":     "scan_requested",
		"command_id": uuidToString(cmdID),
	})
}

// ActiveScan handles GET /api/v1/endpoints/{id}/active-scan.
// Returns the latest non-terminal run_scan command for this endpoint so the UI
// can resume tracking a scan after navigating away and back.
func (h *EndpointHandler) ActiveScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid endpoint ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	cmd, err := h.q.GetActiveRunScanByAgent(ctx, sqlcgen.GetActiveRunScanByAgentParams{
		AgentID:  id,
		TenantID: tid,
	})
	if err != nil {
		if isNotFound(err) {
			WriteJSON(w, http.StatusOK, map[string]any{"command": nil})
			return
		}
		slog.ErrorContext(ctx, "get active run_scan", "endpoint_id", chi.URLParam(r, "id"), "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get active scan")
		return
	}

	resp := map[string]any{
		"command": map[string]any{
			"id":            uuidToString(cmd.ID),
			"agent_id":      uuidToString(cmd.AgentID),
			"type":          cmd.Type,
			"status":        cmd.Status,
			"created_at":    cmd.CreatedAt,
			"delivered_at":  nullableTime(cmd.DeliveredAt),
			"completed_at":  nullableTime(cmd.CompletedAt),
			"error_message": nullableText(cmd.ErrorMessage),
		},
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListCVEs handles GET /api/v1/endpoints/{id}/cves.
func (h *EndpointHandler) ListCVEs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid endpoint ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.ListEndpointCVEsAffected(ctx, sqlcgen.ListEndpointCVEsAffectedParams{
		EndpointID: id,
		TenantID:   tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list endpoint CVEs", "endpoint_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list endpoint CVEs")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{"data": rows, "total_count": len(rows)})
}

// ListPackages handles GET /api/v1/endpoints/{id}/packages.
func (h *EndpointHandler) ListPackages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid endpoint ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	pkgs, err := h.q.ListEndpointPackagesByEndpoint(ctx, sqlcgen.ListEndpointPackagesByEndpointParams{
		EndpointID: id,
		TenantID:   tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list endpoint packages", "endpoint_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list endpoint packages")
		return
	}

	items := make([]endpointPackageItem, len(pkgs))
	for i, p := range pkgs {
		item := endpointPackageItem{
			ID:          uuidToString(p.ID),
			PackageName: p.PackageName,
			Version:     p.Version,
			CreatedAt:   p.CreatedAt.Time.Format(time.RFC3339),
		}
		if p.Arch.Valid {
			item.Arch = &p.Arch.String
		}
		if p.Source.Valid {
			item.Source = &p.Source.String
		}
		if p.Release.Valid {
			item.Release = &p.Release.String
		}
		items[i] = item
	}

	WriteJSON(w, http.StatusOK, map[string]any{"data": items, "total_count": len(items)})
}

// endpointPackageItem is the JSON shape for a single package in ListPackages response.
type endpointPackageItem struct {
	ID          string  `json:"id"`
	PackageName string  `json:"package_name"`
	Version     string  `json:"version"`
	Arch        *string `json:"arch"`
	Source      *string `json:"source"`
	Release     *string `json:"release"`
	CreatedAt   string  `json:"created_at"`
}

// ListDeploymentHistory handles GET /api/v1/endpoints/{id}/deployments.
func (h *EndpointHandler) ListDeploymentHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid endpoint ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	targets, err := h.q.ListDeploymentTargetsByEndpoint(ctx, sqlcgen.ListDeploymentTargetsByEndpointParams{
		EndpointID: id,
		TenantID:   tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list endpoint deployment history", "endpoint_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list endpoint deployment history")
		return
	}

	items := make([]endpointDeploymentItem, len(targets))
	for i, t := range targets {
		item := endpointDeploymentItem{
			ID:           uuidToString(t.ID),
			DeploymentID: uuidToString(t.DeploymentID),
			PatchID:      uuidToString(t.PatchID),
			Status:       t.Status,
			CreatedAt:    t.CreatedAt.Time.Format(time.RFC3339),
		}
		if t.StartedAt.Valid {
			s := t.StartedAt.Time.Format(time.RFC3339)
			item.StartedAt = &s
		}
		if t.CompletedAt.Valid {
			s := t.CompletedAt.Time.Format(time.RFC3339)
			item.CompletedAt = &s
		}
		if t.StartedAt.Valid && t.CompletedAt.Valid {
			d := int64(t.CompletedAt.Time.Sub(t.StartedAt.Time).Seconds())
			item.DurationSeconds = &d
		}
		if t.ErrorMessage.Valid {
			item.ErrorMessage = &t.ErrorMessage.String
		}
		items[i] = item
	}

	WriteJSON(w, http.StatusOK, map[string]any{"data": items, "total_count": len(items)})
}

// ListPatches handles GET /api/v1/endpoints/{id}/patches.
func (h *EndpointHandler) ListPatches(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid endpoint ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.ListPatchesForEndpoint(ctx, sqlcgen.ListPatchesForEndpointParams{
		EndpointID: id,
		TenantID:   tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list endpoint patches", "endpoint_id", chi.URLParam(r, "id"), "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list endpoint patches")
		return
	}

	availableRows, err := h.q.ListAvailablePatchesForEndpointByOS(ctx, sqlcgen.ListAvailablePatchesForEndpointByOSParams{
		EndpointID: id,
		TenantID:   tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list available patches for endpoint", "endpoint_id", chi.URLParam(r, "id"), "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list available patches for endpoint")
		return
	}

	items := make([]endpointPatchItem, 0, len(rows)+len(availableRows))
	for _, row := range rows {
		items = append(items, endpointPatchItem{
			ID:          uuidToString(row.ID),
			Name:        row.Name,
			Version:     row.Version,
			Severity:    row.Severity,
			OsFamily:    row.OsFamily,
			Status:      row.DeployStatus,
			Source:      row.SourceRepo.String,
			HighestCVSS: float64(row.HighestCvss),
			CVECount:    row.CveCount,
			CreatedAt:   row.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	for _, row := range availableRows {
		items = append(items, endpointPatchItem{
			ID:          uuidToString(row.ID),
			Name:        row.Name,
			Version:     row.Version,
			Severity:    row.Severity,
			OsFamily:    row.OsFamily,
			Status:      row.DeployStatus,
			Source:      row.SourceRepo.String,
			HighestCVSS: float64(row.HighestCvss),
			CVECount:    row.CveCount,
			CreatedAt:   row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	WriteJSON(w, http.StatusOK, map[string]any{"data": items, "total_count": len(items)})
}

// endpointPatchItem is the JSON shape for GET /api/v1/endpoints/{id}/patches.
type endpointPatchItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Version     string  `json:"version"`
	Severity    string  `json:"severity"`
	OsFamily    string  `json:"os_family"`
	Status      string  `json:"status"`
	Source      string  `json:"source"`
	HighestCVSS float64 `json:"highest_cvss"`
	CVECount    int64   `json:"cve_count"`
	CreatedAt   string  `json:"created_at"`
}

// Export handles GET /api/v1/endpoints/export — streams a CSV of filtered endpoints.
func (h *EndpointHandler) Export(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var exportTagID pgtype.UUID
	if tidParam := r.URL.Query().Get("tag_id"); tidParam != "" {
		if exportTagID, err = scanUUID(tidParam); err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_TAG_ID", "invalid tag_id: not a valid UUID")
			return
		}
	}

	rows, err := h.q.ListEndpointsForExport(ctx, sqlcgen.ListEndpointsForExportParams{
		TenantID: tid,
		Status:   r.URL.Query().Get("status"),
		OsFamily: r.URL.Query().Get("os_family"),
		Search:   r.URL.Query().Get("search"),
		TagID:    exportTagID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "export endpoints", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to export endpoints")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=endpoints-export.csv")

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"Hostname", "OS Family", "OS Version", "Status", "Agent Version",
		"IP Address", "Architecture", "Kernel Version", "Last Seen",
		"Pending Patches", "Critical Patches", "CVEs", "Tags",
	})

	for _, row := range rows {
		lastSeen := ""
		if row.LastSeen.Valid {
			lastSeen = row.LastSeen.Time.Format(time.RFC3339)
		}

		_ = cw.Write([]string{
			row.Hostname,
			row.OsFamily,
			row.OsVersion,
			row.Status,
			row.AgentVersion.String,
			row.IpAddress.String,
			row.Arch.String,
			row.KernelVersion.String,
			lastSeen,
			strconv.FormatInt(row.PendingPatchesCount, 10),
			strconv.FormatInt(row.CriticalPatchCount, 10),
			strconv.FormatInt(row.CveCount, 10),
			string(tagsToJSON(row.Tags)),
		})
	}
	cw.Flush()
}

// tagsToJSON converts a database tags value (string, []byte, or other) into a JSON byte slice,
// defaulting to "[]" if nil or unrecognizable.
func tagsToJSON(v interface{}) json.RawMessage {
	switch t := v.(type) {
	case string:
		return json.RawMessage(t)
	case []byte:
		return json.RawMessage(t)
	default:
		if v != nil {
			if b, err := json.Marshal(v); err == nil {
				return b
			}
		}
	}
	return json.RawMessage("[]")
}

// endpointDeploymentItem is the JSON shape for a single deployment target in ListDeploymentHistory response.
type endpointDeploymentItem struct {
	ID              string  `json:"id"`
	DeploymentID    string  `json:"deployment_id"`
	PatchID         string  `json:"patch_id"`
	Status          string  `json:"status"`
	StartedAt       *string `json:"started_at"`
	CompletedAt     *string `json:"completed_at"`
	DurationSeconds *int64  `json:"duration_seconds"`
	ErrorMessage    *string `json:"error_message"`
	CreatedAt       string  `json:"created_at"`
}
