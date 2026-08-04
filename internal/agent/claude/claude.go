// Package claude implements agent.Agent on top of the Claude API.
package claude

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"tuti-server/internal/agent"
)

// defaultSystemPrompt instructs the model to act as Tuti, a friendly math
// tutor for K-12 students working through the app's Snap & Solve and Work
// Review flows.
const defaultSystemPrompt = `You are Tuti, a patient, encouraging math tutor for K-12 students.

A student may share a photo of a math problem and ask for help solving it
step by step, or share a photo of their own written work and ask for
feedback. In both cases:
  - Never just give the final answer up front. Guide the student toward it
    with questions and hints, one step at a time.
  - When reviewing a student's work, point out specifically what they did
    right before addressing mistakes, and explain *why* a step is wrong,
    not just that it is.
  - Use simple, age-appropriate language. Avoid jargon unless you define it.
  - Keep responses short and conversational — this is a chat, not an essay.
  - If a photo is blurry, cropped, or otherwise unreadable, say so and ask
    the student to retake it rather than guessing at the problem.`

const defaultMaxTokens = 2048

// Agent talks to the Claude API to power the tutoring chat.
type Agent struct {
	client       anthropic.Client
	model        anthropic.Model
	name         string
	systemPrompt string
	maxTokens    int64
}

// Option configures an Agent constructed via New.
type Option func(*Agent)

// WithName overrides the human-readable agent name (default "Tuti").
func WithName(name string) Option {
	return func(a *Agent) { a.name = name }
}

// WithSystemPrompt overrides the default tutoring system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(a *Agent) { a.systemPrompt = prompt }
}

// WithMaxTokens overrides the default response length cap.
func WithMaxTokens(maxTokens int64) Option {
	return func(a *Agent) { a.maxTokens = maxTokens }
}

// New constructs a Claude-backed agent.Agent. model should be a Claude model
// ID (e.g. "claude-opus-5"). Credentials are resolved the standard way
// (ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, or an `ant auth login` profile);
// pass an explicit apiKey to override.
func New(model string, apiKey string, opts ...Option) *Agent {
	clientOpts := []option.RequestOption{}
	if apiKey != "" {
		clientOpts = append(clientOpts, option.WithAPIKey(apiKey))
	}

	a := &Agent{
		client:       anthropic.NewClient(clientOpts...),
		model:        anthropic.Model(model),
		name:         "Tuti",
		systemPrompt: defaultSystemPrompt,
		maxTokens:    defaultMaxTokens,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *Agent) Name() string { return a.name }

func (a *Agent) SendMessage(ctx context.Context, message string, history []agent.Message, attachment *agent.Attachment) (<-chan agent.Chunk, error) {
	messages := make([]anthropic.MessageParam, 0, len(history)+1)
	for _, h := range history {
		messages = append(messages, anthropic.MessageParam{
			Role:    roleFor(h.Role),
			Content: []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock(h.Text)},
		})
	}

	userBlocks := make([]anthropic.ContentBlockParamUnion, 0, 2)
	if attachment != nil {
		userBlocks = append(userBlocks, anthropic.NewImageBlockBase64(
			attachment.ContentType,
			base64.StdEncoding.EncodeToString(attachment.Bytes),
		))
	}
	userBlocks = append(userBlocks, anthropic.NewTextBlock(message))
	messages = append(messages, anthropic.MessageParam{
		Role:    anthropic.MessageParamRoleUser,
		Content: userBlocks,
	})

	stream := a.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     a.model,
		MaxTokens: a.maxTokens,
		System:    []anthropic.TextBlockParam{{Text: a.systemPrompt}},
		Messages:  messages,
	})

	out := make(chan agent.Chunk)
	go func() {
		defer close(out)
		for stream.Next() {
			event := stream.Current()
			delta, ok := event.AsAny().(anthropic.ContentBlockDeltaEvent)
			if !ok {
				continue
			}
			textDelta, ok := delta.Delta.AsAny().(anthropic.TextDelta)
			if !ok || textDelta.Text == "" {
				continue
			}
			select {
			case out <- agent.Chunk{Text: textDelta.Text}:
			case <-ctx.Done():
				return
			}
		}
		if err := stream.Err(); err != nil {
			select {
			case out <- agent.Chunk{Err: fmt.Errorf("claude: %w", err)}:
			case <-ctx.Done():
			}
		}
	}()

	return out, nil
}

func roleFor(r agent.Role) anthropic.MessageParamRole {
	if r == agent.RoleAgent {
		return anthropic.MessageParamRoleAssistant
	}
	return anthropic.MessageParamRoleUser
}
