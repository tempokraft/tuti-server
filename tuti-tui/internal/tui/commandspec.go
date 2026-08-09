package tui

// paramKind distinguishes how a command parameter is entered. paramFile
// additionally means Ctrl+F opens the file picker for it.
type paramKind int

const (
	paramText paramKind = iota
	paramFile
	paramChoice
)

// paramSpec describes one input a command needs.
type paramSpec struct {
	Name        string
	Kind        paramKind
	Placeholder string
	Default     string
	Choices     []string // paramChoice only
	Optional    bool
}

// commandSpec is one entry in the left-hand menu. It implements
// list.DefaultItem (Title/Description/FilterValue) so it can be used
// directly as a bubbles/list item.
type commandSpec struct {
	ID     string
	Name   string
	Desc   string
	Params []paramSpec
}

func (c commandSpec) Title() string       { return c.Name }
func (c commandSpec) Description() string { return c.Desc }
func (c commandSpec) FilterValue() string { return c.Name }

// commandCatalog is the full set of RPCs (plus a couple of local
// utilities) exposed in the menu, in display order.
var commandCatalog = []commandSpec{
	{ID: "health", Name: "Health Check", Desc: "GET /healthz"},
	{ID: "init", Name: "Init Snap & Solve", Desc: "Start a new Snap & Solve session"},
	{ID: "snap", Name: "Submit Snap", Desc: "Upload a photo for the current snap session", Params: []paramSpec{
		{Name: "Photo", Kind: paramFile, Placeholder: "path to a photo — Ctrl+F to browse"},
	}},
	{ID: "respond", Name: "Submit Response", Desc: "Submit the student's chosen action", Params: []paramSpec{
		{Name: "Response", Kind: paramChoice, Choices: []string{"check_work", "solve", "explain"}},
	}},
	{ID: "upload", Name: "Upload Screenshot", Desc: "Upload a screenshot (standalone)", Params: []paramSpec{
		{Name: "Photo", Kind: paramFile, Placeholder: "path to a screenshot — Ctrl+F to browse"},
	}},
	{ID: "captures", Name: "List Captures", Desc: "List uploaded captures, most recent first"},
	{ID: "session", Name: "Create Session", Desc: "Open a new solve session"},
	{ID: "analyze", Name: "Analyze Assets", Desc: "Analyze captures attached to a session", Params: []paramSpec{
		{Name: "Asset IDs", Kind: paramText, Placeholder: "comma-separated, defaults to last capture", Optional: true},
	}},
	{ID: "lesson", Name: "Get Lesson", Desc: "Fetch lesson content", Params: []paramSpec{
		{Name: "Lesson ID", Kind: paramText, Placeholder: "e.g. linear_equations"},
		{Name: "Language", Kind: paramText, Placeholder: "en", Default: "en", Optional: true},
	}},
	{ID: "devprompt", Name: "Dev Prompt", Desc: "Send a raw prompt to the LLM (requires DEV_MODE=true on server)", Params: []paramSpec{
		{Name: "Prompt", Kind: paramText, Placeholder: "enter your prompt"},
	}},
	{ID: "clear", Name: "Clear Log", Desc: "Clear the results log"},
}
