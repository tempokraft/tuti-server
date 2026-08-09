package httpapi

import (
	"net/http"

	tutiv1 "tuti-server/internal/genproto/tutiv1"
	"tuti-server/internal/tracing"
)

// handleUploadScreenshot implements TutiService.UploadScreenshot: a
// protojson body carrying the image bytes directly (base64), rather than
// multipart/form-data.
func (s *Server) handleUploadScreenshot(w http.ResponseWriter, r *http.Request) {
	ctx, span := s.Tracer.StartSpan(r.Context(), "captures.upload")
	defer span.End()

	var req tutiv1.UploadScreenshotRequest
	if err := readProto(w, r, &req, s.MaxUploadBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.GetData()) == 0 {
		writeJSONError(w, http.StatusBadRequest, "data is required")
		return
	}

	contentType, err := detectImageContentType(req.GetData(), req.GetFilename())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	obj, err := s.Store.Save(ctx, req.GetFilename(), contentType, req.GetData())
	if err != nil {
		span.RecordError(err)
		writeJSONError(w, http.StatusInternalServerError, "failed to store upload")
		return
	}
	span.SetAttributes(tracing.String("capture_id", obj.ID), tracing.Int("size_bytes", int(obj.SizeBytes)))

	writeProto(w, http.StatusCreated, &tutiv1.UploadScreenshotResponse{
		Capture: &tutiv1.Capture{
			Id:           obj.ID,
			Name:         obj.Name,
			Data:         req.GetData(),
			UploadedAtMs: obj.UploadedAt.UnixMilli(),
		},
	})
}

// handleListCaptures implements TutiService.ListCaptures. Per the proto,
// Capture embeds its bytes directly, so each listed capture requires a
// follow-up Store.Get — acceptable for a local dev/single-instance store,
// worth revisiting if the capture list ever grows large.
func (s *Server) handleListCaptures(w http.ResponseWriter, r *http.Request) {
	ctx, span := s.Tracer.StartSpan(r.Context(), "captures.list")
	defer span.End()

	objs, err := s.Store.List(ctx)
	if err != nil {
		span.RecordError(err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list captures")
		return
	}

	captures := make([]*tutiv1.Capture, 0, len(objs))
	for _, obj := range objs {
		_, data, err := s.Store.Get(ctx, obj.ID)
		if err != nil {
			span.RecordError(err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load capture")
			return
		}
		captures = append(captures, &tutiv1.Capture{
			Id:           obj.ID,
			Name:         obj.Name,
			Data:         data,
			UploadedAtMs: obj.UploadedAt.UnixMilli(),
		})
	}

	writeProto(w, http.StatusOK, &tutiv1.ListCapturesResponse{Captures: captures})
}
