package analysistest

import (
	"tuti-server/internal/analysis"
	tutiv1 "tuti-server/internal/genproto/tutiv1"
)

// Canned analysis results, for tests that need a plausible "the model
// looked at the photo" response without hand-building a Problem literal
// every time. Each is a function, not a package var: analysis results are
// proto messages (pointers), and returning a fresh copy per call keeps one
// test's mutation from bleeding into another's.

// BlankPage is a canned ExtractResult for a photo with no attempted work.
func BlankPage() analysis.ExtractResult {
	return analysis.ExtractResult{Blank: true}
}

// SolvedProblem is a canned ExtractResult for a fully worked linear
// equation, for tests that need a non-blank Extract response but don't
// care about the specific content.
func SolvedProblem() analysis.ExtractResult {
	return analysis.ExtractResult{
		Blank:   false,
		Problem: linearEquationProblem(),
	}
}

// EvaluatedWithMistake is a canned EvaluateResult: the same problem as
// SolvedProblem, plus one mistake in the student's shown work.
func EvaluatedWithMistake() analysis.EvaluateResult {
	return analysis.EvaluateResult{
		Problem: linearEquationProblem(),
		Mistakes: []*tutiv1.Mistake{
			{
				Id:            "mock_m1",
				Description:   "Subtracted 3 from both sides instead of adding, flipping the sign on the constant term.",
				StepReference: "Step 1",
			},
		},
	}
}

// EvaluatedNoMistakes is a canned EvaluateResult for shown work that's
// fully correct.
func EvaluatedNoMistakes() analysis.EvaluateResult {
	return analysis.EvaluateResult{
		Problem:  linearEquationProblem(),
		Mistakes: nil,
	}
}

func linearEquationProblem() *tutiv1.Problem {
	return &tutiv1.Problem{
		Id:         "mock_problem",
		Title:      "Solve for x",
		Topic:      "Algebra",
		Difficulty: tutiv1.DifficultyLevel_DIFFICULTY_LEVEL_EASY,
		Statement: []*tutiv1.ContentBlock{
			textBlock("Solve the equation for x:"),
			mathBlock("2x + 3 = 11"),
		},
		Hints: []string{
			"Start by isolating the term with x.",
		},
		Solution: []*tutiv1.ContentBlock{
			stepBlock(1, "Subtract 3 from both sides", mathBlock("2x = 8")),
			stepBlock(2, "Divide both sides by 2", mathBlock("x = 4")),
		},
	}
}

func textBlock(text string) *tutiv1.ContentBlock {
	return &tutiv1.ContentBlock{Block: &tutiv1.ContentBlock_Text{Text: &tutiv1.TextBlock{Text: text}}}
}

func mathBlock(expression string) *tutiv1.ContentBlock {
	return &tutiv1.ContentBlock{Block: &tutiv1.ContentBlock_Math{Math: &tutiv1.MathBlock{Expression: expression}}}
}

func stepBlock(number int32, label string, content ...*tutiv1.ContentBlock) *tutiv1.ContentBlock {
	return &tutiv1.ContentBlock{Block: &tutiv1.ContentBlock_Step{Step: &tutiv1.StepBlock{
		StepNumber: number,
		Label:      label,
		Content:    content,
	}}}
}
