//go:build integration

package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DashboardExpected holds expected counts for dashboard summary verification.
type DashboardExpected struct {
	EndpointsTotal   int32
	PatchesAvailable int32
	CvesUnpatched    int32
}

// WaitFor polls fn at the given interval until it returns true or the timeout
// elapses. If the timeout is reached, the test fails with the provided
// description.
func WaitFor(t *testing.T, timeout, interval time.Duration, desc string, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("WaitFor timed out after %s: %s", timeout, desc)
}

// AssertEndpointExists verifies that at least one endpoint exists for the given
// tenant and returns its ID. Fails the test if none is found.
func AssertEndpointExists(t *testing.T, pool *pgxpool.Pool, tenantID string) string {
	t.Helper()

	id := TryGetEndpointID(t, pool, tenantID)
	if id == "" {
		t.Fatalf("assert endpoint exists: no endpoint found for tenant %s", tenantID)
	}
	return id
}

// TryGetEndpointID returns the first endpoint ID for the given tenant, or ""
// if none is found. Does not fail the test.
func TryGetEndpointID(t *testing.T, pool *pgxpool.Pool, tenantID string) string {
	t.Helper()

	var id string
	err := pool.QueryRow(
		context.Background(),
		`SELECT id::text FROM endpoints WHERE tenant_id = $1 LIMIT 1`,
		tenantID,
	).Scan(&id)
	if err != nil {
		return ""
	}
	return id
}

// AssertPackageCount verifies that the number of packages for the given
// endpoint is at least minCount. Fails the test if the count is too low.
func AssertPackageCount(t *testing.T, pool *pgxpool.Pool, tenantID, endpointID string, minCount int) {
	t.Helper()

	count := TryPackageCount(t, pool, tenantID, endpointID)
	if count < minCount {
		t.Fatalf("assert package count: got %d, want at least %d (tenant=%s, endpoint=%s)",
			count, minCount, tenantID, endpointID)
	}
}

// TryPackageCount returns the number of packages for the given endpoint, or 0
// if the query fails. Does not fail the test.
func TryPackageCount(t *testing.T, pool *pgxpool.Pool, tenantID, endpointID string) int {
	t.Helper()

	var count int
	err := pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*)::int FROM endpoint_packages WHERE tenant_id = $1 AND endpoint_id = $2`,
		tenantID, endpointID,
	).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// AssertPackageExists verifies that a specific package exists for the given
// endpoint. Fails the test if the package is not found.
func AssertPackageExists(t *testing.T, pool *pgxpool.Pool, tenantID, endpointID, packageName string) {
	t.Helper()

	var exists bool
	err := pool.QueryRow(
		context.Background(),
		`SELECT EXISTS(SELECT 1 FROM endpoint_packages WHERE tenant_id = $1 AND endpoint_id = $2 AND package_name = $3)`,
		tenantID, endpointID, packageName,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("assert package exists: query failed: %v", err)
	}
	if !exists {
		t.Fatalf("assert package exists: package %q not found (tenant=%s, endpoint=%s)",
			packageName, tenantID, endpointID)
	}
}

// AssertCVEExists verifies that a CVE record with the given cve_id text
// (e.g. "CVE-2024-1234") exists for the tenant. Fails the test if not found.
func AssertCVEExists(t *testing.T, pool *pgxpool.Pool, tenantID, cveID string) {
	t.Helper()

	var exists bool
	err := pool.QueryRow(
		context.Background(),
		`SELECT EXISTS(SELECT 1 FROM cves WHERE tenant_id = $1 AND cve_id = $2)`,
		tenantID, cveID,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("assert CVE exists: query failed: %v", err)
	}
	if !exists {
		t.Fatalf("assert CVE exists: CVE %q not found (tenant=%s)", cveID, tenantID)
	}
}

// AssertPatchExists verifies that a patch with the given package name exists
// for the tenant and returns its ID. Fails the test if not found.
func AssertPatchExists(t *testing.T, pool *pgxpool.Pool, tenantID, packageName string) string {
	t.Helper()

	var id string
	err := pool.QueryRow(
		context.Background(),
		`SELECT id::text FROM patches WHERE tenant_id = $1 AND name = $2 LIMIT 1`,
		tenantID, packageName,
	).Scan(&id)
	if err != nil {
		t.Fatalf("assert patch exists: patch %q not found (tenant=%s): %v",
			packageName, tenantID, err)
	}
	return id
}

// AssertDeploymentStatus verifies that the deployment has the expected status.
// Fails the test if the status does not match.
func AssertDeploymentStatus(t *testing.T, pool *pgxpool.Pool, tenantID, deploymentID, expectedStatus string) {
	t.Helper()

	actual := TryDeploymentStatus(t, pool, tenantID, deploymentID)
	if actual == "" {
		t.Fatalf("assert deployment status: deployment %s not found (tenant=%s)",
			deploymentID, tenantID)
	}
	if actual != expectedStatus {
		t.Fatalf("assert deployment status: got %q, want %q (tenant=%s, deployment=%s)",
			actual, expectedStatus, tenantID, deploymentID)
	}
}

// TryDeploymentStatus returns the status of the given deployment, or "" if the
// query fails or the deployment is not found. Does not fail the test.
func TryDeploymentStatus(t *testing.T, pool *pgxpool.Pool, tenantID, deploymentID string) string {
	t.Helper()

	var status string
	err := pool.QueryRow(
		context.Background(),
		`SELECT status FROM deployments WHERE tenant_id = $1 AND id = $2`,
		tenantID, deploymentID,
	).Scan(&status)
	if err != nil {
		return ""
	}
	return status
}

// AssertAuditEventTypes verifies that the given audit event types all exist
// for the tenant. Fails the test if any expected type is missing.
func AssertAuditEventTypes(t *testing.T, pool *pgxpool.Pool, tenantID string, expectedTypes []string) {
	t.Helper()

	rows, err := pool.Query(
		context.Background(),
		`SELECT DISTINCT type FROM audit_events WHERE tenant_id = $1`,
		tenantID,
	)
	if err != nil {
		t.Fatalf("assert audit event types: query failed: %v", err)
	}
	defer rows.Close()

	found := make(map[string]bool)
	for rows.Next() {
		var eventType string
		if err := rows.Scan(&eventType); err != nil {
			t.Fatalf("assert audit event types: scan failed: %v", err)
		}
		found[eventType] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("assert audit event types: rows iteration failed: %v", err)
	}

	var missing []string
	for _, et := range expectedTypes {
		if !found[et] {
			missing = append(missing, et)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("assert audit event types: missing types %v (tenant=%s, found=%v)",
			missing, tenantID, found)
	}
}

// AssertDashboardSummary verifies aggregate counts for the tenant dashboard.
// It checks endpoint count, available patch count, and unpatched CVE count.
// Fails the test if any count does not match.
func AssertDashboardSummary(t *testing.T, pool *pgxpool.Pool, tenantID string, expected DashboardExpected) {
	t.Helper()

	ctx := context.Background()
	var errs []string

	var endpointCount int32
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM endpoints WHERE tenant_id = $1`,
		tenantID,
	).Scan(&endpointCount)
	if err != nil {
		t.Fatalf("assert dashboard summary: endpoint count query failed: %v", err)
	}
	if endpointCount != expected.EndpointsTotal {
		errs = append(errs, fmt.Sprintf("endpoints: got %d, want %d", endpointCount, expected.EndpointsTotal))
	}

	var patchCount int32
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM patches WHERE tenant_id = $1 AND status = 'available'`,
		tenantID,
	).Scan(&patchCount)
	if err != nil {
		t.Fatalf("assert dashboard summary: patch count query failed: %v", err)
	}
	if patchCount != expected.PatchesAvailable {
		errs = append(errs, fmt.Sprintf("patches_available: got %d, want %d", patchCount, expected.PatchesAvailable))
	}

	var cveCount int32
	err = pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT c.id)::int FROM cves c
		 LEFT JOIN endpoint_cves ec ON ec.cve_id = c.id AND ec.status = 'patched'
		 WHERE c.tenant_id = $1 AND ec.id IS NULL`,
		tenantID,
	).Scan(&cveCount)
	if err != nil {
		t.Fatalf("assert dashboard summary: CVE count query failed: %v", err)
	}
	if cveCount != expected.CvesUnpatched {
		errs = append(errs, fmt.Sprintf("cves_unpatched: got %d, want %d", cveCount, expected.CvesUnpatched))
	}

	if len(errs) > 0 {
		t.Fatalf("assert dashboard summary (tenant=%s): mismatches: %v", tenantID, errs)
	}
}
