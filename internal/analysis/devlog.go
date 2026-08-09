package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// structuredLogReq is the request shape written to dev log files for
// callStructured calls. Image bytes are intentionally omitted — only
// the count is recorded to keep files small.
type structuredLogReq struct {
	System     string `json:"system"`
	Prompt     string `json:"prompt"`
	ToolName   string `json:"tool_name"`
	ImageCount int    `json:"image_count"`
	MaxTokens  int64  `json:"max_tokens"`
}

type textLogReq struct {
	Prompt    string `json:"prompt"`
	MaxTokens int64  `json:"max_tokens"`
}

type providerLogEntry struct {
	Timestamp string          `json:"timestamp"`
	Op        string          `json:"op"`
	LatencyMs int64           `json:"latency_ms"`
	Request   json.RawMessage `json:"request"`
	Response  json.RawMessage `json:"response,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// writeDevLog writes a single JSON file capturing one provider round trip.
// Error calls get an ERROR_ prefix in the filename so they sort visibly.
// Errors are silently swallowed — this is instrumentation, not a critical path.
func writeDevLog(dir, op string, req any, response json.RawMessage, callErr error, latency time.Duration) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return
	}

	entry := providerLogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Op:        op,
		LatencyMs: latency.Milliseconds(),
		Request:   reqBytes,
		Response:  response,
	}
	if callErr != nil {
		entry.Error = callErr.Error()
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return
	}

	prefix := op
	if callErr != nil {
		prefix = "ERROR_" + op
	}
	name := fmt.Sprintf("%s_%s_%s.json",
		time.Now().UTC().Format("2006-01-02_15-04-05.000"),
		prefix,
		randSuffix(),
	)
	_ = os.WriteFile(filepath.Join(dir, name), data, 0o644)
}
