package httpapi

import (
	"context"
	"errors"
	"net/http"

	"tuti-server/internal/analysis"
	"tuti-server/internal/catalog"
	tutiv1 "tuti-server/internal/genproto/tutiv1"
	"tuti-server/internal/session"
	"tuti-server/internal/tracing"
)

// handleInitializeSnapAndSolve starts a new Snap & Solve session, always
// beginning with CaptureSnapStep.
func (s *Server) handleInitializeSnapAndSolve(w http.ResponseWriter, r *http.Request) {
	_, span := s.Tracer.StartSpan(r.Context(), "snap.initialize")
	defer span.End()

	var req tutiv1.InitializeSnapAndSolveRequest
	if err := readProto(w, r, &req, defaultMaxBodyBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := s.SnapStore.Init()
	if err != nil {
		span.RecordError(err)
		writeJSONError(w, http.StatusInternalServerError, "failed to start session")
		return
	}

	writeProto(w, http.StatusOK, &tutiv1.InitializeSnapAndSolveResponse{
		SessionId: id,
		NextStep:  &tutiv1.NextStep{Step: &tutiv1.NextStep_CaptureSnap{CaptureSnap: &tutiv1.CaptureSnapStep{}}},
	})
}

// handleSubmitSnap stores the uploaded photo and advances the session to
// CaptureSnapResponseStep with the fixed set of Snap & Solve options.
func (s *Server) handleSubmitSnap(w http.ResponseWriter, r *http.Request) {
	ctx, span := s.Tracer.StartSpan(r.Context(), "snap.submit")
	defer span.End()

	var req tutiv1.SubmitSnapRequest
	if err := readProto(w, r, &req, s.MaxUploadBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.GetSessionId() == "" {
		writeJSONError(w, http.StatusBadRequest, "sessionId is required")
		return
	}
	if len(req.GetData()) == 0 {
		writeJSONError(w, http.StatusBadRequest, "data is required")
		return
	}

	contentType := http.DetectContentType(req.GetData())
	obj, err := s.Store.Save(ctx, req.GetFilename(), contentType, req.GetData())
	if err != nil {
		span.RecordError(err)
		writeJSONError(w, http.StatusInternalServerError, "failed to store upload")
		return
	}

	if err := s.SnapStore.SubmitSnap(req.GetSessionId(), obj.ID); err != nil {
		writeSnapSessionError(w, err)
		return
	}
	span.SetAttributes(tracing.String("session_id", req.GetSessionId()), tracing.String("capture_id", obj.ID))

	writeProto(w, http.StatusOK, &tutiv1.SubmitSnapResult{
		NextStep: &tutiv1.NextStep{Step: &tutiv1.NextStep_CaptureSnapResponse{
			CaptureSnapResponse: &tutiv1.CaptureSnapResponseStep{Options: catalog.SnapOptions},
		}},
	})
}

// handleSubmitSnapResponse runs the analysis matching the student's chosen
// option and returns DisplayAnalysisStep. "check_work" evaluates the shown
// attempt for mistakes; any other option (solve/explain) extracts the
// problem and returns a full solution walk-through with no mistakes.
func (s *Server) handleSubmitSnapResponse(w http.ResponseWriter, r *http.Request) {
	ctx, span := s.Tracer.StartSpan(r.Context(), "snap.submit_response")
	defer span.End()

	var req tutiv1.SubmitSnapResponseRequest
	if err := readProto(w, r, &req, defaultMaxBodyBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.GetSessionId() == "" {
		writeJSONError(w, http.StatusBadRequest, "sessionId is required")
		return
	}
	if req.GetResponseId() == "" {
		writeJSONError(w, http.StatusBadRequest, "responseId is required")
		return
	}

	captureID, err := s.SnapStore.SubmitResponse(req.GetSessionId())
	if err != nil {
		writeSnapSessionError(w, err)
		return
	}

	obj, data, err := s.Store.Get(ctx, captureID)
	if err != nil {
		span.RecordError(err)
		writeJSONError(w, http.StatusInternalServerError, "failed to load capture")
		return
	}
	img := analysis.Image{Bytes: data, ContentType: obj.ContentType}

	step, err := s.buildDisplayAnalysis(ctx, req.GetResponseId(), img)
	if err != nil {
		span.RecordError(err)
		writeJSONError(w, http.StatusBadGateway, "analysis failed")
		return
	}

	writeProto(w, http.StatusOK, &tutiv1.SubmitSnapResponseResult{
		NextStep: &tutiv1.NextStep{Step: &tutiv1.NextStep_DisplayAnalysis{DisplayAnalysis: step}},
	})
}

func (s *Server) buildDisplayAnalysis(ctx context.Context, responseID string, img analysis.Image) (*tutiv1.DisplayAnalysisStep, error) {
	if responseID == "check_work" {
		result, err := s.Analyzer.Evaluate(ctx, []analysis.Image{img})
		if err != nil {
			return nil, err
		}
		lessons, similar := catalog.LessonsAndProblemsForTopic(result.Problem.GetTopic())
		return &tutiv1.DisplayAnalysisStep{
			LessonsToReview: lessons,
			ProblemsCaptured: []*tutiv1.EvaluatedProblem{{
				Problem:         result.Problem,
				Mistakes:        result.Mistakes,
				SimilarProblems: similar,
			}},
		}, nil
	}

	result, err := s.Analyzer.Extract(ctx, []analysis.Image{img})
	if err != nil {
		return nil, err
	}
	if result.Blank {
		lessons, _ := catalog.LessonsAndProblemsForTopic("")
		return &tutiv1.DisplayAnalysisStep{LessonsToReview: lessons}, nil
	}
	lessons, similar := catalog.LessonsAndProblemsForTopic(result.Problem.GetTopic())
	return &tutiv1.DisplayAnalysisStep{
		LessonsToReview: lessons,
		ProblemsCaptured: []*tutiv1.EvaluatedProblem{{
			Problem:         result.Problem,
			SimilarProblems: similar,
		}},
	}, nil
}

func writeSnapSessionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, session.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "session not found")
	case errors.Is(err, session.ErrWrongStep):
		writeJSONError(w, http.StatusConflict, "call is out of order for this session")
	default:
		writeJSONError(w, http.StatusInternalServerError, "session error")
	}
}
