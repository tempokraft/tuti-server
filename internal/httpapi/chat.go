package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"tuti-server/internal/agent"
	"tuti-server/internal/storage"
	"tuti-server/internal/tracing"
)

type chatMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type chatRequest struct {
	Message   string        `json:"message"`
	History   []chatMessage `json:"history"`
	CaptureID string        `json:"captureId"`
}

// handleChat streams the agent's reply back as a sequence of raw text
// chunks over a chunked HTTP response (Content-Type: text/plain). This
// mirrors the Flutter client's Agent.sendMessage, which models a reply as
// a Stream<String> of incremental chunks — a client reads the response
// body incrementally and appends each chunk to the in-progress message.
//
// On success but before any output has partially been sent, errors are
// reported as a JSON error body with an appropriate status code. Once
// streaming has begun, an error simply ends the connection early; the
// client sees a short reply.
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Message == "" {
		writeJSONError(w, http.StatusBadRequest, "message is required")
		return
	}

	ctx, span := s.Tracer.StartSpan(r.Context(), "chat.send_message",
		tracing.Int("history_len", len(req.History)),
		tracing.String("capture_id", req.CaptureID),
	)
	defer span.End()

	history := make([]agent.Message, 0, len(req.History))
	for _, m := range req.History {
		role := agent.RoleUser
		if m.Role == string(agent.RoleAgent) {
			role = agent.RoleAgent
		}
		history = append(history, agent.Message{Role: role, Text: m.Text})
	}

	var attachment *agent.Attachment
	if req.CaptureID != "" {
		obj, data, err := s.Store.Get(ctx, req.CaptureID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "capture not found")
				return
			}
			span.RecordError(err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load capture")
			return
		}
		attachment = &agent.Attachment{Bytes: data, ContentType: obj.ContentType}
	}

	chunks, err := s.Agent.SendMessage(ctx, req.Message, history, attachment)
	if err != nil {
		span.RecordError(err)
		writeJSONError(w, http.StatusBadGateway, "agent request failed")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for chunk := range chunks {
		if chunk.Err != nil {
			span.RecordError(chunk.Err)
			return
		}
		if _, err := w.Write([]byte(chunk.Text)); err != nil {
			return
		}
		flusher.Flush()
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
