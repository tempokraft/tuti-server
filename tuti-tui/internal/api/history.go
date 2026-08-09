package api

import (
	"context"
	"fmt"
	"time"
)

// maxHistoryEntries caps how many past request/response pairs the client
// retains — older entries are dropped as new ones arrive.
const maxHistoryEntries = 20

// requestRedactThreshold is the per-string-value length above which the
// history panel redacts a request param instead of showing it raw.
const requestRedactThreshold = 500

// replayTimeout bounds a replayed call — the original may have had no
// client-side deadline (LLM-backed RPCs), but a replay triggered by hand
// from the history panel shouldn't be able to hang the TUI indefinitely.
const replayTimeout = 60 * time.Second

// HistoryEntry records one HTTP round trip made by the client, for display
// in the TUI's history panel and, via Replay, for resending unchanged.
type HistoryEntry struct {
	RPC      string
	Time     time.Time
	Duration time.Duration
	Request  string // pretty-printed request body, params over requestRedactThreshold chars redacted; or "(empty)"
	Response string // pretty-printed response body received so far
	Status   string // e.g. "200 OK", or "error" if the round trip never got a response
	Err      error

	// method, path, and rawRequest capture exactly what went over the
	// wire, unredacted, so Replay can resend it byte-for-byte — Request
	// above is display-only and may have had large params redacted.
	method     string
	path       string
	rawRequest []byte
}

// recordHistory appends entry to the client's history, evicting the oldest
// entry once maxHistoryEntries is exceeded. Called for every request the
// client makes, success or failure.
func (c *Client) recordHistory(method, rpc, path string, reqBody, respBody []byte, status string, dur time.Duration, err error) {
	if err != nil && status == "" {
		status = "error"
	}
	entry := HistoryEntry{
		RPC:        rpc,
		Time:       time.Now(),
		Duration:   dur,
		Request:    PrettyRawRedacted(reqBody, requestRedactThreshold),
		Response:   PrettyRaw(respBody),
		Status:     status,
		Err:        err,
		method:     method,
		path:       path,
		rawRequest: reqBody,
	}

	c.historyMu.Lock()
	defer c.historyMu.Unlock()
	c.history = append(c.history, entry)
	if len(c.history) > maxHistoryEntries {
		c.history = c.history[len(c.history)-maxHistoryEntries:]
	}
}

// Replay resends e's original request — same method, path, and raw
// (unredacted) body — and returns the raw response body. Like any other
// call, it records its own new HistoryEntry, so a replay shows up at the
// top of the panel just like the call it repeated.
func (c *Client) Replay(ctx context.Context, e HistoryEntry) ([]byte, error) {
	if e.method == "" || e.path == "" {
		return nil, fmt.Errorf("replay: %s has no recorded request to resend", e.RPC)
	}
	ctx, cancel := context.WithTimeout(ctx, replayTimeout)
	defer cancel()
	return c.do(ctx, e.method, e.RPC, e.path, e.rawRequest)
}

// History returns a snapshot of the most recent requests, oldest first.
func (c *Client) History() []HistoryEntry {
	c.historyMu.Lock()
	defer c.historyMu.Unlock()
	out := make([]HistoryEntry, len(c.history))
	copy(out, c.history)
	return out
}
