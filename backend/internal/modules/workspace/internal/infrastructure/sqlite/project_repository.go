package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	projectDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/project"
)

type projectRepository struct {
	db *sql.DB
}

func NewProjectRepository(config Config) (application.ProjectRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &projectRepository{db: config.DB}, nil
}

func (r *projectRepository) Create(ctx context.Context, value projectDomain.Project) error {
	assetIDs, err := json.Marshal(value.AssetIDs())
	if err != nil {
		return fmt.Errorf("encode project asset ids: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO workspace_projects(
		id, workspace_id, name, description, status, asset_ids, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		value.ID(), value.WorkspaceID(), value.Name(), value.Description(), value.Status(), string(assetIDs),
		value.CreatedAt().Format(time.RFC3339Nano), value.UpdatedAt().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert workspace project: %w", err)
	}
	return nil
}

func (r *projectRepository) FindByID(ctx context.Context, workspaceID, projectID string) (projectDomain.Project, error) {
	return scanProject(r.db.QueryRowContext(ctx, `SELECT
		id, workspace_id, name, description, status, asset_ids, created_at, updated_at
	FROM workspace_projects WHERE workspace_id = ? AND id = ?`, workspaceID, projectID))
}

func (r *projectRepository) List(ctx context.Context, workspaceID, status string) ([]projectDomain.Project, error) {
	query := `SELECT id, workspace_id, name, description, status, asset_ids, created_at, updated_at
		FROM workspace_projects WHERE workspace_id = ?`
	arguments := []any{workspaceID}
	if status != "" {
		query += ` AND status = ?`
		arguments = append(arguments, status)
	}
	query += ` ORDER BY created_at DESC, id ASC`
	rows, err := r.db.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list workspace projects: %w", err)
	}
	defer rows.Close()
	values := make([]projectDomain.Project, 0)
	for rows.Next() {
		value, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace projects: %w", err)
	}
	return values, nil
}

func (r *projectRepository) Search(ctx context.Context, query application.ProjectSearchQuery) ([]application.ProjectSearchResult, int, error) {
	statement := `SELECT id, workspace_id, name, description, status, asset_ids, created_at, updated_at
		FROM workspace_projects WHERE workspace_id = ?`
	arguments := []any{query.WorkspaceID}
	if !query.IncludeClosed {
		statement += ` AND status NOT IN (?, ?)`
		arguments = append(arguments, projectDomain.StatusCompleted, projectDomain.StatusCancelled)
	}
	rows, err := r.db.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return nil, 0, fmt.Errorf("query searchable workspace projects: %w", err)
	}
	defer rows.Close()
	type rankedResult struct {
		application.ProjectSearchResult
		rank int
	}
	ranked := make([]rankedResult, 0)
	for rows.Next() {
		value, err := scanProject(rows)
		if err != nil {
			return nil, 0, err
		}
		rank, source, snippet, matches := matchProject(value, query.Phrase, query.Terms)
		if matches {
			ranked = append(ranked, rankedResult{
				ProjectSearchResult: application.ProjectSearchResult{Project: value, MatchSource: source, MatchedSnippet: snippet},
				rank:                rank,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate searchable workspace projects: %w", err)
	}
	sort.Slice(ranked, func(left, right int) bool {
		if ranked[left].rank != ranked[right].rank {
			return ranked[left].rank < ranked[right].rank
		}
		leftUpdated := ranked[left].Project.UpdatedAt()
		rightUpdated := ranked[right].Project.UpdatedAt()
		if !leftUpdated.Equal(rightUpdated) {
			return leftUpdated.After(rightUpdated)
		}
		return ranked[left].Project.ID() < ranked[right].Project.ID()
	})
	total := len(ranked)
	if query.Offset >= total {
		return []application.ProjectSearchResult{}, total, nil
	}
	end := query.Offset + query.Limit
	if end > total {
		end = total
	}
	results := make([]application.ProjectSearchResult, end-query.Offset)
	for index := range results {
		results[index] = ranked[query.Offset+index].ProjectSearchResult
	}
	return results, total, nil
}

func (r *projectRepository) Update(ctx context.Context, value projectDomain.Project) error {
	assetIDs, err := json.Marshal(value.AssetIDs())
	if err != nil {
		return fmt.Errorf("encode project asset ids: %w", err)
	}
	result, err := r.db.ExecContext(ctx, `UPDATE workspace_projects
		SET name = ?, description = ?, status = ?, asset_ids = ?, updated_at = ?
		WHERE workspace_id = ? AND id = ?`,
		value.Name(), value.Description(), value.Status(), string(assetIDs), value.UpdatedAt().Format(time.RFC3339Nano),
		value.WorkspaceID(), value.ID(),
	)
	if err != nil {
		return fmt.Errorf("update workspace project: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect workspace project update: %w", err)
	}
	if rows == 0 {
		return application.ErrProjectRecordNotFound
	}
	return nil
}

func (r *projectRepository) DeleteWithDependents(ctx context.Context, workspaceID, projectID string, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin project deletion: %w", err)
	}
	defer tx.Rollback()
	var found string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM workspace_projects WHERE workspace_id = ? AND id = ?`, workspaceID, projectID).Scan(&found); errors.Is(err, sql.ErrNoRows) {
		return application.ErrProjectRecordNotFound
	} else if err != nil {
		return fmt.Errorf("select project for deletion: %w", err)
	}
	if err := deleteProjectRequirementAuthority(ctx, tx, workspaceID, projectID); err != nil {
		return err
	}
	timestamp := now.UTC().Format(time.RFC3339Nano)
	statements := []struct {
		query string
		args  []any
	}{
		{`DELETE FROM workspace_project_actor_relations WHERE workspace_id = ? AND project_id = ?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_pins WHERE workspace_id = ? AND item_type = 'project' AND item_id = ?`, []any{workspaceID, projectID}},
		{`UPDATE workspace_todos SET project_id = NULL, updated_at = ? WHERE workspace_id = ? AND project_id = ?`, []any{timestamp, workspaceID, projectID}},
		{`UPDATE workspace_issues SET project_id = NULL, updated_at = ? WHERE workspace_id = ? AND project_id = ?`, []any{timestamp, workspaceID, projectID}},
		{`DELETE FROM workspace_requirement_versions WHERE requirement_id IN (
			SELECT id FROM workspace_requirements WHERE workspace_id = ? AND project_id = ?
		)`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_requirements WHERE workspace_id = ? AND project_id = ?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_project_resources WHERE workspace_id = ? AND project_id = ?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_project_resource_sets WHERE workspace_id = ? AND project_id = ?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_projects WHERE workspace_id = ? AND id = ?`, []any{workspaceID, projectID}},
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement.query, statement.args...); err != nil {
			return fmt.Errorf("apply project deletion cleanup: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit project deletion: %w", err)
	}
	return nil
}

type projectScanner interface {
	Scan(...any) error
}

func scanProject(scanner projectScanner) (projectDomain.Project, error) {
	var id, rowWorkspaceID, name, description, status, rawAssetIDs, rawCreatedAt, rawUpdatedAt string
	err := scanner.Scan(&id, &rowWorkspaceID, &name, &description, &status, &rawAssetIDs, &rawCreatedAt, &rawUpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return projectDomain.Project{}, application.ErrProjectRecordNotFound
	}
	if err != nil {
		return projectDomain.Project{}, fmt.Errorf("select workspace project: %w", err)
	}
	var assetIDs []string
	if err := json.Unmarshal([]byte(rawAssetIDs), &assetIDs); err != nil {
		return projectDomain.Project{}, fmt.Errorf("decode workspace project asset ids: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, rawCreatedAt)
	if err != nil {
		return projectDomain.Project{}, fmt.Errorf("parse workspace project created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, rawUpdatedAt)
	if err != nil {
		return projectDomain.Project{}, fmt.Errorf("parse workspace project updated_at: %w", err)
	}
	value, err := projectDomain.Rehydrate(id, rowWorkspaceID, name, description, status, assetIDs, createdAt, updatedAt)
	if err != nil {
		return projectDomain.Project{}, fmt.Errorf("map workspace project: %w", err)
	}
	return value, nil
}

func matchProject(value projectDomain.Project, phrase string, terms []string) (int, string, string, bool) {
	name := strings.ToLower(value.Name())
	description := strings.ToLower(value.Description())
	phraseInName := strings.Contains(name, phrase)
	phraseInDescription := strings.Contains(description, phrase)
	allInName := allTermsMatch(terms, func(term string) bool { return strings.Contains(name, term) })
	allAnywhere := allTermsMatch(terms, func(term string) bool {
		return strings.Contains(name, term) || strings.Contains(description, term)
	})
	if !phraseInName && !phraseInDescription && !allAnywhere {
		return 0, "", "", false
	}
	rank := 4
	source := "description"
	snippet := value.Description()
	switch {
	case name == phrase:
		rank = 0
	case strings.HasPrefix(name, phrase):
		rank = 1
	case phraseInName:
		rank = 2
	case allInName:
		rank = 3
	}
	if phraseInName || allInName {
		source = "name"
		snippet = value.Name()
	}
	return rank, source, truncateSnippet(snippet, 160), true
}

func allTermsMatch(terms []string, matches func(string) bool) bool {
	if len(terms) == 0 {
		return false
	}
	for _, term := range terms {
		if !matches(term) {
			return false
		}
	}
	return true
}

func truncateSnippet(value string, limit int) string {
	characters := []rune(value)
	if len(characters) <= limit {
		return value
	}
	return string(characters[:limit])
}
