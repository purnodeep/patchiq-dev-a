package compliance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// CustomFrameworkQuerier defines the queries needed to load custom frameworks.
type CustomFrameworkQuerier interface {
	GetCustomFramework(ctx context.Context, arg sqlcgen.GetCustomFrameworkParams) (sqlcgen.CustomComplianceFramework, error)
	ListCustomControls(ctx context.Context, arg sqlcgen.ListCustomControlsParams) ([]sqlcgen.CustomComplianceControl, error)
}

// DBCustomFrameworkLoader loads custom frameworks from PostgreSQL.
type DBCustomFrameworkLoader struct {
	q CustomFrameworkQuerier
}

// NewDBCustomFrameworkLoader creates a loader backed by the given querier.
func NewDBCustomFrameworkLoader(q CustomFrameworkQuerier) *DBCustomFrameworkLoader {
	return &DBCustomFrameworkLoader{q: q}
}

// LoadCustomFramework loads a custom framework by its UUID string ID and converts it
// to the Framework struct used by the evaluator.
func (l *DBCustomFrameworkLoader) LoadCustomFramework(ctx context.Context, tenantID pgtype.UUID, frameworkID string) (*Framework, error) {
	fwUUID, err := uuid.Parse(frameworkID)
	if err != nil {
		return nil, fmt.Errorf("parse custom framework ID %q: %w", frameworkID, err)
	}

	fwPgID := pgtype.UUID{Bytes: fwUUID, Valid: true}

	dbFW, err := l.q.GetCustomFramework(ctx, sqlcgen.GetCustomFrameworkParams{
		ID:       fwPgID,
		TenantID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("get custom framework %s: %w", frameworkID, err)
	}

	dbControls, err := l.q.ListCustomControls(ctx, sqlcgen.ListCustomControlsParams{
		FrameworkID: fwPgID,
		TenantID:    tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("list controls for custom framework %s: %w", frameworkID, err)
	}

	fw := &Framework{
		ID:                   frameworkID,
		Name:                 dbFW.Name,
		Version:              dbFW.Version,
		Description:          textVal(dbFW.Description),
		DefaultScoringMethod: dbFW.ScoringMethod,
	}

	for _, dbc := range dbControls {
		control := Control{
			ID:              dbc.ControlID,
			Name:            dbc.Name,
			Description:     textVal(dbc.Description),
			Category:        dbc.Category,
			RemediationHint: textVal(dbc.RemediationHint),
			CheckType:       dbc.CheckType,
			CheckConfig:     dbc.CheckConfig,
		}

		// Parse SLA tiers from JSONB.
		if len(dbc.SlaTiers) > 0 {
			var tiers []SeverityTier
			if err := json.Unmarshal(dbc.SlaTiers, &tiers); err == nil {
				control.SLATiers = tiers
			}
		}

		fw.Controls = append(fw.Controls, control)
	}

	return fw, nil
}

func textVal(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}
