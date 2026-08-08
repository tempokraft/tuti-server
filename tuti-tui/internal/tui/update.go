package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"tuti-tui/internal/api"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.applySize()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case healthResultMsg:
		if msg.err != nil {
			m.health = healthDown
			m.statusMsg = "server unreachable: " + msg.err.Error()
		} else {
			m.health = healthOK
			m.statusMsg = "connected"
		}
		return m, nil

	case resultMsg:
		return m.handleResult(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "enter":
		if m.busy {
			return m, nil
		}
		line := strings.TrimSpace(m.input.Value())
		if line == "" {
			return m, nil
		}
		m.input.Reset()
		m.appendLine(promptStyle.Render("> ") + line)

		cmd, immediate, quit := m.dispatch(line)
		if quit {
			m.quitting = true
			return m, tea.Quit
		}
		if immediate != nil {
			return m.handleResult(*immediate)
		}
		if cmd != nil {
			m.busy = true
			m.statusMsg = "running " + strings.Fields(line)[0] + "..."
			return m, cmd
		}
		return m, nil
	}

	if m.busy {
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// handleResult renders a completed (or immediate, non-networked) command
// result into the transcript and applies any state updates it carries.
func (m Model) handleResult(msg resultMsg) (tea.Model, tea.Cmd) {
	m.busy = false

	if msg.title == "__clear__" {
		m.rendered = nil
		m.syncViewport()
		return m, nil
	}

	if msg.err != nil {
		m.statusMsg = msg.title + ": error"
		m.appendLine(errorStyle.Render(msg.title+" error: ") + msg.err.Error())
		return m, nil
	}

	m.statusMsg = msg.title + ": ok"
	if msg.setSnapSessionID != "" {
		m.snapSessionID = msg.setSnapSessionID
	}
	if msg.setSolveSessionID != "" {
		m.solveSessionID = msg.setSolveSessionID
	}
	if msg.setLastCaptureID != "" {
		m.lastCaptureID = msg.setLastCaptureID
	}

	var b strings.Builder
	if msg.summary != "" {
		b.WriteString(summaryStyle.Render(msg.summary))
	}
	if msg.body != nil {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(jsonStyle.Render(api.Pretty(msg.body)))
	}
	m.appendLine(b.String())

	return m, nil
}

func (m *Model) appendLine(line string) {
	m.rendered = append(m.rendered, line)
	m.syncViewport()
}

func (m *Model) syncViewport() {
	m.viewport.SetContent(strings.Join(m.rendered, "\n\n"))
	m.viewport.GotoBottom()
}

func (m *Model) applySize() {
	headerH := 2
	statusH := 1
	inputH := 1
	vpH := m.height - headerH - statusH - inputH - 2 // border
	if vpH < 3 {
		vpH = 3
	}

	if !m.ready {
		m.viewport = viewport.New(m.width, vpH)
		m.ready = true
	} else {
		m.viewport.Width = m.width
		m.viewport.Height = vpH
	}
	m.input.Width = m.width - 4
	m.syncViewport()
}
