// Package api is a hand-rolled client for tuti-server's HTTP/JSON binding
// of TutiService (see tuti/proto/tuti_service.proto), used by the TUI to
// exercise every RPC by hand. Every RPC is POST /v1/<RpcName> with a
// protojson-encoded body — see types.go for the wire shapes.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client talks to a tuti-server instance.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a Client pointed at baseURL (e.g. "http://localhost:8080").
func New(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 0},
	}
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

// call POSTs req (protojson-shaped by encoding/json's normal rules — see
// types.go's comment on why that's equivalent here) to /v1/rpc and decodes
// the response into resp. A nil req sends an empty body ("{}"), matching
// how the server treats empty-request RPCs. A zero timeout means no
// client-side deadline (used for RPCs that call out to an LLM and can
// legitimately take a while).
func (c *Client) call(ctx context.Context, rpc string, req, resp any, timeout time.Duration) error {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	body := []byte("{}")
	if req != nil {
		var err error
		body, err = json.Marshal(req)
		if err != nil {
			return fmt.Errorf("%s: encoding request: %w", rpc, err)
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/"+rpc, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("%s: reading response: %w", rpc, err)
	}
	if httpResp.StatusCode >= 300 {
		return apiError(rpc, httpResp.Status, respBody)
	}
	if resp == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, resp); err != nil {
		return fmt.Errorf("%s: decoding response: %w (body: %s)", rpc, err, string(respBody))
	}
	return nil
}

// shortTimeout bounds RPCs that never call out to an LLM.
const shortTimeout = 10 * time.Second

// Health hits GET /healthz.
func (c *Client) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/healthz", nil)
	if err != nil {
		return err
	}
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

// UploadScreenshot reads the file at path and uploads it.
func (c *Client) UploadScreenshot(path string) (*Capture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var resp uploadScreenshotResponse
	req := uploadScreenshotRequest{Data: data, Filename: filepath.Base(path)}
	if err := c.call(context.Background(), "UploadScreenshot", req, &resp, shortTimeout); err != nil {
		return nil, err
	}
	return resp.Capture, nil
}

// ListCaptures fetches every uploaded capture, most recent first.
func (c *Client) ListCaptures() ([]*Capture, error) {
	var resp listCapturesResponse
	if err := c.call(context.Background(), "ListCaptures", nil, &resp, shortTimeout); err != nil {
		return nil, err
	}
	return resp.Captures, nil
}

// CreateSession opens a new solve session.
func (c *Client) CreateSession() (*SolveSession, error) {
	var resp createSessionResponse
	if err := c.call(context.Background(), "CreateSession", nil, &resp, shortTimeout); err != nil {
		return nil, err
	}
	return resp.Session, nil
}

// AnalyzeAssets classifies the given captures as blank or containing a
// written problem. No client-side timeout: this calls out to an LLM.
func (c *Client) AnalyzeAssets(sessionID string, assetIDs []string) (*AnalyzeAssetsResult, error) {
	var resp AnalyzeAssetsResult
	req := analyzeAssetsRequest{SessionID: sessionID, AssetIDs: assetIDs}
	if err := c.call(context.Background(), "AnalyzeAssets", req, &resp, 0); err != nil {
		return nil, err
	}
	return &resp, nil
}

// InitializeSnapAndSolve starts a new Snap & Solve session.
func (c *Client) InitializeSnapAndSolve() (sessionID string, next *NextStep, err error) {
	var resp initializeSnapAndSolveResponse
	if err := c.call(context.Background(), "InitializeSnapAndSolve", nil, &resp, shortTimeout); err != nil {
		return "", nil, err
	}
	return resp.SessionID, resp.NextStep, nil
}

// SubmitSnap uploads the photo at path for sessionID.
func (c *Client) SubmitSnap(sessionID, path string) (*NextStep, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var resp submitSnapResult
	req := submitSnapRequest{SessionID: sessionID, Data: data, Filename: filepath.Base(path)}
	if err := c.call(context.Background(), "SubmitSnap", req, &resp, shortTimeout); err != nil {
		return nil, err
	}
	return resp.NextStep, nil
}

// SubmitSnapResponse submits the student's chosen action. No client-side
// timeout: this calls out to an LLM.
func (c *Client) SubmitSnapResponse(sessionID, responseID string) (*NextStep, error) {
	var resp submitSnapResponseResult
	req := submitSnapResponseRequest{SessionID: sessionID, ResponseID: responseID}
	if err := c.call(context.Background(), "SubmitSnapResponse", req, &resp, 0); err != nil {
		return nil, err
	}
	return resp.NextStep, nil
}

// GetLessonContent fetches a lesson by id and language ("" defaults to "en"
// server-side).
func (c *Client) GetLessonContent(lessonID, language string) (*LessonContent, error) {
	var resp getLessonContentResponse
	req := getLessonContentRequest{LessonID: lessonID, Language: language}
	if err := c.call(context.Background(), "GetLessonContent", req, &resp, shortTimeout); err != nil {
		return nil, err
	}
	return resp.Content, nil
}
