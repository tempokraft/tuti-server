package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)

	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	promptStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	summaryStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	jsonStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	errorStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))

	healthOKStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	healthDownStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	healthUnkStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	statusBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if !m.ready {
		return "starting up...\n"
	}

	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString("\n")
	b.WriteString(boxStyle.Width(m.width - 2).Render(m.viewport.View()))
	b.WriteString("\n")
	b.WriteString(m.statusBar())
	b.WriteString("\n")
	if m.busy {
		b.WriteString(m.spinner.View() + " " + subtleStyle.Render("working..."))
	} else {
		b.WriteString(m.input.View())
	}
	return b.String()
}

func (m Model) header() string {
	return titleStyle.Render("tuti-tui") + " " + subtleStyle.Render(m.client.BaseURL)
}

func (m Model) healthBadge() string {
	switch m.health {
	case healthOK:
		return healthOKStyle.Render("● connected")
	case healthDown:
		return healthDownStyle.Render("● unreachable")
	default:
		return healthUnkStyle.Render(m.spinner.View() + " checking")
	}
}

func (m Model) statusBar() string {
	var ctx []string
	if m.snapSessionID != "" {
		ctx = append(ctx, "snap: "+m.snapSessionID)
	}
	if m.solveSessionID != "" {
		ctx = append(ctx, "solve: "+m.solveSessionID)
	}
	if m.lastCaptureID != "" {
		ctx = append(ctx, "capture: "+m.lastCaptureID)
	}

	line := m.healthBadge()
	if len(ctx) > 0 {
		line += "  " + subtleStyle.Render(strings.Join(ctx, "  "))
	}
	if m.statusMsg != "" {
		line += "  " + statusBarStyle.Render(m.statusMsg)
	}
	return line
}
