package httpapi

import (
	"net/http"

	"tuti-server/internal/tracing"
)

// withMiddleware wraps mux with CORS handling, panic recovery, and
// per-request tracing.
func withMiddleware(mux http.Handler, tracer tracing.Tracer) http.Handler {
	return withCORS(withRecover(withTracing(mux, tracer)))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func withTracing(next http.Handler, tracer tracing.Tracer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.StartSpan(r.Context(), "http.request",
			tracing.String("method", r.Method),
			tracing.String("path", r.URL.Path),
		)
		defer span.End()

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r.WithContext(ctx))

		span.SetAttributes(tracing.Int("status", rec.status))
	})
}

// statusRecorder captures the response status code for tracing, since
// http.ResponseWriter doesn't expose it after the fact.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Flush forwards to the underlying ResponseWriter's http.Flusher, if it has
// one. Without this, wrapping http.ResponseWriter here would silently hide
// flush support from handlers that stream a response (e.g. the chat
// endpoint), since a type assertion to http.Flusher only sees methods
// statusRecorder itself declares.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
