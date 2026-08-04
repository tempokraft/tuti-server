// Package slogtracer is a tracing.Tracer implementation that emits
// structured log lines via log/slog. It's the default tracer: no external
// collector required, but the tracing.Tracer interface it satisfies means a
// real exporter (OpenTelemetry, etc.) can replace it later with no call-site
// changes.
package slogtracer

import (
	"context"
	"log/slog"
	"time"

	"tuti-server/internal/tracing"
)

// Tracer logs span start/end as structured log records.
type Tracer struct {
	logger *slog.Logger
}

// New returns a Tracer that logs via logger. If logger is nil, slog.Default
// is used.
func New(logger *slog.Logger) *Tracer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Tracer{logger: logger}
}

func (t *Tracer) StartSpan(ctx context.Context, name string, attrs ...tracing.Attr) (context.Context, tracing.Span) {
	span := &span{
		logger: t.logger,
		name:   name,
		start:  time.Now(),
		attrs:  attrs,
	}
	return ctx, span
}

type span struct {
	logger *slog.Logger
	name   string
	start  time.Time
	attrs  []tracing.Attr
	err    error
}

func (s *span) SetAttributes(attrs ...tracing.Attr) {
	s.attrs = append(s.attrs, attrs...)
}

func (s *span) RecordError(err error) {
	if err != nil {
		s.err = err
	}
}

func (s *span) End() {
	args := make([]any, 0, len(s.attrs)*2+4)
	for _, a := range s.attrs {
		args = append(args, a.Key, a.Value)
	}
	args = append(args, "duration_ms", time.Since(s.start).Milliseconds())

	if s.err != nil {
		args = append(args, "error", s.err.Error())
		s.logger.Error(s.name, args...)
		return
	}
	s.logger.Info(s.name, args...)
}
