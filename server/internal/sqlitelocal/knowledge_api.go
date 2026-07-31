package sqlitelocal

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/workspacepermissions"
)

func (s *Server) requireKnowledge(w http.ResponseWriter) bool {
	if s.knowledgeService == nil || s.knowledgeStore == nil {
		writeError(w, http.StatusServiceUnavailable, errKnowledgeDisabled.Error())
		return false
	}
	return true
}

func (s *Server) listKnowledge(w http.ResponseWriter, r *http.Request) {
	if !s.requireKnowledge(w) {
		return
	}
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectID != "" && !s.belongsToWorkspace(r.Context(), "projects", projectID, workspaceValue.ID) {
		writeError(w, http.StatusBadRequest, "project not found in workspace")
		return
	}
	limit := parseKnowledgeLimit(r.URL.Query().Get("limit"))
	query := knowledge.SearchQuery{
		WorkspaceID: workspaceValue.ID,
		ProjectID:   projectID,
		Text:        strings.TrimSpace(r.URL.Query().Get("query")),
		Limit:       limit,
		Cursor:      strings.TrimSpace(r.URL.Query().Get("cursor")),
	}
	if rawKind := strings.TrimSpace(r.URL.Query().Get("kind")); rawKind != "" {
		query.Kinds = []knowledge.Kind{knowledge.Kind(rawKind)}
	}
	page, err := s.knowledgeStore.Search(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to search knowledge")
		return
	}
	items := make([]map[string]any, 0, len(page.Results))
	for _, result := range page.Results {
		response := knowledgeEntryResponse(result.Entry)
		response["score"] = result.Score
		response["matched_by"] = result.MatchedBy
		response["citation"] = result.Citation
		items = append(items, response)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"entries":     items,
		"total":       len(items),
		"next_cursor": nullable(page.NextCursor),
	})
}

func (s *Server) getKnowledgeEntry(w http.ResponseWriter, r *http.Request) {
	if !s.requireKnowledge(w) {
		return
	}
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	entry, err := s.knowledgeStore.GetEntry(r.Context(), workspaceValue.ID, chi.URLParam(r, "id"))
	if errors.Is(err, knowledge.ErrNotFound) {
		writeError(w, http.StatusNotFound, "knowledge entry not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load knowledge")
		return
	}
	writeJSON(w, http.StatusOK, knowledgeEntryResponse(entry))
}

func (s *Server) listKnowledgeRevisions(w http.ResponseWriter, r *http.Request) {
	entry, ok := s.resolvePublishedKnowledgeEntry(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"knowledge_id": entry.ID,
		"revisions":    entry.Revisions,
	})
}

func (s *Server) listKnowledgeSources(w http.ResponseWriter, r *http.Request) {
	entry, ok := s.resolvePublishedKnowledgeEntry(w, r)
	if !ok {
		return
	}
	sources := make([]knowledge.SourceRef, 0)
	seen := make(map[string]struct{})
	for _, revision := range entry.Revisions {
		for _, source := range revision.SourceRefs {
			key := source.Type + "\x00" + source.ID + "\x00" + source.Revision
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			sources = append(sources, source)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"knowledge_id": entry.ID,
		"sources":      sources,
	})
}

func (s *Server) resolvePublishedKnowledgeEntry(
	w http.ResponseWriter,
	r *http.Request,
) (knowledge.Entry, bool) {
	if !s.requireKnowledge(w) {
		return knowledge.Entry{}, false
	}
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return knowledge.Entry{}, false
	}
	entry, err := s.knowledgeStore.GetEntry(r.Context(), workspaceValue.ID, chi.URLParam(r, "id"))
	if errors.Is(err, knowledge.ErrNotFound) {
		writeError(w, http.StatusNotFound, "knowledge entry not found")
		return knowledge.Entry{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load knowledge")
		return knowledge.Entry{}, false
	}
	return entry, true
}

func (s *Server) proposeKnowledge(w http.ResponseWriter, r *http.Request) {
	if !s.requireKnowledge(w) {
		return
	}
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	var request struct {
		ProjectID   string                `json:"project_id"`
		KnowledgeID string                `json:"knowledge_id"`
		Kind        knowledge.Kind        `json:"kind"`
		Title       string                `json:"title"`
		Content     string                `json:"content"`
		Reason      string                `json:"reason"`
		SourceRefs  []knowledge.SourceRef `json:"source_refs"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	request.ProjectID = strings.TrimSpace(request.ProjectID)
	if request.ProjectID != "" &&
		!s.belongsToWorkspace(r.Context(), "projects", request.ProjectID, workspaceValue.ID) {
		writeError(w, http.StatusBadRequest, "project not found in workspace")
		return
	}
	candidate, err := s.knowledgeService.Propose(r.Context(), knowledge.ProposalInput{
		WorkspaceID:   workspaceValue.ID,
		ProjectID:     request.ProjectID,
		TargetEntryID: strings.TrimSpace(request.KnowledgeID),
		Kind:          request.Kind,
		Title:         request.Title,
		Content:       request.Content,
		Reason:        request.Reason,
		ProposedBy:    currentUserID(r),
		SourceRefs:    request.SourceRefs,
	})
	if err != nil {
		writeKnowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, knowledgeCandidateResponse(candidate))
}

func (s *Server) listKnowledgeCandidates(w http.ResponseWriter, r *http.Request) {
	if !s.requireKnowledge(w) {
		return
	}
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	projectIDs, ok := s.requireKnowledgeReviewer(w, r, workspaceValue.ID, projectID)
	if !ok {
		return
	}
	query := knowledge.CandidateQuery{
		WorkspaceID: workspaceValue.ID,
		ProjectID:   projectID,
		ProjectIDs:  projectIDs,
		Limit:       parseKnowledgeLimit(r.URL.Query().Get("limit")),
		Cursor:      strings.TrimSpace(r.URL.Query().Get("cursor")),
	}
	if rawStatus := strings.TrimSpace(r.URL.Query().Get("status")); rawStatus != "" {
		query.Statuses = []knowledge.Status{knowledge.Status(rawStatus)}
	} else {
		query.Statuses = []knowledge.Status{
			knowledge.StatusCandidate,
			knowledge.StatusInReview,
		}
	}
	if rawKind := strings.TrimSpace(r.URL.Query().Get("kind")); rawKind != "" {
		query.Kinds = []knowledge.Kind{knowledge.Kind(rawKind)}
	}
	page, err := s.knowledgeStore.ListCandidates(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list knowledge candidates")
		return
	}
	items := make([]map[string]any, 0, len(page.Candidates))
	for _, candidate := range page.Candidates {
		items = append(items, knowledgeCandidateResponse(candidate))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"candidates":  items,
		"total":       len(items),
		"next_cursor": nullable(page.NextCursor),
	})
}

func (s *Server) reviewKnowledgeCandidate(w http.ResponseWriter, r *http.Request) {
	if !s.requireKnowledge(w) {
		return
	}
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	candidate, err := s.knowledgeStore.GetCandidate(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, knowledge.ErrNotFound) || candidate.WorkspaceID != workspaceValue.ID {
		writeError(w, http.StatusNotFound, "knowledge record not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load knowledge candidate")
		return
	}
	if _, ok := s.requireKnowledgeReviewer(w, r, workspaceValue.ID, candidate.ProjectID); !ok {
		return
	}
	var request struct {
		Action           knowledge.ReviewAction `json:"action"`
		ExpectedRevision int64                  `json:"expected_revision"`
		Rationale        string                 `json:"rationale"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	candidate, entry, err := s.knowledgeService.Review(r.Context(), knowledge.ReviewInput{
		WorkspaceID:      workspaceValue.ID,
		CandidateID:      chi.URLParam(r, "id"),
		ExpectedRevision: request.ExpectedRevision,
		Action:           request.Action,
		ReviewerID:       currentUserID(r),
		Rationale:        request.Rationale,
	})
	if err != nil {
		writeKnowledgeError(w, err)
		return
	}
	var entryResponse any
	if entry != nil {
		entryResponse = knowledgeEntryResponse(*entry)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"candidate": knowledgeCandidateResponse(candidate),
		"entry":     entryResponse,
	})
}

func (s *Server) getKnowledgeHealth(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	if !s.requireWorkspaceRole(
		w,
		r,
		workspaceValue.ID,
		workspacepermissions.RoleOwner,
		workspacepermissions.RoleAdmin,
	) {
		return
	}
	if s.knowledgeService == nil || s.knowledgeStore == nil {
		reason := "knowledge store unavailable"
		if errors.Is(s.knowledgeUnavailable, errKnowledgeDisabled) {
			reason = "knowledge store disabled"
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled":   !errors.Is(s.knowledgeUnavailable, errKnowledgeDisabled),
			"available": false,
			"reason":    reason,
		})
		return
	}
	capabilities, err := s.knowledgeStore.Capabilities(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to inspect knowledge store")
		return
	}
	outboxStats, err := s.knowledgeOutboxStats(r.Context(), workspaceValue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to inspect knowledge outbox")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":        true,
		"available":      true,
		"schema_version": capabilities.SchemaVersion,
		"journal_mode":   capabilities.JournalMode,
		"fts5":           capabilities.FTS5,
		"outbox":         outboxStats,
	})
}

func knowledgeCandidateResponse(candidate knowledge.Candidate) map[string]any {
	return map[string]any{
		"id":              candidate.ID,
		"workspace_id":    candidate.WorkspaceID,
		"project_id":      nullable(candidate.ProjectID),
		"knowledge_id":    nullable(candidate.TargetEntryID),
		"target_revision": candidate.TargetRevision,
		"kind":            candidate.Kind,
		"title":           candidate.Title,
		"content":         candidate.Content,
		"reason":          candidate.Reason,
		"status":          candidate.Status,
		"revision":        candidate.Revision,
		"proposed_by":     candidate.ProposedBy,
		"source_refs":     candidate.SourceRefs,
		"created_at":      candidate.CreatedAt,
		"updated_at":      candidate.UpdatedAt,
	}
}

func knowledgeEntryResponse(entry knowledge.Entry) map[string]any {
	return map[string]any{
		"id":               entry.ID,
		"workspace_id":     entry.WorkspaceID,
		"project_id":       nullable(entry.ProjectID),
		"candidate_id":     nullable(entry.CandidateID),
		"kind":             entry.Kind,
		"status":           entry.Status,
		"current_revision": entry.CurrentRevision,
		"revisions":        entry.Revisions,
		"created_at":       entry.CreatedAt,
		"updated_at":       entry.UpdatedAt,
	}
}

func parseKnowledgeLimit(value string) int {
	limit, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func writeKnowledgeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, knowledge.ErrNotFound):
		writeError(w, http.StatusNotFound, "knowledge record not found")
	case errors.Is(err, knowledge.ErrRevisionConflict):
		writeError(w, http.StatusConflict, "knowledge revision conflict")
	case errors.Is(err, knowledge.ErrWorkspaceMismatch):
		writeError(w, http.StatusForbidden, "knowledge access denied")
	case errors.Is(err, knowledge.ErrProjectScope):
		writeError(w, http.StatusBadRequest, "project not found in workspace")
	case errors.Is(err, knowledge.ErrReasonRequired),
		errors.Is(err, knowledge.ErrRationaleRequired),
		errors.Is(err, knowledge.ErrReviewerRequired),
		errors.Is(err, knowledge.ErrInvalidProposal),
		errors.Is(err, knowledge.ErrRevisionTarget),
		errors.Is(err, knowledge.ErrInvalidReview):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "knowledge operation failed")
	}
}
