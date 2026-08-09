package api

import (
	"encoding/json"
	"fmt"
)

// Pretty JSON-indents v for display in the TUI transcript, truncating any
// long string value (image bytes travel as base64, which is otherwise an
// unreadable wall of characters) regardless of which field it's in.
func Pretty(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "<error: " + err.Error() + ">"
	}
	return PrettyRaw(data)
}

// PrettyRaw indents already-encoded JSON bytes for display, truncating long
// string values the same way Pretty does. Returns "(empty)" for a nil/empty
// input, e.g. a GET request with no body.
func PrettyRaw(data []byte) string {
	if len(data) == 0 {
		return "(empty)"
	}

	var generic any
	if err := json.Unmarshal(data, &generic); err != nil {
		return string(data)
	}
	truncateStrings(generic)

	out, err := json.MarshalIndent(generic, "", "  ")
	if err != nil {
		return string(data)
	}
	return string(out)
}

const maxDisplayStringLen = 80

func truncateStrings(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if s, ok := val.(string); ok && len(s) > maxDisplayStringLen {
				t[k] = s[:maxDisplayStringLen] + "..."
			} else {
				truncateStrings(val)
			}
		}
	case []any:
		for i, val := range t {
			if s, ok := val.(string); ok && len(s) > maxDisplayStringLen {
				t[i] = s[:maxDisplayStringLen] + "..."
			} else {
				truncateStrings(val)
			}
		}
	}
}

// PrettyRawRedacted indents already-encoded JSON bytes like PrettyRaw, but
// replaces (rather than truncates) any string value longer than maxLen —
// used for the history panel, which shows the request in full otherwise;
// a redaction keeps a large param (base64 image data, a long prompt) from
// flooding the panel while still surfacing that it was there.
func PrettyRawRedacted(data []byte, maxLen int) string {
	if len(data) == 0 {
		return "(empty)"
	}

	var generic any
	if err := json.Unmarshal(data, &generic); err != nil {
		return string(data)
	}
	redactLongStrings(generic, maxLen)

	out, err := json.MarshalIndent(generic, "", "  ")
	if err != nil {
		return string(data)
	}
	return string(out)
}

func redactLongStrings(v any, maxLen int) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if s, ok := val.(string); ok && len(s) > maxLen {
				t[k] = fmt.Sprintf("[redacted: %d chars]", len(s))
			} else {
				redactLongStrings(val, maxLen)
			}
		}
	case []any:
		for i, val := range t {
			if s, ok := val.(string); ok && len(s) > maxLen {
				t[i] = fmt.Sprintf("[redacted: %d chars]", len(s))
			} else {
				redactLongStrings(val, maxLen)
			}
		}
	}
}
