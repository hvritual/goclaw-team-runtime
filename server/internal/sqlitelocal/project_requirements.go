package sqlitelocal

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/projectrequirements"
	"github.com/multica-ai/multica/server/internal/workspacepermissions"
)

type projectRequirementDraftRequest struct {
	ExpectedRevision int                         `json:"expected_revision"`
	Content          projectrequirements.Content `json:"content"`
	ChangeSummary    string                      `json:"change_summary"`
}

type projectRequirementTransitionRequest struct {
	ExpectedRevision int `json:"expected_revision"`
}

func (s *Server) getProjectRequirementBaseline(w http.ResponseWriter, r *http.Request) {
	projectValue, ok := s.loadProject(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	record, err := s.requirements.Get(r.Context(), projectValue.WorkspaceID, projectValue.ID)
	if errors.Is(err, projectrequirements.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"baseline": nil, "history": []any{}})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load project requirements")
		return
	}
	writeJSON(w, http.StatusOK, requirementRecordResponse(record))
}

func (s *Server) listProjectRequirementHistory(w http.ResponseWriter, r *http.Request) {
	projectValue, ok := s.loadProject(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	record, err := s.requirements.Get(r.Context(), projectValue.WorkspaceID, projectValue.ID)
	if errors.Is(err, projectrequirements.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"history": []any{}})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load project requirement history")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": record.History})
}

func (s *Server) saveProjectRequirementDraft(w http.ResponseWriter, r *http.Request) {
	projectValue, ok := s.loadProject(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var request projectRequirementDraftRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.ExpectedRevision < 0 || !validRequirementContent(request.Content) {
		writeError(w, http.StatusBadRequest, "invalid project requirement content")
		return
	}
	record, err := s.requirements.SaveDraft(r.Context(), projectrequirements.SaveDraftInput{
		WorkspaceID: projectValue.WorkspaceID, ProjectID: projectValue.ID, ActorID: currentUserID(r),
		ExpectedRevision: request.ExpectedRevision, Content: request.Content, ChangeSummary: strings.TrimSpace(request.ChangeSummary),
	})
	if !writeProjectRequirementError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, requirementRecordResponse(record))
}

func (s *Server) submitProjectRequirementReview(w http.ResponseWriter, r *http.Request) {
	s.transitionProjectRequirement(w, r, func(input projectrequirements.TransitionInput) (projectrequirements.Record, error) {
		return s.requirements.SubmitReview(r.Context(), input)
	}, false)
}

func (s *Server) approveProjectRequirement(w http.ResponseWriter, r *http.Request) {
	s.transitionProjectRequirement(w, r, func(input projectrequirements.TransitionInput) (projectrequirements.Record, error) {
		return s.requirements.Approve(r.Context(), input)
	}, true)
}

func (s *Server) withdrawProjectRequirementReview(w http.ResponseWriter, r *http.Request) {
	s.transitionProjectRequirement(w, r, func(input projectrequirements.TransitionInput) (projectrequirements.Record, error) {
		return s.requirements.Withdraw(r.Context(), input)
	}, true)
}

func (s *Server) transitionProjectRequirement(w http.ResponseWriter, r *http.Request, transition func(projectrequirements.TransitionInput) (projectrequirements.Record, error), requiresApprover bool) {
	projectValue, ok := s.loadProject(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if requiresApprover && !s.requireProjectRequirementApprover(w, r, projectValue) {
		return
	}
	var request projectRequirementTransitionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.ExpectedRevision < 1 {
		writeError(w, http.StatusBadRequest, "expected revision is required")
		return
	}
	record, err := transition(projectrequirements.TransitionInput{WorkspaceID: projectValue.WorkspaceID, ProjectID: projectValue.ID, ActorID: currentUserID(r), ExpectedRevision: request.ExpectedRevision})
	if !writeProjectRequirementError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, requirementRecordResponse(record))
}

func (s *Server) requireProjectRequirementApprover(w http.ResponseWriter, r *http.Request, projectValue project) bool {
	role, err := workspaceRole(r.Context(), s.db, projectValue.WorkspaceID, currentUserID(r))
	if err != nil {
		writeError(w, http.StatusForbidden, "workspace access denied")
		return false
	}
	if role == workspacepermissions.RoleOwner || role == workspacepermissions.RoleAdmin ||
		(projectValue.LeadType.Valid && projectValue.LeadType.String == "member" && projectValue.LeadID.Valid && projectValue.LeadID.String == currentUserID(r)) {
		return true
	}
	writeError(w, http.StatusForbidden, "project requirement approval requires project lead or workspace administrator")
	return false
}

func validRequirementContent(content projectrequirements.Content) bool {
	return validRequirementItems(content.Goals) && validRequirementItems(content.InScope) && validRequirementItems(content.OutOfScope) &&
		validRequirementItems(content.Constraints) && validRequirementItems(content.AcceptanceCriteria) && validRequirementItems(content.Dependencies)
}

func validRequirementItems(items []projectrequirements.Item) bool {
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		if key == "" || strings.TrimSpace(item.Text) == "" {
			return false
		}
		if _, exists := seen[key]; exists {
			return false
		}
		seen[key] = struct{}{}
	}
	return true
}

func writeProjectRequirementError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, projectrequirements.ErrNotFound):
		writeError(w, http.StatusNotFound, "project requirement baseline not found")
	case errors.Is(err, projectrequirements.ErrRevisionConflict):
		writeError(w, http.StatusConflict, "project requirement revision conflict")
	case errors.Is(err, projectrequirements.ErrInvalidTransition):
		writeError(w, http.StatusConflict, "invalid project requirement transition")
	default:
		writeError(w, http.StatusInternalServerError, "failed to update project requirements")
	}
	return false
}

func requirementRecordResponse(record projectrequirements.Record) map[string]any {
	return map[string]any{"baseline": record.Baseline, "current_content": record.CurrentContent, "effective_content": record.EffectiveContent, "history": record.History}
}
