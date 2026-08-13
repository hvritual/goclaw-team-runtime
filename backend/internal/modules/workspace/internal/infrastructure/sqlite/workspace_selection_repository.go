package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type WorkspaceSelectionRepository struct{ db *sql.DB }

func NewWorkspaceSelectionRepository(config Config) (*WorkspaceSelectionRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &WorkspaceSelectionRepository{db: config.DB}, nil
}

func (r *WorkspaceSelectionRepository) ListByIDs(ctx context.Context, ids []string) ([]contract.WorkspaceSelection, error) {
	if len(ids) == 0 {
		return []contract.WorkspaceSelection{}, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	arguments := make([]any, len(ids))
	for index, id := range ids {
		arguments[index] = id
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,slug,description,context,settings,repos,issue_prefix,avatar_url,created_at,updated_at
		FROM workspaces WHERE id IN (`+placeholders+`) ORDER BY created_at ASC, id ASC`, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list authorized workspaces: %w", err)
	}
	defer rows.Close()
	values := make([]contract.WorkspaceSelection, 0)
	for rows.Next() {
		value, err := scanWorkspaceSelection(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate authorized workspaces: %w", err)
	}
	return values, nil
}

func (r *WorkspaceSelectionRepository) FindIDBySlugAndIDs(ctx context.Context, slug string, ids []string) (string, error) {
	if len(ids) == 0 {
		return "", contract.ErrWorkspaceNotFound
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	arguments := make([]any, 0, len(ids)+1)
	arguments = append(arguments, slug)
	for _, id := range ids {
		arguments = append(arguments, id)
	}
	var workspaceID string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM workspaces WHERE slug = ? AND id IN (`+placeholders+`)`, arguments...).Scan(&workspaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", contract.ErrWorkspaceNotFound
	}
	if err != nil {
		return "", fmt.Errorf("resolve authorized workspace slug: %w", err)
	}
	return workspaceID, nil
}

type selectionScanner interface{ Scan(...any) error }

func scanWorkspaceSelection(scanner selectionScanner) (contract.WorkspaceSelection, error) {
	var value contract.WorkspaceSelection
	var description, contextValue, avatarURL sql.NullString
	var rawSettings, rawRepos string
	if err := scanner.Scan(&value.ID, &value.Name, &value.Slug, &description, &contextValue, &rawSettings, &rawRepos, &value.IssuePrefix, &avatarURL, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return contract.WorkspaceSelection{}, fmt.Errorf("scan authorized workspace: %w", err)
	}
	if description.Valid {
		value.Description = &description.String
	}
	if contextValue.Valid {
		value.Context = &contextValue.String
	}
	if avatarURL.Valid {
		value.AvatarURL = &avatarURL.String
	}
	if err := json.Unmarshal([]byte(rawSettings), &value.Settings); err != nil {
		return contract.WorkspaceSelection{}, fmt.Errorf("decode workspace settings: %w", err)
	}
	if err := json.Unmarshal([]byte(rawRepos), &value.Repos); err != nil {
		return contract.WorkspaceSelection{}, fmt.Errorf("decode workspace repos: %w", err)
	}
	if value.Settings == nil {
		value.Settings = map[string]any{}
	}
	if value.Repos == nil {
		value.Repos = []contract.WorkspaceRepo{}
	}
	return value, nil
}

var _ application.WorkspaceSelectionRepository = (*WorkspaceSelectionRepository)(nil)
