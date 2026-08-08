// Package tui implements an interactive terminal client for exercising a
// running tuti-server instance: every TutiService RPC, driven from a
// single command line, with results shown in a scrollable transcript.
package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tuti-tui/internal/api"
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

	viewport viewport.Model
	input    textinput.Model
	spinner  spinner.Model

	rendered []string // transcript lines, one entry per printed block
	busy     bool     // a command is in flight; input is locked

	// Context carried between commands, so a test session doesn't require
	// re-pasting ids by hand.
	snapSessionID  string
	solveSessionID string
	lastCaptureID  string

	health    healthState
	statusMsg string

	width, height int
	ready         bool

	quitting bool
}

// New builds the initial Model pointed at the given server base URL.
func New(baseURL string) Model {
	ti := textinput.New()
	ti.Placeholder = "type a command — 'help' to list them"
	ti.Focus()
	ti.CharLimit = 2000

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		client:    api.New(baseURL),
		input:     ti,
		spinner:   sp,
		health:    healthUnknown,
		statusMsg: "checking server...",
	}
}

// Init kicks off an initial health check and prints the welcome banner.
func (m Model) Init() tea.Cmd {
	return tea.Batch(checkHealth(m.client), m.spinner.Tick)
}
