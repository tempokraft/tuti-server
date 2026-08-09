package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"tuti-tui/internal/api"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)

	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	summaryStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	jsonStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	errorStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))

	healthOKStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	healthDownStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	healthUnkStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	statusBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

	fieldLabelStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	fieldLabelFocusStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	choiceStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	choiceSelectedStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	fileHintStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	actionButtonStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	sendingStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	scrollActiveStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	scrollDimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	boxFocusStyle = boxStyle.
			BorderForeground(lipgloss.Color("205"))

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

	menuBox := boxStyle
	formBox := boxStyle
	historyBox := boxStyle
	switch m.focus {
	case focusMenu:
		menuBox = boxFocusStyle
	case focusForm, focusPicker:
		formBox = boxFocusStyle
	case focusHistory:
		historyBox = boxFocusStyle
	}

	menuW, formW, historyW := m.columnWidths()

	left := menuBox.Width(menuW - 2).Render(m.menu.View())

	var mid string
	if m.focus == focusPicker {
		mid = formBox.Width(formW - 2).Render(m.pickerView())
	} else {
		mid = formBox.Width(formW - 2).Render(m.formView())
	}

	right := historyBox.Width(historyW - 2).Render(m.historyView())

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, " ", mid, " ", right))
	b.WriteString("\n")
	b.WriteString(boxStyle.Width(m.width - 2).Render(m.viewport.View()))
	b.WriteString("\n")
	b.WriteString(m.statusBar())
	b.WriteString("\n")
	b.WriteString(m.footer())
	return b.String()
}

// columnWidths splits the top row into three side-by-side boxes — menu,
// form, and history — each returned width being the total on-screen width
// of that box (border included), so callers pass width-2 to Style.Width to
// account for the border.
func (m Model) columnWidths() (menuW, formW, historyW int) {
	menuW = m.width * 3 / 10
	if menuW < 24 {
		menuW = 24
	}
	remaining := m.width - menuW - 2 // two single-space gaps between the three boxes
	if remaining < 0 {
		remaining = 0
	}
	formW = remaining * 3 / 5
	historyW = remaining - formW
	if formW < 10 {
		formW = 10
	}
	if historyW < 10 {
		historyW = 10
	}
	return
}

// topRowHeight is the shared height budget for the menu/form/history row.
func (m Model) topRowHeight() int {
	topH := m.height * 2 / 5
	if topH < 8 {
		topH = 8
	}
	return topH
}

// historyEntriesNewestFirst returns every retained history entry
// (client.History() is oldest-first) reordered newest first, matching how
// the panel displays and indexes them.
func (m Model) historyEntriesNewestFirst() []api.HistoryEntry {
	entries := m.client.History()
	out := make([]api.HistoryEntry, len(entries))
	for i, e := range entries {
		out[len(entries)-1-i] = e
	}
	return out
}

// historyContentWidth is the text width history content is wrapped/
// truncated to, derived from the panel's column width.
func (m Model) historyContentWidth() int {
	_, _, historyW := m.columnWidths()
	return historyW - 6
}

// historyContent renders every retained entry as scrollable text: a
// summary line, then the raw request and response (params over 500 chars
// redacted in the request — see PrettyRawRedacted). Whichever entry is
// selected gets an explicit "Replay" button (or a "sending" message while
// busy) attached under its summary. selStart/selEnd is the 0-based line
// range of the selected entry within the returned content, so the caller
// can keep it scrolled into view.
func (m Model) historyContent(width int) (content string, selStart, selEnd int) {
	entries := m.historyEntriesNewestFirst()
	if len(entries) == 0 {
		return subtleStyle.Render("(no requests yet)"), 0, -1
	}

	var blocks []string
	line := 0
	for i, e := range entries {
		selected := m.focus == focusHistory && i == m.historyIdx

		status := e.Status
		if status == "" {
			status = "-"
		}
		marker := "  "
		if selected {
			marker = "▸ "
		}
		summary := truncateLine(marker+fmt.Sprintf("%s %s %s %s",
			e.Time.Format("15:04:05"), e.RPC, status, e.Duration.Round(time.Millisecond)), width)

		var lines []string
		switch {
		case e.Err != nil:
			lines = append(lines, errorStyle.Render(summary))
		case selected:
			lines = append(lines, fieldLabelFocusStyle.Render(summary))
		default:
			lines = append(lines, subtleStyle.Render(summary))
		}

		if selected {
			if m.busy {
				lines = append(lines, sendingStyle.Render(truncateLine("    ⏳ Sending — waiting for response...", width)))
			} else {
				lines = append(lines, actionButtonStyle.Render(truncateLine("    [ enter ▶ Replay ]", width)))
			}
		}

		lines = append(lines, subtleStyle.Render(truncateLine("  request:", width)))
		for _, l := range strings.Split(e.Request, "\n") {
			lines = append(lines, jsonStyle.Render(truncateLine(l, width)))
		}
		lines = append(lines, subtleStyle.Render(truncateLine("  response:", width)))
		for _, l := range strings.Split(e.Response, "\n") {
			lines = append(lines, jsonStyle.Render(truncateLine(l, width)))
		}

		if selected {
			selStart = line
			selEnd = line + len(lines) - 1
		}
		line += len(lines)
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	return strings.Join(blocks, "\n"), selStart, selEnd
}

// historyView renders the fixed "History" heading — with a position
// counter and ▲▼ scroll-availability indicators, so it's visible at a
// glance whether there's more content above/below without having to
// scroll to find out — plus the scrollable viewport beneath it. See
// syncHistoryViewport for how that viewport's content and scroll position
// get kept up to date.
func (m Model) historyView() string {
	header := titleStyle.Render("History")

	count := len(m.client.History())
	if count > 0 {
		if m.focus == focusHistory {
			header += " " + subtleStyle.Render(fmt.Sprintf("%d/%d", m.historyIdx+1, count))
		} else {
			header += " " + subtleStyle.Render(fmt.Sprintf("(%d)", count))
		}

		up, down := scrollDimStyle, scrollDimStyle
		if !m.historyViewport.AtTop() {
			up = scrollActiveStyle
		}
		if !m.historyViewport.AtBottom() {
			down = scrollActiveStyle
		}
		header += " " + up.Render("▲") + down.Render("▼")

		if m.focus != focusHistory {
			header += " " + subtleStyle.Render("(tab to scroll)")
		}
	}

	return header + "\n" + m.historyViewport.View()
}

func truncateLine(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	if n == 1 {
		return s[:1]
	}
	return s[:n-1] + "…"
}

func (m Model) header() string {
	return titleStyle.Render("tuti-tui") + " " + subtleStyle.Render(m.client.BaseURL)
}

func (m Model) formView() string {
	c := m.selectedCommand()

	var b strings.Builder
	b.WriteString(titleStyle.Render(c.Name))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render(c.Desc))
	b.WriteString("\n")

	if len(c.Params) == 0 {
		b.WriteString("\n")
		b.WriteString(subtleStyle.Render("No parameters."))
	} else {
		for i, f := range m.fields {
			focused := m.focus == focusForm && i == m.fieldI
			label := fieldLabelStyle
			if focused {
				label = fieldLabelFocusStyle
			}
			marker := "  "
			if focused {
				marker = "▸ "
			}
			req := ""
			if !f.spec.Optional {
				req = " *"
			}

			b.WriteString("\n")
			b.WriteString(marker + label.Render(f.spec.Name+req))
			b.WriteString("\n")

			switch f.spec.Kind {
			case paramChoice:
				b.WriteString("    " + renderChoices(f))
			case paramFile:
				b.WriteString("    " + f.input.View())
				b.WriteString("\n    " + fileHintStyle.Render("Ctrl+F to browse your machine for a file"))
			default:
				b.WriteString("    " + f.input.View())
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	if m.busy && (m.focus == focusMenu || m.focus == focusForm) {
		b.WriteString(sendingStyle.Render("⏳ Sending — waiting for response..."))
	} else {
		b.WriteString(actionButtonStyle.Render("[ enter ▶ Submit ]"))
	}
	return b.String()
}

func renderChoices(f formField) string {
	parts := make([]string, len(f.spec.Choices))
	for i, choice := range f.spec.Choices {
		if i == f.choiceIdx {
			parts[i] = choiceSelectedStyle.Render("● " + choice)
		} else {
			parts[i] = choiceStyle.Render("○ " + choice)
		}
	}
	return strings.Join(parts, "   ")
}

func (m Model) pickerView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Choose a file"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render(m.picker.CurrentDirectory))
	b.WriteString("\n\n")
	b.WriteString(m.picker.View())
	return b.String()
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
	if m.busy {
		line += "  " + m.spinner.View() + " " + sendingStyle.Render("sending — waiting for response...")
	}
	if len(ctx) > 0 {
		line += "  " + subtleStyle.Render(strings.Join(ctx, "  "))
	}
	if m.statusMsg != "" {
		line += "  " + statusBarStyle.Render(m.statusMsg)
	}
	return line
}

func (m Model) footer() string {
	switch m.focus {
	case focusMenu:
		return helpStyle.Render("↑/↓ select · enter run/edit params · tab history panel · ctrl+c quit")
	case focusHistory:
		return helpStyle.Render("↑/↓ select call · pgup/pgdn/home/end scroll · enter replay · esc/tab back to menu · ctrl+c quit")
	case focusPicker:
		return helpStyle.Render("↑/↓ navigate · enter open/select · esc cancel")
	default:
		return helpStyle.Render("tab/shift+tab move · ←/→ choose · ctrl+f browse file · enter/ctrl+s run · esc back · ctrl+c quit")
	}
}
