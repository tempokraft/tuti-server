package httpapi

import (
	"errors"
	"fmt"
	"net/http"

	"tuti-server/internal/analysis"
	"tuti-server/internal/catalog"
	tutiv1 "tuti-server/internal/genproto/tutiv1"
	"tuti-server/internal/storage"
)

// handleCreateSession opens a new solve session for a subsequent
// AnalyzeAssets call.
func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	_, span := s.Tracer.StartSpan(r.Context(), "solve.create_session")
	defer span.End()

	var req tutiv1.CreateSessionRequest
	if err := readProto(w, r, &req, defaultMaxBodyBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	sess, err := s.SolveStore.Create()
	if err != nil {
		span.RecordError(err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	writeProto(w, http.StatusOK, &tutiv1.CreateSessionResponse{
		Session: &tutiv1.SolveSession{
			SessionId:   sess.ID,
			CreatedAtMs: sess.CreatedAt.UnixMilli(),
		},
	})
}

// handleAnalyzeAssets classifies the given captures as blank or containing
// a written problem. similarProblems/topics/lessons are resolved
// deterministically from internal/catalog; only the blank/written call and
// the problem extraction itself come from internal/analysis.
func (s *Server) handleAnalyzeAssets(w http.ResponseWriter, r *http.Request) {
	ctx, span := s.Tracer.StartSpan(r.Context(), "solve.analyze_assets")
	defer span.End()

	var req tutiv1.AnalyzeAssetsRequest
	if err := readProto(w, r, &req, defaultMaxBodyBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.GetSessionId() == "" {
		writeJSONError(w, http.StatusBadRequest, "sessionId is required")
		return
	}
	if !s.SolveStore.Exists(req.GetSessionId()) {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	if len(req.GetAssetIds()) == 0 {
		writeJSONError(w, http.StatusBadRequest, "assetIds is required")
		return
	}

	images := make([]analysis.Image, 0, len(req.GetAssetIds()))
	for _, id := range req.GetAssetIds() {
		obj, data, err := s.Store.Get(ctx, id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, fmt.Sprintf("capture %s not found", id))
				return
			}
			span.RecordError(err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load capture")
			return
		}
		images = append(images, analysis.Image{Bytes: data, ContentType: obj.ContentType})
	}

	result, err := s.Analyzer.Extract(ctx, images)
	if err != nil {
		span.RecordError(err)
		writeJSONError(w, http.StatusBadGateway, "analysis failed")
		return
	}

	resp := &tutiv1.AnalyzeAssetsResponse{}
	if result.Blank {
		resp.Result = &tutiv1.AnalyzeAssetsResponse_Blank{Blank: &tutiv1.BlankProblemsResult{
			Topics:          catalog.Topics,
			SimilarProblems: []*tutiv1.Problem{catalog.Problems["prob_quad_1"], catalog.Problems["prob_pyth_1"]},
		}}
	} else {
		_, similar := catalog.LessonsAndProblemsForTopic(result.Problem.GetTopic())
		resp.Result = &tutiv1.AnalyzeAssetsResponse_ProblemsFound{ProblemsFound: &tutiv1.ProblemsFoundResult{
			DetectedProblems: []*tutiv1.Problem{result.Problem},
			SimilarProblems:  similar,
		}}
	}

	writeProto(w, http.StatusOK, resp)
}
