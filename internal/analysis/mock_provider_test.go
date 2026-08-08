package analysis

import (
	"context"
	"encoding/json"
	"fmt"
)

// mockProvider is a scriptable provider for unit-testing llmAnalyzer's
// orchestration — prompt/tool selection, JSON decoding, Validate() — with
// no real backend involved. Set Func to control the raw JSON (or error) a
// call returns. Every request llmAnalyzer built is recorded in Requests
// so a test can assert on it: right tool name, right prompt, images
// passed through unchanged, etc.
type mockProvider struct {
	Func func(ctx context.Context, req structuredCallRequest) (json.RawMessage, error)

	Requests []structuredCallRequest
}

func (m *mockProvider) callStructured(ctx context.Context, req structuredCallRequest) (json.RawMessage, error) {
	m.Requests = append(m.Requests, req)
	if m.Func == nil {
		return nil, fmt.Errorf("mockProvider: Func not set")
	}
	return m.Func(ctx, req)
}
