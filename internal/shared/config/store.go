package config

import (
	"context"
	"encoding/json"
	"errors"
)

// ErrNoOverride is returned when no override exists for the given scope.
var ErrNoOverride = errors.New("no config override found")

// ConfigStore abstracts database access for config overrides.
type ConfigStore interface {
	GetOverride(ctx context.Context, tenantID, scopeType, scopeID, module string) (json.RawMessage, error)
}
