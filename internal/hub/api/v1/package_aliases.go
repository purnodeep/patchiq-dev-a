package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// PackageAliasQuerier abstracts the sqlcgen queries used by PackageAliasHandler.
type PackageAliasQuerier interface {
	ListPackageAliases(ctx context.Context, arg sqlcgen.ListPackageAliasesParams) ([]sqlcgen.PackageAlias, error)
	ListPackageAliasesByProduct(ctx context.Context, feedProduct string) ([]sqlcgen.PackageAlias, error)
	UpsertPackageAlias(ctx context.Context, arg sqlcgen.UpsertPackageAliasParams) (sqlcgen.PackageAlias, error)
	UpdatePackageAliasById(ctx context.Context, arg sqlcgen.UpdatePackageAliasByIdParams) (sqlcgen.PackageAlias, error)
	DeletePackageAlias(ctx context.Context, id pgtype.UUID) error
	GetPackageAlias(ctx context.Context, arg sqlcgen.GetPackageAliasParams) (sqlcgen.PackageAlias, error)
}

// PackageAliasHandler serves package alias CRUD endpoints.
type PackageAliasHandler struct {
	queries  PackageAliasQuerier
	eventBus domain.EventBus
}

// NewPackageAliasHandler creates a new PackageAliasHandler.
func NewPackageAliasHandler(queries PackageAliasQuerier, eventBus domain.EventBus) *PackageAliasHandler {
	return &PackageAliasHandler{queries: queries, eventBus: eventBus}
}

type packageAliasRequest struct {
	FeedProduct    string  `json:"feed_product"`
	OsFamily       string  `json:"os_family"`
	OsDistribution string  `json:"os_distribution"`
	OsPackageName  string  `json:"os_package_name"`
	Confidence     *string `json:"confidence,omitempty"`
}

func (req *packageAliasRequest) validate() error {
	if req.FeedProduct == "" {
		return fmt.Errorf("feed_product is required")
	}
	if req.OsFamily == "" {
		return fmt.Errorf("os_family is required")
	}
	if req.OsPackageName == "" {
		return fmt.Errorf("os_package_name is required")
	}
	return nil
}

// List handles GET /api/v1/package-aliases.
// Supports ?limit=&offset= (defaults: 50, 0) and ?product= for filtered listing.
func (h *PackageAliasHandler) List(w http.ResponseWriter, r *http.Request) {
	if product := r.URL.Query().Get("product"); product != "" {
		aliases, err := h.queries.ListPackageAliasesByProduct(r.Context(), product)
		if err != nil {
			slog.ErrorContext(r.Context(), "list package aliases by product", "product", product, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "list package aliases by product: internal error")
			return
		}
		if aliases == nil {
			aliases = []sqlcgen.PackageAlias{}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"aliases": aliases}); err != nil {
			slog.ErrorContext(r.Context(), "encode list package aliases by product response", "error", err)
		}
		return
	}

	limit := queryParamInt(r, "limit", 50)
	limit = min(limit, 100)
	if limit < 1 {
		limit = 50
	}
	offset := queryParamInt(r, "offset", 0)

	aliases, err := h.queries.ListPackageAliases(r.Context(), sqlcgen.ListPackageAliasesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "list package aliases", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list package aliases: internal error")
		return
	}
	if aliases == nil {
		aliases = []sqlcgen.PackageAlias{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"aliases": aliases,
		"limit":   limit,
		"offset":  offset,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode list package aliases response", "error", err)
	}
}

// Create handles POST /api/v1/package-aliases.
func (h *PackageAliasHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req packageAliasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode request body: %s", err))
		return
	}

	if err := req.validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	confidence := "manual"
	if req.Confidence != nil {
		confidence = *req.Confidence
	}
	params := sqlcgen.UpsertPackageAliasParams{
		FeedProduct:    req.FeedProduct,
		OsFamily:       req.OsFamily,
		OsDistribution: req.OsDistribution,
		OsPackageName:  req.OsPackageName,
		Confidence:     confidence,
	}

	alias, err := h.queries.UpsertPackageAlias(r.Context(), params)
	if err != nil {
		slog.ErrorContext(r.Context(), "upsert package alias", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "create package alias: internal error")
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	aliasIDStr := uuidToString(alias.ID)
	evt := domain.NewSystemEvent(events.PackageAliasCreated, tenantID, "package_alias", aliasIDStr, "create", alias)
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit package_alias.created event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]any{"alias": alias}); err != nil {
		slog.ErrorContext(r.Context(), "encode create package alias response", "error", err)
	}
}

// Delete handles DELETE /api/v1/package-aliases/{id}.
func (h *PackageAliasHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse package alias id: %s", err))
		return
	}

	if err := h.queries.DeletePackageAlias(r.Context(), id); err != nil {
		slog.ErrorContext(r.Context(), "delete package alias", "id", uuidToString(id), "error", err)
		writeJSONError(w, http.StatusInternalServerError, "delete package alias: internal error")
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	evt := domain.NewSystemEvent(events.PackageAliasDeleted, tenantID, "package_alias", chi.URLParam(r, "id"), "deleted", nil)
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit package_alias.deleted event", "error", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// Update handles PUT /api/v1/package-aliases/{id}.
func (h *PackageAliasHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse package alias id: %s", err))
		return
	}

	var req packageAliasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode request body: %s", err))
		return
	}

	if err := req.validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	confidence := "manual"
	if req.Confidence != nil {
		confidence = *req.Confidence
	}

	alias, err := h.queries.UpdatePackageAliasById(r.Context(), sqlcgen.UpdatePackageAliasByIdParams{
		ID:             id,
		FeedProduct:    req.FeedProduct,
		OsFamily:       req.OsFamily,
		OsDistribution: req.OsDistribution,
		OsPackageName:  req.OsPackageName,
		Confidence:     confidence,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, fmt.Sprintf("package alias %s not found", uuidToString(id)))
			return
		}
		slog.ErrorContext(r.Context(), "update package alias by id", "id", uuidToString(id), "error", err)
		writeJSONError(w, http.StatusInternalServerError, "update package alias: internal error")
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	aliasIDStr := uuidToString(alias.ID)
	evt := domain.NewSystemEvent(events.PackageAliasUpdated, tenantID, "package_alias", aliasIDStr, "updated", alias)
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit package_alias.updated event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"alias": alias}); err != nil {
		slog.ErrorContext(r.Context(), "encode update package alias response", "error", err)
	}
}

type discoverPackageEntry struct {
	OsPackageName  string `json:"os_package_name"`
	OsFamily       string `json:"os_family"`
	OsDistribution string `json:"os_distribution"`
}

type discoverRequest struct {
	Packages []discoverPackageEntry `json:"packages"`
}

// normalizePkgToFeedProduct attempts to map an OS package name back to a
// canonical feed product name using known naming conventions.
func normalizePkgToFeedProduct(osPackageName string) string {
	name := osPackageName

	// Strip common suffixes first
	for _, suffix := range []string{"-devel", "-dev", "-libs", "-common", "-utils", "-tools", "-doc", "-dbg", "-dbgsym"} {
		if strings.HasSuffix(name, suffix) {
			name = strings.TrimSuffix(name, suffix)
			break
		}
	}

	// Strip trailing version qualifiers like "3", "3t64", "1.1"
	// e.g., "libssl3" -> "libssl", "libcurl4" -> "libcurl"
	if idx := strings.LastIndexAny(name, "0123456789"); idx > 0 {
		// Only strip if the digit portion is at the end
		prefix := name[:idx]
		// Ensure we don't strip too aggressively — keep at least 3 chars
		if len(prefix) >= 3 {
			name = prefix
		}
	}

	// Strip "lib" prefix for library packages
	// e.g., "libcurl" -> "curl", "libssl" -> "ssl"
	if strings.HasPrefix(name, "lib") && len(name) > 3 {
		name = strings.TrimPrefix(name, "lib")
	}

	return name
}

// Discover handles POST /api/v1/package-aliases/discover.
// For each package, tries an exact match upsert with confidence="discovery".
func (h *PackageAliasHandler) Discover(w http.ResponseWriter, r *http.Request) {
	var req discoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode request body: %s", err))
		return
	}

	if len(req.Packages) == 0 {
		writeJSONError(w, http.StatusBadRequest, "packages must not be empty")
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	confidence := "discovery"
	created := make([]sqlcgen.PackageAlias, 0, len(req.Packages))

	for _, pkg := range req.Packages {
		if pkg.OsPackageName == "" || pkg.OsFamily == "" {
			slog.WarnContext(r.Context(), "skip discover entry: missing required fields",
				"os_package_name", pkg.OsPackageName, "os_family", pkg.OsFamily)
			continue
		}

		feedProduct := normalizePkgToFeedProduct(pkg.OsPackageName)
		params := sqlcgen.UpsertPackageAliasParams{
			FeedProduct:    feedProduct,
			OsFamily:       pkg.OsFamily,
			OsDistribution: pkg.OsDistribution,
			OsPackageName:  pkg.OsPackageName,
			Confidence:     confidence,
		}

		alias, err := h.queries.UpsertPackageAlias(r.Context(), params)
		if err != nil {
			slog.ErrorContext(r.Context(), "discover upsert package alias", "os_package_name", pkg.OsPackageName, "error", err)
			continue
		}

		aliasIDStr := uuidToString(alias.ID)
		evt := domain.NewSystemEvent(events.PackageAliasCreated, tenantID, "package_alias", aliasIDStr, "create", alias)
		if err := h.eventBus.Emit(r.Context(), evt); err != nil {
			slog.ErrorContext(r.Context(), "emit package_alias.created event during discover", "error", err)
		}

		created = append(created, alias)
	}

	if len(created) == 0 && len(req.Packages) > 0 {
		writeJSONError(w, http.StatusBadRequest, "no valid packages to discover")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]any{"aliases": created}); err != nil {
		slog.ErrorContext(r.Context(), "encode discover package aliases response", "error", err)
	}
}
