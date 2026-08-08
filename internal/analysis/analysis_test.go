package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func testImages() []Image {
	return []Image{{Bytes: []byte("fake-jpeg"), ContentType: "image/jpeg"}}
}

func TestExtract_Blank(t *testing.T) {
	mock := &mockProvider{Func: func(context.Context, structuredCallRequest) (json.RawMessage, error) {
		return json.RawMessage(`{"blank": true}`), nil
	}}
	a := &llmAnalyzer{backend: mock}

	result, err := a.Extract(context.Background(), testImages())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if !result.Blank || result.Problem != nil {
		t.Fatalf("Extract: want blank result with no problem, got %+v", result)
	}

	if len(mock.Requests) != 1 {
		t.Fatalf("want 1 request, got %d", len(mock.Requests))
	}
	req := mock.Requests[0]
	if req.ToolName != extractToolName {
		t.Errorf("ToolName = %q, want %q", req.ToolName, extractToolName)
	}
	if req.System != systemPrompt {
		t.Errorf("System prompt not passed through")
	}
	if req.Prompt != extractPrompt {
		t.Errorf("Prompt = %q, want extractPrompt", req.Prompt)
	}
	if req.MaxTokens != defaultMaxTokens {
		t.Errorf("MaxTokens = %d, want %d", req.MaxTokens, defaultMaxTokens)
	}
}

func TestExtract_Problem(t *testing.T) {
	const raw = `{
		"blank": false,
		"problem": {
			"title": "Solve for x",
			"topic": "Algebra",
			"difficulty": "easy",
			"statement": [{"kind": "text", "text": "Solve:"}, {"kind": "math", "expression": "2x+3=11"}],
			"hints": ["Isolate x"],
			"solution": [
				{"stepNumber": 1, "label": "Subtract 3", "content": [{"kind": "math", "expression": "2x=8"}]},
				{"stepNumber": 2, "label": "Divide by 2", "content": [{"kind": "math", "expression": "x=4"}]}
			]
		}
	}`
	mock := &mockProvider{Func: func(context.Context, structuredCallRequest) (json.RawMessage, error) {
		return json.RawMessage(raw), nil
	}}
	a := &llmAnalyzer{backend: mock}

	result, err := a.Extract(context.Background(), testImages())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if result.Blank {
		t.Fatal("Extract: want non-blank result")
	}
	if result.Problem == nil || result.Problem.Title != "Solve for x" {
		t.Fatalf("Extract: unexpected problem %+v", result.Problem)
	}
	if len(result.Problem.Solution) != 2 {
		t.Fatalf("Extract: want 2 solution steps, got %d", len(result.Problem.Solution))
	}
	if !strings.HasPrefix(result.Problem.Id, "detected_") {
		t.Errorf("Problem.Id = %q, want detected_ prefix", result.Problem.Id)
	}
}

func TestExtract_BlankFalseWithNoProblem(t *testing.T) {
	mock := &mockProvider{Func: func(context.Context, structuredCallRequest) (json.RawMessage, error) {
		return json.RawMessage(`{"blank": false}`), nil
	}}
	a := &llmAnalyzer{backend: mock}

	if _, err := a.Extract(context.Background(), testImages()); err == nil {
		t.Fatal("want error when blank=false and problem is missing")
	}
}

func TestExtract_UnknownDifficultyRejected(t *testing.T) {
	const raw = `{"blank": false, "problem": {
		"title": "T", "topic": "Algebra", "difficulty": "impossible",
		"statement": [{"kind":"text","text":"x"}], "hints": [],
		"solution": [{"stepNumber":1,"label":"L","content":[{"kind":"text","text":"y"}]}]
	}}`
	mock := &mockProvider{Func: func(context.Context, structuredCallRequest) (json.RawMessage, error) {
		return json.RawMessage(raw), nil
	}}
	a := &llmAnalyzer{backend: mock}

	if _, err := a.Extract(context.Background(), testImages()); err == nil {
		t.Fatal("want error for unknown difficulty enum value")
	}
}

func TestExtract_UnknownBlockKindRejected(t *testing.T) {
	const raw = `{"blank": false, "problem": {
		"title": "T", "topic": "Algebra", "difficulty": "easy",
		"statement": [{"kind":"diagram","text":"x"}], "hints": [],
		"solution": [{"stepNumber":1,"label":"L","content":[{"kind":"text","text":"y"}]}]
	}}`
	mock := &mockProvider{Func: func(context.Context, structuredCallRequest) (json.RawMessage, error) {
		return json.RawMessage(raw), nil
	}}
	a := &llmAnalyzer{backend: mock}

	if _, err := a.Extract(context.Background(), testImages()); err == nil {
		t.Fatal("want error for unknown block kind")
	}
}

func TestExtract_MalformedJSON(t *testing.T) {
	mock := &mockProvider{Func: func(context.Context, structuredCallRequest) (json.RawMessage, error) {
		return json.RawMessage(`{not valid json`), nil
	}}
	a := &llmAnalyzer{backend: mock}

	if _, err := a.Extract(context.Background(), testImages()); err == nil {
		t.Fatal("want decode error for malformed JSON")
	}
}

func TestExtract_ProviderError(t *testing.T) {
	wantErr := errors.New("boom")
	mock := &mockProvider{Func: func(context.Context, structuredCallRequest) (json.RawMessage, error) {
		return nil, wantErr
	}}
	a := &llmAnalyzer{backend: mock}

	if _, err := a.Extract(context.Background(), testImages()); !errors.Is(err, wantErr) {
		t.Fatalf("want wrapped provider error, got %v", err)
	}
}

func TestEvaluate_WithMistakes(t *testing.T) {
	const raw = `{
		"problem": {
			"title": "Solve for x", "topic": "Algebra", "difficulty": "medium",
			"statement": [{"kind":"math","expression":"2x+3=11"}],
			"hints": [],
			"solution": [{"stepNumber":1,"label":"Subtract 3","content":[{"kind":"math","expression":"2x=8"}]}]
		},
		"mistakes": [
			{"description": "Added instead of subtracted", "stepReference": "Step 1"},
			{"description": "Arithmetic slip"}
		]
	}`
	mock := &mockProvider{Func: func(context.Context, structuredCallRequest) (json.RawMessage, error) {
		return json.RawMessage(raw), nil
	}}
	a := &llmAnalyzer{backend: mock}

	result, err := a.Evaluate(context.Background(), testImages())
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(result.Mistakes) != 2 {
		t.Fatalf("want 2 mistakes, got %d", len(result.Mistakes))
	}
	if result.Mistakes[0].Id != result.Problem.Id+"_m1" || result.Mistakes[1].Id != result.Problem.Id+"_m2" {
		t.Errorf("mistake IDs not derived from problem ID: %+v", result.Mistakes)
	}

	if req := mock.Requests[0]; req.ToolName != evaluateToolName {
		t.Errorf("ToolName = %q, want %q", req.ToolName, evaluateToolName)
	}
}

func TestEvaluate_EmptyMistakeDescriptionRejected(t *testing.T) {
	const raw = `{
		"problem": {
			"title": "T", "topic": "Algebra", "difficulty": "easy",
			"statement": [{"kind":"text","text":"x"}], "hints": [],
			"solution": [{"stepNumber":1,"label":"L","content":[{"kind":"text","text":"y"}]}]
		},
		"mistakes": [{"description": ""}]
	}`
	mock := &mockProvider{Func: func(context.Context, structuredCallRequest) (json.RawMessage, error) {
		return json.RawMessage(raw), nil
	}}
	a := &llmAnalyzer{backend: mock}

	if _, err := a.Evaluate(context.Background(), testImages()); err == nil {
		t.Fatal("want error for empty mistake description")
	}
}
