//go:build integration

package grpc_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	servergrpc "github.com/skenzeriq/patchiq/internal/server/grpc"
	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

const (
	integTestDBName  = "patchiq_integ_test"
	integTestDBUser  = "postgres"
	integTestDBPass  = "postgres"
	integAppRolePass = "test_app_pass"
)

// setupIntegrationDB starts a PostgreSQL 16 container, runs migrations, and
// returns a superuser pool plus cleanup function. This mirrors the store
// package's setupTestDB but lives in the grpc_test package for integration tests.
func setupIntegrationDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(integTestDBName),
		postgres.WithUsername(integTestDBUser),
		postgres.WithPassword(integTestDBPass),
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
		t.Fatalf("get connection string: %v", err)
	}

	superPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("create super pool: %v", err)
	}

	// Find migrations directory relative to this test file.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed: cannot determine test file path")
	}
	migrationsDir := filepath.Join(filepath.Dir(filename), "..", "store", "migrations")

	if err := runIntegMigrations(ctx, superPool, migrationsDir); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// Set password for patchiq_app role so we can connect as it.
	if _, err := superPool.Exec(ctx,
		fmt.Sprintf("ALTER ROLE patchiq_app WITH PASSWORD '%s'", integAppRolePass),
	); err != nil {
		t.Fatalf("set patchiq_app password: %v", err)
	}

	cleanup := func() {
		superPool.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("terminate container: %v", err)
		}
	}
	return superPool, cleanup
}

// runIntegMigrations executes all *.sql migration files in ascending order,
// extracting the "-- +goose Up" section from each file.
func runIntegMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
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

	if len(files) == 0 {
		return fmt.Errorf("no .sql migration files found in %s", dir)
	}

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read migration file %s: %w", f, err)
		}

		upSQL, err := extractIntegUpSection(string(content))
		if err != nil {
			return fmt.Errorf("extract Up section from %s: %w", filepath.Base(f), err)
		}
		if upSQL == "" {
			continue
		}

		if _, err := pool.Exec(ctx, upSQL); err != nil {
			return fmt.Errorf("execute migration %s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

// extractIntegUpSection returns SQL between "-- +goose Up" and "-- +goose Down".
func extractIntegUpSection(content string) (string, error) {
	const upMarker = "-- +goose Up"
	const downMarker = "-- +goose Down"

	upIdx := strings.Index(content, upMarker)
	if upIdx == -1 {
		return "", fmt.Errorf("missing %q marker in migration file", upMarker)
	}
	upIdx += len(upMarker)

	downIdx := strings.Index(content, downMarker)
	if downIdx == -1 {
		return strings.TrimSpace(content[upIdx:]), nil
	}
	return strings.TrimSpace(content[upIdx:downIdx]), nil
}

// integAppPool returns a pgxpool connected as the patchiq_app role.
func integAppPool(t *testing.T, superPool *pgxpool.Pool) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	cfg := superPool.Config().ConnConfig
	appConnStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=patchiq_app password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database, integAppRolePass,
	)
	pool, err := pgxpool.New(ctx, appConnStr)
	if err != nil {
		t.Fatalf("create app pool: %v", err)
	}
	return pool
}

// seedIntegTenantAndEndpoint creates a tenant and endpoint via superuser,
// returning the tenant ID string and endpoint UUID.
func seedIntegTenantAndEndpoint(t *testing.T, ctx context.Context, superPool *pgxpool.Pool, label string) (string, uuid.UUID) {
	t.Helper()

	var tenantIDStr string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO tenants (name, slug) VALUES ($1, $2) RETURNING id::text",
		"Integ Tenant "+label, "integ-tenant-"+label,
	).Scan(&tenantIDStr); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	var endpointID pgtype.UUID
	if err := superPool.QueryRow(ctx,
		"INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status) VALUES ($1, $2, 'linux', '22.04', 'online') RETURNING id",
		tenantIDStr, "integ-host-"+label,
	).Scan(&endpointID); err != nil {
		t.Fatalf("seed endpoint: %v", err)
	}

	return tenantIDStr, uuid.UUID(endpointID.Bytes)
}

// sendInventoryViaStream builds an OutboxMessage from the given InventoryReport,
// sends it over a SyncOutbox gRPC stream with the agent ID in metadata, and
// returns the received OutboxAck.
func sendInventoryViaStream(
	t *testing.T,
	client pb.AgentServiceClient,
	agentID uuid.UUID,
	report *pb.InventoryReport,
) *pb.OutboxAck {
	t.Helper()

	payload, err := proto.Marshal(report)
	if err != nil {
		t.Fatalf("marshal inventory report: %v", err)
	}

	md := metadata.Pairs("x-agent-id", agentID.String())
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	stream, err := client.SyncOutbox(ctx)
	if err != nil {
		t.Fatalf("open SyncOutbox stream: %v", err)
	}

	msg := &pb.OutboxMessage{
		MessageId: "integ-msg-" + uuid.New().String()[:8],
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_INVENTORY,
		Payload:   payload,
	}

	if err := stream.Send(msg); err != nil {
		t.Fatalf("send outbox message: %v", err)
	}

	// Close the send side so the server can process and respond.
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("close send: %v", err)
	}

	ack, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv ack: %v", err)
	}

	return ack
}

func TestIntegration_InventoryPipeline_3Packages(t *testing.T) {
	superPool, cleanup := setupIntegrationDB(t)
	defer cleanup()
	ctx := context.Background()

	app := integAppPool(t, superPool)
	defer app.Close()

	st := store.NewStore(app)
	svc := servergrpc.NewAgentServiceServer(st, noopEventBus{}, slog.Default())

	client, grpcCleanup := setupBufconn(t, svc)
	defer grpcCleanup()

	tenantIDStr, endpointUUID := seedIntegTenantAndEndpoint(t, ctx, superPool, "3pkg")

	// Build an InventoryReport with 3 packages (nginx has RPM release field).
	report := &pb.InventoryReport{
		AgentId:         endpointUUID.String(),
		ProtocolVersion: 1,
		CollectedAt:     timestamppb.Now(),
		InstalledPackages: []*pb.PackageInfo{
			{Name: "curl", Version: "7.88.1", Architecture: "amd64", Source: "apt"},
			{Name: "vim", Version: "9.0.1378", Architecture: "amd64", Source: "apt"},
			{Name: "nginx", Version: "1.24.0", Architecture: "x86_64", Source: "yum", Release: "1.el9"},
		},
	}

	ack := sendInventoryViaStream(t, client, endpointUUID, report)

	// Verify ACK is accepted (RejectionCode == UNSPECIFIED means accepted).
	if ack.GetRejectionCode() != pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_UNSPECIFIED {
		t.Fatalf("expected accepted ack, got rejection: %v — %s", ack.GetRejectionCode(), ack.GetRejectionDetail())
	}

	// Verify persistence: query the DB as the tenant.
	tenantUUID := pgtype.UUID{}
	if err := tenantUUID.Scan(tenantIDStr); err != nil {
		t.Fatalf("parse tenant UUID: %v", err)
	}
	endpointPgUUID := pgtype.UUID{Bytes: endpointUUID, Valid: true}

	tenantCtx := tenant.WithTenantID(ctx, tenantIDStr)
	tx, err := st.BeginTx(tenantCtx)
	if err != nil {
		t.Fatalf("BeginTx for verification: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := sqlcgen.New(tx)

	// Verify endpoint_inventories has 1 record with package_count=3.
	inv, err := qtx.GetLatestEndpointInventory(ctx, sqlcgen.GetLatestEndpointInventoryParams{
		EndpointID: endpointPgUUID,
		TenantID:   tenantUUID,
	})
	if err != nil {
		t.Fatalf("GetLatestEndpointInventory: %v", err)
	}
	if inv.PackageCount != 3 {
		t.Errorf("inventory package_count = %d, want 3", inv.PackageCount)
	}

	// Verify endpoint_packages has 3 rows.
	pkgs, err := qtx.ListEndpointPackages(ctx, sqlcgen.ListEndpointPackagesParams{
		InventoryID: inv.ID,
		TenantID:    tenantUUID,
	})
	if err != nil {
		t.Fatalf("ListEndpointPackages: %v", err)
	}
	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}

	// Verify package names (sorted by package_name in sqlc query).
	expectedNames := []string{"curl", "nginx", "vim"}
	for i, pkg := range pkgs {
		if pkg.PackageName != expectedNames[i] {
			t.Errorf("pkgs[%d].PackageName = %q, want %q", i, pkg.PackageName, expectedNames[i])
		}
	}

	// Verify nginx has RPM release field set.
	for _, pkg := range pkgs {
		if pkg.PackageName == "nginx" {
			if !pkg.Release.Valid || pkg.Release.String != "1.el9" {
				t.Errorf("nginx release = %v, want '1.el9'", pkg.Release)
			}
		}
	}

	// Verify curl and vim versions.
	for _, pkg := range pkgs {
		switch pkg.PackageName {
		case "curl":
			if pkg.Version != "7.88.1" {
				t.Errorf("curl version = %q, want %q", pkg.Version, "7.88.1")
			}
		case "vim":
			if pkg.Version != "9.0.1378" {
				t.Errorf("vim version = %q, want %q", pkg.Version, "9.0.1378")
			}
		}
	}
}

func TestIntegration_InventoryPipeline_500Packages(t *testing.T) {
	superPool, cleanup := setupIntegrationDB(t)
	defer cleanup()
	ctx := context.Background()

	app := integAppPool(t, superPool)
	defer app.Close()

	st := store.NewStore(app)
	svc := servergrpc.NewAgentServiceServer(st, noopEventBus{}, slog.Default())

	client, grpcCleanup := setupBufconn(t, svc)
	defer grpcCleanup()

	tenantIDStr, endpointUUID := seedIntegTenantAndEndpoint(t, ctx, superPool, "500pkg")

	// Build an InventoryReport with 500 generated packages.
	packages := make([]*pb.PackageInfo, 500)
	for i := range packages {
		packages[i] = &pb.PackageInfo{
			Name:         fmt.Sprintf("pkg-%04d", i),
			Version:      fmt.Sprintf("%d.0.0", i),
			Architecture: "amd64",
			Source:       "apt",
		}
	}

	report := &pb.InventoryReport{
		AgentId:           endpointUUID.String(),
		ProtocolVersion:   1,
		CollectedAt:       timestamppb.Now(),
		InstalledPackages: packages,
	}

	ack := sendInventoryViaStream(t, client, endpointUUID, report)

	if ack.GetRejectionCode() != pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_UNSPECIFIED {
		t.Fatalf("expected accepted ack, got rejection: %v — %s", ack.GetRejectionCode(), ack.GetRejectionDetail())
	}

	// Verify persistence.
	tenantUUID := pgtype.UUID{}
	if err := tenantUUID.Scan(tenantIDStr); err != nil {
		t.Fatalf("parse tenant UUID: %v", err)
	}
	endpointPgUUID := pgtype.UUID{Bytes: endpointUUID, Valid: true}

	tenantCtx := tenant.WithTenantID(ctx, tenantIDStr)
	tx, err := st.BeginTx(tenantCtx)
	if err != nil {
		t.Fatalf("BeginTx for verification: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := sqlcgen.New(tx)

	inv, err := qtx.GetLatestEndpointInventory(ctx, sqlcgen.GetLatestEndpointInventoryParams{
		EndpointID: endpointPgUUID,
		TenantID:   tenantUUID,
	})
	if err != nil {
		t.Fatalf("GetLatestEndpointInventory: %v", err)
	}
	if inv.PackageCount != 500 {
		t.Errorf("inventory package_count = %d, want 500", inv.PackageCount)
	}

	pkgs, err := qtx.ListEndpointPackages(ctx, sqlcgen.ListEndpointPackagesParams{
		InventoryID: inv.ID,
		TenantID:    tenantUUID,
	})
	if err != nil {
		t.Fatalf("ListEndpointPackages: %v", err)
	}
	if len(pkgs) != 500 {
		t.Fatalf("expected 500 packages, got %d", len(pkgs))
	}
}

func TestIntegration_InventoryPipeline_InvalidPayload(t *testing.T) {
	superPool, cleanup := setupIntegrationDB(t)
	defer cleanup()
	ctx := context.Background()

	app := integAppPool(t, superPool)
	defer app.Close()

	st := store.NewStore(app)
	svc := servergrpc.NewAgentServiceServer(st, noopEventBus{}, slog.Default())

	client, grpcCleanup := setupBufconn(t, svc)
	defer grpcCleanup()

	tenantIDStr, endpointUUID := seedIntegTenantAndEndpoint(t, ctx, superPool, "invalid")

	// Send garbage bytes as inventory payload.
	md := metadata.Pairs("x-agent-id", endpointUUID.String())
	streamCtx := metadata.NewOutgoingContext(context.Background(), md)

	stream, err := client.SyncOutbox(streamCtx)
	if err != nil {
		t.Fatalf("open SyncOutbox stream: %v", err)
	}

	msg := &pb.OutboxMessage{
		MessageId: "integ-msg-invalid",
		Type:      pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_INVENTORY,
		Payload:   []byte("this is not valid protobuf data"),
	}

	if err := stream.Send(msg); err != nil {
		t.Fatalf("send invalid message: %v", err)
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("close send: %v", err)
	}

	ack, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv ack: %v", err)
	}

	// Verify PAYLOAD_INVALID rejection.
	if ack.GetRejectionCode() != pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_PAYLOAD_INVALID {
		t.Errorf("expected PAYLOAD_INVALID rejection, got: %v", ack.GetRejectionCode())
	}
	if ack.GetMessageId() != "integ-msg-invalid" {
		t.Errorf("ack message_id = %q, want %q", ack.GetMessageId(), "integ-msg-invalid")
	}

	// Verify NO rows were inserted into endpoint_inventories.
	tenantUUID := pgtype.UUID{}
	if err := tenantUUID.Scan(tenantIDStr); err != nil {
		t.Fatalf("parse tenant UUID: %v", err)
	}
	endpointPgUUID := pgtype.UUID{Bytes: endpointUUID, Valid: true}

	tenantCtx := tenant.WithTenantID(ctx, tenantIDStr)
	tx, err := st.BeginTx(tenantCtx)
	if err != nil {
		t.Fatalf("BeginTx for verification: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := sqlcgen.New(tx)
	invs, err := qtx.ListEndpointInventories(ctx, sqlcgen.ListEndpointInventoriesParams{
		EndpointID: endpointPgUUID,
		TenantID:   tenantUUID,
	})
	if err != nil {
		t.Fatalf("ListEndpointInventories: %v", err)
	}
	if len(invs) != 0 {
		t.Errorf("expected 0 inventory records after invalid payload, got %d", len(invs))
	}
}
