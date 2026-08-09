// Package tui implements an interactive terminal client for exercising a
// running tuti-server instance: every TutiService RPC, driven from a
// menu (left) of commands and a form (right) of their parameters, with
// results shown in a scrollable log underneath.
package tui

import (
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
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

// focus tracks which region of the screen has keyboard input.
type focus int

const (
	focusMenu focus = iota
	focusForm
	focusPicker
	focusHistory
)

// formField pairs a paramSpec with its live editable state — a textinput
// for text/file kinds, a selected index for choice kinds.
type formField struct {
	spec      paramSpec
	input     textinput.Model
	choiceIdx int
}

func newFormField(spec paramSpec) formField {
	f := formField{spec: spec}
	if spec.Kind != paramChoice {
		ti := textinput.New()
		ti.Placeholder = spec.Placeholder
		ti.SetValue(spec.Default)
		ti.CharLimit = 2000
		f.input = ti
	}
	return f
}

// Value returns the field's current value regardless of kind.
func (f formField) Value() string {
	if f.spec.Kind == paramChoice {
		if len(f.spec.Choices) == 0 {
			return ""
		}
		return f.spec.Choices[f.choiceIdx]
	}
	return f.input.Value()
}

// Model is the root Bubble Tea model for the TUI.
type Model struct {
	client *api.Client

	menu   list.Model
	fields []formField
	fieldI int // index into fields of the currently focused one

	picker       filepicker.Model
	pickerTarget int // index into fields the picker result will fill

	historyIdx      int // index into historyEntriesNewestFirst() of the selected entry, when focusHistory
	historyViewport viewport.Model

	viewport viewport.Model
	spinner  spinner.Model

	focus    focus
	rendered []string // results log lines
	busy     bool

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
	items := make([]list.Item, len(commandCatalog))
	for i, c := range commandCatalog {
		items[i] = c
	}
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	menu := list.New(items, delegate, 0, 0)
	menu.Title = "Commands"
	menu.SetShowStatusBar(false)
	menu.SetShowHelp(false)
	menu.SetFilteringEnabled(false)
	menu.Styles.Title = titleStyle

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	pk := filepicker.New()
	pk.DirAllowed = false
	pk.FileAllowed = true
	pk.AllowedTypes = []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".heic"}

	m := Model{
		client:    api.New(baseURL),
		menu:      menu,
		picker:    pk,
		spinner:   sp,
		health:    healthUnknown,
		statusMsg: "checking server...",
	}
	m.fields = fieldsFor(commandCatalog[0])
	return m
}

func fieldsFor(c commandSpec) []formField {
	fields := make([]formField, len(c.Params))
	for i, p := range c.Params {
		fields[i] = newFormField(p)
	}
	if len(fields) > 0 && fields[0].spec.Kind != paramChoice {
		fields[0].input.Focus()
	}
	return fields
}

func (m Model) selectedCommand() commandSpec {
	if c, ok := m.menu.SelectedItem().(commandSpec); ok {
		return c
	}
	return commandSpec{}
}

// Init kicks off an initial health check.
func (m Model) Init() tea.Cmd {
	return tea.Batch(checkHealth(m.client), m.spinner.Tick)
}
