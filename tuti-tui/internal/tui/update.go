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

	default:
		// filepicker.Init()/Update() emit their own message types
		// (readDirMsg, errorMsg, ...) that only it knows how to handle —
		// route anything we don't recognize to it while it's active, or
		// its directory listing never arrives.
		if m.focus == focusPicker {
			var cmd tea.Cmd
			m.picker, cmd = m.picker.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}
	if m.busy {
		return m, nil
	}

	switch m.focus {
	case focusPicker:
		return m.handlePickerKey(msg)
	case focusForm:
		return m.handleFormKey(msg)
	case focusHistory:
		return m.handleHistoryKey(msg)
	default:
		return m.handleMenuKey(msg)
	}
}

func (m Model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	before := m.selectedCommand().ID

	switch msg.String() {
	case "enter":
		c := m.selectedCommand()
		if len(c.Params) == 0 {
			return m.runNow()
		}
		m.focus = focusForm
		m.fieldI = 0
		if len(m.fields) > 0 && m.fields[0].spec.Kind != paramChoice {
			m.fields[0].input.Focus()
		}
		return m, nil

	case "tab":
		if len(m.historyEntriesNewestFirst()) > 0 {
			m.focus = focusHistory
			m.historyIdx = 0
			m.syncHistoryViewport()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.menu, cmd = m.menu.Update(msg)
	if m.selectedCommand().ID != before {
		m.fields = fieldsFor(m.selectedCommand())
		m.fieldI = 0
	}
	return m, cmd
}

// handleHistoryKey drives the history panel: browsing past requests,
// firing an explicit replay of whichever one is selected, and scrolling
// (both by moving the selection and directly via page up/down) to see the
// full request/response of entries that don't fit in the panel at once.
func (m Model) handleHistoryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	entries := m.historyEntriesNewestFirst()

	switch msg.String() {
	case "esc", "tab":
		m.focus = focusMenu
		return m, nil

	case "up", "k":
		if m.historyIdx > 0 {
			m.historyIdx--
		}
		m.syncHistoryViewport()
		return m, nil

	case "down", "j":
		if m.historyIdx < len(entries)-1 {
			m.historyIdx++
		}
		m.syncHistoryViewport()
		return m, nil

	case "enter", "r":
		if m.historyIdx < 0 || m.historyIdx >= len(entries) {
			return m, nil
		}
		entry := entries[m.historyIdx]
		m.busy = true
		m.statusMsg = "replaying " + entry.RPC + "..."
		m.syncHistoryViewport()
		return m, replayCmd(m.client, entry)

	case "pgup", "pgdown":
		var cmd tea.Cmd
		m.historyViewport, cmd = m.historyViewport.Update(msg)
		return m, cmd

	case "home":
		m.historyViewport.GotoTop()
		return m, nil

	case "end":
		m.historyViewport.GotoBottom()
		return m, nil
	}
	return m, nil
}

func (m Model) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.fields) == 0 {
		m.focus = focusMenu
		return m, nil
	}
	current := m.fields[m.fieldI].spec.Kind

	switch msg.String() {
	case "esc":
		m.blurField()
		m.focus = focusMenu
		return m, nil

	case "tab":
		m.blurField()
		if m.fieldI == len(m.fields)-1 {
			m.focus = focusMenu
			return m, nil
		}
		m.fieldI++
		m.focusField()
		return m, nil

	case "shift+tab":
		m.blurField()
		if m.fieldI == 0 {
			m.focus = focusMenu
			return m, nil
		}
		m.fieldI--
		m.focusField()
		return m, nil

	case "ctrl+f":
		if current == paramFile {
			return m.openPicker()
		}
		return m, nil

	case "ctrl+s":
		return m.runNow()

	case "enter":
		return m.runNow()

	case "left", "right":
		if current == paramChoice {
			f := &m.fields[m.fieldI]
			n := len(f.spec.Choices)
			if msg.String() == "left" {
				f.choiceIdx = (f.choiceIdx - 1 + n) % n
			} else {
				f.choiceIdx = (f.choiceIdx + 1) % n
			}
			return m, nil
		}
	}

	if current == paramChoice {
		return m, nil
	}
	var cmd tea.Cmd
	m.fields[m.fieldI].input, cmd = m.fields[m.fieldI].input.Update(msg)
	return m, cmd
}

func (m *Model) blurField() {
	if m.fields[m.fieldI].spec.Kind != paramChoice {
		m.fields[m.fieldI].input.Blur()
	}
}

func (m *Model) focusField() {
	if m.fields[m.fieldI].spec.Kind != paramChoice {
		m.fields[m.fieldI].input.Focus()
	}
}

func (m Model) openPicker() (tea.Model, tea.Cmd) {
	m.blurField()
	m.pickerTarget = m.fieldI
	m.focus = focusPicker
	return m, m.picker.Init()
}

func (m Model) handlePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.focus = focusForm
		m.focusField()
		return m, nil
	}

	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)

	if ok, path := m.picker.DidSelectFile(msg); ok {
		m.fields[m.pickerTarget].input.SetValue(path)
		m.focus = focusForm
		m.fieldI = m.pickerTarget
		m.focusField()
		return m, nil
	}
	return m, cmd
}

func (m Model) runNow() (tea.Model, tea.Cmd) {
	cmd, immediate := m.run()
	if immediate != nil {
		return m.handleResult(*immediate)
	}
	if cmd != nil {
		m.busy = true
		m.statusMsg = "running " + m.selectedCommand().Name + "..."
		return m, cmd
	}
	return m, nil
}

// handleResult renders a completed (or immediate, non-networked) command
// result into the log and applies any state updates it carries.
func (m Model) handleResult(msg resultMsg) (tea.Model, tea.Cmd) {
	m.busy = false
	m.syncHistoryViewport()

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

// syncHistoryViewport rebuilds the history panel's scrollable content and,
// if the selected entry isn't already fully visible, scrolls to its start
// — called whenever the entry list, the selection, the busy state, or the
// panel's size changes. Anchoring on the start (rather than pulling the
// end into view) matters when an entry is taller than the viewport: it
// keeps the entry's header and Replay button on screen, with the rest of
// its request/response reachable by scrolling further, instead of jumping
// straight to the tail and hiding what's selected.
func (m *Model) syncHistoryViewport() {
	content, selStart, selEnd := m.historyContent(m.historyContentWidth())
	m.historyViewport.SetContent(content)

	if selEnd < selStart {
		return
	}
	top := m.historyViewport.YOffset
	bottom := top + m.historyViewport.Height - 1
	if selStart < top || selEnd > bottom {
		m.historyViewport.SetYOffset(selStart)
	}
}

func (m *Model) applySize() {
	headerH := 2
	statusH := 1
	footerH := 1

	menuW, formW, historyW := m.columnWidths()
	topH := m.topRowHeight()
	logH := m.height - headerH - statusH - footerH - topH - 2
	if logH < 3 {
		logH = 3
	}
	// historyH leaves one line for the "History" heading rendered above
	// the viewport in historyView.
	historyH := topH - 3
	if historyH < 1 {
		historyH = 1
	}

	m.menu.SetSize(menuW-2, topH-2)
	for i := range m.fields {
		m.fields[i].input.Width = formW - 6
	}
	m.picker.SetHeight(topH - 2)

	if !m.ready {
		m.viewport = viewport.New(m.width, logH)
		m.historyViewport = viewport.New(historyW-6, historyH)
		m.ready = true
	} else {
		m.viewport.Width = m.width
		m.viewport.Height = logH
		m.historyViewport.Width = historyW - 6
		m.historyViewport.Height = historyH
	}
	m.syncViewport()
	m.syncHistoryViewport()
}
