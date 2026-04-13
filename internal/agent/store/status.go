package store

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/skenzeriq/patchiq/internal/agent"
	"github.com/skenzeriq/patchiq/internal/agent/api"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

var _ api.StatusProvider = (*StatusProvider)(nil)

// inventoryHealth is the subset of inventory.Module used by StatusProvider.
// Defined as an interface so the store package does not import internal/agent/inventory.
type inventoryHealth interface {
	LastCollectedAt() time.Time
	LastErrors() []string
	CollectorNames() []string
}

// StatusProvider implements api.StatusProvider using live agent state.
type StatusProvider struct {
	state     *comms.AgentState
	version   string
	serverURL string
	startTime time.Time
	db        *sql.DB

	mu            sync.Mutex
	lastHeartbeat *time.Time

	invHealth inventoryHealth
}

// NewStatusProvider creates a StatusProvider.
func NewStatusProvider(state *comms.AgentState, version, serverURL string, db *sql.DB) *StatusProvider {
	return &StatusProvider{
		state:     state,
		version:   version,
		serverURL: serverURL,
		startTime: time.Now(),
		db:        db,
	}
}

// SetInventoryHealth wires the inventory module so Status() can report collection health.
func (p *StatusProvider) SetInventoryHealth(inv inventoryHealth) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.invHealth = inv
}

// SetLastHeartbeat updates the last heartbeat time.
func (p *StatusProvider) SetLastHeartbeat(t time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastHeartbeat = &t
}

// Status returns the current agent status.
func (p *StatusProvider) Status() api.StatusInfo {
	hostname, err := os.Hostname()
	if err != nil {
		slog.Error("get hostname for status", "error", err)
		hostname = "unknown"
	}

	agentID, err := p.state.Get(context.Background(), "agent_id")
	if err != nil {
		slog.Error("get agent_id from state", "error", err)
	}
	enrollStatus, err := p.state.Get(context.Background(), "enrollment_status")
	if err != nil {
		slog.Error("get enrollment_status from state", "error", err)
	}
	if enrollStatus == "" {
		if agentID != "" {
			enrollStatus = "enrolled"
		} else {
			enrollStatus = "pending"
		}
	}

	p.mu.Lock()
	hb := p.lastHeartbeat
	inv := p.invHealth
	p.mu.Unlock()

	// Status() does not accept a context parameter (interface constraint), so use
	// context.Background() to satisfy the slog convention of including trace_id.
	ctx := context.Background()
	var pendingCount, installedCount, failedCount int64
	if p.db != nil {
		if err := p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pending_patches`).Scan(&pendingCount); err != nil {
			slog.ErrorContext(ctx, "query pending patch count", "error", err)
		}
		if err := p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM patch_history WHERE result = 'success'`).Scan(&installedCount); err != nil {
			slog.ErrorContext(ctx, "query installed patch count", "error", err)
		}
		if err := p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM patch_history WHERE result = 'failed'`).Scan(&failedCount); err != nil {
			slog.ErrorContext(ctx, "query failed patch count", "error", err)
		}
	}

	info := api.StatusInfo{
		AgentID:           agentID,
		Hostname:          hostname,
		OSFamily:          runtime.GOOS,
		OSVersion:         agent.OSVersion(),
		AgentVersion:      p.version,
		EnrollmentStatus:  enrollStatus,
		ServerURL:         p.serverURL,
		LastHeartbeat:     hb,
		UptimeSeconds:     int64(time.Since(p.startTime).Seconds()),
		PendingPatchCount: pendingCount,
		InstalledCount:    installedCount,
		FailedCount:       failedCount,
	}

	if inv != nil {
		info.CollectorCount = len(inv.CollectorNames())
		info.CollectionErrors = inv.LastErrors()
		if lca := inv.LastCollectedAt(); !lca.IsZero() {
			info.LastCollectionAt = &lca
		}
	}

	return info
}
