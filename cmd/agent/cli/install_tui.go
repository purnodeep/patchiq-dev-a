package cli

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type installStep int

const (
	stepServerInput installStep = iota
	stepTokenInput
	stepConnecting
	stepInstallingService
	stepDone
	stepError
)

// enrollResultMsg carries the result of the enrollment process.
type enrollResultMsg struct {
	agentID string
	err     error
}

// serviceResultMsg carries the result of installing/starting the Windows service.
type serviceResultMsg struct {
	err error
}

type installModel struct {
	step        installStep
	serverInput textinput.Model
	tokenInput  textinput.Model
	spinner     spinner.Model
	opts        installOpts
	err         error
	agentID     string
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	errorStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	successStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	dimStyle     = lipgloss.NewStyle().Faint(true)
)

func newInstallModel(opts installOpts) installModel {
	si := textinput.New()
	si.Placeholder = "10.0.5.13:50451"
	si.CharLimit = 256
	si.SetWidth(40)

	ti := textinput.New()
	ti.Placeholder = "paste enrollment token"
	ti.CharLimit = 512
	ti.SetWidth(40)

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	// If a default server address is baked in (release builds), skip the
	// server-input step entirely and jump straight to the token field.
	startStep := stepServerInput
	if DefaultServerAddress != "" && opts.server == "" {
		opts.server = DefaultServerAddress
		startStep = stepTokenInput
		ti.Focus()
	} else {
		si.Focus()
	}

	return installModel{
		step:        startStep,
		serverInput: si,
		tokenInput:  ti,
		spinner:     sp,
		opts:        opts,
	}
}

func (m installModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m installModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.err = fmt.Errorf("install cancelled by user")
			return m, tea.Quit
		case "enter":
			return m.handleEnter()
		}

	case tea.WindowSizeMsg:
		return m, nil

	case enrollResultMsg:
		if msg.err != nil {
			m.step = stepError
			m.err = msg.err
			return m, tea.Quit
		}
		m.agentID = msg.agentID
		m.step = stepInstallingService
		return m, tea.Batch(m.spinner.Tick, m.doInstallService())

	case serviceResultMsg:
		if msg.err != nil {
			m.step = stepError
			m.err = msg.err
			return m, tea.Quit
		}
		m.step = stepDone
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Forward to active input.
	var cmd tea.Cmd
	switch m.step {
	case stepServerInput:
		m.serverInput, cmd = m.serverInput.Update(msg)
	case stepTokenInput:
		m.tokenInput, cmd = m.tokenInput.Update(msg)
	}
	return m, cmd
}

func (m installModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case stepServerInput:
		server := m.serverInput.Value()
		if server == "" {
			server = "10.0.5.13:50451"
		}
		m.opts.server = server
		m.step = stepTokenInput
		m.serverInput.Blur()
		return m, m.tokenInput.Focus()

	case stepTokenInput:
		token := m.tokenInput.Value()
		if token == "" {
			// Stay on this step; user must provide a token.
			return m, nil
		}
		m.opts.token = token
		m.step = stepConnecting
		m.tokenInput.Blur()
		return m, tea.Batch(m.spinner.Tick, m.doEnroll())
	}
	return m, nil
}

// doEnroll returns a tea.Cmd that performs the full enrollment flow.
func (m installModel) doEnroll() tea.Cmd {
	opts := m.opts
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// TUI manages its own step display, so logStatus is a no-op.
		agentID, err := performEnroll(ctx, opts, func(string) {})
		if err != nil {
			return enrollResultMsg{err: err}
		}
		return enrollResultMsg{agentID: agentID}
	}
}

// doInstallService returns a tea.Cmd that registers the agent as a native
// service and starts it. The platform-specific installAndStartService function
// handles the details (systemd on Linux, SCM on Windows, error on others).
func (m installModel) doInstallService() tea.Cmd {
	return func() tea.Msg {
		if err := installAndStartService(); err != nil {
			return serviceResultMsg{err: err}
		}
		return serviceResultMsg{}
	}
}

func (m installModel) View() tea.View {
	title := titleStyle.Render("PatchIQ Agent Setup")
	hint := dimStyle.Render("Enter to continue, Esc to cancel")

	var s string
	switch m.step {
	case stepServerInput:
		s = fmt.Sprintf(
			"%s\n\nServer address:\n%s\n\n%s\n",
			title, m.serverInput.View(), hint,
		)

	case stepTokenInput:
		s = fmt.Sprintf(
			"%s\n\nServer: %s\nEnrollment token:\n%s\n\n%s\n",
			title, m.opts.server, m.tokenInput.View(), hint,
		)

	case stepConnecting:
		s = fmt.Sprintf(
			"%s\n\n%s Connecting to %s...\n",
			title, m.spinner.View(), m.opts.server,
		)

	case stepInstallingService:
		s = fmt.Sprintf(
			"%s\n\n%s Installing PatchIQ as a background service...\n",
			title, m.spinner.View(),
		)

	case stepDone:
		configPath := m.opts.configPath
		if configPath == "" {
			configPath = defaultConfigPath
		}
		s = fmt.Sprintf(
			"%s\n\n%s\n  Agent ID:    %s\n  Config:      %s\n  Service:     %s (running)\n\n%s\n",
			title,
			successStyle.Render("Setup complete!"),
			m.agentID,
			configPath,
			serviceNameForCurrentOS(),
			dimStyle.Render("The agent is now running as a background service and will start automatically on boot."),
		)

	case stepError:
		s = fmt.Sprintf(
			"%s\n\n%s\n  %v\n",
			title,
			errorStyle.Render("Error:"),
			m.err,
		)
	}

	return tea.NewView(s)
}

// serviceNameForCurrentOS returns the platform-native service name shown in
// the stepDone summary screen.
func serviceNameForCurrentOS() string {
	switch runtime.GOOS {
	case "windows":
		return "PatchIQAgent"
	case "linux":
		return "patchiq-agent"
	default:
		return "patchiq-agent"
	}
}
