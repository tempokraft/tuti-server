// Package agent defines the abstraction over a conversational tutoring
// agent, mirroring the Agent interface on the Flutter client
// (lib/services/agent/agent.dart) so the two sides compose the same way.
package agent

import "context"

// Role identifies who authored a Message.
type Role string

const (
	RoleUser  Role = "user"
	RoleAgent Role = "agent"
)

// Message is a single turn of prior conversation history, provided as
// context for the next SendMessage call.
type Message struct {
	Role Role
	Text string
}

// Chunk is one piece of an in-progress reply. Err is set (with Text empty)
// when the stream terminates abnormally; the channel is closed after a
// chunk with a non-nil Err, or after the reply completes normally.
type Chunk struct {
	Text string
	Err  error
}

// Attachment is an optional image attached to a message, e.g. a captured
// photo of a math problem from the Snap & Solve flow.
type Attachment struct {
	Bytes       []byte
	ContentType string
}

// Agent sends a message to a conversational tutoring backend and streams
// the reply back in incremental chunks.
type Agent interface {
	// Name is a human-readable name for the agent, e.g. shown in a chat UI.
	Name() string

	// SendMessage sends message (with optional attachment and prior
	// history for context) and streams the reply back over the returned
	// channel. The channel is always closed by the implementation.
	SendMessage(ctx context.Context, message string, history []Message, attachment *Attachment) (<-chan Chunk, error)
}
