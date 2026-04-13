package config

// SourceLevel identifies which hierarchy level set a config value.
type SourceLevel string

const (
	SourceSystem   SourceLevel = "system"
	SourceTenant   SourceLevel = "tenant"
	SourceTag      SourceLevel = "tag"
	SourceEndpoint SourceLevel = "endpoint"
)

// ResolveParams holds the identifiers needed to resolve config for an endpoint.
// Groups were removed in migration 060; the "tag" level walks the endpoint's
// tag assignments and merges each matching tag-scope override in key order.
type ResolveParams struct {
	TenantID   string
	TagIDs     []string
	EndpointID string
	Module     string
}

// ResolvedConfig holds the effective config and per-field source attribution.
type ResolvedConfig[T any] struct {
	Effective T                      `json:"effective"`
	Sources   map[string]SourceLevel `json:"sources"`
}
