package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
)

// handleDevPrompt forwards a raw text prompt to the active LLM backend and
// returns its plain-text reply. Only registered when Server.DevMode is true.
func (s *Server) handleDevPrompt(w http.ResponseWriter, r *http.Request) {
	_, span := s.Tracer.StartSpan(r.Context(), "dev.prompt")
	defer span.End()

	r.Body = http.MaxBytesReader(w, r.Body, defaultMaxBodyBytes)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "request body too large or unreadable")
		return
	}

	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Prompt == "" {
		writeJSONError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	reply, err := s.Prompter.RawPrompt(r.Context(), req.Prompt)
	if err != nil {
		span.RecordError(err)
		writeJSONError(w, http.StatusBadGateway, "provider error: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"reply": reply})
}
