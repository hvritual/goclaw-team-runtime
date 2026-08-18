package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type knowledgeReviewRepository struct{ db *sql.DB }

func NewKnowledgeReviewRepository(c Config) (application.KnowledgeReviewRepository, error) {
	if c.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &knowledgeReviewRepository{db: c.DB}, nil
}

func (r *knowledgeReviewRepository) CreateKnowledgeCandidate(ctx context.Context, command application.CreateKnowledgeCandidateCommand) (application.CreatedKnowledgeCandidate, error) {
	tx, err := r.db.Conn(ctx)
	if err != nil {
		return application.CreatedKnowledgeCandidate{}, err
	}
	defer tx.Close()
	if _, err = tx.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return application.CreatedKnowledgeCandidate{}, err
	}
	if _, err = tx.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return application.CreatedKnowledgeCandidate{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = tx.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()
	var hash, body string
	err = tx.QueryRowContext(ctx, `SELECT request_hash,response_body FROM workspace_mutation_idempotency WHERE workspace_id=? AND action='workspace.knowledge.propose' AND idempotency_key=?`, command.Candidate.WorkspaceID, command.IdempotencyKey).Scan(&hash, &body)
	if err == nil {
		if hash != command.RequestHash {
			return application.CreatedKnowledgeCandidate{}, contract.ErrKnowledgeIdempotencyConflict
		}
		var replay contract.KnowledgeCandidate
		if json.Unmarshal([]byte(body), &replay) != nil {
			return application.CreatedKnowledgeCandidate{}, errors.New("invalid Knowledge proposal replay")
		}
		return application.CreatedKnowledgeCandidate{Candidate: replay, Replayed: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return application.CreatedKnowledgeCandidate{}, err
	}
	c := command.Candidate
	if c.KnowledgeID != nil {
		var revision int
		if err := tx.QueryRowContext(ctx, `SELECT current_revision FROM workspace_governed_knowledge WHERE workspace_id=? AND id=? AND status='published'`, c.WorkspaceID, *c.KnowledgeID).Scan(&revision); errors.Is(err, sql.ErrNoRows) {
			return application.CreatedKnowledgeCandidate{}, contract.ErrInvalidKnowledgeReview
		} else if err != nil {
			return application.CreatedKnowledgeCandidate{}, err
		} else {
			c.TargetRevision = revision
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO workspace_knowledge_candidates(id,workspace_id,project_id,knowledge_id,target_revision,kind,title,content,reason,status,revision,proposed_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, c.ID, c.WorkspaceID, c.ProjectID, c.KnowledgeID, c.TargetRevision, c.Kind, c.Title, c.Content, c.Reason, c.Status, c.Revision, c.ProposedBy, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return application.CreatedKnowledgeCandidate{}, err
	}
	for i, source := range c.SourceRefs {
		if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_knowledge_candidate_sources(candidate_id,ordinal,source_type,source_id,source_revision,citation,asset_id,asset_version_id) VALUES(?,?,?,?,?,?,?,?)`, c.ID, i, source.Type, source.ID, source.Revision, source.Citation, source.AssetID, source.AssetVersionID); err != nil {
			return application.CreatedKnowledgeCandidate{}, err
		}
	}
	metadata, _ := json.Marshal(map[string]any{"version": "knowledge-review-v1", "status": "candidate", "kind": c.Kind, "target_revision": c.TargetRevision})
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_audit_entries(workspace_id,occurred_at,id,actor_type,actor_id,action,resource_kind,resource_id,resource_revision,request_id,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, c.WorkspaceID, c.CreatedAt, command.AuditID, "member", c.ProposedBy, "workspace.knowledge.propose", "knowledge_candidate", c.ID, 1, command.IdempotencyKey, string(metadata)); err != nil {
		return application.CreatedKnowledgeCandidate{}, err
	}
	response, _ := json.Marshal(c)
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_mutation_idempotency(workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,resource_revision,response_status,response_body,created_at,expires_at) VALUES(?,'workspace.knowledge.propose',?,?,?,?,?,201,?,?,NULL)`, c.WorkspaceID, command.IdempotencyKey, command.RequestHash, "knowledge_candidate", c.ID, 1, string(response), c.CreatedAt); err != nil {
		return application.CreatedKnowledgeCandidate{}, err
	}
	if _, err = tx.ExecContext(ctx, `COMMIT`); err != nil {
		return application.CreatedKnowledgeCandidate{}, err
	}
	committed = true
	return application.CreatedKnowledgeCandidate{Candidate: c}, nil
}

func (r *knowledgeReviewRepository) ListKnowledgeCandidates(ctx context.Context, workspaceID string) ([]contract.KnowledgeCandidate, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,workspace_id,project_id,knowledge_id,target_revision,kind,title,content,reason,status,revision,proposed_by,created_at,updated_at FROM workspace_knowledge_candidates WHERE workspace_id=? AND status IN ('candidate','in_review','quarantined') ORDER BY updated_at DESC,id ASC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []contract.KnowledgeCandidate
	for rows.Next() {
		c, scanErr := scanKnowledgeCandidate(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		sources, loadErr := loadCandidateSources(ctx, r.db, c.ID)
		if loadErr != nil {
			return nil, loadErr
		}
		c.SourceRefs = sources
		result = append(result, c)
	}
	if result == nil {
		result = []contract.KnowledgeCandidate{}
	}
	return result, rows.Err()
}

type candidateScanner interface{ Scan(...any) error }

func scanKnowledgeCandidate(row candidateScanner) (contract.KnowledgeCandidate, error) {
	var c contract.KnowledgeCandidate
	var projectID, knowledgeID sql.NullString
	err := row.Scan(&c.ID, &c.WorkspaceID, &projectID, &knowledgeID, &c.TargetRevision, &c.Kind, &c.Title, &c.Content, &c.Reason, &c.Status, &c.Revision, &c.ProposedBy, &c.CreatedAt, &c.UpdatedAt)
	if projectID.Valid {
		c.ProjectID = &projectID.String
	}
	if knowledgeID.Valid {
		c.KnowledgeID = &knowledgeID.String
	}
	return c, err
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func loadCandidateSources(ctx context.Context, q queryer, id string) ([]contract.KnowledgeSourceRef, error) {
	rows, err := q.QueryContext(ctx, `SELECT source_type,source_id,source_revision,citation,asset_id,asset_version_id FROM workspace_knowledge_candidate_sources WHERE candidate_id=? ORDER BY ordinal`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []contract.KnowledgeSourceRef{}
	for rows.Next() {
		var s contract.KnowledgeSourceRef
		var asset, version sql.NullString
		if err := rows.Scan(&s.Type, &s.ID, &s.Revision, &s.Citation, &asset, &version); err != nil {
			return nil, err
		}
		if asset.Valid {
			s.AssetID = &asset.String
			s.AssetVersionID = &version.String
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *knowledgeReviewRepository) ReviewKnowledgeCandidate(ctx context.Context, command application.ReviewKnowledgeCandidateCommand) (contract.ReviewKnowledgeResponse, error) {
	tx, err := r.db.Conn(ctx)
	if err != nil {
		return contract.ReviewKnowledgeResponse{}, err
	}
	defer tx.Close()
	if _, err = tx.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return contract.ReviewKnowledgeResponse{}, err
	}
	if _, err = tx.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return contract.ReviewKnowledgeResponse{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = tx.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()
	c, err := scanKnowledgeCandidate(tx.QueryRowContext(ctx, `SELECT id,workspace_id,project_id,knowledge_id,target_revision,kind,title,content,reason,status,revision,proposed_by,created_at,updated_at FROM workspace_knowledge_candidates WHERE workspace_id=? AND id=?`, command.WorkspaceID, command.CandidateID))
	if errors.Is(err, sql.ErrNoRows) {
		return contract.ReviewKnowledgeResponse{}, contract.ErrKnowledgeCandidateNotFound
	}
	if err != nil {
		return contract.ReviewKnowledgeResponse{}, err
	}
	if c.Revision != command.ExpectedRevision {
		return contract.ReviewKnowledgeResponse{}, &contract.KnowledgeRevisionConflictError{Resource: "candidate", CurrentRevision: c.Revision}
	}
	if c.ProposedBy == command.ActorID && !command.AllowSelfReview {
		return contract.ReviewKnowledgeResponse{}, contract.ErrKnowledgeSelfReview
	}
	next, terminal, valid := knowledgeTransition(c.Status, command.Action, c.KnowledgeID != nil)
	if !valid {
		return contract.ReviewKnowledgeResponse{}, contract.ErrInvalidKnowledgeReview
	}
	sources, err := loadCandidateSources(ctx, tx, c.ID)
	if err != nil {
		return contract.ReviewKnowledgeResponse{}, err
	}
	c.SourceRefs = sources
	if terminal && len(sources) == 0 {
		return contract.ReviewKnowledgeResponse{}, contract.ErrInvalidKnowledgeReview
	}
	if terminal && command.ValidateSources != nil {
		if err := command.ValidateSources(ctx, c.WorkspaceID, sources); err != nil {
			return contract.ReviewKnowledgeResponse{}, err
		}
	}
	var entry *contract.GovernedKnowledgeEntry
	if terminal {
		entry, err = r.applyKnowledgePublication(ctx, tx, c, command)
		if err != nil {
			return contract.ReviewKnowledgeResponse{}, err
		}
	}
	updated := command.OccurredAt.UTC().Format(time.RFC3339Nano)
	newRevision := c.Revision + 1
	result, err := tx.ExecContext(ctx, `UPDATE workspace_knowledge_candidates SET status=?,revision=?,updated_at=? WHERE workspace_id=? AND id=? AND revision=?`, next, newRevision, updated, c.WorkspaceID, c.ID, c.Revision)
	if err != nil {
		return contract.ReviewKnowledgeResponse{}, err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return contract.ReviewKnowledgeResponse{}, &contract.KnowledgeRevisionConflictError{Resource: "candidate", CurrentRevision: c.Revision}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_knowledge_review_events(candidate_id,candidate_revision,action,actor_id,rationale,emergency,occurred_at) VALUES(?,?,?,?,?,?,?)`, c.ID, newRevision, command.Action, command.ActorID, command.Rationale, command.Emergency, updated); err != nil {
		return contract.ReviewKnowledgeResponse{}, err
	}
	metadata, _ := json.Marshal(map[string]any{"version": "knowledge-review-v1", "action": command.Action, "status": next, "emergency": command.Emergency})
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_audit_entries(workspace_id,occurred_at,id,actor_type,actor_id,action,resource_kind,resource_id,resource_revision,request_id,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, c.WorkspaceID, updated, command.AuditID, "member", command.ActorID, "workspace.knowledge."+command.Action, "knowledge_candidate", c.ID, newRevision, command.AuditID, string(metadata)); err != nil {
		return contract.ReviewKnowledgeResponse{}, err
	}
	c.Status, c.Revision, c.UpdatedAt = next, newRevision, updated
	if _, err = tx.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.ReviewKnowledgeResponse{}, err
	}
	committed = true
	return contract.ReviewKnowledgeResponse{Candidate: c, Entry: entry}, nil
}

func knowledgeTransition(status, action string, target bool) (string, bool, bool) {
	switch {
	case status == "candidate" && action == "approve":
		return "in_review", false, true
	case status == "in_review" && action == "reject":
		return "rejected", false, true
	case status == "in_review" && action == "quarantine":
		return "quarantined", false, true
	case status == "quarantined" && action == "return":
		return "in_review", false, true
	case status == "in_review" && action == "publish" && !target:
		return "published", true, true
	case status == "in_review" && action == "supersede" && target:
		return "published", true, true
	case status == "in_review" && action == "invalidate" && target:
		return "published", true, true
	default:
		return "", false, false
	}
}

type knowledgeMutationConnection interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (r *knowledgeReviewRepository) applyKnowledgePublication(ctx context.Context, tx knowledgeMutationConnection, c contract.KnowledgeCandidate, command application.ReviewKnowledgeCandidateCommand) (*contract.GovernedKnowledgeEntry, error) {
	when := command.OccurredAt.UTC().Format(time.RFC3339Nano)
	knowledgeID := command.PublicationID
	if command.Action == "supersede" || command.Action == "invalidate" {
		var current int
		var status string
		if err := tx.QueryRowContext(ctx, `SELECT current_revision,status FROM workspace_governed_knowledge WHERE workspace_id=? AND id=?`, c.WorkspaceID, *c.KnowledgeID).Scan(&current, &status); errors.Is(err, sql.ErrNoRows) {
			return nil, &contract.KnowledgeRevisionConflictError{Resource: "knowledge", CurrentRevision: 0}
		} else if err != nil {
			return nil, err
		}
		if current != c.TargetRevision || status != "published" {
			return nil, &contract.KnowledgeRevisionConflictError{Resource: "knowledge", CurrentRevision: current}
		}
		nextStatus := "superseded"
		if command.Action == "invalidate" {
			nextStatus = "invalidated"
		}
		result, err := tx.ExecContext(ctx, `UPDATE workspace_governed_knowledge SET status=?,updated_at=? WHERE workspace_id=? AND id=? AND current_revision=? AND status='published'`, nextStatus, when, c.WorkspaceID, *c.KnowledgeID, c.TargetRevision)
		if err != nil {
			return nil, err
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			return nil, &contract.KnowledgeRevisionConflictError{Resource: "knowledge", CurrentRevision: current}
		}
		if command.Action == "invalidate" {
			knowledgeID = ""
		}
	}
	var projected *contract.GovernedKnowledgeEntry
	if command.Action != "invalidate" {
		if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_governed_knowledge(id,workspace_id,project_id,candidate_id,kind,status,current_revision,created_at,updated_at) VALUES(?,?,?,?,?,'published',1,?,?)`, knowledgeID, c.WorkspaceID, c.ProjectID, c.ID, c.Kind, when, when); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_knowledge_revisions(knowledge_id,revision,supersedes_revision,title,content,created_by,created_at) VALUES(?,1,0,?,?,?,?)`, knowledgeID, c.Title, c.Content, command.ActorID, when); err != nil {
			return nil, err
		}
		for i, s := range c.SourceRefs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_knowledge_source_refs(knowledge_id,revision,ordinal,source_type,source_id,source_revision,citation,asset_id,asset_version_id) VALUES(?,1,?,?,?,?,?,?,?)`, knowledgeID, i, s.Type, s.ID, s.Revision, s.Citation, s.AssetID, s.AssetVersionID); err != nil {
				return nil, err
			}
		}
		revision := contract.KnowledgeRevision{Number: 1, Title: c.Title, Content: c.Content, CreatedBy: command.ActorID, CreatedAt: when, SourceRefs: c.SourceRefs}
		projected = &contract.GovernedKnowledgeEntry{ID: knowledgeID, WorkspaceID: c.WorkspaceID, ProjectID: c.ProjectID, CandidateID: &c.ID, Kind: c.Kind, Status: "published", CurrentRevision: 1, Revision: revision, Revisions: []contract.KnowledgeRevision{revision}, Citation: c.SourceRefs[0].Citation, MatchedBy: "detail", CreatedAt: when, UpdatedAt: when}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_knowledge_publications(candidate_id,knowledge_id,target_knowledge_id,action,created_at) VALUES(?,?,?,?,?)`, c.ID, nullableKnowledgePublicationID(knowledgeID), c.KnowledgeID, command.Action, when); err != nil {
		return nil, err
	}
	return projected, nil
}

func nullableKnowledgePublicationID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

var _ application.KnowledgeReviewRepository = (*knowledgeReviewRepository)(nil)
