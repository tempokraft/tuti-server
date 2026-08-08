// Package analysistest provides a scriptable analysis.Analyzer — a stand-in
// for the vision LLM — for tests that exercise code depending on it (most
// notably internal/httpapi's handlers) without calling a real backend.
package analysistest

import (
	"context"
	"fmt"
	"sync"

	"tuti-server/internal/analysis"
)

// Analyzer is a scriptable analysis.Analyzer. Set ExtractFunc and/or
// EvaluateFunc to whatever result (or error) the test needs the model to
// have produced; the canned results in responses.go (BlankPage,
// SolvedProblem, EvaluatedWithMistake, EvaluatedNoMistakes) are ready to
// return as-is for tests that don't care about the specific content. Every
// call is recorded and available via Calls, for asserting what the code
// under test actually sent. Safe for concurrent use.
type Analyzer struct {
	ExtractFunc  func(ctx context.Context, images []analysis.Image) (analysis.ExtractResult, error)
	EvaluateFunc func(ctx context.Context, images []analysis.Image) (analysis.EvaluateResult, error)

	mu    sync.Mutex
	calls []Call
}

var _ analysis.Analyzer = (*Analyzer)(nil)

// Call records one Extract or Evaluate invocation.
type Call struct {
	Method string // "Extract" or "Evaluate"
	Images []analysis.Image
}

// Calls returns every call made so far, in order.
func (a *Analyzer) Calls() []Call {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]Call(nil), a.calls...)
}

func (a *Analyzer) record(method string, images []analysis.Image) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, Call{Method: method, Images: images})
}

// Extract implements analysis.Analyzer by recording the call and
// delegating to ExtractFunc. It errors if ExtractFunc wasn't set, rather
// than silently returning a zero result, so an unscripted call fails the
// test loudly instead of masking a bug.
func (a *Analyzer) Extract(ctx context.Context, images []analysis.Image) (analysis.ExtractResult, error) {
	a.record("Extract", images)
	if a.ExtractFunc == nil {
		return analysis.ExtractResult{}, fmt.Errorf("analysistest: Analyzer.ExtractFunc not set")
	}
	return a.ExtractFunc(ctx, images)
}

// Evaluate implements analysis.Analyzer by recording the call and
// delegating to EvaluateFunc. See Extract for why an unset func errors.
func (a *Analyzer) Evaluate(ctx context.Context, images []analysis.Image) (analysis.EvaluateResult, error) {
	a.record("Evaluate", images)
	if a.EvaluateFunc == nil {
		return analysis.EvaluateResult{}, fmt.Errorf("analysistest: Analyzer.EvaluateFunc not set")
	}
	return a.EvaluateFunc(ctx, images)
}
