package cli

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
	"github.com/skenzeriq/patchiq/internal/agent/inventory"
	"google.golang.org/protobuf/proto"
)

var dryRunStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))

type scanOpts struct {
	dryRun     bool
	configPath string
	dataDir    string
}

func parseScanFlags(args []string) (scanOpts, error) {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	var opts scanOpts
	fs.BoolVar(&opts.dryRun, "dry-run", false, "scan without queuing results to outbox")
	fs.StringVar(&opts.configPath, "config", "", "path to agent config file")
	fs.StringVar(&opts.dataDir, "data-dir", "", "path to agent data directory")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: patchiq-agent scan [flags]

Trigger an immediate inventory scan, bypassing the schedule.

Examples:
  patchiq-agent scan
  patchiq-agent scan --dry-run

Flags:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return scanOpts{}, fmt.Errorf("parse scan flags: %w", err)
	}
	return opts, nil
}

type scanPhase int

const (
	scanPhaseScanning scanPhase = iota
	scanPhaseResults
	scanPhaseError
)

// scanResultMsg carries the result of the inventory scan.
type scanResultMsg struct {
	packages []*pb.PackageInfo
	payload  []byte // raw outbox payload for queuing
	err      error
}

type scanModel struct {
	phase    scanPhase
	opts     scanOpts
	spinner  spinner.Model
	table    table.Model
	packages []*pb.PackageInfo
	payload  []byte
	err      error
}

func newScanModel(opts scanOpts) scanModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return scanModel{
		phase:   scanPhaseScanning,
		opts:    opts,
		spinner: sp,
	}
}

func (m scanModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.doScan())
}

func (m scanModel) doScan() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		mod := inventory.New()
		deps := agent.ModuleDeps{
			Logger:         logger,
			ConfigProvider: agent.NoopConfigProvider{},
			EventEmitter:   agent.NoopEventEmitter{},
			FileCache:      agent.NoopFileCache{},
		}
		if err := mod.Init(ctx, deps); err != nil {
			return scanResultMsg{err: fmt.Errorf("scan: init inventory module: %w", err)}
		}

		items, err := mod.Collect(ctx)
		if err != nil {
			return scanResultMsg{err: fmt.Errorf("scan: collect inventory: %w", err)}
		}

		if len(items) == 0 {
			return scanResultMsg{}
		}

		var report pb.InventoryReport
		if err := proto.Unmarshal(items[0].Payload, &report); err != nil {
			return scanResultMsg{err: fmt.Errorf("scan: unmarshal inventory report: %w", err)}
		}

		return scanResultMsg{
			packages: report.GetInstalledPackages(),
			payload:  items[0].Payload,
		}
	}
}

func (m scanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}
		// Forward to table for navigation in results phase.
		if m.phase == scanPhaseResults {
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		return m, nil

	case scanResultMsg:
		if msg.err != nil {
			m.phase = scanPhaseError
			m.err = msg.err
			return m, tea.Quit
		}
		m.packages = msg.packages
		m.payload = msg.payload
		m.phase = scanPhaseResults

		// Build table.
		columns := []table.Column{
			{Title: "Name", Width: 30},
			{Title: "Version", Width: 20},
			{Title: "Source", Width: 15},
		}
		var rows []table.Row
		for _, p := range m.packages {
			rows = append(rows, table.Row{p.GetName(), p.GetVersion(), p.GetSource()})
		}
		height := len(rows)
		if height > 20 {
			height = 20
		}
		if height < 1 {
			height = 1
		}
		m.table = table.New(
			table.WithColumns(columns),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(height),
		)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m scanModel) View() tea.View {
	title := titleStyle.Render("PatchIQ Scan")

	var dryRunBanner string
	if m.opts.dryRun {
		dryRunBanner = dryRunStyle.Render("[DRY RUN]") + "\n"
	}

	var s string
	switch m.phase {
	case scanPhaseScanning:
		s = fmt.Sprintf("%s\n%s\n%s Scanning installed packages...\n",
			title, dryRunBanner, m.spinner.View())

	case scanPhaseResults:
		count := len(m.packages)
		s = fmt.Sprintf("%s\n%s\n%s\n\n%s\n\n%s\n",
			title,
			dryRunBanner,
			successStyle.Render(fmt.Sprintf("Found %d packages", count)),
			m.table.View(),
			dimStyle.Render("↑/↓ navigate • q quit"),
		)

	case scanPhaseError:
		s = fmt.Sprintf("%s\n\n%s\n  %v\n",
			title,
			errorStyle.Render("Error:"),
			m.err,
		)
	}

	return tea.NewView(s)
}

// RunScan implements the "scan" subcommand.
func RunScan(args []string) int {
	opts, err := parseScanFlags(args)
	if err != nil {
		slog.Error("scan: failed to parse flags", "error", err)
		return ExitError
	}

	model := newScanModel(opts)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		slog.Error("scan: TUI failed", "error", err)
		return ExitError
	}

	final, ok := finalModel.(scanModel)
	if !ok {
		slog.Error("scan: unexpected TUI model type", "type", fmt.Sprintf("%T", finalModel))
		return ExitError
	}

	if final.err != nil {
		return ExitError
	}

	// If not dry-run and we have results, queue to outbox.
	if !opts.dryRun && len(final.payload) > 0 {
		dataDir := opts.dataDir
		if dataDir == "" {
			dataDir = DefaultDataDir()
		}
		dbPath := filepath.Join(dataDir, "agent.db")
		db, err := comms.OpenDB(dbPath)
		if err != nil {
			slog.Error("scan: failed to open database", "path", dbPath, "error", err)
			return ExitError
		}
		defer db.Close()

		outbox := comms.NewOutbox(db)
		if _, err := outbox.Add(context.Background(), "inventory", final.payload); err != nil {
			slog.Error("scan: failed to queue results", "error", err)
			return ExitError
		}
		slog.Info("scan results queued to outbox")
	}

	return ExitOK
}
