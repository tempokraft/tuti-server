package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// defaultMaxBodyBytes bounds request bodies for RPCs that don't carry
// image bytes (everything except UploadScreenshot/SubmitSnap, which use
// Server.MaxUploadBytes instead).
const defaultMaxBodyBytes = 1 << 20 // 1 MiB

// writeProto protojson-encodes msg as the response body.
func writeProto(w http.ResponseWriter, status int, msg proto.Message) {
	data, err := protojson.Marshal(msg)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

// readProto decodes a protojson request body into msg, bounded by
// maxBytes. An empty body is treated as an empty message (protojson
// itself requires at least "{}").
func readProto(w http.ResponseWriter, r *http.Request, msg proto.Message, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("request body too large or unreadable: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	if err := protojson.Unmarshal(data, msg); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	return nil
}

// writeJSONError is the transport-level error envelope for failures that
// happen outside the proto contract (bad JSON, oversized body, internal
// errors) — the proto has no Error message of its own.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
