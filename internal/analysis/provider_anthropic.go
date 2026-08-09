package analysis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// anthropicProvider calls Claude with a forced tool_choice so the model
// has no option but to respond by calling the one tool we ask for.
type anthropicProvider struct {
	client anthropic.Client
	model  anthropic.Model
}

// newAnthropicProvider builds a Claude-backed provider. apiKey may be
// empty to fall back to the SDK's default credential resolution
// (ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, or an `ant auth login`
// profile).
func newAnthropicProvider(model, apiKey string) *anthropicProvider {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &anthropicProvider{
		client: anthropic.NewClient(opts...),
		model:  anthropic.Model(model),
	}
}

func (p *anthropicProvider) callStructured(ctx context.Context, req structuredCallRequest) (json.RawMessage, error) {
	if len(req.Images) == 0 {
		return nil, fmt.Errorf("analysis: at least one image is required")
	}

	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(req.Images)+1)
	for _, img := range req.Images {
		blocks = append(blocks, anthropic.NewImageBlockBase64(img.ContentType, base64.StdEncoding.EncodeToString(img.Bytes)))
	}
	blocks = append(blocks, anthropic.NewTextBlock(req.Prompt))

	tool := anthropic.ToolParam{
		Name:        req.ToolName,
		Description: anthropic.String(req.Description),
		// Strict guarantees the tool_use input validates exactly against
		// the schema (types, enums, no stray keys) instead of best-effort.
		Strict: anthropic.Bool(true),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties:  req.Properties,
			Required:    req.Required,
			ExtraFields: map[string]any{"additionalProperties": false},
		},
	}

	message, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     p.model,
		MaxTokens: req.MaxTokens,
		System:    []anthropic.TextBlockParam{{Text: req.System}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(blocks...)},
		Tools:     []anthropic.ToolUnionParam{{OfTool: &tool}},
		ToolChoice: anthropic.ToolChoiceUnionParam{
			OfTool: &anthropic.ToolChoiceToolParam{Name: req.ToolName},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("analysis: %w", err)
	}
	if message.StopReason == anthropic.StopReasonRefusal {
		return nil, fmt.Errorf("analysis: model declined to respond (refusal)")
	}

	for _, block := range message.Content {
		if toolUse := block.AsToolUse(); toolUse.Name == req.ToolName {
			return toolUse.Input, nil
		}
	}
	return nil, fmt.Errorf("analysis: model did not call %s (stop_reason=%s)", req.ToolName, message.StopReason)
}

func (p *anthropicProvider) callText(ctx context.Context, prompt string, maxTokens int64) (string, error) {
	message, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     p.model,
		MaxTokens: maxTokens,
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
	})
	if err != nil {
		return "", fmt.Errorf("analysis: %w", err)
	}
	if message.StopReason == anthropic.StopReasonRefusal {
		return "", fmt.Errorf("analysis: model declined to respond (refusal)")
	}
	for _, block := range message.Content {
		if tb := block.AsText(); tb.Text != "" {
			return tb.Text, nil
		}
	}
	return "", fmt.Errorf("analysis: no text in response (stop_reason=%s)", message.StopReason)
}
