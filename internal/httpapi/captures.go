package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"tuti-server/internal/storage"
	"tuti-server/internal/tracing"
)

type captureResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	SizeBytes  int64     `json:"sizeBytes"`
	UploadedAt time.Time `json:"uploadedAt"`
	URL        string    `json:"url"`
}

func toCaptureResponse(obj storage.Object) captureResponse {
	return captureResponse{
		ID:         obj.ID,
		Name:       obj.Name,
		SizeBytes:  obj.SizeBytes,
		UploadedAt: obj.UploadedAt,
		URL:        fmt.Sprintf("/v1/captures/%s/content", obj.ID),
	}
}

// handleUploadCapture accepts a multipart/form-data upload with a "file"
// field and stores it via the configured storage.Store.
func (s *Server) handleUploadCapture(w http.ResponseWriter, r *http.Request) {
	ctx, span := s.Tracer.StartSpan(r.Context(), "captures.upload")
	defer span.End()

	r.Body = http.MaxBytesReader(w, r.Body, s.MaxUploadBytes)
	if err := r.ParseMultipartForm(s.MaxUploadBytes); err != nil {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "upload too large or malformed")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "missing \"file\" field")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		span.RecordError(err)
		writeJSONError(w, http.StatusInternalServerError, "failed to read upload")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	obj, err := s.Store.Save(ctx, header.Filename, contentType, data)
	if err != nil {
		span.RecordError(err)
		writeJSONError(w, http.StatusInternalServerError, "failed to store upload")
		return
	}
	span.SetAttributes(tracing.String("capture_id", obj.ID), tracing.Int("size_bytes", int(obj.SizeBytes)))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toCaptureResponse(obj))
}

// handleListCaptures returns previously uploaded captures, most recent
// first.
func (s *Server) handleListCaptures(w http.ResponseWriter, r *http.Request) {
	objs, err := s.Store.List(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list captures")
		return
	}

	resp := make([]captureResponse, len(objs))
	for i, obj := range objs {
		resp[i] = toCaptureResponse(obj)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleCaptureContent serves the raw bytes of a previously uploaded
// capture.
func (s *Server) handleCaptureContent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	obj, data, err := s.Store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "capture not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load capture")
		return
	}

	w.Header().Set("Content-Type", obj.ContentType)
	w.Write(data)
}
