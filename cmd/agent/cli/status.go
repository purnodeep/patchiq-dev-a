package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

type statusOpts struct {
	watch      bool
	jsonOutput bool
	configPath string
	dataDir    string
}

// StatusInfo holds agent status information for display or JSON output.
type StatusInfo struct {
	AgentID       string `json:"agent_id"`
	Connection    string `json:"connection"`
	LastHeartbeat string `json:"last_heartbeat"`
	LastScan      string `json:"last_scan"`
	QueueDepth    int    `json:"queue_depth"`
}

func parseStatusFlags(args []string) (statusOpts, error) {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	var opts statusOpts
	fs.BoolVar(&opts.watch, "watch", false, "Continuously watch agent status")
	fs.BoolVar(&opts.jsonOutput, "json", false, "Output as JSON")
	fs.StringVar(&opts.configPath, "config", "", "Path to agent config file")
	fs.StringVar(&opts.dataDir, "data-dir", "", "Path to agent data directory")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: patchiq-agent status [flags]

Display agent health, connection state, and operational metrics.

Examples:
  patchiq-agent status
  patchiq-agent status --json
  patchiq-agent status --watch

Flags:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return statusOpts{}, fmt.Errorf("parse status flags: %w", err)
	}
	return opts, nil
}

func collectStatusInfo(ctx context.Context, state *comms.AgentState, outbox *comms.Outbox) (StatusInfo, error) {
	agentID, err := state.Get(ctx, "agent_id")
	if err != nil {
		return StatusInfo{}, fmt.Errorf("collect status agent_id: %w", err)
	}

	lastHeartbeat, err := state.Get(ctx, "last_heartbeat")
	if err != nil {
		return StatusInfo{}, fmt.Errorf("collect status last_heartbeat: %w", err)
	}

	lastScan, err := state.Get(ctx, "last_scan")
	if err != nil {
		return StatusInfo{}, fmt.Errorf("collect status last_scan: %w", err)
	}

	pending, err := outbox.Pending(ctx, 10000)
	if err != nil {
		return StatusInfo{}, fmt.Errorf("collect status pending: %w", err)
	}

	info := StatusInfo{
		AgentID:       agentID,
		LastHeartbeat: lastHeartbeat,
		LastScan:      lastScan,
		QueueDepth:    len(pending),
	}

	if info.AgentID == "" {
		info.AgentID = "(not enrolled)"
	}
	if info.LastHeartbeat == "" {
		info.LastHeartbeat = "(never)"
	}
	if info.LastScan == "" {
		info.LastScan = "(never)"
	}

	if agentID != "" && lastHeartbeat != "" {
		info.Connection = "connected"
	} else {
		info.Connection = "disconnected"
	}

	return info, nil
}

var (
	labelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	greenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	redStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func printStatus(info StatusInfo) {
	connStyle := redStyle
	if info.Connection == "connected" {
		connStyle = greenStyle
	}

	fmt.Fprintf(os.Stdout, "%s %s\n", labelStyle.Render("Agent ID:"), valueStyle.Render(info.AgentID))
	fmt.Fprintf(os.Stdout, "%s %s\n", labelStyle.Render("Connection:"), connStyle.Render(info.Connection))
	fmt.Fprintf(os.Stdout, "%s %s\n", labelStyle.Render("Last Heartbeat:"), valueStyle.Render(info.LastHeartbeat))
	fmt.Fprintf(os.Stdout, "%s %s\n", labelStyle.Render("Last Scan:"), valueStyle.Render(info.LastScan))
	fmt.Fprintf(os.Stdout, "%s %s\n", labelStyle.Render("Queue Depth:"), valueStyle.Render(fmt.Sprintf("%d", info.QueueDepth)))
}

func runStatusWatch(state *comms.AgentState, outbox *comms.Outbox) int {
	model := newStatusModel(state, outbox)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		slog.Error("status watch TUI error", "error", err)
		return ExitError
	}
	return ExitOK
}

// RunStatus executes the status command.
func RunStatus(args []string) int {
	opts, err := parseStatusFlags(args)
	if err != nil {
		slog.Error("status: invalid flags", "error", err)
		return ExitError
	}

	cfg, err := LoadAgentConfig(opts.configPath)
	if err != nil {
		slog.Error("status: load config", "error", err)
		return ExitError
	}

	dataDir := opts.dataDir
	if dataDir == "" {
		dataDir = cfg.DataDir
	}

	dbPath := filepath.Join(dataDir, "agent.db")
	db, err := comms.OpenDB(dbPath)
	if err != nil {
		slog.Error("status: open database", "path", dbPath, "error", err)
		return ExitConnectionError
	}
	defer db.Close()

	state := comms.NewAgentState(db)
	outbox := comms.NewOutbox(db)

	if opts.watch {
		return runStatusWatch(state, outbox)
	}

	ctx := context.Background()
	info, err := collectStatusInfo(ctx, state, outbox)
	if err != nil {
		slog.Error("status: collect info", "error", err)
		return ExitError
	}

	if opts.jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(info); err != nil {
			slog.Error("status: encode JSON", "error", err)
			return ExitError
		}
		return ExitOK
	}

	printStatus(info)
	return ExitOK
}
