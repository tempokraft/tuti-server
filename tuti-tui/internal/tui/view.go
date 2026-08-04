package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)

	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	userStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	agentStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	systemStyle = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("243"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

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

	switch m.mode {
	case modeUploadPrompt:
		return m.viewUploadPrompt()
	case modeCapturePicker:
		return m.viewCapturePicker()
	default:
		return m.viewChat()
	}
}

func (m Model) header() string {
	title := titleStyle.Render("tuti-tui") + " " + subtleStyle.Render(m.client.BaseURL)
	return title
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
	capture := ""
	if m.captureID != "" {
		capture = "  " + subtleStyle.Render("capture: "+m.captureID)
	}
	streaming := ""
	if m.streaming {
		streaming = "  " + m.spinner.View() + " streaming..."
	}
	line := m.healthBadge() + capture + streaming
	if m.statusMsg != "" {
		line += "  " + statusBarStyle.Render(m.statusMsg)
	}
	return line
}

func (m Model) footer() string {
	return helpStyle.Render("enter send · ctrl+u upload capture · ctrl+l list captures · esc detach capture · ctrl+r recheck health · ctrl+c quit")
}

func (m Model) viewChat() string {
	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString("\n")
	b.WriteString(boxStyle.Width(m.width - 2).Render(m.viewport.View()))
	b.WriteString("\n")
	b.WriteString(m.statusBar())
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n")
	b.WriteString(m.footer())
	return b.String()
}

func (m Model) viewUploadPrompt() string {
	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString("\n\n")
	b.WriteString(titleStyle.Render("Upload a capture"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("Enter a local file path, then press Enter. Esc to cancel."))
	b.WriteString("\n\n")
	b.WriteString(boxStyle.Width(m.width - 2).Render(m.uploadInput.View()))
	b.WriteString("\n")
	if m.statusMsg != "" {
		b.WriteString(statusBarStyle.Render(m.statusMsg))
	}
	return b.String()
}

func (m Model) viewCapturePicker() string {
	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString("\n\n")
	b.WriteString(titleStyle.Render("Select a capture to attach"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("↑/↓ to move, enter to attach, esc to cancel."))
	b.WriteString("\n\n")

	if len(m.captures) == 0 {
		b.WriteString(subtleStyle.Render("no captures uploaded yet"))
		return b.String()
	}

	for i, c := range m.captures {
		line := fmt.Sprintf("%s  %-24s  %8d bytes  %s", c.ID, c.Name, c.SizeBytes, c.UploadedAt.Local().Format("2006-01-02 15:04:05"))
		if i == m.captureSel {
			b.WriteString(userStyle.Render("> " + line))
		} else {
			b.WriteString("  " + subtleStyle.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}
