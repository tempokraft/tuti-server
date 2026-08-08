package api

// These types mirror tuti/proto/tuti_service.proto's JSON (protojson)
// encoding by hand, since this module can't import tuti-server's generated
// Go types (internal/, and a separate module besides). A few conventions
// to keep in mind when extending this file:
//
//   - proto `bytes` fields (e.g. Data below) are Go []byte, which
//     encoding/json already base64-encodes/decodes — matching protojson.
//   - proto `int64` fields (timestamps) are protojson-encoded as JSON
//     strings, so they're plain Go `string` here, not int64.
//   - a proto `oneof` (NextStep.step, ContentBlock.block,
//     AnalyzeAssetsResponse.result) is protojson-encoded as a flat object
//     with exactly one variant field set — represented here as a struct
//     with every variant as an `omitempty` pointer field.

// Capture is a stored screenshot/photo.
type Capture struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	Data         []byte `json:"data,omitempty"`
	UploadedAtMs string `json:"uploadedAtMs,omitempty"`
	URL          string `json:"url,omitempty"`
}

// SolveSession is returned by CreateSession.
type SolveSession struct {
	SessionID   string   `json:"sessionId,omitempty"`
	AssetIDs    []string `json:"assetIds,omitempty"`
	CreatedAtMs string   `json:"createdAtMs,omitempty"`
}

// SnapOption is one Snap & Solve action offered after a photo is submitted.
type SnapOption struct {
	ID          string `json:"id,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	IconName    string `json:"iconName,omitempty"`
}

// LessonReference points at a lesson worth reviewing.
type LessonReference struct {
	LessonID    string `json:"lessonId,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	IconName    string `json:"iconName,omitempty"`
}

// Mistake is a single error found in a student's shown work.
type Mistake struct {
	ID                string              `json:"id,omitempty"`
	Description       string              `json:"description,omitempty"`
	StepReference     string              `json:"stepReference,omitempty"`
	SocraticQuestions []*SocraticQuestion `json:"socraticQuestions,omitempty"`
}

// SocraticQuestion probes whether the student understands why a mistake
// was a mistake, rather than just stating the fix.
type SocraticQuestion struct {
	ID          string            `json:"id,omitempty"`
	Question    string            `json:"question,omitempty"`
	Options     []*QuestionOption `json:"options,omitempty"`
	Explanation string            `json:"explanation,omitempty"` // shown after the student answers
}

type QuestionOption struct {
	ID        string `json:"id,omitempty"`
	Text      string `json:"text,omitempty"`
	IsCorrect bool   `json:"isCorrect,omitempty"`
}

// TopicRecommendation is shown when a captured page is blank.
type TopicRecommendation struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	LessonID    string `json:"lessonId,omitempty"`
	IconName    string `json:"iconName,omitempty"`
}

// Problem is a math problem: statement, hints, and a worked solution.
type Problem struct {
	ID         string          `json:"id,omitempty"`
	Title      string          `json:"title,omitempty"`
	Topic      string          `json:"topic,omitempty"`
	Difficulty string          `json:"difficulty,omitempty"` // enum name, e.g. "DIFFICULTY_LEVEL_EASY"
	Statement  []*ContentBlock `json:"statement,omitempty"`
	Hints      []string        `json:"hints,omitempty"`
	Solution   []*ContentBlock `json:"solution,omitempty"`
}

// EvaluatedProblem pairs a Problem with mistakes found in a student's
// attempt and problems similar to it.
type EvaluatedProblem struct {
	Problem         *Problem   `json:"problem,omitempty"`
	Mistakes        []*Mistake `json:"mistakes,omitempty"`
	SimilarProblems []*Problem `json:"similarProblems,omitempty"`
}

// ContentBlock is a oneof: exactly one field is set. Graph2D/Geometry/
// Figure3D only carry a title here — the TUI can't render shapes, so it
// shows a placeholder rather than modeling every nested field.
type ContentBlock struct {
	Text     *TextBlock   `json:"text,omitempty"`
	Math     *MathBlock   `json:"math,omitempty"`
	Graph2D  *TitledBlock `json:"graph2d,omitempty"`
	Geometry *TitledBlock `json:"geometry,omitempty"`
	Figure3D *TitledBlock `json:"figure3d,omitempty"`
	Step     *StepBlock   `json:"step,omitempty"`
	Hint     *HintBlock   `json:"hint,omitempty"`
}

type TextBlock struct {
	Text   string `json:"text,omitempty"`
	Bold   bool   `json:"bold,omitempty"`
	Italic bool   `json:"italic,omitempty"`
}

type MathBlock struct {
	Expression string `json:"expression,omitempty"`
	Inline     bool   `json:"inline,omitempty"`
}

type TitledBlock struct {
	Title string `json:"title,omitempty"`
}

type StepBlock struct {
	StepNumber int32           `json:"stepNumber,omitempty"`
	Label      string          `json:"label,omitempty"`
	Content    []*ContentBlock `json:"content,omitempty"`
}

type HintBlock struct {
	Hint string `json:"hint,omitempty"`
}

// LessonSection is one heading + content of a lesson.
type LessonSection struct {
	Heading string          `json:"heading,omitempty"`
	Content []*ContentBlock `json:"content,omitempty"`
}

// LessonContent is the full body of a lesson in one language.
type LessonContent struct {
	ID            string           `json:"id,omitempty"`
	Title         string           `json:"title,omitempty"`
	Topic         string           `json:"topic,omitempty"`
	IconName      string           `json:"iconName,omitempty"`
	Language      string           `json:"language,omitempty"`
	Context       []*ContentBlock  `json:"context,omitempty"`
	Sections      []*LessonSection `json:"sections,omitempty"`
	WorkedExample *LessonSection   `json:"workedExample,omitempty"`
}

// NextStep is a oneof: exactly one field is set.
type NextStep struct {
	CaptureSnap         *CaptureSnapStep         `json:"captureSnap,omitempty"`
	CaptureSnapResponse *CaptureSnapResponseStep `json:"captureSnapResponse,omitempty"`
	DisplayAnalysis     *DisplayAnalysisStep     `json:"displayAnalysis,omitempty"`
}

type CaptureSnapStep struct{}

type CaptureSnapResponseStep struct {
	Options []*SnapOption `json:"options,omitempty"`
}

type DisplayAnalysisStep struct {
	LessonsToReview  []*LessonReference  `json:"lessonsToReview,omitempty"`
	ProblemsCaptured []*EvaluatedProblem `json:"problemsCaptured,omitempty"`
}

// ── request/response envelopes, one pair per RPC ────────────────────────────

type uploadScreenshotRequest struct {
	Data     []byte `json:"data,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type uploadScreenshotResponse struct {
	Capture *Capture `json:"capture,omitempty"`
}

type listCapturesResponse struct {
	Captures []*Capture `json:"captures,omitempty"`
}

type createSessionResponse struct {
	Session *SolveSession `json:"session,omitempty"`
}

type analyzeAssetsRequest struct {
	SessionID string   `json:"sessionId,omitempty"`
	AssetIDs  []string `json:"assetIds,omitempty"`
}

// AnalyzeAssetsResult is a oneof: exactly one field is set.
type AnalyzeAssetsResult struct {
	Blank         *BlankProblemsResult `json:"blank,omitempty"`
	ProblemsFound *ProblemsFoundResult `json:"problemsFound,omitempty"`
}

type BlankProblemsResult struct {
	Topics          []*TopicRecommendation `json:"topics,omitempty"`
	SimilarProblems []*Problem             `json:"similarProblems,omitempty"`
}

type ProblemsFoundResult struct {
	DetectedProblems []*Problem `json:"detectedProblems,omitempty"`
	SimilarProblems  []*Problem `json:"similarProblems,omitempty"`
}

type initializeSnapAndSolveResponse struct {
	SessionID string    `json:"sessionId,omitempty"`
	NextStep  *NextStep `json:"nextStep,omitempty"`
}

type submitSnapRequest struct {
	SessionID string `json:"sessionId,omitempty"`
	Data      []byte `json:"data,omitempty"`
	Filename  string `json:"filename,omitempty"`
}

type submitSnapResult struct {
	NextStep *NextStep `json:"nextStep,omitempty"`
}

type submitSnapResponseRequest struct {
	SessionID  string `json:"sessionId,omitempty"`
	ResponseID string `json:"responseId,omitempty"`
}

type submitSnapResponseResult struct {
	NextStep *NextStep `json:"nextStep,omitempty"`
}

type getLessonContentRequest struct {
	LessonID string `json:"lessonId,omitempty"`
	Language string `json:"language,omitempty"`
}

type getLessonContentResponse struct {
	Content *LessonContent `json:"content,omitempty"`
}
