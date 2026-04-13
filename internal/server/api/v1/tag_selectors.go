package v1

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/skenzeriq/patchiq/internal/server/targeting"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// SelectorValidator is the Resolver interface slice used by the
// selector-validate endpoint. Count(ctx, tenantID, sel) evaluates the
// selector against the tenant's endpoints and returns the match count —
// the UI uses this for live preview as the user builds a selector.
type SelectorValidator interface {
	Count(ctx context.Context, tenantID string, sel *targeting.Selector) (int, error)
}

// TagSelectorHandler serves selector validation endpoints.
type TagSelectorHandler struct {
	resolver SelectorValidator
}

func NewTagSelectorHandler(resolver SelectorValidator) *TagSelectorHandler {
	if resolver == nil {
		panic("tag_selectors: NewTagSelectorHandler called with nil resolver")
	}
	return &TagSelectorHandler{resolver: resolver}
}

type validateSelectorRequest struct {
	Selector *targeting.Selector `json:"selector"`
}

type validateSelectorResponse struct {
	Valid        bool   `json:"valid"`
	Error        string `json:"error,omitempty"`
	MatchedCount int    `json:"matched_count"`
}

// Validate handles POST /api/v1/tags/selectors/validate. Returns whether
// the supplied selector is structurally valid plus, if so, the number of
// endpoints that currently match it inside the caller's tenant.
func (h *TagSelectorHandler) Validate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	var body validateSelectorRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}

	// Reject empty/null selector explicitly. Without this, Count(nil)
	// would return the full tenant endpoint count and the UI's live
	// preview would silently show "matched N of N" for a submission with
	// no selector — a frontend bug that pipes an empty body to this
	// endpoint would look like a legitimate match-all.
	if body.Selector == nil {
		WriteFieldError(w, http.StatusBadRequest, "INVALID_SELECTOR",
			"selector is required; send an explicit match-all shape if that is the intent",
			"selector")
		return
	}

	if body.Selector != nil {
		if err := targeting.Validate(*body.Selector); err != nil {
			// ErrMalformedSelector is the expected case and produces a 200
			// with valid:false so the UI can show inline feedback without
			// treating validation failures as hard errors.
			if errors.Is(err, targeting.ErrMalformedSelector) {
				WriteJSON(w, http.StatusOK, validateSelectorResponse{
					Valid: false,
					Error: err.Error(),
				})
				return
			}
			slog.ErrorContext(ctx, "validate selector unexpected error", "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "selector validation failed")
			return
		}
	}

	count, err := h.resolver.Count(ctx, tenantID, body.Selector)
	if err != nil {
		slog.ErrorContext(ctx, "resolver count", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count matching endpoints")
		return
	}

	WriteJSON(w, http.StatusOK, validateSelectorResponse{
		Valid:        true,
		MatchedCount: count,
	})
}
