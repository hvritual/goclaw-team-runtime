package sqlitelocal

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/knowledge"
)

type acceptanceConclusionRequest struct {
	Result       string   `json:"result"`
	Rationale    string   `json:"rationale"`
	EvidenceRefs []string `json:"evidence_refs"`
}

func (r *acceptanceConclusionRequest) validate() error {
	r.Result = strings.TrimSpace(r.Result)
	r.Rationale = strings.TrimSpace(r.Rationale)
	switch r.Result {
	case "accepted", "conditional", "rejected":
	default:
		return errors.New("result must be accepted, conditional, or rejected")
	}
	if r.Rationale == "" {
		return errors.New("rationale is required")
	}
	r.EvidenceRefs = cleanStrings(r.EvidenceRefs)
	return nil
}

type issueAcceptanceConclusion struct {
	ID, WorkspaceID, IssueID, Result, Rationale, ActorID, CreatedAt, UpdatedAt string
	EvidenceRefs                                                               []string
}

func (v issueAcceptanceConclusion) response() map[string]any {
	return map[string]any{
		"id": v.ID, "workspace_id": v.WorkspaceID, "issue_id": v.IssueID,
		"result": v.Result, "rationale": v.Rationale, "evidence_refs": v.EvidenceRefs,
		"actor_id": v.ActorID, "created_at": v.CreatedAt, "updated_at": v.UpdatedAt,
	}
}

func insertIssueAcceptanceConclusion(
	ctx context.Context,
	tx *sql.Tx,
	issueValue issue,
	actorID string,
	request acceptanceConclusionRequest,
) (issueAcceptanceConclusion, error) {
	timestamp := now()
	value := issueAcceptanceConclusion{
		ID: newID(), WorkspaceID: issueValue.WorkspaceID, IssueID: issueValue.ID,
		Result: request.Result, Rationale: request.Rationale, EvidenceRefs: request.EvidenceRefs,
		ActorID: actorID, CreatedAt: timestamp, UpdatedAt: timestamp,
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO issue_acceptance_conclusion(
		id, workspace_id, issue_id, result, rationale, evidence_refs, actor_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, value.ID, value.WorkspaceID, value.IssueID,
		value.Result, value.Rationale, encodeJSON(value.EvidenceRefs, "[]"), value.ActorID,
		value.CreatedAt, value.UpdatedAt)
	return value, err
}

func (s *Server) createIssueAcceptanceConclusion(w http.ResponseWriter, r *http.Request) {
	issueValue, _, ok := s.loadIssue(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if issueValue.Status != "done" {
		writeError(w, http.StatusConflict, "issue must be done before recording an acceptance conclusion")
		return
	}
	var request acceptanceConclusionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := request.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record acceptance conclusion")
		return
	}
	defer tx.Rollback()
	value, err := insertIssueAcceptanceConclusion(r.Context(), tx, issueValue, currentUserID(r), request)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record acceptance conclusion")
		return
	}
	if err := enqueueKnowledgeEvidence(r.Context(), tx, acceptanceConclusionEvidence(issueValue, value)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record acceptance evidence")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record acceptance conclusion")
		return
	}
	s.dispatchKnowledgeEvidence(r.Context())
	writeJSON(w, http.StatusCreated, value.response())
}

func (s *Server) listIssueAcceptanceConclusions(w http.ResponseWriter, r *http.Request) {
	issueValue, _, ok := s.loadIssue(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT id, workspace_id, issue_id, result, rationale,
		evidence_refs, actor_id, created_at, updated_at FROM issue_acceptance_conclusion
		WHERE issue_id = ? ORDER BY created_at DESC, id DESC`, issueValue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list acceptance conclusions")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var value issueAcceptanceConclusion
		var refs string
		if err := rows.Scan(&value.ID, &value.WorkspaceID, &value.IssueID, &value.Result,
			&value.Rationale, &refs, &value.ActorID, &value.CreatedAt, &value.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list acceptance conclusions")
			return
		}
		value.EvidenceRefs = decodeStrings(refs)
		items = append(items, value.response())
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list acceptance conclusions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"acceptance_conclusions": items, "total": len(items)})
}

type retrospectiveRequest struct {
	Summary      string   `json:"summary"`
	Successes    []string `json:"successes"`
	Problems     []string `json:"problems"`
	Lessons      []string `json:"lessons"`
	FollowUpRefs []string `json:"follow_up_refs"`
}

func (r *retrospectiveRequest) validate() error {
	r.Summary = strings.TrimSpace(r.Summary)
	r.Successes = cleanStrings(r.Successes)
	r.Problems = cleanStrings(r.Problems)
	r.Lessons = cleanStrings(r.Lessons)
	r.FollowUpRefs = cleanStrings(r.FollowUpRefs)
	if r.Summary == "" {
		return errors.New("summary is required")
	}
	if len(r.Lessons) == 0 {
		return errors.New("at least one lesson is required")
	}
	return nil
}

type projectRetrospective struct {
	ID, WorkspaceID, ProjectID, Summary, ActorID, CreatedAt, UpdatedAt string
	Successes, Problems, Lessons, FollowUpRefs                         []string
}

func (v projectRetrospective) response() map[string]any {
	return map[string]any{
		"id": v.ID, "workspace_id": v.WorkspaceID, "project_id": v.ProjectID,
		"summary": v.Summary, "successes": v.Successes, "problems": v.Problems,
		"lessons": v.Lessons, "follow_up_refs": v.FollowUpRefs, "actor_id": v.ActorID,
		"created_at": v.CreatedAt, "updated_at": v.UpdatedAt,
	}
}

func (s *Server) createProjectRetrospective(w http.ResponseWriter, r *http.Request) {
	projectValue, ok := s.loadProject(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var request retrospectiveRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := request.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	timestamp := now()
	value := projectRetrospective{
		ID: newID(), WorkspaceID: projectValue.WorkspaceID, ProjectID: projectValue.ID,
		Summary: request.Summary, Successes: request.Successes, Problems: request.Problems,
		Lessons: request.Lessons, FollowUpRefs: request.FollowUpRefs,
		ActorID: currentUserID(r), CreatedAt: timestamp, UpdatedAt: timestamp,
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record retrospective")
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), `INSERT INTO project_retrospective(
		id, workspace_id, project_id, summary, successes, problems, lessons, follow_up_refs,
		actor_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, value.ID, value.WorkspaceID, value.ProjectID,
		value.Summary, encodeJSON(value.Successes, "[]"), encodeJSON(value.Problems, "[]"),
		encodeJSON(value.Lessons, "[]"), encodeJSON(value.FollowUpRefs, "[]"), value.ActorID,
		value.CreatedAt, value.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record retrospective")
		return
	}
	if err := enqueueKnowledgeEvidence(r.Context(), tx, retrospectiveEvidence(projectValue, value)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record retrospective evidence")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record retrospective")
		return
	}
	s.dispatchKnowledgeEvidence(r.Context())
	writeJSON(w, http.StatusCreated, value.response())
}

func (s *Server) listProjectRetrospectives(w http.ResponseWriter, r *http.Request) {
	projectValue, ok := s.loadProject(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT id, workspace_id, project_id, summary,
		successes, problems, lessons, follow_up_refs, actor_id, created_at, updated_at
		FROM project_retrospective WHERE project_id = ? ORDER BY created_at DESC, id DESC`, projectValue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list retrospectives")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var value projectRetrospective
		var successes, problems, lessons, followUps string
		if err := rows.Scan(&value.ID, &value.WorkspaceID, &value.ProjectID, &value.Summary,
			&successes, &problems, &lessons, &followUps, &value.ActorID, &value.CreatedAt,
			&value.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list retrospectives")
			return
		}
		value.Successes, value.Problems = decodeStrings(successes), decodeStrings(problems)
		value.Lessons, value.FollowUpRefs = decodeStrings(lessons), decodeStrings(followUps)
		items = append(items, value.response())
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list retrospectives")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"retrospectives": items, "total": len(items)})
}

func acceptanceConclusionEvidence(issueValue issue, value issueAcceptanceConclusion) knowledge.Evidence {
	content := fmt.Sprintf("Result: %s\n\n%s", value.Result, value.Rationale)
	if len(value.EvidenceRefs) > 0 {
		content += "\n\nEvidence:\n- " + strings.Join(value.EvidenceRefs, "\n- ")
	}
	return structuredEvidence(value.WorkspaceID, optionalString(issueValue.ProjectID),
		"acceptance_conclusion", issueValue.ID, issueValue.UpdatedAt, value.CreatedAt, "issue.acceptance_concluded",
		knowledge.KindRequirement, "Acceptance: "+issueValue.Title, content, value.ActorID, true)
}

func retrospectiveEvidence(projectValue project, value projectRetrospective) knowledge.Evidence {
	content := value.Summary
	if len(value.Lessons) > 0 {
		content += "\n\nLessons:\n- " + strings.Join(value.Lessons, "\n- ")
	}
	if len(value.Successes) > 0 {
		content += "\n\nSuccesses:\n- " + strings.Join(value.Successes, "\n- ")
	}
	if len(value.Problems) > 0 {
		content += "\n\nProblems:\n- " + strings.Join(value.Problems, "\n- ")
	}
	if len(value.FollowUpRefs) > 0 {
		content += "\n\nFollow-ups:\n- " + strings.Join(value.FollowUpRefs, "\n- ")
	}
	return structuredEvidence(value.WorkspaceID, value.ProjectID, "retrospective", value.ID,
		value.UpdatedAt, value.CreatedAt, "project.retrospective_recorded", knowledge.KindLesson,
		"Retrospective: "+projectValue.Title, content, value.ActorID, true)
}

func structuredEvidence(workspaceID, projectID, sourceType, sourceID, revision, occurredAtValue, eventType string,
	kind knowledge.Kind, title, content, actorID string, terminal bool) knowledge.Evidence {
	checksum := sha256.Sum256([]byte(content))
	occurredAt, err := time.Parse(time.RFC3339Nano, occurredAtValue)
	if err != nil {
		occurredAt = time.Now().UTC()
	}
	return knowledge.NewEvidence(knowledge.EvidenceDraft{
		WorkspaceID: workspaceID, ProjectID: projectID, SourceType: sourceType, SourceID: sourceID,
		SourceRevision: fmt.Sprintf("%s@sha256:%x", revision, checksum), EventType: eventType,
		Kind: kind, Title: title, Content: content, ActorID: actorID, OccurredAt: occurredAt,
		Terminal: terminal,
	})
}

func cleanStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func decodeStrings(value string) []string {
	var decoded []string
	if json.Unmarshal([]byte(value), &decoded) != nil || decoded == nil {
		return []string{}
	}
	return decoded
}
