package analysis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

// openaiProvider calls an OpenAI model through the Responses API with a
// forced tool_choice, mirroring anthropicProvider: the model has no option
// but to respond by calling the one function we ask for.
type openaiProvider struct {
	client openai.Client
	model  string
}

// newOpenAIProvider builds an OpenAI-backed provider. apiKey may be empty
// to fall back to the SDK's default credential resolution (OPENAI_API_KEY
// and friends).
func newOpenAIProvider(model, apiKey string) *openaiProvider {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &openaiProvider{
		client: openai.NewClient(opts...),
		model:  model,
	}
}

func (p *openaiProvider) callStructured(ctx context.Context, req structuredCallRequest) (json.RawMessage, error) {
	if len(req.Images) == 0 {
		return nil, fmt.Errorf("analysis: at least one image is required")
	}

	content := make(responses.ResponseInputMessageContentListParam, 0, len(req.Images)+1)
	for _, img := range req.Images {
		part := responses.ResponseInputContentParamOfInputImage(responses.ResponseInputImageDetailAuto)
		part.OfInputImage.ImageURL = openai.String("data:" + img.ContentType + ";base64," + base64.StdEncoding.EncodeToString(img.Bytes))
		content = append(content, part)
	}
	content = append(content, responses.ResponseInputContentParamOfInputText(req.Prompt))

	// FunctionToolParam.Parameters is a raw map — unlike Anthropic's
	// ToolInputSchemaParam it doesn't supply "type": "object" itself, so we
	// build the full object schema here.
	parameters := map[string]any{
		"type":       "object",
		"properties": req.Properties,
		"required":   req.Required,
		// Best-effort here, not server-enforced: true strict mode on
		// OpenAI additionally requires every property to be listed in
		// "required" (optional fields become nullable unions instead),
		// which the shared schema in schema.go doesn't do. See the
		// comment at the top of schema.go.
		"additionalProperties": false,
	}

	resp, err := p.client.Responses.New(ctx, responses.ResponseNewParams{
		Model:           p.model,
		Instructions:    openai.String(req.System),
		MaxOutputTokens: openai.Int(req.MaxTokens),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				{OfMessage: &responses.EasyInputMessageParam{
					Role: responses.EasyInputMessageRoleUser,
					Content: responses.EasyInputMessageContentUnionParam{
						OfInputItemContentList: content,
					},
				}},
			},
		},
		Tools: []responses.ToolUnionParam{{OfFunction: &responses.FunctionToolParam{
			Name:        req.ToolName,
			Description: openai.String(req.Description),
			Parameters:  parameters,
			Strict:      openai.Bool(false),
		}}},
		ToolChoice: responses.ResponseNewParamsToolChoiceUnion{
			OfFunctionTool: &responses.ToolChoiceFunctionParam{Name: req.ToolName},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("analysis: %w", err)
	}

	for _, item := range resp.Output {
		if item.Type != "function_call" {
			continue
		}
		if fc := item.AsFunctionCall(); fc.Name == req.ToolName {
			return json.RawMessage(fc.Arguments), nil
		}
	}
	if resp.Error.Message != "" {
		return nil, fmt.Errorf("analysis: %s", resp.Error.Message)
	}
	return nil, fmt.Errorf("analysis: model did not call %s (status=%s)", req.ToolName, resp.Status)
}
