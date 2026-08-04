package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
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

	case capturesLoadedMsg:
		if msg.err != nil {
			m.statusMsg = "failed to list captures: " + msg.err.Error()
			m.mode = modeChat
			return m, nil
		}
		m.captures = msg.captures
		m.captureSel = 0
		m.mode = modeCapturePicker
		return m, nil

	case uploadDoneMsg:
		m.mode = modeChat
		m.input.Focus()
		if msg.err != nil {
			m.statusMsg = "upload failed: " + msg.err.Error()
		} else {
			m.captureID = msg.id
			m.statusMsg = "attached capture " + msg.id
			m.appendLine(systemStyle.Render(fmt.Sprintf("[attached capture %s]", msg.id)))
		}
		return m, nil

	case chatChunkMsg:
		m.streamBuf.WriteString(msg.text)
		m.refreshStreamingLine()
		return m, waitForChunk(m.chunkCh)

	case chatDoneMsg:
		m.streaming = false
		final := m.streamBuf.String()
		m.streamBuf.Reset()
		if msg.err != nil {
			m.statusMsg = "chat error: " + msg.err.Error()
			if final == "" {
				m.appendLine(errorStyle.Render("error: " + msg.err.Error()))
				return m, nil
			}
		}
		if final != "" {
			m.history = append(m.history, api.Message{Role: "agent", Text: final})
		}
		m.finalizeStreamingLine(final)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeUploadPrompt:
		return m.handleUploadKey(msg)
	case modeCapturePicker:
		return m.handleCapturePickerKey(msg)
	default:
		return m.handleChatKey(msg)
	}
}

func (m Model) handleChatKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "esc":
		if m.captureID != "" {
			m.captureID = ""
			m.statusMsg = "capture detached"
			return m, nil
		}

	case "ctrl+u":
		if m.streaming {
			return m, nil
		}
		m.mode = modeUploadPrompt
		m.uploadInput.Reset()
		m.uploadInput.Focus()
		m.input.Blur()
		return m, textarea.Blink

	case "ctrl+l":
		if m.streaming {
			return m, nil
		}
		m.statusMsg = "loading captures..."
		return m, loadCaptures(m.client)

	case "ctrl+r":
		m.statusMsg = "checking server..."
		return m, checkHealth(m.client)

	case "enter":
		if m.streaming {
			return m, nil
		}
		text := strings.TrimSpace(m.input.Value())
		if text == "" {
			return m, nil
		}
		return m.startChat(text)
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) handleUploadKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.mode = modeChat
		m.input.Focus()
		return m, nil
	case "enter":
		path := strings.TrimSpace(m.uploadInput.Value())
		if path == "" {
			m.mode = modeChat
			m.input.Focus()
			return m, nil
		}
		m.statusMsg = "uploading " + path + "..."
		return m, doUpload(m.client, path)
	}

	var cmd tea.Cmd
	m.uploadInput, cmd = m.uploadInput.Update(msg)
	return m, cmd
}

func (m Model) handleCapturePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.mode = modeChat
		m.input.Focus()
		return m, nil
	case "up", "k":
		if m.captureSel > 0 {
			m.captureSel--
		}
	case "down", "j":
		if m.captureSel < len(m.captures)-1 {
			m.captureSel++
		}
	case "enter":
		m.mode = modeChat
		m.input.Focus()
		if len(m.captures) > 0 {
			sel := m.captures[m.captureSel]
			m.captureID = sel.ID
			m.statusMsg = "attached capture " + sel.ID
			m.appendLine(systemStyle.Render(fmt.Sprintf("[attached capture %s (%s)]", sel.ID, sel.Name)))
		}
	}
	return m, nil
}

// startChat renders the user's message immediately and kicks off a
// streaming request for the reply.
func (m Model) startChat(text string) (tea.Model, tea.Cmd) {
	m.appendLine(userStyle.Render("you  ") + " " + text)

	req := api.ChatRequest{
		Message:   text,
		History:   append([]api.Message(nil), m.history...),
		CaptureID: m.captureID,
	}
	m.history = append(m.history, api.Message{Role: "user", Text: text})
	m.captureID = ""

	m.input.Reset()
	m.streaming = true
	m.streamBuf.Reset()
	m.appendLine("") // placeholder line for the streaming reply
	m.chunkCh = make(chan string)

	return m, tea.Batch(sendChat(m.client, req, m.chunkCh), waitForChunk(m.chunkCh))
}

// appendLine adds a finished line to the transcript and scrolls to bottom.
func (m *Model) appendLine(line string) {
	m.rendered = append(m.rendered, line)
	m.syncViewport()
}

// refreshStreamingLine rewrites the last transcript line with the
// in-progress agent reply.
func (m *Model) refreshStreamingLine() {
	if len(m.rendered) == 0 {
		return
	}
	m.rendered[len(m.rendered)-1] = agentStyle.Render("tuti ") + " " + m.streamBuf.String()
	m.syncViewport()
}

func (m *Model) finalizeStreamingLine(final string) {
	if len(m.rendered) == 0 {
		return
	}
	if final == "" {
		final = "(no reply)"
	}
	m.rendered[len(m.rendered)-1] = agentStyle.Render("tuti ") + " " + final
	m.syncViewport()
}

func (m *Model) syncViewport() {
	m.viewport.SetContent(strings.Join(m.rendered, "\n\n"))
	m.viewport.GotoBottom()
}

func (m *Model) applySize() {
	headerH := 3
	statusH := 1
	inputH := m.input.Height() + 2 // border
	vpH := m.height - headerH - statusH - inputH
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
	m.input.SetWidth(m.width - 4)
	m.uploadInput.SetWidth(m.width - 4)
	m.syncViewport()
}
