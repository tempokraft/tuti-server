// Package tracing defines a minimal, backend-agnostic abstraction over
// request/operation tracing so the concrete exporter (structured logs today,
// OpenTelemetry or another backend later) can be swapped without touching
// call sites.
package tracing

import "context"

// Attr is a single key/value attribute attached to a span.
type Attr struct {
	Key   string
	Value any
}

// String builds a string-valued Attr.
func String(key, value string) Attr { return Attr{Key: key, Value: value} }

// Int builds an int-valued Attr.
func Int(key string, value int) Attr { return Attr{Key: key, Value: value} }

// Span represents a single traced operation.
type Span interface {
	// SetAttributes attaches additional attributes to the span.
	SetAttributes(attrs ...Attr)

	// RecordError marks the span as having failed with err. A nil err is a
	// no-op.
	RecordError(err error)

	// End finalizes the span. Implementations must be safe to call exactly
	// once; callers should defer it immediately after StartSpan.
	End()
}

// Tracer starts spans for named operations.
type Tracer interface {
	// StartSpan begins a new span named name, returning a derived context
	// that implementations may use to propagate span identity, and the
	// Span itself.
	StartSpan(ctx context.Context, name string, attrs ...Attr) (context.Context, Span)
}
