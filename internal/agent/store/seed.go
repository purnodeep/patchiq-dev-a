package store

import (
	"database/sql"
	"fmt"
)

// Seed populates the agent SQLite database with realistic development data.
// Uses INSERT OR IGNORE so it is safe to call multiple times (idempotent).
// Activated via PATCHIQ_AGENT_SEED=true environment variable.
func Seed(db *sql.DB) error {
	if err := seedPatchData(db); err != nil {
		return fmt.Errorf("seed patches: %w", err)
	}
	if err := seedHistoryData(db); err != nil {
		return fmt.Errorf("seed history: %w", err)
	}
	if err := seedLogData(db); err != nil {
		return fmt.Errorf("seed logs: %w", err)
	}
	return nil
}

func seedPatchData(db *sql.DB) error {
	patches := []struct {
		id, name, version, severity, status, queuedAt string
		cvss                                          float64
		size, cveIDs, source                          string
	}{
		{"p001", "openssl", "3.0.8-1", "critical", "queued", "2026-03-16T08:00:00Z", 9.8, "12 MB", `["CVE-2024-0727","CVE-2023-6237","CVE-2023-5678"]`, "apt"},
		{"p002", "linux-image-6.1.0", "6.1.0-28", "critical", "queued", "2026-03-16T07:30:00Z", 9.1, "89 MB", `["CVE-2024-1086","CVE-2024-0646"]`, "apt"},
		{"p003", "curl", "7.88.1-10", "critical", "installing", "2026-03-16T07:00:00Z", 8.8, "3 MB", `["CVE-2023-38545"]`, "apt"},
		{"p004", "libssl3", "3.0.8-1", "high", "queued", "2026-03-15T20:00:00Z", 7.5, "4 MB", `["CVE-2024-0727"]`, "apt"},
		{"p005", "nginx", "1.22.1-9", "high", "queued", "2026-03-15T18:00:00Z", 7.3, "1 MB", `["CVE-2023-44487"]`, "apt"},
		{"p006", "python3", "3.11.2-6", "high", "downloading", "2026-03-15T16:00:00Z", 7.0, "28 MB", `["CVE-2023-40217"]`, "apt"},
		{"p007", "git", "1:2.39.2-1", "high", "queued", "2026-03-15T14:00:00Z", 6.8, "11 MB", `["CVE-2024-32004"]`, "apt"},
		{"p008", "sudo", "1.9.13p3-1", "high", "queued", "2026-03-15T12:00:00Z", 6.7, "2 MB", `["CVE-2023-42465"]`, "apt"},
		{"p009", "libc6", "2.36-9", "medium", "queued", "2026-03-15T10:00:00Z", 5.5, "13 MB", `[]`, "apt"},
		{"p010", "bash", "5.2.15-2", "medium", "queued", "2026-03-15T09:00:00Z", 5.3, "2 MB", `["CVE-2022-3715"]`, "apt"},
		{"p011", "ssh", "1:9.2p1-2", "medium", "queued", "2026-03-15T08:30:00Z", 5.1, "1 MB", `[]`, "apt"},
		{"p012", "vim", "2:9.0.1378-2", "medium", "queued", "2026-03-14T20:00:00Z", 4.8, "3 MB", `[]`, "apt"},
		{"p013", "wget", "1.21.3-1", "medium", "queued", "2026-03-14T18:00:00Z", 4.5, "1 MB", `[]`, "apt"},
	}
	for _, p := range patches {
		_, err := db.Exec(`INSERT OR IGNORE INTO pending_patches
			(id, name, version, severity, status, queued_at, cvss_score, size, cve_ids, source)
			VALUES (?,?,?,?,?,?,?,?,?,?)`,
			p.id, p.name, p.version, p.severity, p.status, p.queuedAt,
			p.cvss, p.size, p.cveIDs, p.source)
		if err != nil {
			return fmt.Errorf("insert patch %s: %w", p.id, err)
		}
	}
	return nil
}

func seedHistoryData(db *sql.DB) error {
	type histRow struct {
		id, patchName, version, action, result, completedAt string
		errMsg, stderr                                      string
		dur, attempt                                        int
		reboot                                              bool
	}
	rows := []histRow{
		{"h001", "openssl", "3.0.7-1", "install", "success", "2026-03-16T06:00:00Z", "", "", 87, 1, false},
		{"h002", "curl", "7.88.1-9", "install", "failed", "2026-03-15T22:00:00Z", "exit code 1", "E: Unable to locate package curl", 12, 1, false},
		{"h003", "linux-image-5.15", "5.15.0-91", "install", "success", "2026-03-15T18:00:00Z", "", "", 234, 1, true},
		{"h004", "sudo", "1.9.13p2", "rollback", "success", "2026-03-15T14:00:00Z", "", "", 45, 1, false},
		{"h005", "nginx", "1.22.1-8", "install", "success", "2026-03-14T10:00:00Z", "", "", 23, 1, false},
		{"h006", "python3", "3.11.2-5", "install", "failed", "2026-03-13T15:00:00Z", "dependency conflict", "dpkg: error processing package python3", 8, 2, false},
		{"h007", "git", "1:2.39.1-1", "install", "success", "2026-03-12T09:00:00Z", "", "", 34, 1, false},
		{"h008", "bash", "5.2.15-1", "install", "success", "2026-03-11T11:00:00Z", "", "", 19, 1, false},
		{"h009", "vim", "2:9.0.1300-1", "install", "success", "2026-03-10T16:00:00Z", "", "", 22, 1, false},
		{"h010", "libc6", "2.35-13", "install", "success", "2026-03-09T14:00:00Z", "", "", 156, 1, true},
	}
	for _, r := range rows {
		rebootInt := 0
		if r.reboot {
			rebootInt = 1
		}
		var errMsg, stderr *string
		if r.errMsg != "" {
			errMsg = &r.errMsg
		}
		if r.stderr != "" {
			stderr = &r.stderr
		}
		_, err := db.Exec(`INSERT OR IGNORE INTO patch_history
			(id, patch_name, patch_version, action, result, error_message, completed_at,
			 duration_seconds, reboot_required, stderr, attempt)
			VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			r.id, r.patchName, r.version, r.action, r.result, errMsg, r.completedAt,
			r.dur, rebootInt, stderr, r.attempt)
		if err != nil {
			return fmt.Errorf("insert history %s: %w", r.id, err)
		}
	}
	return nil
}

func seedLogData(db *sql.DB) error {
	logs := []struct{ id, level, message, source, ts string }{
		{"l001", "info", "Agent started successfully", "main", "2026-03-16T08:00:00Z"},
		{"l002", "info", "Enrolled with server at grpc.patchiq.internal:50051", "comms", "2026-03-16T08:00:01Z"},
		{"l003", "info", "Heartbeat sent", "comms", "2026-03-16T08:00:30Z"},
		{"l004", "info", "Inventory scan started", "inventory", "2026-03-16T08:01:00Z"},
		{"l005", "info", "Found 247 installed packages", "inventory", "2026-03-16T08:01:12Z"},
		{"l006", "info", "13 pending patches discovered", "inventory", "2026-03-16T08:01:13Z"},
		{"l007", "info", "Installing openssl 3.0.7-1", "patcher", "2026-03-16T06:00:00Z"},
		{"l008", "info", "openssl installed successfully in 87s", "patcher", "2026-03-16T06:01:27Z"},
		{"l009", "warn", "Package curl installation attempt 1 failed, retrying", "patcher", "2026-03-15T22:00:00Z"},
		{"l010", "error", "curl installation failed after 1 attempt: exit code 1", "patcher", "2026-03-15T22:00:12Z"},
		{"l011", "info", "Heartbeat sent", "comms", "2026-03-16T08:01:00Z"},
		{"l012", "debug", "SQLite WAL checkpoint completed", "store", "2026-03-16T08:02:00Z"},
		{"l013", "info", "Outbox sync: 3 messages sent", "comms", "2026-03-16T08:02:30Z"},
		{"l014", "warn", "Server connection latency high: 450ms", "comms", "2026-03-16T08:03:00Z"},
		{"l015", "info", "Command received: SCAN_NOW", "comms", "2026-03-16T08:03:15Z"},
		{"l016", "info", "Scan triggered by server command", "inventory", "2026-03-16T08:03:16Z"},
		{"l017", "error", "gRPC stream disconnected, reconnecting in 5s", "comms", "2026-03-16T07:55:00Z"},
		{"l018", "info", "gRPC stream reconnected", "comms", "2026-03-16T07:55:05Z"},
		{"l019", "debug", "Config reloaded from /etc/patchiq/agent.yaml", "config", "2026-03-16T08:00:00Z"},
		{"l020", "info", "Next scan scheduled at 2026-03-17T02:00:00Z", "inventory", "2026-03-16T08:01:14Z"},
	}
	for _, l := range logs {
		_, err := db.Exec(`INSERT OR IGNORE INTO agent_logs (id, level, message, source, timestamp)
			VALUES (?,?,?,?,?)`, l.id, l.level, l.message, l.source, l.ts)
		if err != nil {
			return fmt.Errorf("insert log %s: %w", l.id, err)
		}
	}
	return nil
}
