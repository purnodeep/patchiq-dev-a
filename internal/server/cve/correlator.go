package cve

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// PatchInfo represents a patch that may be correlated with a CVE.
type PatchInfo struct {
	ID      string
	Name    string
	Version string
}

// CVEPatchLink represents a link between a patch and a CVE database entry.
type CVEPatchLink struct {
	PatchID string
	CVEDBID string
}

// PatchLister retrieves patches by package name for a given tenant.
type PatchLister interface {
	ListPatchesByName(ctx context.Context, tenantID, packageName string) ([]PatchInfo, error)
}

// CVELinker persists a link between a patch and a CVE.
type CVELinker interface {
	LinkPatchCVE(ctx context.Context, tenantID, patchID, cveDBID, versionEndExcluding, versionEndIncluding string) error
}

// Correlator matches CVE records against known patches and links them.
type Correlator struct {
	lister PatchLister
	linker CVELinker
}

// NewCorrelator creates a Correlator with the given dependencies.
func NewCorrelator(lister PatchLister, linker CVELinker) *Correlator {
	return &Correlator{lister: lister, linker: linker}
}

// Correlate links CVE records to patches for the given tenant. Returns the number of links created.
// Errors from individual ListPatchesByName or LinkPatchCVE calls are logged and collected;
// processing continues so that one failure does not block the entire batch.
func (c *Correlator) Correlate(ctx context.Context, tenantID string, records []CVERecord, cveDBIDs map[string]string) (int, error) {
	linked := 0
	var errs []error
	for _, rec := range records {
		dbID, ok := cveDBIDs[rec.CVEID]
		if !ok {
			continue
		}
		for _, pkg := range rec.AffectedPackages {
			patches, err := c.lister.ListPatchesByName(ctx, tenantID, pkg.PackageName)
			if err != nil {
				slog.ErrorContext(ctx, "correlate: list patches failed",
					"tenant_id", tenantID, "cve_id", rec.CVEID, "package", pkg.PackageName, "error", err)
				errs = append(errs, fmt.Errorf("list patches for %s/%s: %w", rec.CVEID, pkg.PackageName, err))
				continue
			}
			for _, p := range patches {
				if err := c.linker.LinkPatchCVE(ctx, tenantID, p.ID, dbID, pkg.VersionEndExcluding, pkg.VersionEndIncluding); err != nil {
					slog.ErrorContext(ctx, "correlate: link patch CVE failed",
						"tenant_id", tenantID, "cve_id", rec.CVEID, "patch_id", p.ID, "error", err)
					errs = append(errs, fmt.Errorf("link patch %s to CVE %s: %w", p.ID, rec.CVEID, err))
					continue
				}
				linked++
			}
		}
	}
	return linked, errors.Join(errs...)
}
