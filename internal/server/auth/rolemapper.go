package auth

import (
	"context"
	"errors"
)

// ErrNoRoleMapping indicates no mapping exists for the external role.
var ErrNoRoleMapping = errors.New("no role mapping found for external role")

// RoleMappingStore looks up external-to-internal role mappings.
type RoleMappingStore interface {
	GetRoleMappingByExternalRole(ctx context.Context, tenantID, externalRole string) (string, error)
}

// RoleMapper maps external IdP roles to PatchIQ role IDs.
type RoleMapper struct {
	store RoleMappingStore
}

// NewRoleMapper creates a new RoleMapper.
func NewRoleMapper(store RoleMappingStore) *RoleMapper {
	return &RoleMapper{store: store}
}

// Map returns the PatchIQ role ID for the given external role.
func (m *RoleMapper) Map(ctx context.Context, tenantID, externalRole string) (string, error) {
	return m.store.GetRoleMappingByExternalRole(ctx, tenantID, externalRole)
}
