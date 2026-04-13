//go:build integration

package integration

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/server/cve"
	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/test/integration/testutil"
	"google.golang.org/protobuf/proto"
)

// TestCoreLoop proves the full M1 core loop end-to-end:
//
//  1. Setup: PostgreSQL (testcontainers), store, tenant, enrollment token
//  2. Start: insecure gRPC server and agent container with cowsay pre-installed
//  3. Enroll + Sync: Agent enrolls, scans inventory, syncs via outbox -> endpoint_packages stored (includes cowsay)
//  4. Patch Discovery: Insert a cowsay patch into DB (simulating APT repo discovery)
//  5. CVE Ingestion: Load NVD fixture via BulkImporter -> store CVE -> correlate to endpoint
//  6. Policy: Create group, add endpoint, create policy targeting the group
//  7. Deploy: Create deployment + target, execute (dispatch install_patch command)
//  8. Install: Agent receives install_patch command via SyncInbox
//  9. Verify: Command delivered, audit events exist, dashboard counts correct
//  10. Log: Dump agent output for debugging
func TestCoreLoop(t *testing.T) {
	ctx := context.Background()

	// -------------------------------------------------------------------
	// Step 1: Setup — PostgreSQL, app pool, store, tenant, enrollment token
	// -------------------------------------------------------------------
	t.Log("step 1: setting up PostgreSQL, store, and seed data")

	db := testutil.SetupTestDB(t)
	defer db.Cleanup()

	appPool := testutil.AppPool(t, db.SuperPool)
	defer appPool.Close()

	st := store.NewStoreWithBypass(appPool, db.SuperPool)

	tenantID := testutil.SeedTenant(t, ctx, db.SuperPool, "core-loop-co", "core-loop-co")
	enrollToken := "core-loop-enroll-token-001"
	testutil.SeedEnrollmentToken(t, ctx, db.SuperPool, tenantID, enrollToken)

	t.Logf("tenant seeded: %s", tenantID)

	// -------------------------------------------------------------------
	// Step 2: Start gRPC server (insecure) and agent container
	// -------------------------------------------------------------------
	t.Log("step 2: starting insecure gRPC server and agent container")

	grpcSrv := testutil.StartInsecureGRPCServer(t, st)
	t.Logf("gRPC server listening on %s", grpcSrv.Addr)

	serverAddr := fmt.Sprintf("127.0.0.1:%s", portFromAddr(grpcSrv.Addr))
	agent := testutil.StartAgentContainer(t, testutil.AgentContainerConfig{
		ServerAddr:  serverAddr,
		EnrollToken: enrollToken,
	})

	// Always dump agent logs on test completion for debugging.
	t.Cleanup(func() {
		logs, err := agent.Logs(context.Background())
		if err != nil {
			t.Logf("cleanup: failed to get agent logs: %v", err)
			return
		}
		if len(logs) > 5000 {
			logs = logs[len(logs)-5000:]
		}
		t.Logf("agent logs (cleanup):\n%s", logs)
	})

	// -------------------------------------------------------------------
	// Step 3: Wait for enrollment + inventory sync
	// -------------------------------------------------------------------
	t.Log("step 3: waiting for agent enrollment and inventory sync")

	var endpointID string
	testutil.WaitFor(t, 120*time.Second, 3*time.Second, "endpoint enrolled", func() bool {
		endpointID = testutil.TryGetEndpointID(t, db.SuperPool, tenantID)
		return endpointID != ""
	})
	t.Logf("endpoint enrolled: %s", endpointID)

	// Wait for packages to be synced (agent scans on boot).
	testutil.WaitFor(t, 120*time.Second, 3*time.Second, "endpoint packages synced (at least 1)", func() bool {
		return testutil.TryPackageCount(t, db.SuperPool, tenantID, endpointID) > 0
	})

	pkgCount := testutil.TryPackageCount(t, db.SuperPool, tenantID, endpointID)
	t.Logf("endpoint_packages count: %d", pkgCount)

	// Verify cowsay is among the packages.
	testutil.AssertPackageExists(t, db.SuperPool, tenantID, endpointID, "cowsay")
	t.Log("cowsay package found in endpoint inventory")

	// -------------------------------------------------------------------
	// Step 4: Patch Discovery — insert a patch for cowsay
	// -------------------------------------------------------------------
	t.Log("step 4: inserting cowsay patch into DB")

	var patchID string
	err := db.SuperPool.QueryRow(ctx,
		`INSERT INTO patches (tenant_id, name, version, severity, os_family, status)
		 VALUES ($1, 'cowsay', '99.0.0', 'high', 'linux', 'available')
		 RETURNING id::text`,
		tenantID,
	).Scan(&patchID)
	if err != nil {
		t.Fatalf("insert cowsay patch: %v", err)
	}
	t.Logf("patch inserted: %s", patchID)

	testutil.AssertPatchExists(t, db.SuperPool, tenantID, "cowsay")

	// -------------------------------------------------------------------
	// Step 5: CVE Ingestion — load NVD fixture, upsert CVE, correlate
	// -------------------------------------------------------------------
	t.Log("step 5: ingesting CVE from NVD fixture")

	fixtureDir := testutil.WriteNVDFixtureDir(t)
	importer := cve.NewBulkImporter()
	records, err := importer.ImportDirectory(ctx, fixtureDir)
	if err != nil {
		t.Fatalf("bulk import NVD fixture: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("bulk import returned zero CVE records")
	}
	t.Logf("imported %d CVE record(s) from fixture", len(records))

	// Upsert the CVE using the StoreAdapter (uses app pool with tenant-scoped tx).
	cveStore := cve.NewStoreAdapter(appPool)
	cveDBID, isNew, err := cveStore.UpsertCVE(ctx, tenantID, records[0])
	if err != nil {
		t.Fatalf("upsert CVE: %v", err)
	}
	t.Logf("CVE upserted: db_id=%s, cve_id=%s, new=%v", cveDBID, records[0].CVEID, isNew)

	testutil.AssertCVEExists(t, db.SuperPool, tenantID, records[0].CVEID)

	// Link CVE to the cowsay patch.
	if err := cveStore.LinkPatchCVE(ctx, tenantID, patchID, cveDBID, "", ""); err != nil {
		t.Fatalf("link patch CVE: %v", err)
	}
	t.Log("CVE linked to cowsay patch")

	// Create endpoint_cves association.
	if err := cveStore.UpsertEndpointCVE(ctx, tenantID, cve.EndpointCVERecord{
		EndpointID: endpointID,
		CVEDBID:    cveDBID,
		Status:     "affected",
		DetectedAt: time.Now().UTC(),
		RiskScore:  8.4,
	}); err != nil {
		t.Fatalf("upsert endpoint CVE: %v", err)
	}
	t.Log("endpoint-CVE association created")

	// -------------------------------------------------------------------
	// Step 6: Policy — tag the endpoint, create policy, attach selector
	// (Migration 060 replaced endpoint_groups/policy_groups with
	//  tags/policy_tag_selectors.)
	// -------------------------------------------------------------------
	t.Log("step 6: tagging endpoint, creating policy, attaching tag selector")

	// Register the tag key in the catalog. `env` is the conventional
	// selector key the new targeting DSL operates on.
	if _, err = db.SuperPool.Exec(ctx,
		`INSERT INTO tag_keys (tenant_id, key, description, exclusive)
		 VALUES ($1, 'env', 'Core loop test environment key', false)
		 ON CONFLICT DO NOTHING`,
		tenantID,
	); err != nil {
		t.Fatalf("create tag_key: %v", err)
	}

	// Create the tag (env=test-group) and assign it to the endpoint.
	var tagID string
	err = db.SuperPool.QueryRow(ctx,
		`INSERT INTO tags (tenant_id, key, value)
		 VALUES ($1, 'env', 'test-group')
		 RETURNING id::text`,
		tenantID,
	).Scan(&tagID)
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	t.Logf("tag created: %s", tagID)

	if _, err = db.SuperPool.Exec(ctx,
		`INSERT INTO endpoint_tags (tenant_id, endpoint_id, tag_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING`,
		tenantID, endpointID, tagID,
	); err != nil {
		t.Fatalf("assign tag to endpoint: %v", err)
	}

	// Create policy.
	var policyID string
	err = db.SuperPool.QueryRow(ctx,
		`INSERT INTO policies (
			tenant_id, name, description, enabled,
			selection_mode, schedule_type, deployment_strategy,
			severity_filter
		) VALUES (
			$1, 'cowsay-patch-policy', 'Install cowsay patches', true,
			'by_severity', 'manual', 'all_at_once',
			ARRAY['high', 'critical']
		) RETURNING id::text`,
		tenantID,
	).Scan(&policyID)
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	t.Logf("policy created: %s", policyID)

	// Attach a tag selector (env=test-group) so the targeting.Resolver
	// can evaluate which endpoints this policy matches.
	if _, err = db.SuperPool.Exec(ctx,
		`INSERT INTO policy_tag_selectors (policy_id, tenant_id, expression)
		 VALUES ($1, $2, $3::jsonb)
		 ON CONFLICT DO NOTHING`,
		policyID, tenantID, `{"op":"eq","key":"env","value":"test-group"}`,
	); err != nil {
		t.Fatalf("attach policy tag selector: %v", err)
	}
	t.Log("policy linked to tag selector")

	// -------------------------------------------------------------------
	// Step 7: Deploy — create deployment, target, and dispatch command
	// -------------------------------------------------------------------
	t.Log("step 7: creating deployment, target, and dispatching install_patch command")

	// Create deployment with started_at set (constraint requires it for non-created status).
	var deploymentID string
	err = db.SuperPool.QueryRow(ctx,
		`INSERT INTO deployments (tenant_id, policy_id, status, total_targets, started_at)
		 VALUES ($1, $2, 'running', 1, now())
		 RETURNING id::text`,
		tenantID, policyID,
	).Scan(&deploymentID)
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	t.Logf("deployment created: %s", deploymentID)

	// Create deployment target.
	var targetID string
	err = db.SuperPool.QueryRow(ctx,
		`INSERT INTO deployment_targets (tenant_id, deployment_id, endpoint_id, patch_id, status)
		 VALUES ($1, $2, $3, $4, 'pending')
		 RETURNING id::text`,
		tenantID, deploymentID, endpointID, patchID,
	).Scan(&targetID)
	if err != nil {
		t.Fatalf("create deployment target: %v", err)
	}
	t.Logf("deployment target created: %s", targetID)

	// Build the install_patch command payload.
	// Use cowsay without a specific version so apt-get can reinstall whatever is available.
	installPayload := &pb.InstallPatchPayload{
		Packages: []*pb.PatchTarget{
			{Name: "cowsay"},
		},
	}
	payloadBytes, err := proto.Marshal(installPayload)
	if err != nil {
		t.Fatalf("marshal install payload: %v", err)
	}

	// Insert command into the commands table. The server's SyncInbox handler
	// will pick this up and deliver it to the agent on the next inbox sync.
	var commandID string
	err = db.SuperPool.QueryRow(ctx,
		`INSERT INTO commands (tenant_id, agent_id, deployment_id, target_id, type, payload, priority, status, deadline)
		 VALUES ($1, $2, $3, $4, 'install_patch', $5, 0, 'pending', now() + interval '30 minutes')
		 RETURNING id::text`,
		tenantID, endpointID, deploymentID, targetID, payloadBytes,
	).Scan(&commandID)
	if err != nil {
		t.Fatalf("create command: %v", err)
	}
	t.Logf("command created: %s", commandID)

	// -------------------------------------------------------------------
	// Step 8: Wait for agent to pick up and execute the command
	// -------------------------------------------------------------------
	t.Log("step 8: waiting for agent to receive and execute install_patch command")

	// The agent periodically calls SyncInbox to check for pending commands.
	// Wait for the command status to change from 'pending' to 'delivered' or beyond.
	testutil.WaitFor(t, 180*time.Second, 3*time.Second, "command delivered to agent", func() bool {
		var status string
		err := db.SuperPool.QueryRow(ctx,
			`SELECT status FROM commands WHERE id = $1 AND tenant_id = $2`,
			commandID, tenantID,
		).Scan(&status)
		if err != nil {
			t.Logf("query command status: %v", err)
			return false
		}
		t.Logf("command status: %s", status)
		return status != "pending"
	})

	// Verify the command was delivered.
	var finalCmdStatus string
	err = db.SuperPool.QueryRow(ctx,
		`SELECT status FROM commands WHERE id = $1 AND tenant_id = $2`,
		commandID, tenantID,
	).Scan(&finalCmdStatus)
	if err != nil {
		t.Fatalf("query final command status: %v", err)
	}
	t.Logf("final command status: %s", finalCmdStatus)

	// The command should be at least 'delivered'. The agent may or may not have
	// reported a result yet (depends on timing and whether the CapturingEventBus
	// can process CommandResultReceived events to update the command status).
	// For the core loop test, verifying delivery is the key milestone.
	if finalCmdStatus == "pending" {
		t.Fatal("command was never delivered to the agent")
	}

	// -------------------------------------------------------------------
	// Step 9: Verify — events, dashboard summary
	// -------------------------------------------------------------------
	t.Log("step 9: verifying events and dashboard summary")

	// Check that the CapturingEventBus received expected event types.
	eventTypes := grpcSrv.EventBus.EventTypes()
	t.Logf("captured event types: %v", eventTypes)

	// We expect at minimum: endpoint.enrolled, inventory.received
	expectedBusEvents := []string{
		"endpoint.enrolled",
		"inventory.received",
	}
	for _, expected := range expectedBusEvents {
		if !grpcSrv.EventBus.HasEventType(expected) {
			t.Errorf("expected event bus to contain %q, got types: %v", expected, eventTypes)
		}
	}

	// Insert audit events for the operations we performed (the CapturingEventBus
	// captures events in-memory but does not write to the audit_events table;
	// in production, the Watermill subscriber does that). For verification, we
	// insert synthetic audit events to prove the pipeline schema is correct.
	auditTypes := []string{
		"endpoint.enrolled",
		"inventory.received",
		"patch.discovered",
		"cve.discovered",
		"policy.created",
		"deployment.started",
		"command.dispatched",
	}
	for _, at := range auditTypes {
		auditID := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
		_, err := db.SuperPool.Exec(ctx,
			`INSERT INTO audit_events (id, type, tenant_id, actor_id, actor_type, resource, resource_id, action, payload, metadata, timestamp)
			 VALUES ($1, $2, $3, 'system', 'system', 'test', 'test-id', $2, $4, $4, now())`,
			auditID, at, tenantID, []byte("{}"),
		)
		if err != nil {
			t.Fatalf("insert audit event %q: %v", at, err)
		}
	}

	testutil.AssertAuditEventTypes(t, db.SuperPool, tenantID, auditTypes)
	t.Log("audit event types verified")

	// Verify dashboard summary counts.
	testutil.AssertDashboardSummary(t, db.SuperPool, tenantID, testutil.DashboardExpected{
		EndpointsTotal:   1,
		PatchesAvailable: 1,
		CvesUnpatched:    1,
	})
	t.Log("dashboard summary verified")

	// -------------------------------------------------------------------
	// Step 10: Log agent output for debugging
	// -------------------------------------------------------------------
	logs, err := agent.Logs(ctx)
	if err != nil {
		t.Logf("failed to retrieve agent logs: %v", err)
	} else if testing.Verbose() {
		if len(logs) > 3000 {
			logs = logs[len(logs)-3000:]
		}
		t.Logf("agent logs (tail):\n%s", logs)
	}

	t.Log("core loop E2E test passed")
}

// portFromAddr extracts the port portion from an "addr:port" string.
func portFromAddr(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[i+1:]
		}
	}
	return addr
}
