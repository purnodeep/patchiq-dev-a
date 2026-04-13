package cli

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

type statusTickMsg time.Time

type statusUpdateMsg struct {
	info StatusInfo
	err  error
}

type statusModel struct {
	state  *comms.AgentState
	outbox *comms.Outbox
	info   StatusInfo
	err    error
}

func newStatusModel(state *comms.AgentState, outbox *comms.Outbox) statusModel {
	return statusModel{
		state:  state,
		outbox: outbox,
	}
}

func (m statusModel) Init() tea.Cmd {
	return tea.Batch(m.refreshCmd(), m.tickCmd())
}

func (m statusModel) tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return statusTickMsg(t)
	})
}

func (m statusModel) refreshCmd() tea.Cmd {
	state := m.state
	outbox := m.outbox
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		info, err := collectStatusInfo(ctx, state, outbox)
		return statusUpdateMsg{info: info, err: err}
	}
}

func (m statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			return m, m.refreshCmd()
		}

	case tea.WindowSizeMsg:
		return m, nil

	case statusTickMsg:
		return m, tea.Batch(m.refreshCmd(), m.tickCmd())

	case statusUpdateMsg:
		m.info = msg.info
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

func (m statusModel) View() tea.View {
	title := titleStyle.Render("PatchIQ Agent Status")
	live := dimStyle.Render("[live]")

	var s string
	if m.err != nil {
		s = fmt.Sprintf(
			"%s %s\n\n%s %v\n\n%s\n",
			title, live,
			errorStyle.Render("Error:"), m.err,
			dimStyle.Render("r: refresh | q: quit"),
		)
		v := tea.NewView(s)
		v.AltScreen = true
		return v
	}

	connStyle := redStyle
	if m.info.Connection == "connected" {
		connStyle = greenStyle
	}

	s = fmt.Sprintf(
		"%s %s\n\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n\n%s\n",
		title, live,
		labelStyle.Render("Agent ID:"), valueStyle.Render(m.info.AgentID),
		labelStyle.Render("Connection:"), connStyle.Render(m.info.Connection),
		labelStyle.Render("Last Heartbeat:"), valueStyle.Render(m.info.LastHeartbeat),
		labelStyle.Render("Last Scan:"), valueStyle.Render(m.info.LastScan),
		labelStyle.Render("Queue Depth:"), valueStyle.Render(fmt.Sprintf("%d", m.info.QueueDepth)),
		dimStyle.Render("r: refresh | q: quit"),
	)

	v := tea.NewView(s)
	v.AltScreen = true
	return v
}
