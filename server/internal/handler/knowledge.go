package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/middleware"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type knowledgeOperationalStore interface {
	knowledge.Repository
	knowledge.SearchIndex
}

type knowledgeCapabilityStore interface {
	Capabilities(context.Context) (knowledge.StoreCapabilities, error)
}

type knowledgeHealthProvider interface {
	KnowledgeHealth(context.Context, string) (map[string]any, error)
}

func (h *Handler) ConfigureKnowledge(
	store knowledgeOperationalStore,
	service *knowledge.Service,
	unavailable error,
) {
	h.knowledgeStore = store
	h.knowledgeService = service
	h.knowledgeUnavailable = unavailable
	h.knowledgeEvidenceEnabled = service != nil
}

func (h *Handler) ConfigureKnowledgeEvidence(enabled bool) {
	h.knowledgeEvidenceEnabled = enabled
}

func (h *Handler) ConfigureKnowledgeHealth(provider knowledgeHealthProvider) {
	h.knowledgeHealth = provider
}

func (h *Handler) RegisterKnowledgeRoutes(r chi.Router) {
	r.Get("/", h.listKnowledge)
	r.Get("/search", h.listKnowledge)
	r.Post("/proposals", h.proposeKnowledge)
	r.Get("/candidates", h.listKnowledgeCandidates)
	r.Post("/candidates/{id}/review", h.reviewKnowledgeCandidate)
	r.Get("/health", h.getKnowledgeHealth)
	r.Get("/{id}/revisions", h.listKnowledgeRevisions)
	r.Get("/{id}/sources", h.listKnowledgeSources)
	r.Get("/{id}", h.getKnowledgeEntry)
}

func (h *Handler) requireKnowledge(w http.ResponseWriter) bool {
	if h.knowledgeStore == nil || h.knowledgeService == nil {
		writeError(w, http.StatusServiceUnavailable, "knowledge store unavailable")
		return false
	}
	return true
}

func knowledgeAccess(r *http.Request) (string, db.Member, bool) {
	workspaceID := middleware.WorkspaceIDFromContext(r.Context())
	member, ok := middleware.MemberFromContext(r.Context())
	return workspaceID, member, ok && workspaceID != ""
}

func (h *Handler) listKnowledge(w http.ResponseWriter, r *http.Request) {
	if !h.requireKnowledge(w) {
		return
	}
	workspaceID, _, ok := knowledgeAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "workspace access denied")
		return
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectID != "" && !h.knowledgeProjectInWorkspace(r.Context(), workspaceID, projectID) {
		writeError(w, http.StatusBadRequest, "project not found in workspace")
		return
	}
	query := knowledge.SearchQuery{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		Text:        strings.TrimSpace(r.URL.Query().Get("query")),
		Limit:       parseKnowledgeLimit(r.URL.Query().Get("limit")),
		Cursor:      strings.TrimSpace(r.URL.Query().Get("cursor")),
	}
	if rawKind := strings.TrimSpace(r.URL.Query().Get("kind")); rawKind != "" {
		query.Kinds = []knowledge.Kind{knowledge.Kind(rawKind)}
	}
	page, err := h.knowledgeStore.Search(r.Context(), query)
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
		"entries": items, "total": len(items), "next_cursor": knowledgeNullable(page.NextCursor),
	})
}

func (h *Handler) getKnowledgeEntry(w http.ResponseWriter, r *http.Request) {
	entry, ok := h.resolveKnowledgeEntry(w, r)
	if ok {
		writeJSON(w, http.StatusOK, knowledgeEntryResponse(entry))
	}
}

func (h *Handler) listKnowledgeRevisions(w http.ResponseWriter, r *http.Request) {
	entry, ok := h.resolveKnowledgeEntry(w, r)
	if ok {
		writeJSON(w, http.StatusOK, map[string]any{"knowledge_id": entry.ID, "revisions": entry.Revisions})
	}
}

func (h *Handler) listKnowledgeSources(w http.ResponseWriter, r *http.Request) {
	entry, ok := h.resolveKnowledgeEntry(w, r)
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
	writeJSON(w, http.StatusOK, map[string]any{"knowledge_id": entry.ID, "sources": sources})
}

func (h *Handler) resolveKnowledgeEntry(w http.ResponseWriter, r *http.Request) (knowledge.Entry, bool) {
	if !h.requireKnowledge(w) {
		return knowledge.Entry{}, false
	}
	workspaceID, _, ok := knowledgeAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "workspace access denied")
		return knowledge.Entry{}, false
	}
	entry, err := h.knowledgeStore.GetEntry(r.Context(), workspaceID, chi.URLParam(r, "id"))
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

func (h *Handler) proposeKnowledge(w http.ResponseWriter, r *http.Request) {
	if !h.requireKnowledge(w) {
		return
	}
	workspaceID, member, ok := knowledgeAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "workspace access denied")
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
	if !decodeKnowledgeJSON(w, r, &request) {
		return
	}
	candidate, err := h.knowledgeService.Propose(r.Context(), knowledge.ProposalInput{
		WorkspaceID: workspaceID, ProjectID: request.ProjectID,
		TargetEntryID: request.KnowledgeID, Kind: request.Kind,
		Title: request.Title, Content: request.Content, Reason: request.Reason,
		ProposedBy: util.UUIDToString(member.UserID), SourceRefs: request.SourceRefs,
	})
	if err != nil {
		writeKnowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, knowledgeCandidateResponse(candidate))
}

func (h *Handler) listKnowledgeCandidates(w http.ResponseWriter, r *http.Request) {
	if !h.requireKnowledge(w) {
		return
	}
	workspaceID, member, ok := knowledgeAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "workspace access denied")
		return
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	projectIDs, ok := h.requireKnowledgeReviewer(w, r, workspaceID, member, projectID)
	if !ok {
		return
	}
	query := knowledge.CandidateQuery{
		WorkspaceID: workspaceID, ProjectID: projectID, ProjectIDs: projectIDs,
		Limit:    parseKnowledgeLimit(r.URL.Query().Get("limit")),
		Cursor:   strings.TrimSpace(r.URL.Query().Get("cursor")),
		Statuses: []knowledge.Status{knowledge.StatusCandidate, knowledge.StatusInReview},
	}
	if rawStatus := strings.TrimSpace(r.URL.Query().Get("status")); rawStatus != "" {
		query.Statuses = []knowledge.Status{knowledge.Status(rawStatus)}
	}
	if rawKind := strings.TrimSpace(r.URL.Query().Get("kind")); rawKind != "" {
		query.Kinds = []knowledge.Kind{knowledge.Kind(rawKind)}
	}
	page, err := h.knowledgeStore.ListCandidates(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list knowledge candidates")
		return
	}
	items := make([]map[string]any, 0, len(page.Candidates))
	for _, candidate := range page.Candidates {
		items = append(items, knowledgeCandidateResponse(candidate))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"candidates": items, "total": len(items), "next_cursor": knowledgeNullable(page.NextCursor),
	})
}

func (h *Handler) reviewKnowledgeCandidate(w http.ResponseWriter, r *http.Request) {
	if !h.requireKnowledge(w) {
		return
	}
	workspaceID, member, ok := knowledgeAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "workspace access denied")
		return
	}
	candidate, err := h.knowledgeStore.GetCandidate(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, knowledge.ErrNotFound) || candidate.WorkspaceID != workspaceID {
		writeError(w, http.StatusNotFound, "knowledge record not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load knowledge candidate")
		return
	}
	if _, ok := h.requireKnowledgeReviewer(w, r, workspaceID, member, candidate.ProjectID); !ok {
		return
	}
	var request struct {
		Action           knowledge.ReviewAction `json:"action"`
		ExpectedRevision int64                  `json:"expected_revision"`
		Rationale        string                 `json:"rationale"`
	}
	if !decodeKnowledgeJSON(w, r, &request) {
		return
	}
	reviewed, entry, err := h.knowledgeService.Review(r.Context(), knowledge.ReviewInput{
		WorkspaceID: workspaceID, CandidateID: candidate.ID,
		ExpectedRevision: request.ExpectedRevision, Action: request.Action,
		ReviewerID: util.UUIDToString(member.UserID), Rationale: request.Rationale,
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
		"candidate": knowledgeCandidateResponse(reviewed), "entry": entryResponse,
	})
}

func (h *Handler) getKnowledgeHealth(w http.ResponseWriter, r *http.Request) {
	workspaceID, member, ok := knowledgeAccess(r)
	if !ok || (member.Role != "owner" && member.Role != "admin") {
		writeError(w, http.StatusForbidden, "insufficient workspace role")
		return
	}
	if h.knowledgeStore == nil || h.knowledgeService == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": h.knowledgeUnavailable != nil, "available": false,
			"reason": "knowledge store unavailable",
		})
		return
	}
	capabilityStore, ok := h.knowledgeStore.(knowledgeCapabilityStore)
	if !ok {
		writeError(w, http.StatusInternalServerError, "knowledge capabilities unavailable")
		return
	}
	capabilities, err := capabilityStore.Capabilities(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to inspect knowledge store")
		return
	}
	response := map[string]any{
		"enabled": true, "available": true, "schema_version": capabilities.SchemaVersion,
		"journal_mode": capabilities.JournalMode, "fts5": capabilities.FTS5,
	}
	if h.knowledgeHealth != nil {
		outboxHealth, err := h.knowledgeHealth.KnowledgeHealth(r.Context(), workspaceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to inspect knowledge outbox")
			return
		}
		response["outbox"] = outboxHealth
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) requireKnowledgeReviewer(
	w http.ResponseWriter,
	r *http.Request,
	workspaceID string,
	member db.Member,
	projectID string,
) ([]string, bool) {
	if member.Role == "owner" || member.Role == "admin" {
		return nil, true
	}
	if h.Queries == nil {
		writeError(w, http.StatusForbidden, "insufficient workspace role")
		return nil, false
	}
	workspaceUUID, err := util.ParseUUID(workspaceID)
	if err != nil {
		writeError(w, http.StatusForbidden, "workspace access denied")
		return nil, false
	}
	projects, err := h.Queries.ListProjects(r.Context(), db.ListProjectsParams{WorkspaceID: workspaceUUID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve knowledge review scope")
		return nil, false
	}
	projectIDs := make([]string, 0)
	for _, project := range projects {
		if project.LeadType.String == "member" && project.LeadID == member.UserID {
			ledProjectID := util.UUIDToString(project.ID)
			projectIDs = append(projectIDs, ledProjectID)
			if projectID != "" && projectID == ledProjectID {
				return projectIDs, true
			}
		}
	}
	if projectID == "" && len(projectIDs) > 0 {
		return projectIDs, true
	}
	writeError(w, http.StatusForbidden, "insufficient workspace role")
	return nil, false
}

func (h *Handler) knowledgeProjectInWorkspace(ctx context.Context, workspaceID, projectID string) bool {
	if h.Queries == nil {
		return false
	}
	workspaceUUID, err := util.ParseUUID(workspaceID)
	if err != nil {
		return false
	}
	projectUUID, err := util.ParseUUID(projectID)
	if err != nil {
		return false
	}
	_, err = h.Queries.GetProjectInWorkspace(ctx, db.GetProjectInWorkspaceParams{
		ID: projectUUID, WorkspaceID: workspaceUUID,
	})
	return err == nil
}

func knowledgeCandidateResponse(candidate knowledge.Candidate) map[string]any {
	return map[string]any{
		"id": candidate.ID, "workspace_id": candidate.WorkspaceID,
		"project_id":      knowledgeNullable(candidate.ProjectID),
		"knowledge_id":    knowledgeNullable(candidate.TargetEntryID),
		"target_revision": candidate.TargetRevision, "kind": candidate.Kind,
		"title": candidate.Title, "content": candidate.Content, "reason": candidate.Reason,
		"status": candidate.Status, "revision": candidate.Revision,
		"proposed_by": candidate.ProposedBy, "source_refs": candidate.SourceRefs,
		"created_at": candidate.CreatedAt, "updated_at": candidate.UpdatedAt,
	}
}

func knowledgeEntryResponse(entry knowledge.Entry) map[string]any {
	return map[string]any{
		"id": entry.ID, "workspace_id": entry.WorkspaceID,
		"project_id":   knowledgeNullable(entry.ProjectID),
		"candidate_id": knowledgeNullable(entry.CandidateID),
		"kind":         entry.Kind, "status": entry.Status,
		"current_revision": entry.CurrentRevision, "revisions": entry.Revisions,
		"created_at": entry.CreatedAt, "updated_at": entry.UpdatedAt,
	}
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

func decodeKnowledgeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
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

func knowledgeNullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
