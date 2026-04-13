package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveConfig(t *testing.T) {
	tests := []struct {
		name       string
		dbFlag     string
		envKey     string
		envVal     string
		wantDB     string
		wantDir    string
		wantErrMsg string
	}{
		{
			name:    "server defaults",
			dbFlag:  "server",
			wantDB:  "postgres://patchiq:patchiq_dev@localhost:5432/patchiq?sslmode=disable",
			wantDir: "internal/server/store/migrations",
		},
		{
			name:    "hub defaults",
			dbFlag:  "hub",
			wantDB:  "postgres://patchiq:patchiq_dev@localhost:5432/patchiq_hub?sslmode=disable",
			wantDir: "internal/hub/store/migrations",
		},
		{
			name:    "server env override",
			dbFlag:  "server",
			envKey:  "PATCHIQ_DB_URL",
			envVal:  "postgres://custom:pass@db:5432/mydb",
			wantDB:  "postgres://custom:pass@db:5432/mydb",
			wantDir: "internal/server/store/migrations",
		},
		{
			name:    "hub env override",
			dbFlag:  "hub",
			envKey:  "PATCHIQ_HUB_DB_URL",
			envVal:  "postgres://custom:pass@db:5432/myhub",
			wantDB:  "postgres://custom:pass@db:5432/myhub",
			wantDir: "internal/hub/store/migrations",
		},
		{
			name:       "invalid db flag",
			dbFlag:     "agent",
			wantErrMsg: "invalid --db value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}
			cfg, err := resolveConfig(tt.dbFlag)
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantDB, cfg.dsn)
			assert.Equal(t, tt.wantDir, cfg.migrationsDir)
		})
	}
}
