//go:build integration

package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/test/integration/testutil"
)

// TestOfflineBehavior proves the agent's store-and-forward pattern:
//
//  1. Agent starts while the gRPC server is unreachable (connection refused).
//  2. The collection runner scans inventory immediately on boot, queuing results
//     into the SQLite outbox.
//  3. The gRPC server comes online on the reserved port.
//  4. The agent reconnects (exponential backoff), enrolls, and drains the outbox.
//  5. The server now has the endpoint record and endpoint_packages in PostgreSQL.
func TestOfflineBehavior(t *testing.T) {
	ctx := context.Background()

	// ---------------------------------------------------------------
	// Step 1: Start PostgreSQL, create app pool and store, seed data.
	// ---------------------------------------------------------------
	db := testutil.SetupTestDB(t)
	defer db.Cleanup()

	appPool := testutil.AppPool(t, db.SuperPool)
	defer appPool.Close()

	st := store.NewStoreWithBypass(appPool, db.SuperPool)

	tenantID := testutil.SeedTenant(t, ctx, db.SuperPool, "offline-co", "offline-co")
	enrollToken := "offline-enroll-token-001"
	testutil.SeedEnrollmentToken(t, ctx, db.SuperPool, tenantID, enrollToken)

	// ---------------------------------------------------------------
	// Step 2: Reserve a port but do NOT start the gRPC server yet.
	// ---------------------------------------------------------------
	port := testutil.FindFreePort(t)
	t.Logf("reserved port %d for gRPC server (not started yet)", port)

	// ---------------------------------------------------------------
	// Step 3: Start the agent container pointing at the reserved port.
	//         The agent will get connection-refused when it tries to
	//         connect, but the collection runner will still run its
	//         first scan immediately and queue results in the outbox.
	// ---------------------------------------------------------------
	serverAddr := fmt.Sprintf("127.0.0.1:%d", port)
	agent := testutil.StartAgentContainer(t, testutil.AgentContainerConfig{
		ServerAddr:  serverAddr,
		EnrollToken: enrollToken,
	})

	// ---------------------------------------------------------------
	// Step 4: Wait for the outbox to have at least one pending item.
	//         The collection runner fires immediately on boot, so this
	//         should happen within ~30 seconds of agent startup.
	// ---------------------------------------------------------------
	testutil.WaitFor(t, 60*time.Second, 2*time.Second, "outbox has pending items", func() bool {
		count, err := agent.OutboxCount(ctx)
		if err != nil {
			t.Logf("outbox count query (may be transient): %v", err)
			return false
		}
		t.Logf("outbox pending count: %d", count)
		return count > 0
	})

	// ---------------------------------------------------------------
	// Step 5: Start the gRPC server on the reserved port (insecure).
	// ---------------------------------------------------------------
	grpcSrv := testutil.StartInsecureGRPCServerOnPort(t, st, port)
	t.Logf("gRPC server started on %s", grpcSrv.Addr)

	// ---------------------------------------------------------------
	// Step 6: Wait for the agent to reconnect and drain the outbox.
	//         The agent uses exponential backoff starting at 1s with
	//         multiplier 2.0, so reconnection should happen within a
	//         few seconds of the server starting. Give up to 120s to
	//         account for enrollment + outbox sync.
	// ---------------------------------------------------------------
	testutil.WaitFor(t, 120*time.Second, 3*time.Second, "outbox drained to zero", func() bool {
		count, err := agent.OutboxCount(ctx)
		if err != nil {
			t.Logf("outbox count query during drain: %v", err)
			return false
		}
		t.Logf("outbox pending count during drain: %d", count)
		return count == 0
	})

	// ---------------------------------------------------------------
	// Step 7: Verify that the server received the enrollment and
	//         inventory data.
	// ---------------------------------------------------------------

	// 7a. Endpoint record should exist in PostgreSQL.
	endpointID := testutil.AssertEndpointExists(t, db.SuperPool, tenantID)
	t.Logf("endpoint enrolled: %s", endpointID)

	// 7b. Endpoint packages should be present (the container has at
	//     least cowsay, sqlite3, ca-certificates installed via apt).
	testutil.AssertPackageCount(t, db.SuperPool, tenantID, endpointID, 1)
	pkgCount := testutil.TryPackageCount(t, db.SuperPool, tenantID, endpointID)
	t.Logf("endpoint_packages count: %d", pkgCount)

	// ---------------------------------------------------------------
	// Step 8: Check agent logs for backoff pattern (informational).
	// ---------------------------------------------------------------
	logs, err := agent.Logs(ctx)
	if err != nil {
		t.Logf("failed to retrieve agent logs: %v", err)
	} else {
		// Look for evidence of reconnection attempts in the logs.
		if strings.Contains(logs, "reconnecting") || strings.Contains(logs, "backoff") ||
			strings.Contains(logs, "heartbeat stream failed") || strings.Contains(logs, "retrying") {
			t.Log("agent logs confirm backoff/reconnect behavior")
		} else {
			t.Log("no explicit backoff log lines found (agent may have connected quickly)")
		}
		// Log a snippet for debugging if the test is run with -v.
		if testing.Verbose() {
			// Truncate to last 2000 chars to avoid flooding output.
			if len(logs) > 2000 {
				logs = logs[len(logs)-2000:]
			}
			t.Logf("agent logs (tail):\n%s", logs)
		}
	}
}
