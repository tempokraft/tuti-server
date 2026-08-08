package httpapi

import (
	"net/http"

	"tuti-server/internal/catalog"
	tutiv1 "tuti-server/internal/genproto/tutiv1"
)

// handleGetLessonContent serves static lesson content from
// internal/catalog. Unknown lesson ids get a "coming soon" stub rather
// than a 404, matching internal/catalog.GetLesson's fallback.
func (s *Server) handleGetLessonContent(w http.ResponseWriter, r *http.Request) {
	_, span := s.Tracer.StartSpan(r.Context(), "lessons.get")
	defer span.End()

	var req tutiv1.GetLessonContentRequest
	if err := readProto(w, r, &req, defaultMaxBodyBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.GetLessonId() == "" {
		writeJSONError(w, http.StatusBadRequest, "lessonId is required")
		return
	}

	lang := req.GetLanguage()
	if lang == "" {
		lang = "en"
	}

	writeProto(w, http.StatusOK, &tutiv1.GetLessonContentResponse{
		Content: catalog.GetLesson(req.GetLessonId(), lang),
	})
}
