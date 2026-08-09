package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	relationDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/relationship"
)

type projectActorRelationRepository struct {
	db *sql.DB
}

func NewProjectActorRelationRepository(config Config) (application.ProjectActorRelationRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &projectActorRelationRepository{db: config.DB}, nil
}

func (r *projectActorRelationRepository) Put(ctx context.Context, value relationDomain.Relation) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO workspace_project_actor_relations(
		workspace_id, project_id, actor_type, actor_id, role
	) VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(workspace_id, project_id, actor_type, actor_id)
	DO UPDATE SET role = excluded.role`,
		value.WorkspaceID(), value.ProjectID(), value.ActorType(), value.ActorID(), value.Role(),
	)
	if err != nil {
		return fmt.Errorf("upsert workspace project actor relation: %w", err)
	}
	return nil
}

func (r *projectActorRelationRepository) List(ctx context.Context, workspaceID, projectID string) ([]relationDomain.Relation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT workspace_id, project_id, actor_type, actor_id, role
		FROM workspace_project_actor_relations
		WHERE workspace_id = ? AND project_id = ?
		ORDER BY role, actor_type, actor_id`, workspaceID, projectID)
	if err != nil {
		return nil, fmt.Errorf("query workspace project actor relations: %w", err)
	}
	defer rows.Close()
	values := make([]relationDomain.Relation, 0)
	for rows.Next() {
		var rowWorkspaceID, rowProjectID, actorType, actorID, role string
		if err := rows.Scan(&rowWorkspaceID, &rowProjectID, &actorType, &actorID, &role); err != nil {
			return nil, fmt.Errorf("scan workspace project actor relation: %w", err)
		}
		value, err := relationDomain.New(rowWorkspaceID, rowProjectID, actorType, actorID, role)
		if err != nil {
			return nil, fmt.Errorf("map workspace project actor relation: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace project actor relations: %w", err)
	}
	return values, nil
}

func (r *projectActorRelationRepository) Delete(ctx context.Context, workspaceID, projectID, actorType, actorID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM workspace_project_actor_relations
		WHERE workspace_id = ? AND project_id = ? AND actor_type = ? AND actor_id = ?`,
		workspaceID, projectID, actorType, actorID,
	)
	if err != nil {
		return fmt.Errorf("delete workspace project actor relation: %w", err)
	}
	return nil
}
