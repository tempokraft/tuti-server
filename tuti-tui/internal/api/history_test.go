package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecordHistoryCapsAtMax(t *testing.T) {
	c := &Client{}
	for i := 0; i < maxHistoryEntries+5; i++ {
		c.recordHistory("POST", fmt.Sprintf("rpc-%d", i), "/v1/rpc", []byte(`{"n":1}`), []byte(`{"ok":true}`), "200 OK", 0, nil)
	}

	got := c.History()
	if len(got) != maxHistoryEntries {
		t.Fatalf("len(History()) = %d, want %d", len(got), maxHistoryEntries)
	}
	// Oldest entries should have been evicted — the first entry left should
	// be rpc-5, since rpc-0..rpc-4 were dropped to stay at the cap.
	if want := "rpc-5"; got[0].RPC != want {
		t.Errorf("History()[0].RPC = %q, want %q", got[0].RPC, want)
	}
	if want := fmt.Sprintf("rpc-%d", maxHistoryEntries+4); got[len(got)-1].RPC != want {
		t.Errorf("History()[last].RPC = %q, want %q", got[len(got)-1].RPC, want)
	}
}

func TestRecordHistoryMarksErrorStatus(t *testing.T) {
	c := &Client{}
	c.recordHistory("POST", "broken", "/v1/broken", nil, nil, "", 0, fmt.Errorf("connection refused"))

	got := c.History()
	if len(got) != 1 {
		t.Fatalf("len(History()) = %d, want 1", len(got))
	}
	if got[0].Status != "error" {
		t.Errorf("Status = %q, want %q", got[0].Status, "error")
	}
	if got[0].Request != "(empty)" {
		t.Errorf("Request = %q, want %q", got[0].Request, "(empty)")
	}
}

func TestRecordHistoryRedactsLongRequestParams(t *testing.T) {
	c := &Client{}
	shortVal := strings.Repeat("a", 500)
	longVal := strings.Repeat("b", 501)
	reqBody := []byte(fmt.Sprintf(`{"short":%q,"long":%q}`, shortVal, longVal))

	c.recordHistory("POST", "upload", "/v1/upload", reqBody, nil, "200 OK", 0, nil)

	got := c.History()[0].Request
	if !strings.Contains(got, shortVal) {
		t.Errorf("Request should keep a 500-char value in full, got: %s", got)
	}
	if strings.Contains(got, longVal) {
		t.Errorf("Request should redact a 501-char value, got: %s", got)
	}
	if !strings.Contains(got, "[redacted: 501 chars]") {
		t.Errorf("Request should contain a redaction marker, got: %s", got)
	}
}

func TestReplayResendsExactRawBody(t *testing.T) {
	var gotBody []byte
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"echo":true}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	longVal := strings.Repeat("x", 600)
	req := struct {
		Data string `json:"data"`
	}{Data: longVal}
	if err := c.call(context.Background(), "Echo", req, nil, shortTimeout); err != nil {
		t.Fatalf("call: %v", err)
	}

	entries := c.History()
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	// The stored display Request must be redacted...
	if strings.Contains(entries[0].Request, longVal) {
		t.Fatalf("displayed Request should have redacted the long value")
	}

	// ...but Replay must still resend the original, unredacted body.
	respBody, err := c.Replay(context.Background(), entries[0])
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if string(respBody) != `{"echo":true}` {
		t.Errorf("Replay response = %q", respBody)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("replayed method = %q, want POST", gotMethod)
	}
	if gotPath != "/v1/Echo" {
		t.Errorf("replayed path = %q, want /v1/Echo", gotPath)
	}
	if !strings.Contains(string(gotBody), longVal) {
		t.Errorf("replayed body should contain the original unredacted value")
	}

	if len(c.History()) != 2 {
		t.Errorf("Replay should record its own history entry, len = %d, want 2", len(c.History()))
	}
}

func TestHistorySnapshotIsIndependentCopy(t *testing.T) {
	c := &Client{}
	c.recordHistory("POST", "first", "/v1/first", nil, nil, "200 OK", 0, nil)

	snap := c.History()
	c.recordHistory("POST", "second", "/v1/second", nil, nil, "200 OK", 0, nil)

	if len(snap) != 1 {
		t.Fatalf("earlier snapshot mutated: len = %d, want 1", len(snap))
	}
}
