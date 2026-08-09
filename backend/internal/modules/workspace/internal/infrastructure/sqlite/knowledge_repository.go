package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	knowledgeDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/knowledge"
)

type knowledgeRepository struct{ db *sql.DB }

func NewKnowledgeRepository(c Config) (application.KnowledgeRepository, error) {
	if c.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &knowledgeRepository{c.DB}, nil
}
func (r *knowledgeRepository) Create(ctx context.Context, v knowledgeDomain.Knowledge) error {
	assets, _ := json.Marshal(v.AssetIDs)
	_, err := r.db.ExecContext(ctx, `INSERT INTO workspace_knowledge(id,workspace_id,title,summary,status,asset_ids,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, v.ID, v.WorkspaceID, v.Title, v.Summary, v.Status, string(assets), v.CreatedAt.Format(time.RFC3339Nano), v.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert Workspace Knowledge: %w", err)
	}
	return nil
}
func (r *knowledgeRepository) FindByID(ctx context.Context, w, id string) (knowledgeDomain.Knowledge, error) {
	var v knowledgeDomain.Knowledge
	var assets, created, updated string
	err := r.db.QueryRowContext(ctx, `SELECT id,workspace_id,title,summary,status,asset_ids,created_at,updated_at FROM workspace_knowledge WHERE workspace_id=? AND id=?`, w, id).Scan(&v.ID, &v.WorkspaceID, &v.Title, &v.Summary, &v.Status, &assets, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return knowledgeDomain.Knowledge{}, application.ErrKnowledgeRecordNotFound
	}
	if err != nil {
		return knowledgeDomain.Knowledge{}, fmt.Errorf("select Workspace Knowledge: %w", err)
	}
	if err = json.Unmarshal([]byte(assets), &v.AssetIDs); err != nil {
		return knowledgeDomain.Knowledge{}, err
	}
	v.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return knowledgeDomain.Knowledge{}, err
	}
	v.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return knowledgeDomain.Knowledge{}, err
	}
	v, err = knowledgeDomain.Rehydrate(v)
	if err != nil {
		return knowledgeDomain.Knowledge{}, fmt.Errorf("map Knowledge: %w", err)
	}
	return v, nil
}
