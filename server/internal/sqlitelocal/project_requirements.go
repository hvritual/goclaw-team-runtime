package sqlitelocal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type projectRequirementLinkRequest struct {
	RequirementKey string `json:"requirement_key"`
	IssueID        string `json:"issue_id"`
	Revision       int    `json:"revision"`
}

type projectRequirementCreateIssueRequest struct {
	Revision int `json:"revision"`
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
	request.Content = normalizeRequirementContent(request.Content)
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
	seen := make(map[string]struct{})
	return validRequirementItems(content.Goals, seen) && validRequirementItems(content.InScope, seen) && validRequirementItems(content.OutOfScope, seen) &&
		validRequirementItems(content.Constraints, seen) && validRequirementItems(content.AcceptanceCriteria, seen) && validRequirementItems(content.Dependencies, seen)
}

func normalizeRequirementContent(content projectrequirements.Content) projectrequirements.Content {
	for _, items := range []*[]projectrequirements.Item{&content.Goals, &content.InScope, &content.OutOfScope, &content.Constraints, &content.AcceptanceCriteria, &content.Dependencies} {
		for index := range *items {
			(*items)[index].Key = strings.TrimSpace((*items)[index].Key)
			(*items)[index].Text = strings.TrimSpace((*items)[index].Text)
		}
	}
	return content
}

func validRequirementItems(items []projectrequirements.Item, seen map[string]struct{}) bool {
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

func (s *Server) getProjectRequirementCoverage(w http.ResponseWriter, r *http.Request) {
	projectValue, ok := s.loadProject(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	coverage, err := s.requirementTracking.Coverage(r.Context(), projectValue.WorkspaceID, projectValue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load requirement coverage")
		return
	}
	writeJSON(w, http.StatusOK, coverage)
}

func (s *Server) linkProjectRequirementIssue(w http.ResponseWriter, r *http.Request) {
	projectValue, ok := s.loadProject(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var request projectRequirementLinkRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	err := s.requirementTracking.Link(r.Context(), projectrequirements.LinkInput{WorkspaceID: projectValue.WorkspaceID, ProjectID: projectValue.ID, RequirementKey: strings.TrimSpace(request.RequirementKey), IssueID: strings.TrimSpace(request.IssueID), ActorID: currentUserID(r), Revision: request.Revision})
	if !writeProjectRequirementTrackingError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) unlinkProjectRequirementIssue(w http.ResponseWriter, r *http.Request) {
	projectValue, ok := s.loadProject(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	revision := 0
	if _, err := fmt.Sscanf(r.URL.Query().Get("revision"), "%d", &revision); err != nil {
		writeError(w, http.StatusBadRequest, "valid requirement revision is required")
		return
	}
	err := s.requirementTracking.Unlink(r.Context(), projectrequirements.LinkInput{WorkspaceID: projectValue.WorkspaceID, ProjectID: projectValue.ID, RequirementKey: chi.URLParam(r, "requirementKey"), IssueID: chi.URLParam(r, "issueID"), ActorID: currentUserID(r), Revision: revision})
	if !writeProjectRequirementTrackingError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeProjectRequirementTrackingError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, projectrequirements.ErrInvalidTracking) {
		writeError(w, http.StatusBadRequest, "issue and requirement must belong to this project and revision")
	} else {
		writeError(w, http.StatusInternalServerError, "failed to update requirement issue links")
	}
	return false
}

func (s *Server) createIssueForProjectRequirement(w http.ResponseWriter, r *http.Request) {
	projectValue, ok := s.loadProject(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var request projectRequirementCreateIssueRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	key := strings.TrimSpace(chi.URLParam(r, "requirementKey"))
	section, text, err := s.requirementItem(r.Context(), projectValue, key, request.Revision)
	if err != nil {
		writeError(w, http.StatusBadRequest, "requirement is not available for this revision")
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create issue")
		return
	}
	defer tx.Rollback()
	issueValue, workspaceValue, err := s.createIssueTx(r.Context(), tx, projectValue.WorkspaceID, currentUserID(r), issueRequest{Title: text, Description: ptr("Requirement " + string(section) + ": " + text + " (" + key + ")"), ProjectID: ptr(projectValue.ID), Status: "todo", Priority: "none"})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create issue")
		return
	}
	if err := s.requirementTrackingSQLite.LinkTx(r.Context(), tx, projectrequirements.LinkInput{WorkspaceID: projectValue.WorkspaceID, ProjectID: projectValue.ID, RequirementKey: key, IssueID: issueValue.ID, ActorID: currentUserID(r), Revision: request.Revision}); err != nil {
		writeProjectRequirementTrackingError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create issue")
		return
	}
	s.dispatchKnowledgeEvidence(r.Context())
	writeJSON(w, http.StatusCreated, issueValue.response(workspaceValue.IssuePrefix))
}

func (s *Server) requirementItem(ctx context.Context, projectValue project, key string, revision int) (projectrequirements.TrackableSection, string, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT content FROM project_requirement_revision WHERE baseline_id = (SELECT id FROM project_requirement_baseline WHERE workspace_id = ? AND project_id = ?) AND revision = ?`, projectValue.WorkspaceID, projectValue.ID, revision).Scan(&raw)
	if err != nil {
		return "", "", err
	}
	var content projectrequirements.Content
	if json.Unmarshal([]byte(raw), &content) != nil {
		return "", "", errors.New("invalid requirement content")
	}
	if item, ok := projectrequirements.FindTrackableItem(content, key); ok {
		return item.Section, item.Item.Text, nil
	}
	return "", "", errors.New("untrackable requirement")
}

func ptr(value string) *string { return &value }
