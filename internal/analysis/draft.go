package analysis

import (
	"fmt"

	tutiv1 "tuti-server/internal/genproto/tutiv1"
)

// blockDraft is a single statement/solution block as extracted by the
// model. Only text/math/hint are supported as leaves — graphs, geometry,
// and 3D figures are curated-content-only (see internal/catalog) and are
// deliberately out of scope for model-generated output.
type blockDraft struct {
	Kind       string `json:"kind"` // "text" | "math" | "hint"
	Text       string `json:"text,omitempty"`
	Expression string `json:"expression,omitempty"`
	Hint       string `json:"hint,omitempty"`
}

// Validate checks invariants the JSON schema's "enum"/"required" can't
// enforce on its own — the schema says "kind" is a string, not that it's
// one of our three values, and a provider without strict-mode enforcement
// (see schema.go) may not check the enum server-side at all.
func (b blockDraft) Validate() error {
	switch b.Kind {
	case "text", "math", "hint":
		return nil
	default:
		return fmt.Errorf("block: unknown kind %q", b.Kind)
	}
}

func (b blockDraft) toProto() *tutiv1.ContentBlock {
	switch b.Kind {
	case "math":
		return &tutiv1.ContentBlock{Block: &tutiv1.ContentBlock_Math{Math: &tutiv1.MathBlock{Expression: b.Expression}}}
	case "hint":
		return &tutiv1.ContentBlock{Block: &tutiv1.ContentBlock_Hint{Hint: &tutiv1.HintBlock{Hint: b.Hint}}}
	default:
		return &tutiv1.ContentBlock{Block: &tutiv1.ContentBlock_Text{Text: &tutiv1.TextBlock{Text: b.Text}}}
	}
}

// stepDraft is one labeled step of a solution walk-through.
type stepDraft struct {
	StepNumber int32        `json:"stepNumber"`
	Label      string       `json:"label"`
	Content    []blockDraft `json:"content"`
}

func (s stepDraft) Validate() error {
	if s.Label == "" {
		return fmt.Errorf("step %d: empty label", s.StepNumber)
	}
	for i, b := range s.Content {
		if err := b.Validate(); err != nil {
			return fmt.Errorf("step %d: content[%d]: %w", s.StepNumber, i, err)
		}
	}
	return nil
}

func (s stepDraft) toProto() *tutiv1.ContentBlock {
	content := make([]*tutiv1.ContentBlock, len(s.Content))
	for i, b := range s.Content {
		content[i] = b.toProto()
	}
	return &tutiv1.ContentBlock{Block: &tutiv1.ContentBlock_Step{Step: &tutiv1.StepBlock{
		StepNumber: s.StepNumber,
		Label:      s.Label,
		Content:    content,
	}}}
}

// problemDraft is the model's extraction of a single math problem shown
// in a photo: its statement, hints, and a full worked solution.
type problemDraft struct {
	Title      string       `json:"title"`
	Topic      string       `json:"topic"`
	Difficulty string       `json:"difficulty"` // "easy" | "medium" | "hard"
	Statement  []blockDraft `json:"statement"`
	Hints      []string     `json:"hints"`
	Solution   []stepDraft  `json:"solution"`
}

// Validate rejects the cases toProto() can't safely represent: an unknown
// difficulty (previously silently mapped to DIFFICULTY_LEVEL_UNSPECIFIED),
// or a missing title/statement/solution — a "correct, complete solution"
// with zero steps is a model failure, not a valid answer.
func (p problemDraft) Validate() error {
	if p.Title == "" {
		return fmt.Errorf("problem: empty title")
	}
	switch p.Difficulty {
	case "easy", "medium", "hard":
	default:
		return fmt.Errorf("problem: unknown difficulty %q", p.Difficulty)
	}
	if len(p.Statement) == 0 {
		return fmt.Errorf("problem: empty statement")
	}
	if len(p.Solution) == 0 {
		return fmt.Errorf("problem: empty solution")
	}
	for i, b := range p.Statement {
		if err := b.Validate(); err != nil {
			return fmt.Errorf("problem: statement[%d]: %w", i, err)
		}
	}
	for i, s := range p.Solution {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("problem: solution[%d]: %w", i, err)
		}
	}
	return nil
}

func (p problemDraft) toProto(id string) *tutiv1.Problem {
	statement := make([]*tutiv1.ContentBlock, len(p.Statement))
	for i, b := range p.Statement {
		statement[i] = b.toProto()
	}
	solution := make([]*tutiv1.ContentBlock, len(p.Solution))
	for i, s := range p.Solution {
		solution[i] = s.toProto()
	}
	return &tutiv1.Problem{
		Id:         id,
		Title:      p.Title,
		Topic:      p.Topic,
		Difficulty: difficultyToProto(p.Difficulty),
		Statement:  statement,
		Hints:      p.Hints,
		Solution:   solution,
	}
}

func difficultyToProto(d string) tutiv1.DifficultyLevel {
	switch d {
	case "easy":
		return tutiv1.DifficultyLevel_DIFFICULTY_LEVEL_EASY
	case "medium":
		return tutiv1.DifficultyLevel_DIFFICULTY_LEVEL_MEDIUM
	case "hard":
		return tutiv1.DifficultyLevel_DIFFICULTY_LEVEL_HARD
	default:
		return tutiv1.DifficultyLevel_DIFFICULTY_LEVEL_UNSPECIFIED
	}
}

// mistakeDraft is a single error the model found in the student's shown
// work.
type mistakeDraft struct {
	Description   string `json:"description"`
	StepReference string `json:"stepReference,omitempty"`
}

func (m mistakeDraft) Validate() error {
	if m.Description == "" {
		return fmt.Errorf("mistake: empty description")
	}
	return nil
}

func (m mistakeDraft) toProto(id string) *tutiv1.Mistake {
	return &tutiv1.Mistake{Id: id, Description: m.Description, StepReference: m.StepReference}
}

// extractOutput is the forced tool-call shape for Extract.
type extractOutput struct {
	Blank   bool          `json:"blank"`
	Problem *problemDraft `json:"problem,omitempty"`
}

func (o extractOutput) Validate() error {
	if o.Blank {
		return nil
	}
	if o.Problem == nil {
		return fmt.Errorf("model reported blank=false with no problem")
	}
	return o.Problem.Validate()
}

// evaluateOutput is the forced tool-call shape for Evaluate.
type evaluateOutput struct {
	Problem  problemDraft   `json:"problem"`
	Mistakes []mistakeDraft `json:"mistakes"`
}

func (o evaluateOutput) Validate() error {
	if err := o.Problem.Validate(); err != nil {
		return err
	}
	for i, m := range o.Mistakes {
		if err := m.Validate(); err != nil {
			return fmt.Errorf("mistakes[%d]: %w", i, err)
		}
	}
	return nil
}
