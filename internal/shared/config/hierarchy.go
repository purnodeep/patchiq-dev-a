package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// MergeConfig merges non-nil fields from overlay into base using reflection.
// Scalar pointer fields: overlay replaces base if non-nil.
// Slice fields: additive (append + deduplicate).
// Struct pointer fields: overlay replaces base if non-nil.
func MergeConfig[T any](base, overlay T) T {
	baseVal := reflect.ValueOf(&base).Elem()
	overlayVal := reflect.ValueOf(&overlay).Elem()
	mergeStructFields(baseVal, overlayVal)
	return base
}

func mergeStructFields(base, overlay reflect.Value) {
	t := base.Type()
	for i := 0; i < t.NumField(); i++ {
		baseField := base.Field(i)
		overlayField := overlay.Field(i)

		switch baseField.Kind() {
		case reflect.Ptr:
			if !overlayField.IsNil() {
				baseField.Set(overlayField)
			}
		case reflect.Slice:
			if overlayField.Len() > 0 {
				merged := deduplicateSlice(baseField, overlayField)
				baseField.Set(merged)
			}
		}
	}
}

func deduplicateSlice(base, overlay reflect.Value) reflect.Value {
	seen := make(map[any]bool)
	result := reflect.MakeSlice(base.Type(), 0, base.Len()+overlay.Len())
	for i := 0; i < base.Len(); i++ {
		v := base.Index(i).Interface()
		if !seen[v] {
			seen[v] = true
			result = reflect.Append(result, base.Index(i))
		}
	}
	for i := 0; i < overlay.Len(); i++ {
		v := overlay.Index(i).Interface()
		if !seen[v] {
			seen[v] = true
			result = reflect.Append(result, overlay.Index(i))
		}
	}
	return result
}

// ResolveConfig resolves the effective configuration by walking the
// hierarchy: system defaults → tenant → tag overrides (one per tag the
// endpoint carries) → endpoint. Each level merges non-nil fields from
// the stored override, and source attribution tracks which level set
// each field. Tag overrides are applied in TagIDs order, so the caller
// controls precedence within the tag layer.
func ResolveConfig[T any](ctx context.Context, store ConfigStore, params ResolveParams, defaults T) (*ResolvedConfig[T], error) {
	effective := defaults
	sources := initSources(defaults, SourceSystem)

	// Tenant first.
	if params.TenantID != "" {
		if err := applyOverride(ctx, store, params.TenantID, ScopeTenant, params.TenantID, params.Module, SourceTenant, &effective, sources); err != nil {
			return nil, err
		}
	}

	// Then each tag the endpoint carries, in the supplied order.
	for _, tagID := range params.TagIDs {
		if tagID == "" {
			continue
		}
		if err := applyOverride(ctx, store, params.TenantID, ScopeTag, tagID, params.Module, SourceTag, &effective, sources); err != nil {
			return nil, err
		}
	}

	// Endpoint override wins.
	if params.EndpointID != "" {
		if err := applyOverride(ctx, store, params.TenantID, ScopeEndpoint, params.EndpointID, params.Module, SourceEndpoint, &effective, sources); err != nil {
			return nil, err
		}
	}

	return &ResolvedConfig[T]{
		Effective: effective,
		Sources:   sources,
	}, nil
}

func applyOverride[T any](ctx context.Context, store ConfigStore, tenantID, scopeType, scopeID, module string, source SourceLevel, effective *T, sources map[string]SourceLevel) error {
	raw, err := store.GetOverride(ctx, tenantID, scopeType, scopeID, module)
	if errors.Is(err, ErrNoOverride) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get config override %s/%s: %w", scopeType, scopeID, err)
	}
	var overlay T
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return fmt.Errorf("unmarshal config override %s/%s: %w", scopeType, scopeID, err)
	}
	updateSources(sources, overlay, source)
	*effective = MergeConfig(*effective, overlay)
	return nil
}

// initSources creates a source map with all non-nil/non-empty fields in cfg attributed to level.
// Field names come from json struct tags.
func initSources[T any](cfg T, level SourceLevel) map[string]SourceLevel {
	sources := make(map[string]SourceLevel)
	v := reflect.ValueOf(&cfg).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := v.Field(i)
		tag := jsonFieldName(t.Field(i))
		if tag == "" {
			continue
		}
		switch field.Kind() {
		case reflect.Ptr:
			if !field.IsNil() {
				sources[tag] = level
			}
		case reflect.Slice:
			if field.Len() > 0 {
				sources[tag] = level
			}
		}
	}
	return sources
}

// updateSources updates the source map for any non-nil/non-empty fields in overlay.
func updateSources[T any](sources map[string]SourceLevel, overlay T, level SourceLevel) {
	v := reflect.ValueOf(&overlay).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := v.Field(i)
		tag := jsonFieldName(t.Field(i))
		if tag == "" {
			continue
		}
		switch field.Kind() {
		case reflect.Ptr:
			if !field.IsNil() {
				sources[tag] = level
			}
		case reflect.Slice:
			if field.Len() > 0 {
				sources[tag] = level
			}
		}
	}
}

// jsonFieldName extracts the JSON field name from a struct field's tag.
// Returns "" if the field has no json tag or is explicitly excluded ("-").
func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}
