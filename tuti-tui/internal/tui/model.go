// Package tui implements an interactive terminal client for exercising a
// running tuti-server instance: chatting, uploading captures, and checking
// server health.
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tuti-tui/internal/api"
)

// mode tracks which "screen" of the TUI is active. The chat view is the
// default; the other two are short modal-style prompts.
type mode int

const (
	modeChat mode = iota
	modeUploadPrompt
	modeCapturePicker
)

type healthState int

const (
	healthUnknown healthState = iota
	healthOK
	healthDown
)

// Model is the root Bubble Tea model for the TUI.
type Model struct {
	client *api.Client

	mode mode

	viewport viewport.Model
	input    textarea.Model
	spinner  spinner.Model

	uploadInput textarea.Model

	history   []api.Message // sent to the server as chat context
	rendered  []string      // rendered transcript lines (user/agent/system)
	streaming bool
	streamBuf strings.Builder
	chunkCh   chan string

	captureID  string // attached to the next outgoing message, if any
	captures   []api.Capture
	captureSel int

	health    healthState
	statusMsg string

	width, height int
	ready         bool

	quitting bool
}

// New builds the initial Model pointed at the given server base URL.
func New(baseURL string) Model {
	ta := textarea.New()
	ta.Placeholder = "Ask a question... (Enter to send, Ctrl+J for newline)"
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.SetHeight(3)
	ta.CharLimit = 8000

	ua := textarea.New()
	ua.Placeholder = "/path/to/screenshot.png"
	ua.ShowLineNumbers = false
	ua.SetHeight(1)
	ua.CharLimit = 1000

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		client:      api.New(baseURL),
		mode:        modeChat,
		input:       ta,
		uploadInput: ua,
		spinner:     sp,
		health:      healthUnknown,
		statusMsg:   "checking server...",
	}
}

// Init kicks off an initial health check.
func (m Model) Init() tea.Cmd {
	return tea.Batch(checkHealth(m.client), m.spinner.Tick)
}

// --- messages ---

type healthResultMsg struct{ err error }

type chatChunkMsg struct{ text string }
type chatDoneMsg struct{ err error }

type capturesLoadedMsg struct {
	captures []api.Capture
	err      error
}

type uploadDoneMsg struct {
	id  string
	err error
}

func checkHealth(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		return healthResultMsg{err: c.Health()}
	}
}

func loadCaptures(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		caps, err := c.ListCaptures()
		return capturesLoadedMsg{captures: caps, err: err}
	}
}

func doUpload(c *api.Client, path string) tea.Cmd {
	return func() tea.Msg {
		id, err := c.UploadCapture(strings.TrimSpace(path))
		return uploadDoneMsg{id: id, err: err}
	}
}

// sendChat streams a reply, pushing chatChunkMsg values through chunkCh as
// they arrive and finishing with a chatDoneMsg read via the returned Cmd.
func sendChat(c *api.Client, req api.ChatRequest, chunkCh chan string) tea.Cmd {
	return func() tea.Msg {
		err := c.Chat(req, func(s string) {
			chunkCh <- s
		})
		close(chunkCh)
		return chatDoneMsg{err: err}
	}
}

// waitForChunk turns the next value off chunkCh into a Bubble Tea message;
// re-issued after every chunk to keep draining the channel.
func waitForChunk(chunkCh chan string) tea.Cmd {
	return func() tea.Msg {
		text, ok := <-chunkCh
		if !ok {
			return nil
		}
		return chatChunkMsg{text: text}
	}
}
