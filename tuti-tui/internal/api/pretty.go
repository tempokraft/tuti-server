package api

import "encoding/json"

// Pretty JSON-indents v for display in the TUI transcript, truncating any
// long string value (image bytes travel as base64, which is otherwise an
// unreadable wall of characters) regardless of which field it's in.
func Pretty(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "<error: " + err.Error() + ">"
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
