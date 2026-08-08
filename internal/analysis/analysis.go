// Package analysis is the only place tuti-server asks an LLM to look at a
// photo and make a judgment call. Everything else about a Snap & Solve
// response (similar problems, lessons to review, snap options) is static
// content resolved deterministically by internal/catalog — this package's
// job is narrowly: classify blank-vs-written, extract a problem statement,
// and (for "check my work") find mistakes in a shown attempt.
//
// Extract/Evaluate below are backend-agnostic: they own the prompts
// (prompts.go), the schema (schema.go), decoding, and validation
// (draft.go), and delegate only the actual model call to a provider
// (provider.go, provider_anthropic.go, provider_openai.go). Adding a
// third backend means implementing provider — nothing here changes.
package analysis

import (
	"context"
	"encoding/json"
	"fmt"

	tutiv1 "tuti-server/internal/genproto/tutiv1"
)

// Image is a single photo to analyze.
type Image struct {
	Bytes       []byte
	ContentType string
}

// ExtractResult is the outcome of looking at a photo to see whether it's
// blank or has a math problem written on it.
type ExtractResult struct {
	Blank   bool
	Problem *tutiv1.Problem // nil when Blank
}

// EvaluateResult is the outcome of checking a student's shown work.
type EvaluateResult struct {
	Problem  *tutiv1.Problem
	Mistakes []*tutiv1.Mistake
}

// Analyzer is the narrow, vision-backed judgment call this package exists
// to make.
type Analyzer interface {
	// Extract classifies the photo(s) as blank or containing a written
	// problem, extracting the problem statement and a full solution when
	// present.
	Extract(ctx context.Context, images []Image) (ExtractResult, error)
	// Evaluate extracts the problem shown and evaluates the student's
	// attempt for mistakes.
	Evaluate(ctx context.Context, images []Image) (EvaluateResult, error)
}

// Backend identifiers accepted by Config.Backend.
const (
	Anthropic = "anthropic"
	OpenAI    = "openai"
)

const defaultMaxTokens = 4096

// Config selects and configures the backend New builds.
type Config struct {
	// Backend is Anthropic (the default, used when empty) or OpenAI.
	Backend string
	// Model is the backend-specific model ID (e.g. "claude-opus-5" or
	// "gpt-5.2").
	Model string
	// APIKey overrides the SDK's default credential resolution
	// (environment variable or CLI-managed profile) when non-empty.
	// Usually left empty in favor of that resolution.
	APIKey string
}

// New builds an Analyzer backed by the provider named in cfg.Backend.
func New(cfg Config) (Analyzer, error) {
	var backend provider
	switch cfg.Backend {
	case "", Anthropic:
		backend = newAnthropicProvider(cfg.Model, cfg.APIKey)
	case OpenAI:
		backend = newOpenAIProvider(cfg.Model, cfg.APIKey)
	default:
		return nil, fmt.Errorf("analysis: unknown backend %q", cfg.Backend)
	}
	return &llmAnalyzer{backend: backend}, nil
}

// llmAnalyzer implements Analyzer against whichever provider it was built
// with. All prompt text, schema shape, and output validation are shared
// across every backend — only the network call in provider differs.
type llmAnalyzer struct {
	backend provider
}

func (a *llmAnalyzer) Extract(ctx context.Context, images []Image) (ExtractResult, error) {
	var out extractOutput
	if err := a.call(ctx, images, extractPrompt, extractToolName, extractToolDescription,
		extractInputProperties(), extractInputRequired(), &out); err != nil {
		return ExtractResult{}, err
	}
	if err := out.Validate(); err != nil {
		return ExtractResult{}, fmt.Errorf("analysis: %w", err)
	}

	result := ExtractResult{Blank: out.Blank}
	if !out.Blank {
		result.Problem = out.Problem.toProto("detected_" + randSuffix())
	}
	return result, nil
}

func (a *llmAnalyzer) Evaluate(ctx context.Context, images []Image) (EvaluateResult, error) {
	var out evaluateOutput
	if err := a.call(ctx, images, evaluatePrompt, evaluateToolName, evaluateToolDescription,
		evaluateInputProperties(), evaluateInputRequired(), &out); err != nil {
		return EvaluateResult{}, err
	}
	if err := out.Validate(); err != nil {
		return EvaluateResult{}, fmt.Errorf("analysis: %w", err)
	}

	problemID := "detected_" + randSuffix()
	mistakes := make([]*tutiv1.Mistake, len(out.Mistakes))
	for i, m := range out.Mistakes {
		mistakes[i] = m.toProto(fmt.Sprintf("%s_m%d", problemID, i+1))
	}
	return EvaluateResult{
		Problem:  out.Problem.toProto(problemID),
		Mistakes: mistakes,
	}, nil
}

// call sends the images + prompt to the backend with a single tool
// forced, and unmarshals that tool's input into out. It never looks at
// the result beyond decoding it — every backend-specific detail (how to
// force the call, how to read the answer back out) lives in provider.
func (a *llmAnalyzer) call(ctx context.Context, images []Image, prompt, toolName, description string, properties map[string]any, required []string, out any) error {
	raw, err := a.backend.callStructured(ctx, structuredCallRequest{
		System:      systemPrompt,
		Prompt:      prompt,
		Images:      images,
		ToolName:    toolName,
		Description: description,
		Properties:  properties,
		Required:    required,
		MaxTokens:   defaultMaxTokens,
	})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("analysis: decoding %s output: %w", toolName, err)
	}
	return nil
}
