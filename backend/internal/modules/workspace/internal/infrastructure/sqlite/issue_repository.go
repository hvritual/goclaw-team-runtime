package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	spacecontract "github.com/hvritual/workspace/internal/modules/space/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
)

const issueColumns = `id, workspace_id, number, identifier, title, description, status, priority,
	assignee_type, assignee_id, creator_type, creator_id, parent_issue_id, project_id, position,
	stage, start_date, due_date, created_at, updated_at, metadata, properties, asset_ids`

type issueRepository struct {
	db                   *sql.DB
	attachmentCleanup    spacecontract.AttachmentCleanupService
	attachmentReferences spacecontract.AttachmentReferenceValidator
}

func NewIssueRepository(config Config) (application.IssueRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &issueRepository{db: config.DB, attachmentCleanup: config.AttachmentCleanup, attachmentReferences: config.AttachmentReferences}, nil
}

func (r *issueRepository) Create(ctx context.Context, value issueDomain.Issue) (created issueDomain.Issue, err error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("acquire Issue transaction connection: %w", err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return issueDomain.Issue{}, fmt.Errorf("configure Issue transaction lock wait: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return issueDomain.Issue{}, fmt.Errorf("begin immediate Issue creation: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()
	if len(value.AssetIDs) > 0 && r.attachmentReferences != nil {
		if err := r.attachmentReferences.ValidateReferences(ctx, connection, value.WorkspaceID, value.AssetIDs); err != nil {
			return issueDomain.Issue{}, fmt.Errorf("validate Issue attachment references: %w", issueAttachmentReferenceError(err))
		}
	}

	var prefix string
	var number int32
	now := value.UpdatedAt.Format(time.RFC3339Nano)
	if err := connection.QueryRowContext(ctx, `UPDATE workspaces
		SET next_issue_number = next_issue_number + 1, updated_at = ?
		WHERE id = ? RETURNING issue_prefix, next_issue_number - 1`, now, value.WorkspaceID).Scan(&prefix, &number); errors.Is(err, sql.ErrNoRows) {
		return issueDomain.Issue{}, application.ErrWorkspaceNotFound
	} else if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("allocate Workspace Issue number: %w", err)
	}
	created, err = value.AssignIdentity(number, prefix)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("assign Workspace Issue identifier: %w", err)
	}
	metadata, properties, assets, err := encodeIssueJSON(created)
	if err != nil {
		return issueDomain.Issue{}, err
	}
	_, err = connection.ExecContext(ctx, `INSERT INTO workspace_issues(
		id, workspace_id, number, identifier, title, description, status, priority,
		assignee_type, assignee_id, creator_type, creator_id, parent_issue_id, project_id,
		position, stage, start_date, due_date, metadata, properties, asset_ids, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		created.ID, created.WorkspaceID, created.Number, created.Identifier, created.Title, nullableString(created.Description),
		created.Status, created.Priority, nullableString(created.AssigneeType), nullableString(created.AssigneeID),
		created.CreatorType, created.CreatorID, nullableString(created.ParentIssueID), nullableString(created.ProjectID),
		created.Position, nullableInt32(created.Stage), nullableString(created.StartDate), nullableString(created.DueDate),
		metadata, properties, assets, created.CreatedAt.Format(time.RFC3339Nano), created.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("insert Workspace Issue: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return issueDomain.Issue{}, fmt.Errorf("commit Workspace Issue creation: %w", err)
	}
	committed = true
	return created, nil
}

func (r *issueRepository) FindByIDOrIdentifier(ctx context.Context, workspaceID, issueID string) (issueDomain.Issue, error) {
	return scanIssue(r.db.QueryRowContext(ctx, `SELECT `+issueColumns+`
		FROM workspace_issues WHERE workspace_id = ? AND (id = ? OR identifier = ?)`, workspaceID, issueID, issueID))
}

func (r *issueRepository) List(ctx context.Context, query application.IssueListQuery) ([]issueDomain.Issue, error) {
	statement := `SELECT ` + issueColumns + ` FROM workspace_issues WHERE workspace_id = ?`
	arguments := []any{query.WorkspaceID}
	filters := []struct {
		column string
		value  *string
	}{
		{"project_id", query.ProjectID}, {"parent_issue_id", query.ParentIssueID},
		{"assignee_type", query.AssigneeType}, {"assignee_id", query.AssigneeID},
		{"creator_type", query.CreatorType}, {"creator_id", query.CreatorID},
	}
	for _, filter := range filters {
		if filter.value != nil {
			statement += ` AND ` + filter.column + ` = ?`
			arguments = append(arguments, *filter.value)
		}
	}
	if query.Status != "" {
		statement += ` AND status = ?`
		arguments = append(arguments, query.Status)
	}
	if query.Priority != "" {
		statement += ` AND priority = ?`
		arguments = append(arguments, query.Priority)
	}
	statement += ` ORDER BY position ASC, created_at DESC, id ASC`
	rows, err := r.db.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list Workspace Issues: %w", err)
	}
	defer rows.Close()
	values := make([]issueDomain.Issue, 0)
	for rows.Next() {
		value, err := scanIssue(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Workspace Issues: %w", err)
	}
	return values, nil
}

func (r *issueRepository) Update(ctx context.Context, command application.IssueUpdateCommand) (updated issueDomain.Issue, err error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("acquire Issue update connection: %w", err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return issueDomain.Issue{}, fmt.Errorf("configure Issue update lock wait: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return issueDomain.Issue{}, fmt.Errorf("begin immediate Issue update: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()

	current, err := scanIssue(connection.QueryRowContext(ctx, `SELECT `+issueColumns+`
		FROM workspace_issues WHERE workspace_id = ? AND id = ?`, command.WorkspaceID, command.IssueID))
	if err != nil {
		return issueDomain.Issue{}, err
	}
	if command.Patch.AssetIDs.Set && !slices.Equal(current.AssetIDs, command.ExpectedAssetIDs) {
		return issueDomain.Issue{}, contract.ErrIssueAttachmentConflict
	}
	if command.Patch.AssetIDs.Set && len(command.Patch.AssetIDs.Values) > 0 && r.attachmentReferences != nil {
		if err := r.attachmentReferences.ValidateReferences(ctx, connection, command.WorkspaceID, command.Patch.AssetIDs.Values); err != nil {
			return issueDomain.Issue{}, fmt.Errorf("validate Issue attachment references: %w", issueAttachmentReferenceError(err))
		}
	}
	updated, err = current.Apply(command.Patch, command.Now)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("apply Workspace Issue update: %w", err)
	}
	statement := `UPDATE workspace_issues SET
		title = ?, description = ?, status = ?, priority = ?, assignee_type = ?, assignee_id = ?,
		parent_issue_id = ?, project_id = ?, position = ?, stage = ?, start_date = ?, due_date = ?`
	arguments := []any{
		updated.Title, nullableString(updated.Description), updated.Status, updated.Priority,
		nullableString(updated.AssigneeType), nullableString(updated.AssigneeID), nullableString(updated.ParentIssueID), nullableString(updated.ProjectID),
		updated.Position, nullableInt32(updated.Stage), nullableString(updated.StartDate), nullableString(updated.DueDate),
	}
	if command.Patch.AssetIDs.Set {
		assets, marshalErr := json.Marshal(updated.AssetIDs)
		if marshalErr != nil {
			return issueDomain.Issue{}, fmt.Errorf("encode Issue assets: %w", marshalErr)
		}
		statement += `, asset_ids = ?`
		arguments = append(arguments, string(assets))
	}
	statement += `, updated_at = ? WHERE workspace_id = ? AND id = ?`
	arguments = append(arguments, updated.UpdatedAt.Format(time.RFC3339Nano), command.WorkspaceID, command.IssueID)
	result, err := connection.ExecContext(ctx, statement, arguments...)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("update Workspace Issue: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("inspect Workspace Issue update: %w", err)
	}
	if rows == 0 {
		return issueDomain.Issue{}, application.ErrIssueRecordNotFound
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return issueDomain.Issue{}, fmt.Errorf("commit Workspace Issue update: %w", err)
	}
	committed = true
	return updated, nil
}

func issueAttachmentReferenceError(err error) error {
	if errors.Is(err, spacecontract.ErrAttachmentNotFound) || errors.Is(err, spacecontract.ErrAttachmentInvalid) {
		return errors.Join(contract.ErrAssetOutsideWorkspace, err)
	}
	return err
}

func (r *issueRepository) Move(ctx context.Context, command application.IssueMoveCommand) (updated issueDomain.Issue, err error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("acquire Issue move connection: %w", err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return issueDomain.Issue{}, fmt.Errorf("configure Issue move lock wait: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return issueDomain.Issue{}, fmt.Errorf("begin immediate Issue move: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()

	current, err := scanIssue(connection.QueryRowContext(ctx, `SELECT `+issueColumns+`
		FROM workspace_issues WHERE workspace_id = ? AND (id = ? OR identifier = ?)`, command.WorkspaceID, command.IssueID, command.IssueID))
	if err != nil {
		return issueDomain.Issue{}, err
	}
	if command.BeforeID != nil && *command.BeforeID == current.ID || command.AfterID != nil && *command.AfterID == current.ID {
		return issueDomain.Issue{}, fmt.Errorf("%w: move anchor cannot be the moved Issue", application.ErrIssueMoveConflict)
	}
	if command.BeforeID != nil && command.AfterID != nil && *command.BeforeID == *command.AfterID {
		return issueDomain.Issue{}, fmt.Errorf("%w: move anchors must be distinct", application.ErrIssueMoveConflict)
	}
	before, err := issueAnchorPosition(ctx, connection, command.WorkspaceID, command.BeforeID)
	if err != nil {
		return issueDomain.Issue{}, err
	}
	after, err := issueAnchorPosition(ctx, connection, command.WorkspaceID, command.AfterID)
	if err != nil {
		return issueDomain.Issue{}, err
	}
	position, err := canonicalIssueMovePosition(current.Position, before, after)
	if err != nil {
		return issueDomain.Issue{}, err
	}
	command.Patch.Position = &position
	updated, err = current.Apply(command.Patch, command.Now)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("%w: %v", application.ErrIssueMoveInvalid, err)
	}
	if updated.ParentIssueID != nil {
		cycle, cycleErr := wouldCreateParentCycleOnConnection(ctx, connection, command.WorkspaceID, updated.ID, *updated.ParentIssueID)
		if cycleErr != nil {
			return issueDomain.Issue{}, cycleErr
		}
		if cycle {
			return issueDomain.Issue{}, fmt.Errorf("%w: circular parent relationship", application.ErrIssueMoveInvalid)
		}
	}
	_, _, assets, err := encodeIssueJSON(updated)
	if err != nil {
		return issueDomain.Issue{}, err
	}
	result, err := connection.ExecContext(ctx, `UPDATE workspace_issues SET
		status=?, assignee_type=?, assignee_id=?, parent_issue_id=?, project_id=?, position=?, asset_ids=?, updated_at=?
		WHERE workspace_id=? AND id=?`,
		updated.Status, nullableString(updated.AssigneeType), nullableString(updated.AssigneeID), nullableString(updated.ParentIssueID), nullableString(updated.ProjectID),
		updated.Position, assets, updated.UpdatedAt.Format(time.RFC3339Nano), updated.WorkspaceID, updated.ID,
	)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("update moved Workspace Issue: %w", err)
	}
	if rows, rowsErr := result.RowsAffected(); rowsErr != nil {
		return issueDomain.Issue{}, fmt.Errorf("inspect moved Workspace Issue: %w", rowsErr)
	} else if rows != 1 {
		return issueDomain.Issue{}, application.ErrIssueRecordNotFound
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return issueDomain.Issue{}, fmt.Errorf("commit Workspace Issue move: %w", err)
	}
	committed = true
	return updated, nil
}

func issueAnchorPosition(ctx context.Context, connection *sql.Conn, workspaceID string, issueID *string) (*float64, error) {
	if issueID == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*issueID)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: move anchor id is required", application.ErrIssueMoveConflict)
	}
	var position float64
	if err := connection.QueryRowContext(ctx, `SELECT position FROM workspace_issues WHERE workspace_id=? AND id=?`, workspaceID, trimmed).Scan(&position); errors.Is(err, sql.ErrNoRows) {
		return nil, application.ErrIssueMoveAnchorNotFound
	} else if err != nil {
		return nil, fmt.Errorf("resolve Workspace Issue move anchor: %w", err)
	}
	return &position, nil
}

func canonicalIssueMovePosition(current float64, before, after *float64) (float64, error) {
	var position float64
	switch {
	case before != nil && after != nil:
		if !(*before < *after) {
			return 0, fmt.Errorf("%w: move anchors are stale or out of order", application.ErrIssueMoveConflict)
		}
		position = *before + (*after-*before)/2
		if !(position > *before && position < *after) {
			return 0, fmt.Errorf("%w: move anchors are too close", application.ErrIssueMoveConflict)
		}
	case before != nil:
		position = *before + 1
	case after != nil:
		position = *after - 1
	default:
		position = current
	}
	if math.IsInf(position, 0) || math.IsNaN(position) {
		return 0, fmt.Errorf("%w: move position is out of range", application.ErrIssueMoveConflict)
	}
	return position, nil
}

func wouldCreateParentCycleOnConnection(ctx context.Context, connection *sql.Conn, workspaceID, issueID, parentID string) (bool, error) {
	seen := map[string]struct{}{}
	for cursor := parentID; cursor != ""; {
		if cursor == issueID {
			return true, nil
		}
		if _, duplicate := seen[cursor]; duplicate {
			return true, nil
		}
		seen[cursor] = struct{}{}
		var next sql.NullString
		if err := connection.QueryRowContext(ctx, `SELECT parent_issue_id FROM workspace_issues WHERE workspace_id=? AND id=?`, workspaceID, cursor).Scan(&next); errors.Is(err, sql.ErrNoRows) {
			return false, application.ErrIssueRecordNotFound
		} else if err != nil {
			return false, fmt.Errorf("walk Workspace Issue move parents: %w", err)
		}
		if !next.Valid {
			return false, nil
		}
		cursor = next.String
	}
	return false, nil
}

func (r *issueRepository) WouldCreateParentCycle(ctx context.Context, workspaceID, issueID, parentID string) (bool, error) {
	seen := map[string]struct{}{}
	cursor := parentID
	for cursor != "" {
		if cursor == issueID {
			return true, nil
		}
		if _, duplicate := seen[cursor]; duplicate {
			return true, nil
		}
		seen[cursor] = struct{}{}
		var next sql.NullString
		err := r.db.QueryRowContext(ctx, `SELECT parent_issue_id FROM workspace_issues
			WHERE workspace_id = ? AND id = ?`, workspaceID, cursor).Scan(&next)
		if errors.Is(err, sql.ErrNoRows) {
			return false, application.ErrIssueRecordNotFound
		}
		if err != nil {
			return false, fmt.Errorf("walk Workspace Issue parents: %w", err)
		}
		if !next.Valid {
			return false, nil
		}
		cursor = next.String
	}
	return false, nil
}

type issueScanner interface{ Scan(...any) error }

func scanIssue(scanner issueScanner) (issueDomain.Issue, error) {
	var value issueDomain.Issue
	var description, assigneeType, assigneeID, parentID, projectID, startDate, dueDate sql.NullString
	var stage sql.NullInt64
	var createdAt, updatedAt, metadata, properties, assets string
	err := scanner.Scan(
		&value.ID, &value.WorkspaceID, &value.Number, &value.Identifier, &value.Title, &description,
		&value.Status, &value.Priority, &assigneeType, &assigneeID, &value.CreatorType, &value.CreatorID,
		&parentID, &projectID, &value.Position, &stage, &startDate, &dueDate,
		&createdAt, &updatedAt, &metadata, &properties, &assets,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return issueDomain.Issue{}, application.ErrIssueRecordNotFound
	}
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("select Workspace Issue: %w", err)
	}
	value.Description, value.AssigneeType, value.AssigneeID = pointer(description), pointer(assigneeType), pointer(assigneeID)
	value.ParentIssueID, value.ProjectID = pointer(parentID), pointer(projectID)
	value.StartDate, value.DueDate = pointer(startDate), pointer(dueDate)
	if stage.Valid {
		converted := int32(stage.Int64)
		value.Stage = &converted
	}
	if err := json.Unmarshal([]byte(metadata), &value.Metadata); err != nil {
		return issueDomain.Issue{}, fmt.Errorf("decode Issue metadata: %w", err)
	}
	if err := json.Unmarshal([]byte(properties), &value.Properties); err != nil {
		return issueDomain.Issue{}, fmt.Errorf("decode Issue properties: %w", err)
	}
	if err := json.Unmarshal([]byte(assets), &value.AssetIDs); err != nil {
		return issueDomain.Issue{}, fmt.Errorf("decode Issue assets: %w", err)
	}
	value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("parse Issue created_at: %w", err)
	}
	value.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("parse Issue updated_at: %w", err)
	}
	value, err = issueDomain.Rehydrate(value)
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("map Workspace Issue: %w", err)
	}
	return value, nil
}

func encodeIssueJSON(value issueDomain.Issue) (string, string, string, error) {
	metadata, err := json.Marshal(value.Metadata)
	if err != nil {
		return "", "", "", fmt.Errorf("encode Issue metadata: %w", err)
	}
	properties, err := json.Marshal(value.Properties)
	if err != nil {
		return "", "", "", fmt.Errorf("encode Issue properties: %w", err)
	}
	assets, err := json.Marshal(value.AssetIDs)
	if err != nil {
		return "", "", "", fmt.Errorf("encode Issue assets: %w", err)
	}
	return string(metadata), string(properties), string(assets), nil
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt32(value *int32) any {
	if value == nil {
		return nil
	}
	return *value
}

var _ application.IssueRepository = (*issueRepository)(nil)
