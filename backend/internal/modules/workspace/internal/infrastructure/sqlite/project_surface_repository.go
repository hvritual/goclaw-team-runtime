package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type ProjectSurfaceRepository struct{ db *sql.DB }

func NewProjectSurfaceRepository(config Config) (*ProjectSurfaceRepository, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("workspace sqlite database is required")
	}
	repository := &ProjectSurfaceRepository{db: config.DB}
	if err := repository.rebuildSearch(context.Background()); err != nil {
		return nil, err
	}
	return repository, nil
}

func (r *ProjectSurfaceRepository) rebuildSearch(ctx context.Context) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Project search projection rebuild: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `DELETE FROM workspace_project_search_documents`); err != nil {
		return fmt.Errorf("clear Project search projection: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_project_search_documents(
		project_id,workspace_id,title,description,status,updated_at
	) SELECT id,workspace_id,goclaw_issue_search_normalize(name),
		goclaw_issue_search_normalize(COALESCE(description,'')),status,updated_at
		FROM workspace_projects`); err != nil {
		return fmt.Errorf("rebuild Project search projection: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit Project search projection rebuild: %w", err)
	}
	return nil
}

const projectSurfaceColumns = `p.id,p.workspace_id,p.name,NULLIF(p.description,''),NULLIF(p.icon,''),p.status,p.priority,p.lead_type,p.lead_id,p.start_date,p.due_date,p.created_at,p.updated_at,
	(SELECT COUNT(*) FROM workspace_issues i WHERE i.workspace_id=p.workspace_id AND i.project_id=p.id),
	(SELECT COUNT(*) FROM workspace_issues i WHERE i.workspace_id=p.workspace_id AND i.project_id=p.id AND i.status='done')`

func (r *ProjectSurfaceRepository) ListProjects(ctx context.Context, workspaceID, status string) ([]contract.ProjectSurfaceProject, error) {
	query := `SELECT ` + projectSurfaceColumns + ` FROM workspace_projects p WHERE p.workspace_id=?`
	arguments := []any{workspaceID}
	if status != "" {
		query += ` AND p.status=?`
		arguments = append(arguments, status)
	}
	query += ` ORDER BY p.updated_at DESC,p.id ASC`
	rows, err := r.db.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list project surface: %w", err)
	}
	defer rows.Close()
	projects := make([]contract.ProjectSurfaceProject, 0)
	for rows.Next() {
		value, err := scanProjectSurface(rows)
		if err != nil {
			return nil, fmt.Errorf("scan project surface: %w", err)
		}
		projects = append(projects, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project surface: %w", err)
	}
	return projects, nil
}

func (r *ProjectSurfaceRepository) GetProject(ctx context.Context, workspaceID, projectID string) (contract.ProjectSurfaceProject, error) {
	value, err := scanProjectSurface(r.db.QueryRowContext(ctx, `SELECT `+projectSurfaceColumns+` FROM workspace_projects p WHERE p.workspace_id=? AND p.id=?`, workspaceID, projectID))
	if errors.Is(err, sql.ErrNoRows) {
		return contract.ProjectSurfaceProject{}, application.ErrProjectSurfaceNotFound
	}
	if err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("get project surface: %w", err)
	}
	return value, nil
}

func (r *ProjectSurfaceRepository) SearchProjects(ctx context.Context, query application.ProjectSurfaceSearchQuery) ([]application.ProjectSurfaceSearchResult, int, error) {
	matchSQL, rankSQL, sourceSQL, matchArgs, rankArgs := projectSearchPredicates(query)
	closedSQL := ""
	if !query.IncludeClosed {
		closedSQL = ` AND d.status NOT IN ('completed','cancelled')`
	}
	countArgs := append([]any{query.WorkspaceID}, matchArgs...)
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_project_search_documents d
		WHERE d.workspace_id=?`+closedSQL+` AND (`+matchSQL+`)`, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count Project search results: %w", err)
	}
	selectArgs := append([]any{}, rankArgs...)
	selectArgs = append(selectArgs, query.WorkspaceID)
	selectArgs = append(selectArgs, matchArgs...)
	selectArgs = append(selectArgs, rankArgs...)
	selectArgs = append(selectArgs, query.Limit, query.Offset)
	rows, err := r.db.QueryContext(ctx, `SELECT `+projectSurfaceColumns+`, `+sourceSQL+`
		FROM workspace_project_search_documents d
		JOIN workspace_projects p ON p.id=d.project_id AND p.workspace_id=d.workspace_id
		WHERE d.workspace_id=?`+closedSQL+` AND (`+matchSQL+`)
		ORDER BY `+rankSQL+` ASC,d.updated_at DESC,d.project_id ASC
		LIMIT ? OFFSET ?`, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query Project search results: %w", err)
	}
	defer rows.Close()
	results := make([]application.ProjectSurfaceSearchResult, 0)
	for rows.Next() {
		value, err := scanProjectSurfaceWithSource(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan Project search result: %w", err)
		}
		results = append(results, value)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate Project search results: %w", err)
	}
	return results, total, nil
}

func projectSearchPredicates(query application.ProjectSurfaceSearchQuery) (matchSQL, rankSQL, sourceSQL string, matchArgs, rankArgs []any) {
	titleTerms := termPredicate("d.title", query.Terms)
	descriptionTerms := termPredicate("d.description", query.Terms)
	matchSQL = `d.title=? OR (` + titleTerms + `) OR (` + descriptionTerms + `)`
	rankSQL = `CASE WHEN d.title=? THEN 0 WHEN (` + titleTerms + `) THEN 1 ELSE 2 END`
	sourceSQL = `CASE WHEN d.title=? OR (` + titleTerms + `) THEN 'title' ELSE 'description' END`
	rankArgs = append([]any{query.Phrase}, stringArgs(query.Terms)...)
	matchArgs = append([]any{}, rankArgs...)
	matchArgs = append(matchArgs, stringArgs(query.Terms)...)
	return matchSQL, rankSQL, sourceSQL, matchArgs, rankArgs
}

func stringArgs(values []string) []any {
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = value
	}
	return result
}

type projectSurfaceAndSourceScanner struct {
	scanner projectSurfaceScanner
	source  *string
}

func (s *projectSurfaceAndSourceScanner) Scan(dest ...any) error {
	return s.scanner.Scan(append(dest, s.source)...)
}

func scanProjectSurfaceWithSource(scanner projectSurfaceScanner) (application.ProjectSurfaceSearchResult, error) {
	var source string
	project, err := scanProjectSurface(&projectSurfaceAndSourceScanner{scanner: scanner, source: &source})
	if err != nil {
		return application.ProjectSurfaceSearchResult{}, err
	}
	result := application.ProjectSurfaceSearchResult{Project: project, MatchSource: source}
	if source == "description" && project.Description != nil {
		result.MatchedSnippet = boundedIssueSearchSnippet(*project.Description, 240)
	}
	return result, nil
}

func (r *ProjectSurfaceRepository) CreateProject(ctx context.Context, value contract.ProjectSurfaceProject) (contract.ProjectSurfaceProject, error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("acquire project creation connection: %w", err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("configure project creation connection: %w", err)
	}
	if _, err := connection.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("begin project creation: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	_, err = connection.ExecContext(ctx, `INSERT INTO workspace_projects(
		id,workspace_id,name,description,icon,status,priority,lead_type,lead_id,start_date,due_date,asset_ids,created_at,updated_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,'[]',?,?)`, value.ID, value.WorkspaceID, value.Title, projectDescriptionValue(value.Description), value.Icon, value.Status, value.Priority, value.LeadType, value.LeadID, value.StartDate, value.DueDate, value.CreatedAt, value.UpdatedAt)
	if err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("create project surface: %w", err)
	}
	created, err := scanProjectSurface(connection.QueryRowContext(ctx, `SELECT `+projectSurfaceColumns+` FROM workspace_projects p WHERE p.workspace_id=? AND p.id=?`, value.WorkspaceID, value.ID))
	if err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("read created project surface: %w", err)
	}
	if _, err := connection.ExecContext(ctx, "COMMIT"); err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("commit project creation: %w", err)
	}
	committed = true
	return created, nil
}

func (r *ProjectSurfaceRepository) UpdateProject(ctx context.Context, value contract.ProjectSurfaceProject) (contract.ProjectSurfaceProject, error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("acquire project update connection: %w", err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("configure project update connection: %w", err)
	}
	if _, err := connection.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("begin project update: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	result, err := connection.ExecContext(ctx, `UPDATE workspace_projects SET name=?,description=?,icon=?,status=?,priority=?,lead_type=?,lead_id=?,start_date=?,due_date=?,updated_at=? WHERE workspace_id=? AND id=?`, value.Title, projectDescriptionValue(value.Description), value.Icon, value.Status, value.Priority, value.LeadType, value.LeadID, value.StartDate, value.DueDate, value.UpdatedAt, value.WorkspaceID, value.ID)
	if err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("update project surface: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected == 0 {
		if err != nil {
			return contract.ProjectSurfaceProject{}, fmt.Errorf("inspect project update: %w", err)
		}
		return contract.ProjectSurfaceProject{}, application.ErrProjectSurfaceNotFound
	}
	updated, err := scanProjectSurface(connection.QueryRowContext(ctx, `SELECT `+projectSurfaceColumns+` FROM workspace_projects p WHERE p.workspace_id=? AND p.id=?`, value.WorkspaceID, value.ID))
	if err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("read updated project surface: %w", err)
	}
	if _, err := connection.ExecContext(ctx, "COMMIT"); err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("commit project update: %w", err)
	}
	committed = true
	return updated, nil
}

func (r *ProjectSurfaceRepository) DeleteProject(ctx context.Context, workspaceID, projectID string, now time.Time) error {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire project delete connection: %w", err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return fmt.Errorf("configure project delete connection: %w", err)
	}
	if _, err := connection.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("begin project delete: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	var found string
	if err := connection.QueryRowContext(ctx, `SELECT id FROM workspace_projects WHERE workspace_id=? AND id=?`, workspaceID, projectID).Scan(&found); errors.Is(err, sql.ErrNoRows) {
		return application.ErrProjectSurfaceNotFound
	} else if err != nil {
		return fmt.Errorf("find project for delete: %w", err)
	}
	timestamp := now.UTC().Format(time.RFC3339Nano)
	statements := []struct {
		query string
		args  []any
	}{
		{`DELETE FROM workspace_project_actor_relations WHERE workspace_id=? AND project_id=?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_pins WHERE workspace_id=? AND item_type='project' AND item_id=?`, []any{workspaceID, projectID}},
		{`UPDATE workspace_todos SET project_id=NULL,updated_at=? WHERE workspace_id=? AND project_id=?`, []any{timestamp, workspaceID, projectID}},
		{`UPDATE workspace_issues SET project_id=NULL,updated_at=? WHERE workspace_id=? AND project_id=?`, []any{timestamp, workspaceID, projectID}},
		{`DELETE FROM workspace_requirement_versions WHERE requirement_id IN (SELECT id FROM workspace_requirements WHERE workspace_id=? AND project_id=?)`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_requirements WHERE workspace_id=? AND project_id=?`, []any{workspaceID, projectID}},
		{`DELETE FROM workspace_projects WHERE workspace_id=? AND id=?`, []any{workspaceID, projectID}},
	}
	for _, statement := range statements {
		if _, err := connection.ExecContext(ctx, statement.query, statement.args...); err != nil {
			return fmt.Errorf("apply project surface delete: %w", err)
		}
	}
	if _, err := connection.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit project surface delete: %w", err)
	}
	committed = true
	return nil
}

func (r *ProjectSurfaceRepository) ListPins(ctx context.Context, workspaceID, userID string) ([]contract.Pin, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,workspace_id,user_id,item_type,item_id,position,created_at
		FROM workspace_pins WHERE workspace_id=? AND user_id=? ORDER BY position,created_at,id`, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("list pins: %w", err)
	}
	defer rows.Close()
	values := make([]contract.Pin, 0)
	for rows.Next() {
		var value contract.Pin
		if err := rows.Scan(&value.ID, &value.WorkspaceID, &value.UserID, &value.ItemType, &value.ItemID, &value.Position, &value.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan pin: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pins: %w", err)
	}
	return values, nil
}

func (r *ProjectSurfaceRepository) InspectPin(ctx context.Context, workspaceID, userID, itemType, itemID string) (bool, bool, error) {
	table := "workspace_issues"
	if itemType == "project" {
		table = "workspace_projects"
	}
	var target string
	if err := r.db.QueryRowContext(ctx, `SELECT id FROM `+table+` WHERE workspace_id=? AND id=?`, workspaceID, itemID).Scan(&target); errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	} else if err != nil {
		return false, false, fmt.Errorf("inspect pin target: %w", err)
	}
	var existing string
	if err := r.db.QueryRowContext(ctx, `SELECT id FROM workspace_pins WHERE workspace_id=? AND user_id=? AND item_type=? AND item_id=?`, workspaceID, userID, itemType, itemID).Scan(&existing); errors.Is(err, sql.ErrNoRows) {
		return true, false, nil
	} else if err != nil {
		return false, false, fmt.Errorf("inspect existing pin: %w", err)
	}
	return true, true, nil
}

func (r *ProjectSurfaceRepository) CreatePin(ctx context.Context, value contract.Pin) (contract.Pin, error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return contract.Pin{}, fmt.Errorf("acquire pin connection: %w", err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return contract.Pin{}, fmt.Errorf("configure pin connection: %w", err)
	}
	if _, err := connection.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return contract.Pin{}, fmt.Errorf("begin pin creation: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	table := "workspace_issues"
	if value.ItemType == "project" {
		table = "workspace_projects"
	}
	var found string
	if err := connection.QueryRowContext(ctx, `SELECT id FROM `+table+` WHERE workspace_id=? AND id=?`, value.WorkspaceID, value.ItemID).Scan(&found); errors.Is(err, sql.ErrNoRows) {
		return contract.Pin{}, application.ErrPinTargetNotFound
	} else if err != nil {
		return contract.Pin{}, fmt.Errorf("find pin target: %w", err)
	}
	var existing string
	err = connection.QueryRowContext(ctx, `SELECT id FROM workspace_pins WHERE workspace_id=? AND user_id=? AND item_type=? AND item_id=?`, value.WorkspaceID, value.UserID, value.ItemType, value.ItemID).Scan(&existing)
	if err == nil {
		return contract.Pin{}, application.ErrPinConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return contract.Pin{}, fmt.Errorf("find existing pin: %w", err)
	}
	if err := connection.QueryRowContext(ctx, `SELECT COALESCE(MAX(position),0)+1 FROM workspace_pins WHERE workspace_id=? AND user_id=?`, value.WorkspaceID, value.UserID).Scan(&value.Position); err != nil {
		return contract.Pin{}, fmt.Errorf("select pin position: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `INSERT INTO workspace_pins(id,workspace_id,user_id,item_type,item_id,position,created_at) VALUES(?,?,?,?,?,?,?)`, value.ID, value.WorkspaceID, value.UserID, value.ItemType, value.ItemID, value.Position, value.CreatedAt); err != nil {
		if strings.Contains(err.Error(), "workspace_pins.workspace_id, workspace_pins.user_id, workspace_pins.item_type, workspace_pins.item_id") {
			return contract.Pin{}, application.ErrPinConflict
		}
		return contract.Pin{}, fmt.Errorf("create pin: %w", err)
	}
	if _, err := connection.ExecContext(ctx, "COMMIT"); err != nil {
		return contract.Pin{}, fmt.Errorf("commit pin: %w", err)
	}
	committed = true
	return value, nil
}

func (r *ProjectSurfaceRepository) DeletePin(ctx context.Context, workspaceID, userID, itemType, itemID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM workspace_pins WHERE workspace_id=? AND user_id=? AND item_type=? AND item_id=?`, workspaceID, userID, itemType, itemID)
	if err != nil {
		return fmt.Errorf("delete pin: %w", err)
	}
	return nil
}

func projectNullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func projectDescriptionValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type projectSurfaceScanner interface{ Scan(...any) error }

func scanProjectSurface(scanner projectSurfaceScanner) (contract.ProjectSurfaceProject, error) {
	var value contract.ProjectSurfaceProject
	var description, icon, leadType, leadID, startDate, dueDate sql.NullString
	if err := scanner.Scan(&value.ID, &value.WorkspaceID, &value.Title, &description, &icon, &value.Status, &value.Priority, &leadType, &leadID, &startDate, &dueDate, &value.CreatedAt, &value.UpdatedAt, &value.IssueCount, &value.DoneCount); err != nil {
		return value, err
	}
	value.Description, value.Icon = projectNullableString(description), projectNullableString(icon)
	value.LeadType, value.LeadID = projectNullableString(leadType), projectNullableString(leadID)
	value.StartDate, value.DueDate = projectNullableString(startDate), projectNullableString(dueDate)
	return value, nil
}
