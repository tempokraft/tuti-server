// Package api is a thin client for the tuti-server HTTP API, used by the
// TUI to exercise /healthz, /v1/chat, and /v1/captures.
package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// reqTimeout bounds the quick, non-streaming requests (health checks,
// listing captures). Chat and uploads are exempt since they can legitimately
// take a while.
func reqTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// apiError builds an error from a non-2xx response, unwrapping the
// server's {"error": "..."} JSON body when present.
func apiError(op, status string, body []byte) error {
	var parsed struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &parsed) == nil && parsed.Error != "" {
		return fmt.Errorf("%s: %s (%s)", op, parsed.Error, status)
	}
	return fmt.Errorf("%s: unexpected status %s: %s", op, status, string(body))
}

// Client talks to a tuti-server instance.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a Client pointed at baseURL (e.g. "http://localhost:8080").
func New(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 0}, // chat responses stream indefinitely
	}
}

// Message mirrors the wire format documented in tuti-server's README.
type Message struct {
	Role string `json:"role"` // "user" | "agent"
	Text string `json:"text"`
}

// Capture is a single entry from GET /v1/captures.
type Capture struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	SizeBytes  int64     `json:"sizeBytes"`
	UploadedAt time.Time `json:"uploadedAt"`
	URL        string    `json:"url"`
}

// Health hits GET /healthz and returns an error if the server didn't
// respond with 2xx within the given timeout.
func (c *Client) Health() error {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/healthz", nil)
	if err != nil {
		return err
	}
	ctx, cancel := reqTimeout()
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("healthz: unexpected status %s", resp.Status)
	}
	return nil
}

// ChatRequest is the body sent to POST /v1/chat.
type ChatRequest struct {
	Message   string    `json:"message"`
	History   []Message `json:"history,omitempty"`
	CaptureID string    `json:"captureId,omitempty"`
}

// Chat streams the tutor's reply, invoking onChunk for each piece of text
// read off the response body as it arrives. It returns once the stream
// ends or ctx-equivalent cancellation happens via the client's timeout.
func (c *Client) Chat(req ChatRequest, onChunk func(string)) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.BaseURL+"/v1/chat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return apiError("chat", resp.Status, msg)
	}

	reader := bufio.NewReader(resp.Body)
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			onChunk(string(buf[:n]))
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// UploadCapture posts a file to POST /v1/captures and returns its new ID.
func (c *Client) UploadCapture(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.BaseURL+"/v1/captures", &buf)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", apiError("upload", resp.Status, respBody)
	}

	var out Capture
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("upload: decoding response: %w (body: %s)", err, string(respBody))
	}
	return out.ID, nil
}

// ListCaptures fetches GET /v1/captures, most recent first.
func (c *Client) ListCaptures() ([]Capture, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/v1/captures", nil)
	if err != nil {
		return nil, err
	}
	ctx, cancel := reqTimeout()
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, apiError("list captures", resp.Status, respBody)
	}

	var out []Capture
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("list captures: decoding response: %w (body: %s)", err, string(respBody))
	}
	return out, nil
}
