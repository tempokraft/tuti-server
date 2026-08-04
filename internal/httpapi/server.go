// Package httpapi wires the HTTP surface for the tutor backend: a
// streaming chat endpoint backed by an agent.Agent, and a capture
// upload/list endpoint backed by a storage.Store.
package httpapi

import (
	"net/http"

	"tuti-server/internal/agent"
	"tuti-server/internal/storage"
	"tuti-server/internal/tracing"
)

// Server holds the dependencies HTTP handlers need. All fields are
// abstractions (agent.Agent, storage.Store, tracing.Tracer) so concrete
// backends can be swapped without touching handler code.
type Server struct {
	Agent          agent.Agent
	Store          storage.Store
	Tracer         tracing.Tracer
	MaxUploadBytes int64
}

// Routes builds the HTTP handler for the server.
//
// There is no authentication yet — every route is open. When auth is
// added, it should be inserted as a middleware here (e.g. wrapping mux
// with an auth check that populates the request context), without
// changing individual handlers.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /v1/chat", s.handleChat)
	mux.HandleFunc("POST /v1/captures", s.handleUploadCapture)
	mux.HandleFunc("GET /v1/captures", s.handleListCaptures)
	mux.HandleFunc("GET /v1/captures/{id}/content", s.handleCaptureContent)

	return withMiddleware(mux, s.Tracer)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
