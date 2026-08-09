package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tuti-tui/internal/api"
)

// ── messages ─────────────────────────────────────────────────────────────

type healthResultMsg struct{ err error }

type resultMsg struct {
	// title is echoed to the log before the result (or error).
	title string
	err   error
	// summary is a human-readable line shown above the raw JSON dump —
	// used for NextStep results, where "here's what happens next" matters
	// more than the wire shape.
	summary string
	body    any // marshaled via api.Pretty when non-nil

	// State updates to apply on success. Zero values leave the
	// corresponding Model field untouched.
	setSnapSessionID  string
	setSolveSessionID string
	setLastCaptureID  string
}

func checkHealth(c *api.Client) tea.Cmd {
	return func() tea.Msg { return healthResultMsg{err: c.Health()} }
}

// replayCmd resends entry's original request byte-for-byte via
// api.Client.Replay and renders the raw response the same way any other
// command result is shown.
func replayCmd(c *api.Client, entry api.HistoryEntry) tea.Cmd {
	title := "replay: " + entry.RPC
	return func() tea.Msg {
		respBody, err := c.Replay(context.Background(), entry)
		if err != nil {
			return resultMsg{title: title, err: err}
		}
		var body any
		if len(respBody) > 0 {
			if err := json.Unmarshal(respBody, &body); err != nil {
				body = string(respBody)
			}
		}
		return resultMsg{title: title, body: body, summary: "replayed " + entry.RPC}
	}
}

// ── run ──────────────────────────────────────────────────────────────────

// fieldValue returns the value of the field named name, or "" if the
// selected command has no such field.
func (m Model) fieldValue(name string) string {
	for _, f := range m.fields {
		if f.spec.Name == name {
			return f.Value()
		}
	}
	return ""
}

// run builds the tea.Cmd for the currently selected command using its
// form field values, or — for validation failures and local utilities —
// an immediate *resultMsg that doesn't need a network round trip.
func (m Model) run() (tea.Cmd, *resultMsg) {
	c := m.selectedCommand()

	switch c.ID {
	case "clear":
		return nil, &resultMsg{title: "__clear__"}

	case "health":
		return func() tea.Msg {
			return resultMsg{title: "health", err: m.client.Health(), summary: "ok"}
		}, nil

	case "init":
		return func() tea.Msg {
			id, next, err := m.client.InitializeSnapAndSolve()
			if err != nil {
				return resultMsg{title: "init", err: err}
			}
			return resultMsg{
				title: "init",
				body: struct {
					SessionID string        `json:"sessionId"`
					NextStep  *api.NextStep `json:"nextStep"`
				}{SessionID: id, NextStep: next},
				summary:          "session: " + id + " — " + summarizeNextStep(next),
				setSnapSessionID: id,
			}
		}, nil

	case "snap":
		path := m.fieldValue("Photo")
		if path == "" {
			return nil, &resultMsg{title: "snap", err: fmt.Errorf("Photo is required (Ctrl+F to browse)")}
		}
		if m.snapSessionID == "" {
			return nil, &resultMsg{title: "snap", err: fmt.Errorf("no snap session — run 'Init Snap & Solve' first")}
		}
		sessionID := m.snapSessionID
		return func() tea.Msg {
			next, err := m.client.SubmitSnap(sessionID, path)
			if err != nil {
				return resultMsg{title: "snap", err: err}
			}
			return resultMsg{title: "snap", body: next, summary: summarizeNextStep(next)}
		}, nil

	case "respond":
		responseID := m.fieldValue("Response")
		if m.snapSessionID == "" {
			return nil, &resultMsg{title: "respond", err: fmt.Errorf("no snap session — run 'Init Snap & Solve' first")}
		}
		sessionID := m.snapSessionID
		return func() tea.Msg {
			next, err := m.client.SubmitSnapResponse(sessionID, responseID)
			if err != nil {
				return resultMsg{title: "respond", err: err}
			}
			return resultMsg{title: "respond", body: next, summary: summarizeNextStep(next)}
		}, nil

	case "upload":
		path := m.fieldValue("Photo")
		if path == "" {
			return nil, &resultMsg{title: "upload", err: fmt.Errorf("Photo is required (Ctrl+F to browse)")}
		}
		return func() tea.Msg {
			capture, err := m.client.UploadScreenshot(path)
			if err != nil {
				return resultMsg{title: "upload", err: err}
			}
			id := ""
			if capture != nil {
				id = capture.ID
			}
			return resultMsg{title: "upload", body: capture, summary: "capture id: " + id, setLastCaptureID: id}
		}, nil

	case "captures":
		return func() tea.Msg {
			captures, err := m.client.ListCaptures()
			if err != nil {
				return resultMsg{title: "captures", err: err}
			}
			return resultMsg{title: "captures", body: captures, summary: fmt.Sprintf("%d capture(s)", len(captures))}
		}, nil

	case "session":
		return func() tea.Msg {
			sess, err := m.client.CreateSession()
			if err != nil {
				return resultMsg{title: "session", err: err}
			}
			id := ""
			if sess != nil {
				id = sess.SessionID
			}
			return resultMsg{title: "session", body: sess, summary: "session id: " + id, setSolveSessionID: id}
		}, nil

	case "analyze":
		raw := strings.TrimSpace(m.fieldValue("Asset IDs"))
		var assetIDs []string
		if raw != "" {
			for _, id := range strings.Split(raw, ",") {
				if id = strings.TrimSpace(id); id != "" {
					assetIDs = append(assetIDs, id)
				}
			}
		} else if m.lastCaptureID != "" {
			assetIDs = []string{m.lastCaptureID}
		}
		if len(assetIDs) == 0 {
			return nil, &resultMsg{title: "analyze", err: fmt.Errorf("Asset IDs is required (or upload something first)")}
		}
		if m.solveSessionID == "" {
			return nil, &resultMsg{title: "analyze", err: fmt.Errorf("no solve session — run 'Create Session' first")}
		}
		sessionID := m.solveSessionID
		return func() tea.Msg {
			result, err := m.client.AnalyzeAssets(sessionID, assetIDs)
			if err != nil {
				return resultMsg{title: "analyze", err: err}
			}
			return resultMsg{title: "analyze", body: result, summary: summarizeAnalysis(result)}
		}, nil

	case "lesson":
		lessonID := m.fieldValue("Lesson ID")
		if lessonID == "" {
			return nil, &resultMsg{title: "lesson", err: fmt.Errorf("Lesson ID is required")}
		}
		lang := m.fieldValue("Language")
		return func() tea.Msg {
			content, err := m.client.GetLessonContent(lessonID, lang)
			if err != nil {
				return resultMsg{title: "lesson", err: err}
			}
			title := ""
			if content != nil {
				title = content.Title
			}
			return resultMsg{title: "lesson", body: content, summary: title}
		}, nil

	case "devprompt":
		prompt := m.fieldValue("Prompt")
		if prompt == "" {
			return nil, &resultMsg{title: "devprompt", err: fmt.Errorf("Prompt is required")}
		}
		return func() tea.Msg {
			reply, err := m.client.DevPrompt(prompt)
			if err != nil {
				return resultMsg{title: "devprompt", err: err}
			}
			return resultMsg{title: "devprompt", body: reply, summary: fmt.Sprintf("%d chars", len(reply))}
		}, nil

	default:
		return nil, &resultMsg{title: c.ID, err: fmt.Errorf("unknown command %q", c.ID)}
	}
}

func summarizeNextStep(next *api.NextStep) string {
	if next == nil {
		return "(no next step)"
	}
	switch {
	case next.CaptureSnap != nil:
		return "next: capture a snap photo"
	case next.CaptureSnapResponse != nil:
		ids := make([]string, len(next.CaptureSnapResponse.Options))
		for i, o := range next.CaptureSnapResponse.Options {
			ids[i] = o.ID
		}
		return "next: choose a response — " + strings.Join(ids, ", ")
	case next.DisplayAnalysis != nil:
		a := next.DisplayAnalysis
		return fmt.Sprintf("next: display analysis — %d lesson(s) to review, %d problem(s) captured",
			len(a.LessonsToReview), len(a.ProblemsCaptured))
	default:
		return "(empty next step)"
	}
}

func summarizeAnalysis(r *api.AnalyzeAssetsResult) string {
	if r == nil {
		return "(no result)"
	}
	switch {
	case r.Blank != nil:
		return fmt.Sprintf("blank page — %d topic(s) suggested", len(r.Blank.Topics))
	case r.ProblemsFound != nil:
		return fmt.Sprintf("problem(s) found — %d detected", len(r.ProblemsFound.DetectedProblems))
	default:
		return "(empty result)"
	}
}
