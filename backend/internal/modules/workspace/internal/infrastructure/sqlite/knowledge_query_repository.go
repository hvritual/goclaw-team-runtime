package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	knowledgeDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/knowledge"
)

type knowledgeQueryRepository struct{ db *sql.DB }

func NewKnowledgeQueryRepository(config Config) (application.KnowledgeQueryRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &knowledgeQueryRepository{db: config.DB}, nil
}

func (r *knowledgeQueryRepository) ListGovernedKnowledge(ctx context.Context, workspaceID string) ([]knowledgeDomain.GovernedEntry, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,workspace_id,project_id,candidate_id,kind,status,current_revision,created_at,updated_at
		FROM workspace_governed_knowledge WHERE workspace_id=? ORDER BY id`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("select governed Knowledge entries: %w", err)
	}
	defer rows.Close()
	values := make([]knowledgeDomain.GovernedEntry, 0)
	for rows.Next() {
		var value knowledgeDomain.GovernedEntry
		var projectID, candidateID sql.NullString
		var created, updated string
		if err := rows.Scan(&value.ID, &value.WorkspaceID, &projectID, &candidateID, &value.Kind, &value.Status, &value.CurrentRevision, &created, &updated); err != nil {
			return nil, fmt.Errorf("scan governed Knowledge entry: %w", err)
		}
		if projectID.Valid {
			value.ProjectID = &projectID.String
		}
		if candidateID.Valid {
			value.CandidateID = &candidateID.String
		}
		value.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, fmt.Errorf("parse governed Knowledge created_at: %w", err)
		}
		value.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
		if err != nil {
			return nil, fmt.Errorf("parse governed Knowledge updated_at: %w", err)
		}
		value.Revisions, err = r.listRevisions(ctx, value.ID)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate governed Knowledge entries: %w", err)
	}
	return values, nil
}

func (r *knowledgeQueryRepository) listRevisions(ctx context.Context, knowledgeID string) ([]knowledgeDomain.Revision, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT revision,supersedes_revision,title,content,created_by,created_at
		FROM workspace_knowledge_revisions WHERE knowledge_id=? ORDER BY revision`, knowledgeID)
	if err != nil {
		return nil, fmt.Errorf("select Knowledge revisions: %w", err)
	}
	defer rows.Close()
	values := make([]knowledgeDomain.Revision, 0)
	for rows.Next() {
		var value knowledgeDomain.Revision
		var created string
		if err := rows.Scan(&value.Number, &value.SupersedesRevision, &value.Title, &value.Content, &value.CreatedBy, &created); err != nil {
			return nil, fmt.Errorf("scan Knowledge revision: %w", err)
		}
		value.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, fmt.Errorf("parse Knowledge revision created_at: %w", err)
		}
		value.SourceRefs, err = r.listSources(ctx, knowledgeID, value.Number)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Knowledge revisions: %w", err)
	}
	return values, nil
}

func (r *knowledgeQueryRepository) listSources(ctx context.Context, knowledgeID string, revision int) ([]knowledgeDomain.SourceRef, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT source_type,source_id,source_revision,citation,asset_id,asset_version_id
		FROM workspace_knowledge_source_refs WHERE knowledge_id=? AND revision=? ORDER BY ordinal`, knowledgeID, revision)
	if err != nil {
		return nil, fmt.Errorf("select Knowledge source references: %w", err)
	}
	defer rows.Close()
	values := make([]knowledgeDomain.SourceRef, 0)
	for rows.Next() {
		var value knowledgeDomain.SourceRef
		var assetID, assetVersionID sql.NullString
		if err := rows.Scan(&value.Type, &value.ID, &value.Revision, &value.Citation, &assetID, &assetVersionID); err != nil {
			return nil, fmt.Errorf("scan Knowledge source reference: %w", err)
		}
		if assetID.Valid {
			value.AssetID = &assetID.String
		}
		if assetVersionID.Valid {
			value.AssetVersionID = &assetVersionID.String
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Knowledge source references: %w", err)
	}
	return values, nil
}

var _ application.KnowledgeQueryRepository = (*knowledgeQueryRepository)(nil)
