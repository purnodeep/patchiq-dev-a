package targeting_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/skenzeriq/patchiq/internal/server/targeting"
)

// resolverTestEnv captures everything a test needs to run against a real
// Postgres instance: a superuser pool for seeding, an app-role pool for the
// resolver itself (RLS is only enforced for non-superusers), and two tenant
// IDs each with a known set of tagged endpoints.
type resolverTestEnv struct {
	super    *pgxpool.Pool
	app      *pgxpool.Pool
	tenantA  string
	tenantB  string
	endpoint map[string]uuid.UUID // name -> id
	cleanup  func()
}

const (
	envTestDBName  = "patchiq_targeting_test"
	envTestDBUser  = "postgres"
	envTestDBPass  = "postgres"
	envAppRolePass = "test_app_pass"
)

func setupResolverEnv(t *testing.T) *resolverTestEnv {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(envTestDBName),
		postgres.WithUsername(envTestDBUser),
		postgres.WithPassword(envTestDBPass),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	super, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("super pool: %v", err)
	}

	// Locate the migrations directory relative to this test file.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	migrationsDir := filepath.Join(filepath.Dir(filename), "..", "store", "migrations")

	if err := runMigrations(ctx, super, migrationsDir); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// ALTER ROLE ... WITH PASSWORD cannot use $1 parameters; the password is
	// a constant so there is no injection risk.
	if _, err := super.Exec(ctx,
		fmt.Sprintf("ALTER ROLE patchiq_app WITH PASSWORD '%s'", envAppRolePass),
	); err != nil {
		t.Fatalf("set app role password: %v", err)
	}

	cfg := super.Config().ConnConfig
	appConnStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=patchiq_app password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database, envAppRolePass,
	)
	app, err := pgxpool.New(ctx, appConnStr)
	if err != nil {
		t.Fatalf("app pool: %v", err)
	}

	env := &resolverTestEnv{
		super:    super,
		app:      app,
		endpoint: make(map[string]uuid.UUID),
		cleanup: func() {
			app.Close()
			super.Close()
			_ = pgContainer.Terminate(ctx)
		},
	}

	// Seed two tenants and a known topology of endpoints + tags.
	env.tenantA = insertTenant(t, super, "Tenant A", "tenant-a")
	env.tenantB = insertTenant(t, super, "Tenant B", "tenant-b")

	// Tenant A endpoints:
	//   a-prod-linux:   env=prod  os=ubuntu  region=us-east
	//   a-staging-mac:  env=stage os=macos   region=us-east
	//   a-prod-debian:  env=prod  os=debian
	//   a-decom:        env=prod  os=ubuntu  (status=decommissioned — must be excluded)
	// Tenant B endpoint:
	//   b-prod-linux:   env=prod  os=ubuntu   (tenant isolation check — must never appear in A's results)
	env.endpoint["a-prod-linux"] = insertEndpoint(t, super, env.tenantA, "a-prod-linux", "online")
	env.endpoint["a-staging-mac"] = insertEndpoint(t, super, env.tenantA, "a-staging-mac", "online")
	env.endpoint["a-prod-debian"] = insertEndpoint(t, super, env.tenantA, "a-prod-debian", "online")
	env.endpoint["a-decom"] = insertEndpoint(t, super, env.tenantA, "a-decom", "decommissioned")
	env.endpoint["b-prod-linux"] = insertEndpoint(t, super, env.tenantB, "b-prod-linux", "online")

	assignTags(t, super, env.tenantA, env.endpoint["a-prod-linux"],
		kv{"env", "prod"}, kv{"os", "ubuntu"}, kv{"region", "us-east"})
	assignTags(t, super, env.tenantA, env.endpoint["a-staging-mac"],
		kv{"env", "stage"}, kv{"os", "macos"}, kv{"region", "us-east"})
	assignTags(t, super, env.tenantA, env.endpoint["a-prod-debian"],
		kv{"env", "prod"}, kv{"os", "debian"})
	assignTags(t, super, env.tenantA, env.endpoint["a-decom"],
		kv{"env", "prod"}, kv{"os", "ubuntu"})
	assignTags(t, super, env.tenantB, env.endpoint["b-prod-linux"],
		kv{"env", "prod"}, kv{"os", "ubuntu"})

	return env
}

type kv struct{ k, v string }

func insertTenant(t *testing.T, pool *pgxpool.Pool, name, slug string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		"INSERT INTO tenants (name, slug) VALUES ($1, $2) RETURNING id::text",
		name, slug,
	).Scan(&id); err != nil {
		t.Fatalf("insert tenant %s: %v", slug, err)
	}
	return id
}

func insertEndpoint(t *testing.T, pool *pgxpool.Pool, tenantID, hostname, status string) uuid.UUID {
	t.Helper()
	var raw string
	if err := pool.QueryRow(context.Background(),
		"INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status) VALUES ($1, $2, 'linux', '22.04', $3) RETURNING id::text",
		tenantID, hostname, status,
	).Scan(&raw); err != nil {
		t.Fatalf("insert endpoint %s: %v", hostname, err)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		t.Fatalf("parse endpoint id %s: %v", raw, err)
	}
	return id
}

// assignTags creates (or reuses) tag rows for each key/value pair and links
// them to the endpoint via endpoint_tags. Uses the superuser pool to bypass
// RLS during setup.
func assignTags(t *testing.T, pool *pgxpool.Pool, tenantID string, endpointID uuid.UUID, pairs ...kv) {
	t.Helper()
	ctx := context.Background()
	for _, p := range pairs {
		var tagID string
		// Migration 060 drops `name` and makes (tenant_id, lower(key),
		// lower(value)) the unique key. Keys are constrained to lowercase
		// by chk_tags_key_lowercase.
		err := pool.QueryRow(ctx,
			`INSERT INTO tags (tenant_id, key, value)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (tenant_id, lower(key), lower(value)) DO UPDATE SET value = EXCLUDED.value
			 RETURNING id::text`,
			tenantID, p.k, p.v,
		).Scan(&tagID)
		if err != nil {
			t.Fatalf("upsert tag %s=%s: %v", p.k, p.v, err)
		}
		if _, err := pool.Exec(ctx,
			"INSERT INTO endpoint_tags (endpoint_id, tag_id, tenant_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
			endpointID, tagID, tenantID,
		); err != nil {
			t.Fatalf("link endpoint_tag: %v", err)
		}
	}
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		upSQL, err := extractUpSection(string(content))
		if err != nil {
			return fmt.Errorf("extract up from %s: %w", filepath.Base(f), err)
		}
		if upSQL == "" {
			continue
		}
		if strings.Contains(string(content), "-- +goose NO TRANSACTION") {
			upSQL = strings.ReplaceAll(upSQL, "CONCURRENTLY ", "")
		}
		if _, err := pool.Exec(ctx, upSQL); err != nil {
			return fmt.Errorf("exec %s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

func extractUpSection(content string) (string, error) {
	const up = "-- +goose Up"
	const down = "-- +goose Down"
	upIdx := strings.Index(content, up)
	if upIdx == -1 {
		return "", fmt.Errorf("missing %q", up)
	}
	upIdx += len(up)
	downIdx := strings.Index(content, down)
	if downIdx == -1 {
		return strings.TrimSpace(content[upIdx:]), nil
	}
	return strings.TrimSpace(content[upIdx:downIdx]), nil
}

// -------- tests --------

func TestResolver_Resolve(t *testing.T) {
	env := setupResolverEnv(t)
	defer env.cleanup()

	r := targeting.NewResolver(env.app)
	ctx := context.Background()

	tests := []struct {
		name    string
		tenant  string
		sel     *targeting.Selector
		wantSet map[string]bool // hostnames that must appear; everything else forbidden
	}{
		{
			"nil selector matches all non-decommissioned in tenant",
			env.tenantA,
			nil,
			map[string]bool{"a-prod-linux": true, "a-staging-mac": true, "a-prod-debian": true},
		},
		{
			"eq env=prod",
			env.tenantA,
			&targeting.Selector{Op: targeting.OpEq, Key: "env", Value: "prod"},
			map[string]bool{"a-prod-linux": true, "a-prod-debian": true},
		},
		{
			"in os {ubuntu,debian}",
			env.tenantA,
			&targeting.Selector{Op: targeting.OpIn, Key: "os", Values: []string{"ubuntu", "debian"}},
			map[string]bool{"a-prod-linux": true, "a-prod-debian": true},
		},
		{
			"exists region",
			env.tenantA,
			&targeting.Selector{Op: targeting.OpExists, Key: "region"},
			map[string]bool{"a-prod-linux": true, "a-staging-mac": true},
		},
		{
			"and env=prod and os=ubuntu",
			env.tenantA,
			&targeting.Selector{Op: targeting.OpAnd, Args: []targeting.Selector{
				{Op: targeting.OpEq, Key: "env", Value: "prod"},
				{Op: targeting.OpEq, Key: "os", Value: "ubuntu"},
			}},
			map[string]bool{"a-prod-linux": true},
		},
		{
			"or env=prod or env=stage",
			env.tenantA,
			&targeting.Selector{Op: targeting.OpOr, Args: []targeting.Selector{
				{Op: targeting.OpEq, Key: "env", Value: "prod"},
				{Op: targeting.OpEq, Key: "env", Value: "stage"},
			}},
			map[string]bool{"a-prod-linux": true, "a-staging-mac": true, "a-prod-debian": true},
		},
		{
			"not env=prod",
			env.tenantA,
			&targeting.Selector{Op: targeting.OpNot, Arg: &targeting.Selector{
				Op: targeting.OpEq, Key: "env", Value: "prod",
			}},
			map[string]bool{"a-staging-mac": true},
		},
		{
			"cross-tenant isolation: tenant B sees only its own endpoint",
			env.tenantB,
			&targeting.Selector{Op: targeting.OpEq, Key: "env", Value: "prod"},
			map[string]bool{"b-prod-linux": true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ids, err := r.Resolve(ctx, tc.tenant, tc.sel)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			got := hostnamesOf(t, env.super, ids)
			assertSameSet(t, got, tc.wantSet)
		})
	}
}

// TestResolver_RLS_SameKeyValueDifferentTenants proves that when both
// tenants carry the same (key,value) tag, the RLS boundary stops a cross-
// tenant read even though the compiled SQL fragment is identical. This is
// the single most important regression guard for the resolver.
func TestResolver_RLS_SameKeyValueDifferentTenants(t *testing.T) {
	env := setupResolverEnv(t)
	defer env.cleanup()
	r := targeting.NewResolver(env.app)
	ctx := context.Background()

	// Both tenants already have env=prod endpoints from seed. Resolve for A
	// and confirm only A's endpoints appear.
	aIDs, err := r.Resolve(ctx, env.tenantA, &targeting.Selector{Op: targeting.OpEq, Key: "env", Value: "prod"})
	if err != nil {
		t.Fatalf("Resolve(A): %v", err)
	}
	got := hostnamesOf(t, env.super, aIDs)
	assertSameSet(t, got, map[string]bool{"a-prod-linux": true, "a-prod-debian": true})
	if got["b-prod-linux"] {
		t.Error("tenant A saw tenant B endpoint — RLS breach")
	}

	bIDs, err := r.Resolve(ctx, env.tenantB, &targeting.Selector{Op: targeting.OpEq, Key: "env", Value: "prod"})
	if err != nil {
		t.Fatalf("Resolve(B): %v", err)
	}
	got = hostnamesOf(t, env.super, bIDs)
	assertSameSet(t, got, map[string]bool{"b-prod-linux": true})
}

// TestResolver_TenantContextDoesNotLeakAcrossConnections pins the
// SET LOCAL semantics: after a Resolve for tenant A, a fresh acquisition
// from the same pool must not carry A's tenant context. A regression to
// SET without LOCAL would make this test fail.
func TestResolver_TenantContextDoesNotLeakAcrossConnections(t *testing.T) {
	env := setupResolverEnv(t)
	defer env.cleanup()
	r := targeting.NewResolver(env.app)
	ctx := context.Background()

	// Force many serialized Resolves so that the pool's connections get
	// reused. If SET LOCAL is not actually local, the second loop would
	// observe tenant A's context on a connection that last served A.
	for i := 0; i < 10; i++ {
		if _, err := r.Resolve(ctx, env.tenantA, &targeting.Selector{Op: targeting.OpEq, Key: "env", Value: "prod"}); err != nil {
			t.Fatalf("warmup Resolve: %v", err)
		}
	}

	// Acquire a raw connection and verify app.current_tenant_id is unset.
	// set_config('app.current_tenant_id', '', false) returns '' for an
	// unset parameter when called with missing_ok=true via current_setting.
	conn, err := env.app.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer conn.Release()

	var leaked string
	// current_setting(name, missing_ok) returns '' for a tx-local setting
	// that is no longer in scope outside its original transaction.
	if err := conn.QueryRow(ctx, "SELECT current_setting('app.current_tenant_id', true)").Scan(&leaked); err != nil {
		t.Fatalf("SELECT current_setting: %v", err)
	}
	if leaked != "" {
		t.Errorf("tenant context leaked onto pooled connection: %q", leaked)
	}
}

// TestResolver_CaseInsensitiveMatching verifies that the lower(...) =
// lower(...) comparisons in the compiled fragment actually behave the way
// the compiler comment promises.
//
// Since migration 059's chk_tags_key_lowercase CHECK constraint, the key
// column is guaranteed lowercase at the DB level — so this test stores a
// canonical lowercase key and asserts that callers passing mixed/upper
// case via the selector still match. Value comparison remains
// case-insensitive in the compiler because tags.value is free-form.
func TestResolver_CaseInsensitiveMatching(t *testing.T) {
	env := setupResolverEnv(t)
	defer env.cleanup()
	ctx := context.Background()

	ep := insertEndpoint(t, env.super, env.tenantA, "case-host", "online")
	// Store a canonical lowercase key with a mixed-case value; the CHECK
	// constraint only governs `key`, so values are free to be uppercase.
	var tagID string
	if err := env.super.QueryRow(ctx,
		`INSERT INTO tags (tenant_id, key, value) VALUES ($1, $2, $3) RETURNING id::text`,
		env.tenantA, "env", "Staging",
	).Scan(&tagID); err != nil {
		t.Fatalf("insert tag: %v", err)
	}
	if _, err := env.super.Exec(ctx,
		"INSERT INTO endpoint_tags (endpoint_id, tag_id, tenant_id) VALUES ($1, $2, $3)",
		ep, tagID, env.tenantA,
	); err != nil {
		t.Fatalf("link endpoint_tag: %v", err)
	}

	r := targeting.NewResolver(env.app)

	// Query with uppercase key + uppercase value; must match the
	// canonical lowercase 'env' + stored 'Staging' via lower() on both
	// sides of the compiled comparison.
	ids, err := r.Resolve(ctx, env.tenantA, &targeting.Selector{Op: targeting.OpEq, Key: "ENV", Value: "STAGING"})
	if err != nil {
		t.Fatalf("Resolve eq uppercase: %v", err)
	}
	if !containsID(ids, ep) {
		t.Error("eq ENV=STAGING did not match env=Staging")
	}

	// OpIn with mixed case too.
	ids, err = r.Resolve(ctx, env.tenantA, &targeting.Selector{
		Op: targeting.OpIn, Key: "Env", Values: []string{"STAGING", "prod"},
	})
	if err != nil {
		t.Fatalf("Resolve in mixed case: %v", err)
	}
	if !containsID(ids, ep) {
		t.Error("in Env {STAGING,prod} did not match env=Staging")
	}

	// OpExists with uppercase key must match the lowercase-canonical row.
	ids, err = r.Resolve(ctx, env.tenantA, &targeting.Selector{Op: targeting.OpExists, Key: "ENV"})
	if err != nil {
		t.Fatalf("Resolve exists uppercase: %v", err)
	}
	if !containsID(ids, ep) {
		t.Error("exists ENV did not match env=Staging")
	}
}

// TestResolver_ResolveForPolicy_UnknownPolicyReturnsErr pins the
// critical safety property: a policyID that does not exist must not
// resolve to "match all endpoints". This is the deployment-to-everyone
// footgun the silent-failure review caught.
func TestResolver_ResolveForPolicy_UnknownPolicyReturnsErr(t *testing.T) {
	env := setupResolverEnv(t)
	defer env.cleanup()
	r := targeting.NewResolver(env.app)
	ctx := context.Background()

	_, err := r.ResolveForPolicy(ctx, env.tenantA, uuid.New().String())
	if !errors.Is(err, targeting.ErrPolicyNotFound) {
		t.Errorf("want ErrPolicyNotFound, got %v", err)
	}

	// Garbage non-UUID input must also fail cleanly.
	_, err = r.ResolveForPolicy(ctx, env.tenantA, "not-a-uuid")
	if !errors.Is(err, targeting.ErrInvalidPolicyID) {
		t.Errorf("want ErrInvalidPolicyID, got %v", err)
	}

	// Garbage tenant input must fail cleanly.
	_, err = r.Resolve(ctx, "not-a-uuid", nil)
	if !errors.Is(err, targeting.ErrInvalidTenantID) {
		t.Errorf("want ErrInvalidTenantID, got %v", err)
	}
}

// TestResolver_ResolveForPolicy_RejectsInvalidStoredSelector guards
// against a malicious or corrupted policy_tag_selectors row: it must be
// rejected at Validate time, never producing broken SQL.
func TestResolver_ResolveForPolicy_RejectsInvalidStoredSelector(t *testing.T) {
	env := setupResolverEnv(t)
	defer env.cleanup()
	ctx := context.Background()

	// Seed distinct policies for each corrupt-expression case. Reuse a
	// single container to keep the test fast — each subtest gets its own
	// policy row.
	cases := []struct {
		name    string
		expr    []byte
		wantSub string
	}{
		{
			"unknown op",
			[]byte(`{"op":"bogus","key":"x"}`),
			"unknown op",
		},
		{
			// Empty composite is the most likely admin-import corruption
			// shape: the JSON parses, the op is known, but Validate rejects
			// it at the op=and 'requires at least one arg' check.
			"empty and composite",
			[]byte(`{"op":"and","args":[]}`),
			"at least one arg",
		},
		{
			// An unrecognised envelope version must be rejected at decode
			// time so a newer row stored by a future binary cannot be
			// silently mis-executed by an older one.
			"unsupported schema version",
			[]byte(`{"v":99,"op":"eq","key":"env","value":"prod"}`),
			"unsupported selector schema version",
		},
	}

	r := targeting.NewResolver(env.app)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var policyID string
			if err := env.super.QueryRow(ctx,
				"INSERT INTO policies (tenant_id, name) VALUES ($1, $2) RETURNING id::text",
				env.tenantA, "bad-selector-"+tc.name,
			).Scan(&policyID); err != nil {
				t.Fatalf("insert policy: %v", err)
			}
			if _, err := env.super.Exec(ctx,
				"INSERT INTO policy_tag_selectors (policy_id, tenant_id, expression) VALUES ($1, $2, $3)",
				policyID, env.tenantA, tc.expr,
			); err != nil {
				t.Fatalf("insert bad selector: %v", err)
			}
			_, err := r.ResolveForPolicy(ctx, env.tenantA, policyID)
			if err == nil {
				t.Fatal("want error for bogus stored selector, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("want error containing %q, got %v", tc.wantSub, err)
			}
		})
	}
}

// TestResolver_ResolveForPolicy_CrossTenantIsolation guards the RLS
// path on policy_tag_selectors: a caller in tenant B must never be able
// to resolve a policy owned by tenant A, even if they guess the UUID.
// Today the safety net comes from policyExistsTx filtering on tenant_id,
// but this test also exercises the RLS policy on policy_tag_selectors —
// if a future refactor drops the tenant_id predicate from policyExistsTx,
// RLS is the second line of defense and we want a test that proves the
// overall outcome (ErrPolicyNotFound) holds regardless.
func TestResolver_ResolveForPolicy_CrossTenantIsolation(t *testing.T) {
	env := setupResolverEnv(t)
	defer env.cleanup()
	ctx := context.Background()

	// Seed a policy + selector in tenant A.
	var policyID string
	if err := env.super.QueryRow(ctx,
		"INSERT INTO policies (tenant_id, name) VALUES ($1, $2) RETURNING id::text",
		env.tenantA, "tenant-a-rollout",
	).Scan(&policyID); err != nil {
		t.Fatalf("insert tenantA policy: %v", err)
	}
	if _, err := env.super.Exec(ctx,
		"INSERT INTO policy_tag_selectors (policy_id, tenant_id, expression) VALUES ($1, $2, $3)",
		policyID, env.tenantA, []byte(`{"op":"eq","key":"env","value":"prod"}`),
	); err != nil {
		t.Fatalf("insert tenantA selector: %v", err)
	}

	r := targeting.NewResolver(env.app)

	// Sanity: tenant A can still resolve its own policy.
	if _, err := r.ResolveForPolicy(ctx, env.tenantA, policyID); err != nil {
		t.Fatalf("tenant A resolve own policy: %v", err)
	}

	// Tenant B passes a valid UUID that happens to belong to tenant A.
	// Must return ErrPolicyNotFound — neither the policy row nor the
	// selector row is visible under tenant B's RLS context.
	_, err := r.ResolveForPolicy(ctx, env.tenantB, policyID)
	if !errors.Is(err, targeting.ErrPolicyNotFound) {
		t.Errorf("cross-tenant ResolveForPolicy: want ErrPolicyNotFound, got %v", err)
	}
}

// containsID returns true if needle is present in ids.
func containsID(ids []uuid.UUID, needle uuid.UUID) bool {
	for _, id := range ids {
		if id == needle {
			return true
		}
	}
	return false
}

func TestResolver_Count(t *testing.T) {
	env := setupResolverEnv(t)
	defer env.cleanup()

	r := targeting.NewResolver(env.app)
	ctx := context.Background()

	n, err := r.Count(ctx, env.tenantA, &targeting.Selector{
		Op: targeting.OpEq, Key: "env", Value: "prod",
	})
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 2 {
		t.Errorf("Count env=prod = %d, want 2", n)
	}
}

func TestResolver_ResolveForPolicy(t *testing.T) {
	env := setupResolverEnv(t)
	defer env.cleanup()
	ctx := context.Background()

	// Seed a policy for tenant A and attach a selector targeting env=prod.
	var policyID string
	if err := env.super.QueryRow(ctx,
		"INSERT INTO policies (tenant_id, name) VALUES ($1, $2) RETURNING id::text",
		env.tenantA, "prod-rollout",
	).Scan(&policyID); err != nil {
		t.Fatalf("insert policy: %v", err)
	}

	expr := []byte(`{"op":"eq","key":"env","value":"prod"}`)
	if _, err := env.super.Exec(ctx,
		"INSERT INTO policy_tag_selectors (policy_id, tenant_id, expression) VALUES ($1, $2, $3)",
		policyID, env.tenantA, expr,
	); err != nil {
		t.Fatalf("insert policy_tag_selectors: %v", err)
	}

	r := targeting.NewResolver(env.app)
	ids, err := r.ResolveForPolicy(ctx, env.tenantA, policyID)
	if err != nil {
		t.Fatalf("ResolveForPolicy: %v", err)
	}
	got := hostnamesOf(t, env.super, ids)
	assertSameSet(t, got, map[string]bool{"a-prod-linux": true, "a-prod-debian": true})

	// Policy with no selector row must match every non-decommissioned endpoint.
	var policyID2 string
	if err := env.super.QueryRow(ctx,
		"INSERT INTO policies (tenant_id, name) VALUES ($1, $2) RETURNING id::text",
		env.tenantA, "all-hosts",
	).Scan(&policyID2); err != nil {
		t.Fatalf("insert policy 2: %v", err)
	}
	ids2, err := r.ResolveForPolicy(ctx, env.tenantA, policyID2)
	if err != nil {
		t.Fatalf("ResolveForPolicy (no selector): %v", err)
	}
	got2 := hostnamesOf(t, env.super, ids2)
	assertSameSet(t, got2, map[string]bool{"a-prod-linux": true, "a-staging-mac": true, "a-prod-debian": true})
}

// hostnamesOf resolves a slice of endpoint UUIDs to their hostnames via the
// superuser pool (bypassing RLS for the assertion helper).
func hostnamesOf(t *testing.T, pool *pgxpool.Pool, ids []uuid.UUID) map[string]bool {
	t.Helper()
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		var name string
		err := pool.QueryRow(context.Background(),
			"SELECT hostname FROM endpoints WHERE id = $1", id.String(),
		).Scan(&name)
		if err != nil {
			t.Fatalf("lookup hostname for %s: %v", id, err)
		}
		out[name] = true
	}
	return out
}

func assertSameSet(t *testing.T, got, want map[string]bool) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("set size mismatch: got %v, want %v", keysOf(got), keysOf(want))
		return
	}
	for k := range want {
		if !got[k] {
			t.Errorf("missing %q from result (got %v)", k, keysOf(got))
		}
	}
	for k := range got {
		if !want[k] {
			t.Errorf("unexpected %q in result (want %v)", k, keysOf(want))
		}
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
