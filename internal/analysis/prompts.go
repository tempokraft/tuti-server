package analysis

// This file is the single place that owns everything the vision model
// reads as instructions: the system prompt, the per-call task prompts, the
// tool/function identity shown to the model, and every JSON-schema field
// description in schema.go (a "description" in a tool's input schema is
// prompt text too — the model reads it when deciding what to put in that
// field). Keeping it all here means tuning model behavior is a one-file
// edit, and both backends in provider_*.go automatically stay in sync
// since they both build their request from these same strings — nothing
// about wording is duplicated per provider.

// systemPrompt sets the model's role and ground rules. Shared by every
// call this package makes, regardless of backend.
const systemPrompt = `You are a math tutor's vision system. You are shown one or more photos of a
student's notebook or worksheet page and must call the provided tool with
your structured findings — never respond in plain text.

Guidelines:
- Statements and solutions must use only "text", "math", and "hint" blocks.
  Do not invent graphs, diagrams, or geometry — describe shapes in words if
  needed.
- Math expressions are LaTeX, without surrounding $ delimiters.
- Solutions must be fully correct and show real work, not just the answer.
- Topic should be one of: Algebra, Geometry, Calculus, Statistics (or the
  closest fit).
- Be specific and concrete. Vague filler ("check your work", "review the
  concept") is not acceptable for a mistake description.`

// Task prompts: the user-turn instruction for each call.
const (
	extractPrompt = "Look at the photo(s). Is the page blank, or is there a math problem written on it? " +
		"If there's a problem, extract it in full (statement, hints, and a complete correct solution)."

	evaluatePrompt = "Look at the photo(s). Extract the math problem being solved, then evaluate the student's " +
		"shown attempt: find every mistake (or none, if the work is fully correct)."
)

// Tool identity: the name the model calls and a description of what it's
// for. Tool choice is forced (the model has no choice but to call it), but
// a clear description still measurably improves the quality of the
// arguments the model fills in.
const (
	extractToolName        = "extract_problem"
	extractToolDescription = "Report whether the photographed page is blank or contains a written math " +
		"problem, and if so extract the full problem statement and a correct step-by-step solution."

	evaluateToolName        = "evaluate_work"
	evaluateToolDescription = "Report the math problem shown in the photo(s) and every mistake found in the " +
		"student's attempted solution."
)

// Field descriptions used by the schema builders in schema.go. Wording
// changes here directly change what the model produces, so tune them here
// rather than inline in the schema-building code.
const (
	descBlank = "True if the page is blank / has no attempted math problem on it."

	descBlock         = "A single piece of statement/solution content. Only 'kind' plus the matching field should be set."
	descBlockText     = "Plain prose. Set when kind=text."
	descBlockMath     = "A LaTeX math expression, no surrounding $ delimiters. Set when kind=math."
	descBlockHint     = "A short nudge, not a full answer. Set when kind=hint."
	descStepLabel     = "Short label for this step, e.g. 'Solve for x'."
	descProblemTopic  = "One of: Algebra, Geometry, Calculus, Statistics (or the closest fit)."
	descStatement     = "The problem as written, as text/math blocks."
	descSolution      = "A full, correct, step-by-step solution."
	descMistake       = "Specific, concrete description of what went wrong."
	descStepReference = "Which step of the student's work this refers to, e.g. 'Step 2'. Empty if not tied to a specific step."
	descMistakesList  = "Every mistake found in the student's shown work. Empty array if the work is fully correct."
)
