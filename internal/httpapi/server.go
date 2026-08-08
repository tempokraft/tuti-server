// Package httpapi wires the HTTP surface for tuti-server: an HTTP/JSON
// binding of the TutiService RPCs defined in tuti/proto/tuti_service.proto
// (see internal/genproto/tutiv1 for the generated types), backed by
// internal/storage for captures, internal/session for the Snap & Solve and
// solve-session flows, internal/catalog for static lesson/problem content,
// and internal/analysis for the one genuinely model-backed judgment call
// (classifying/extracting a problem from a photo).
package httpapi

import (
	"net/http"

	"tuti-server/internal/analysis"
	"tuti-server/internal/session"
	"tuti-server/internal/storage"
	"tuti-server/internal/tracing"
)

// Server holds the dependencies HTTP handlers need. All fields are
// abstractions so concrete backends can be swapped without touching
// handler code.
type Server struct {
	Store          storage.Store
	Analyzer       analysis.Analyzer
	SnapStore      *session.SnapStore
	SolveStore     *session.SolveStore
	Tracer         tracing.Tracer
	MaxUploadBytes int64
}

// Routes builds the HTTP handler for the server.
//
// The proto defines no HTTP bindings, so each RPC is exposed as
// POST /v1/<RpcName>, request/response bodies protojson-encoded. There is
// no authentication yet — every route is open. When auth is added, it
// should be inserted as a middleware here, without changing individual
// handlers.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)

	mux.HandleFunc("POST /v1/UploadScreenshot", s.handleUploadScreenshot)
	mux.HandleFunc("POST /v1/ListCaptures", s.handleListCaptures)

	mux.HandleFunc("POST /v1/InitializeSnapAndSolve", s.handleInitializeSnapAndSolve)
	mux.HandleFunc("POST /v1/SubmitSnap", s.handleSubmitSnap)
	mux.HandleFunc("POST /v1/SubmitSnapResponse", s.handleSubmitSnapResponse)

	mux.HandleFunc("POST /v1/CreateSession", s.handleCreateSession)
	mux.HandleFunc("POST /v1/AnalyzeAssets", s.handleAnalyzeAssets)

	mux.HandleFunc("POST /v1/GetLessonContent", s.handleGetLessonContent)

	return withMiddleware(mux, s.Tracer)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
