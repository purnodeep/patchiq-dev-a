package system

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent"
	"google.golang.org/protobuf/proto"
)

// Module handles system-level commands: reboot and update_config.
type Module struct {
	logger     *slog.Logger
	db         *sql.DB
	rebootFunc func(ctx context.Context, mode pb.RebootMode, gracePeriod int32, msg string) error
}

// New creates a new system module.
func New() *Module {
	return &Module{}
}

func (m *Module) Name() string                                          { return "system" }
func (m *Module) Version() string                                       { return "0.1.0" }
func (m *Module) Capabilities() []string                                { return []string{"system_management"} }
func (m *Module) SupportedCommands() []string                           { return []string{"reboot", "update_config"} }
func (m *Module) CollectInterval() time.Duration                        { return 0 }
func (m *Module) Start(_ context.Context) error                         { return nil }
func (m *Module) Stop(_ context.Context) error                          { return nil }
func (m *Module) HealthCheck(_ context.Context) error                   { return nil }
func (m *Module) Collect(_ context.Context) ([]agent.OutboxItem, error) { return nil, nil }

func (m *Module) Init(_ context.Context, deps agent.ModuleDeps) error {
	m.logger = deps.Logger
	if m.logger == nil {
		m.logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	m.db = deps.LocalDB
	if m.rebootFunc == nil {
		m.rebootFunc = platformReboot
	}
	return nil
}

func (m *Module) HandleCommand(ctx context.Context, cmd agent.Command) (agent.Result, error) {
	switch cmd.Type {
	case "reboot":
		return m.handleReboot(ctx, cmd)
	case "update_config":
		return m.handleUpdateConfig(ctx, cmd)
	default:
		return agent.Result{}, fmt.Errorf("system: unsupported command type %q", cmd.Type)
	}
}

func (m *Module) handleReboot(ctx context.Context, cmd agent.Command) (agent.Result, error) {
	var payload pb.RebootPayload
	if err := proto.Unmarshal(cmd.Payload, &payload); err != nil {
		return agent.Result{ErrorMessage: fmt.Sprintf("system: unmarshal reboot payload: %v", err)}, nil
	}

	m.logger.InfoContext(ctx, "system: reboot requested",
		"command_id", cmd.ID,
		"mode", payload.Mode.String(),
		"grace_period", payload.GracePeriodSeconds,
	)

	// Set post-reboot scan flag if requested.
	if payload.PostRebootScan && m.db != nil {
		if _, err := m.db.ExecContext(ctx,
			`INSERT INTO agent_state (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			"reboot_pending_scan", "true",
		); err != nil {
			m.logger.WarnContext(ctx, "system: failed to set reboot_pending_scan", "error", err)
		}
	}

	if err := m.rebootFunc(ctx, payload.Mode, payload.GracePeriodSeconds, payload.Message); err != nil {
		return agent.Result{ErrorMessage: fmt.Sprintf("system: reboot failed: %v", err)}, nil
	}

	return agent.Result{}, nil
}

func (m *Module) handleUpdateConfig(ctx context.Context, cmd agent.Command) (agent.Result, error) {
	var payload pb.UpdateConfigPayload
	if err := proto.Unmarshal(cmd.Payload, &payload); err != nil {
		return agent.Result{ErrorMessage: fmt.Sprintf("system: unmarshal update_config payload: %v", err)}, nil
	}

	if m.db == nil {
		return agent.Result{ErrorMessage: "system: no database available for config update"}, nil
	}

	for key, value := range payload.Settings {
		if _, err := m.db.ExecContext(ctx,
			`INSERT INTO agent_state (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			key, value,
		); err != nil {
			return agent.Result{ErrorMessage: fmt.Sprintf("system: write setting %q: %v", key, err)}, nil
		}
		m.logger.InfoContext(ctx, "system: config updated", "key", key, "command_id", cmd.ID)
	}

	return agent.Result{}, nil
}

// Verify Module satisfies the interface at compile time.
var _ agent.Module = (*Module)(nil)
