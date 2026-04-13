package auth_test

import (
	"context"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/auth"
)

type mockMappingStore struct {
	mappings map[string]string // external_role -> patchiq_role_id
}

func (m *mockMappingStore) GetRoleMappingByExternalRole(_ context.Context, tenantID, externalRole string) (string, error) {
	id, ok := m.mappings[externalRole]
	if !ok {
		return "", auth.ErrNoRoleMapping
	}
	return id, nil
}

func TestRoleMapper_Map(t *testing.T) {
	tests := []struct {
		name         string
		mappings     map[string]string
		externalRole string
		wantRoleID   string
		wantErr      error
	}{
		{
			name:         "mapping found",
			mappings:     map[string]string{"patchiq:admin": "role-uuid-1"},
			externalRole: "patchiq:admin",
			wantRoleID:   "role-uuid-1",
		},
		{
			name:         "no mapping returns error",
			mappings:     map[string]string{},
			externalRole: "patchiq:unknown",
			wantErr:      auth.ErrNoRoleMapping,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockMappingStore{mappings: tt.mappings}
			mapper := auth.NewRoleMapper(store)

			roleID, err := mapper.Map(context.Background(), "tenant-1", tt.externalRole)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if roleID != tt.wantRoleID {
				t.Errorf("roleID = %q, want %q", roleID, tt.wantRoleID)
			}
		})
	}
}
