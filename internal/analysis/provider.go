package analysis

import (
	"context"
	"encoding/json"
)

// provider is the seam between backend-agnostic orchestration (prompts,
// schema, decoding, validation — all in analysis.go/prompts.go/schema.go)
// and a specific LLM API. An implementation's only job is: force the named
// tool/function call, with these images and this prompt, and hand back
// whatever JSON arguments the model produced for it. Everything about what
// to ask for and what to do with the answer lives above this interface,
// not inside it — that's what lets Extract/Evaluate be written once and
// reused unchanged across backends.
type provider interface {
	callStructured(ctx context.Context, req structuredCallRequest) (json.RawMessage, error)
	callText(ctx context.Context, prompt string, maxTokens int64) (string, error)
}

// structuredCallRequest describes one forced structured call: send this
// system/user prompt and these images, and require the model to respond
// by calling ToolName with input matching the given schema.
type structuredCallRequest struct {
	System string
	Prompt string
	Images []Image

	ToolName    string
	Description string
	Properties  map[string]any
	Required    []string

	MaxTokens int64
}
