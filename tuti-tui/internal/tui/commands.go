package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tuti-tui/internal/api"
)

// ── messages ─────────────────────────────────────────────────────────────

type healthResultMsg struct{ err error }

type resultMsg struct {
	// title is echoed to the transcript before the result (or error).
	title string
	err   error
	// summary is an optional human-readable line shown above the raw
	// JSON dump — used for NextStep results, where "here's what happens
	// next" matters more than the wire shape.
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

// ── dispatch ─────────────────────────────────────────────────────────────

const helpText = `Commands:
  health                         recheck server health
  init                           start a Snap & Solve session
  snap <path>                    upload a photo for the current snap session
  respond <check_work|solve|explain>
                                  submit the student's chosen action
  upload <path>                  upload a screenshot (standalone, not the snap flow)
  captures                       list uploaded captures
  session                        create a solve session
  analyze [id1,id2,...]          analyze captures (defaults to the last upload)
  lesson <id> [lang]             fetch lesson content (lang defaults to en)
  clear                          clear the transcript
  help                           show this message
  quit / ctrl+c                  exit

Context carried between commands: snap session (from 'init'), solve session
(from 'session'), and last uploaded capture id — shown in the status bar.`

// dispatch parses a command line and returns the tea.Cmd that runs it, or
// nil plus a message to show immediately (help/clear/unknown/quit) when no
// network call is needed.
func (m Model) dispatch(line string) (tea.Cmd, *resultMsg, bool /* quit */) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil, nil, false
	}
	cmd, args := strings.ToLower(fields[0]), fields[1:]

	switch cmd {
	case "quit", "exit":
		return nil, nil, true

	case "help", "?":
		return nil, &resultMsg{title: "help", body: nil, summary: helpText}, false

	case "clear":
		return nil, &resultMsg{title: "__clear__"}, false

	case "health":
		return func() tea.Msg {
			err := m.client.Health()
			return resultMsg{title: "health", err: err, summary: "ok"}
		}, nil, false

	case "init":
		return func() tea.Msg {
			id, next, err := m.client.InitializeSnapAndSolve()
			if err != nil {
				return resultMsg{title: "init", err: err}
			}
			return resultMsg{title: "init", body: next, summary: summarizeNextStep(next), setSnapSessionID: id}
		}, nil, false

	case "snap":
		if len(args) < 1 {
			return nil, &resultMsg{title: "snap", err: fmt.Errorf("usage: snap <path>")}, false
		}
		if m.snapSessionID == "" {
			return nil, &resultMsg{title: "snap", err: fmt.Errorf("no snap session — run 'init' first")}, false
		}
		path, sessionID := args[0], m.snapSessionID
		return func() tea.Msg {
			next, err := m.client.SubmitSnap(sessionID, path)
			if err != nil {
				return resultMsg{title: "snap", err: err}
			}
			return resultMsg{title: "snap", body: next, summary: summarizeNextStep(next)}
		}, nil, false

	case "respond":
		if len(args) < 1 {
			return nil, &resultMsg{title: "respond", err: fmt.Errorf("usage: respond <check_work|solve|explain>")}, false
		}
		if m.snapSessionID == "" {
			return nil, &resultMsg{title: "respond", err: fmt.Errorf("no snap session — run 'init' first")}, false
		}
		responseID, sessionID := args[0], m.snapSessionID
		return func() tea.Msg {
			next, err := m.client.SubmitSnapResponse(sessionID, responseID)
			if err != nil {
				return resultMsg{title: "respond", err: err}
			}
			return resultMsg{title: "respond", body: next, summary: summarizeNextStep(next)}
		}, nil, false

	case "upload":
		if len(args) < 1 {
			return nil, &resultMsg{title: "upload", err: fmt.Errorf("usage: upload <path>")}, false
		}
		path := args[0]
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
		}, nil, false

	case "captures":
		return func() tea.Msg {
			captures, err := m.client.ListCaptures()
			if err != nil {
				return resultMsg{title: "captures", err: err}
			}
			return resultMsg{title: "captures", body: captures, summary: fmt.Sprintf("%d capture(s)", len(captures))}
		}, nil, false

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
		}, nil, false

	case "analyze":
		assetIDs := args
		if len(assetIDs) == 0 && m.lastCaptureID != "" {
			assetIDs = []string{m.lastCaptureID}
		}
		if len(assetIDs) == 0 {
			return nil, &resultMsg{title: "analyze", err: fmt.Errorf("usage: analyze <id1,id2,...> (or upload something first)")}, false
		}
		if m.solveSessionID == "" {
			return nil, &resultMsg{title: "analyze", err: fmt.Errorf("no solve session — run 'session' first")}, false
		}
		// allow comma-separated ids in a single arg too
		if len(assetIDs) == 1 {
			assetIDs = strings.Split(assetIDs[0], ",")
		}
		sessionID := m.solveSessionID
		return func() tea.Msg {
			result, err := m.client.AnalyzeAssets(sessionID, assetIDs)
			if err != nil {
				return resultMsg{title: "analyze", err: err}
			}
			return resultMsg{title: "analyze", body: result, summary: summarizeAnalysis(result)}
		}, nil, false

	case "lesson":
		if len(args) < 1 {
			return nil, &resultMsg{title: "lesson", err: fmt.Errorf("usage: lesson <id> [lang]")}, false
		}
		lessonID := args[0]
		lang := ""
		if len(args) > 1 {
			lang = args[1]
		}
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
		}, nil, false

	default:
		return nil, &resultMsg{title: cmd, err: fmt.Errorf("unknown command %q — try 'help'", cmd)}, false
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
