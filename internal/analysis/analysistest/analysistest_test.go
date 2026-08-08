package analysistest

import (
	"context"
	"testing"

	"tuti-server/internal/analysis"
)

func TestAnalyzer_ScriptedResponses(t *testing.T) {
	mock := &Analyzer{
		ExtractFunc: func(ctx context.Context, images []analysis.Image) (analysis.ExtractResult, error) {
			return SolvedProblem(), nil
		},
		EvaluateFunc: func(ctx context.Context, images []analysis.Image) (analysis.EvaluateResult, error) {
			return EvaluatedWithMistake(), nil
		},
	}

	img := []analysis.Image{{Bytes: []byte("fake-jpeg"), ContentType: "image/jpeg"}}

	extract, err := mock.Extract(context.Background(), img)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if extract.Blank || extract.Problem == nil || extract.Problem.Title != "Solve for x" {
		t.Fatalf("Extract: unexpected result %+v", extract)
	}

	evaluate, err := mock.Evaluate(context.Background(), img)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(evaluate.Mistakes) != 1 {
		t.Fatalf("Evaluate: want 1 mistake, got %d", len(evaluate.Mistakes))
	}

	calls := mock.Calls()
	if len(calls) != 2 || calls[0].Method != "Extract" || calls[1].Method != "Evaluate" {
		t.Fatalf("Calls: unexpected recording %+v", calls)
	}
}

func TestAnalyzer_UnscriptedCallErrors(t *testing.T) {
	mock := &Analyzer{}
	if _, err := mock.Extract(context.Background(), nil); err == nil {
		t.Fatal("Extract: want error when ExtractFunc is unset, got nil")
	}
}
